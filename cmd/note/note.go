package note

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var bucket = []byte("note")

type note struct {
	ID    uint64    `json:"id"`
	Body  string    `json:"body"`
	Board string    `json:"board"`
	Star  bool      `json:"star"`
	Born  time.Time `json:"born"`
}

type choose func([]note) ([]note, error)

func key(id uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	return buf
}

func clean(board string) string {
	board = strings.TrimSpace(board)
	if board == "" {
		return "general"
	}
	return board
}

func open() (*bbolt.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bbolt.Open(filepath.Join(dir, "note.db"), 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	return db, nil
}

func update(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func view(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func load(b *bbolt.Bucket) ([]note, error) {
	var out []note

	err := b.ForEach(func(_, v []byte) error {
		var n note
		if err := json.Unmarshal(v, &n); err != nil {
			return fmt.Errorf("failed to decode note: %w", err)
		}
		out = append(out, n)
		return nil
	})

	return out, err
}

func save(b *bbolt.Bucket, n note) error {
	raw, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to encode note: %w", err)
	}

	if err := b.Put(key(n.ID), raw); err != nil {
		return fmt.Errorf("failed to save note: %w", err)
	}

	return nil
}

func order(in []note) []note {
	seen := map[string]bool{}
	var boards []string

	for _, n := range in {
		if !seen[n.Board] {
			seen[n.Board] = true
			boards = append(boards, n.Board)
		}
	}

	var out []note
	for _, name := range boards {
		for _, n := range in {
			if n.Board == name {
				out = append(out, n)
			}
		}
	}

	return out
}

func fetch() ([]note, error) {
	var out []note

	err := view(func(b *bbolt.Bucket) error {
		all, err := load(b)
		out = order(all)
		return err
	})

	return out, err
}

func numbers(args []string) choose {
	return func(all []note) ([]note, error) {
		var out []note
		seen := map[int]bool{}

		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil || num < 1 || num > len(all) {
				return nil, fmt.Errorf("no note numbered %q", arg)
			}
			if seen[num] {
				continue
			}
			seen[num] = true
			out = append(out, all[num-1])
		}

		return out, nil
	}
}

func ident(id uint64) choose {
	return func(all []note) ([]note, error) {
		for _, n := range all {
			if n.ID == id {
				return []note{n}, nil
			}
		}
		return nil, errors.New("note no longer exists")
	}
}

func change(sel choose, fn func(*note)) ([]note, error) {
	var out []note

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err := sel(order(all))
		if err != nil {
			return err
		}

		for _, n := range hit {
			fn(&n)
			if err := save(b, n); err != nil {
				return err
			}
			out = append(out, n)
		}

		return nil
	})

	return out, err
}

func remove(sel choose) ([]note, error) {
	var hit []note

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err = sel(order(all))
		if err != nil {
			return err
		}

		for _, n := range hit {
			if err := b.Delete(key(n.ID)); err != nil {
				return fmt.Errorf("failed to delete note: %w", err)
			}
		}

		return nil
	})

	return hit, err
}

func insert(body, board string) (note, int, error) {
	var fresh note
	var num int

	err := update(func(b *bbolt.Bucket) error {
		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("failed to allocate id: %w", err)
		}

		fresh = note{ID: id, Body: body, Board: board, Born: time.Now()}
		if err := save(b, fresh); err != nil {
			return err
		}

		all, err := load(b)
		if err != nil {
			return err
		}

		for i, n := range order(all) {
			if n.ID == id {
				num = i + 1
			}
		}

		return nil
	})

	return fresh, num, err
}

func pin(n *note) {
	n.Star = !n.Star
}

func piped() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice == 0
}

func editor() []string {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if parts := strings.Fields(os.Getenv(name)); len(parts) > 0 {
			return parts
		}
	}
	return []string{"vi"}
}

func prepare(initial string) (string, *exec.Cmd, error) {
	file, err := os.CreateTemp("", "note-*.md")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := file.WriteString(initial); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	parts := editor()

	return file.Name(), exec.Command(parts[0], append(parts[1:], file.Name())...), nil
}

func collect(path string) (string, error) {
	defer os.Remove(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read temp file: %w", err)
	}

	return strings.TrimSpace(string(raw)), nil
}

func compose(initial string) (string, error) {
	path, cmd, err := prepare(initial)
	if err != nil {
		return "", err
	}

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("editor failed: %w", err)
	}

	return collect(path)
}

func gather(args []string, initial string) (string, error) {
	var text string
	var err error

	switch {
	case len(args) > 0:
		text = strings.Join(args, " ")
	case piped():
		var raw []byte
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		text = string(raw)
	default:
		text, err = compose(initial)
		if err != nil {
			return "", err
		}
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("note is empty")
	}

	return text, nil
}

func echo(hit []note) {
	for _, n := range hit {
		fmt.Println("  " + item(n, fit(cols())))
	}
}

func Note() *cobra.Command {
	root := &cobra.Command{
		Use:          "note",
		Short:        "Keep quick notes across boards",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return paint(nil, "")
		},
	}

	root.AddCommand(add(), show(), edit(), star(), move(), rm(), find(), ls(), tui())

	return root
}

func add() *cobra.Command {
	var board string

	cmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a note",
		RunE: func(_ *cobra.Command, args []string) error {
			text, err := gather(args, "")
			if err != nil {
				return err
			}

			fresh, num, err := insert(text, clean(board))
			if err != nil {
				return err
			}

			line, _ := first(fresh.Body)
			fmt.Println(faint.Render("  added ") + pink.Render(strconv.Itoa(num)) + " " + clip(line, fit(cols())))
			return nil
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "general", "board that holds the note")

	return cmd
}

func show() *cobra.Command {
	return &cobra.Command{
		Use:   "show <number>",
		Short: "Display a note in full",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			all, err := fetch()
			if err != nil {
				return err
			}

			hit, err := numbers(args)(all)
			if err != nil {
				return err
			}

			num, _ := strconv.Atoi(args[0])
			fmt.Println("\n" + detail(hit[0], num, cols()) + "\n")
			return nil
		},
	}
}

func edit() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <number> [text]",
		Short: "Change the text of a note",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			all, err := fetch()
			if err != nil {
				return err
			}

			cur, err := numbers(args[:1])(all)
			if err != nil {
				return err
			}

			text, err := gather(args[1:], cur[0].Body)
			if err != nil {
				return err
			}

			hit, err := change(ident(cur[0].ID), func(n *note) { n.Body = text })
			if err != nil {
				return err
			}

			echo(hit)
			return nil
		},
	}
}

func star() *cobra.Command {
	return &cobra.Command{
		Use:   "star <number>...",
		Short: "Toggle the star of notes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := change(numbers(args), pin)
			if err != nil {
				return err
			}

			echo(hit)
			return nil
		},
	}
}

func move() *cobra.Command {
	return &cobra.Command{
		Use:   "move <number>... <board>",
		Short: "Move notes to another board",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			board := clean(args[len(args)-1])

			hit, err := change(numbers(args[:len(args)-1]), func(n *note) { n.Board = board })
			if err != nil {
				return err
			}

			echo(hit)
			return nil
		},
	}
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <number>...",
		Short: "Remove notes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := remove(numbers(args))
			if err != nil {
				return err
			}

			for _, n := range hit {
				line, _ := first(n.Body)
				fmt.Println(faint.Render("  removed ") + clip(line, fit(cols())))
			}
			return nil
		},
	}
}

func find() *cobra.Command {
	return &cobra.Command{
		Use:   "find <query>",
		Short: "Search notes by text or board",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := strings.ToLower(strings.Join(args, " "))

			return paint(func(n note) bool {
				return strings.Contains(strings.ToLower(n.Body), query) || strings.Contains(strings.ToLower(n.Board), query)
			}, "no notes match "+query)
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [board]",
		Short: "List notes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return paint(nil, "")
			}

			board := args[0]
			return paint(func(n note) bool { return n.Board == board }, "no notes on board "+board)
		},
	}
}
