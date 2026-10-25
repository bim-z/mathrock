package todo

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const (
	pending = iota
	begun
	closed
)

var bucket = []byte("todo")

type task struct {
	ID    uint64    `json:"id"`
	Text  string    `json:"text"`
	Board string    `json:"board"`
	State int       `json:"state"`
	Star  bool      `json:"star"`
	Born  time.Time `json:"born"`
}

type choose func([]task) ([]task, error)

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

	db, err := bbolt.Open(filepath.Join(dir, "todo.db"), 0o600, &bbolt.Options{Timeout: time.Second})
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

func load(b *bbolt.Bucket) ([]task, error) {
	var out []task

	err := b.ForEach(func(_, v []byte) error {
		var t task
		if err := json.Unmarshal(v, &t); err != nil {
			return fmt.Errorf("failed to decode task: %w", err)
		}
		out = append(out, t)
		return nil
	})

	return out, err
}

func save(b *bbolt.Bucket, t task) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to encode task: %w", err)
	}

	if err := b.Put(key(t.ID), raw); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	return nil
}

func order(in []task) []task {
	seen := map[string]bool{}
	var boards []string

	for _, t := range in {
		if !seen[t.Board] {
			seen[t.Board] = true
			boards = append(boards, t.Board)
		}
	}

	var out []task
	for _, name := range boards {
		for _, t := range in {
			if t.Board == name {
				out = append(out, t)
			}
		}
	}

	return out
}

func fetch() ([]task, error) {
	var out []task

	err := view(func(b *bbolt.Bucket) error {
		all, err := load(b)
		out = order(all)
		return err
	})

	return out, err
}

func numbers(args []string) choose {
	return func(all []task) ([]task, error) {
		var out []task
		seen := map[int]bool{}

		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil || num < 1 || num > len(all) {
				return nil, fmt.Errorf("no task numbered %q", arg)
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
	return func(all []task) ([]task, error) {
		for _, t := range all {
			if t.ID == id {
				return []task{t}, nil
			}
		}
		return nil, fmt.Errorf("task no longer exists")
	}
}

func change(sel choose, fn func(*task)) ([]task, error) {
	var out []task

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err := sel(order(all))
		if err != nil {
			return err
		}

		for _, t := range hit {
			fn(&t)
			if err := save(b, t); err != nil {
				return err
			}
			out = append(out, t)
		}

		return nil
	})

	return out, err
}

func remove(sel choose) ([]task, error) {
	var hit []task

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err = sel(order(all))
		if err != nil {
			return err
		}

		for _, t := range hit {
			if err := b.Delete(key(t.ID)); err != nil {
				return fmt.Errorf("failed to delete task: %w", err)
			}
		}

		return nil
	})

	return hit, err
}

func insert(text, board string) (task, int, error) {
	var fresh task
	var num int

	err := update(func(b *bbolt.Bucket) error {
		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("failed to allocate id: %w", err)
		}

		fresh = task{ID: id, Text: text, Board: board, Born: time.Now()}
		if err := save(b, fresh); err != nil {
			return err
		}

		all, err := load(b)
		if err != nil {
			return err
		}

		for i, t := range order(all) {
			if t.ID == id {
				num = i + 1
			}
		}

		return nil
	})

	return fresh, num, err
}

func finish(t *task) {
	if t.State == closed {
		t.State = pending
		return
	}
	t.State = closed
}

func progress(t *task) {
	if t.State == begun {
		t.State = pending
		return
	}
	t.State = begun
}

func pin(t *task) {
	t.Star = !t.Star
}

func echo(hit []task) {
	for _, t := range hit {
		fmt.Println("  " + item(t))
	}
}

func Todo() *cobra.Command {
	root := &cobra.Command{
		Use:          "todo",
		Short:        "Keep track of tasks across boards",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return paint("")
		},
	}

	root.AddCommand(add(), toggle("done", "Toggle tasks as done", finish), toggle("begin", "Toggle tasks as in progress", progress), toggle("star", "Toggle the star of tasks", pin), edit(), move(), rm(), purge(), ls(), tui())

	return root
}

func toggle(use, short string, fn func(*task)) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <number>...",
		Short: short,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := change(numbers(args), fn)
			if err != nil {
				return err
			}

			echo(hit)
			return nil
		},
	}
}

func add() *cobra.Command {
	var board string

	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			fresh, num, err := insert(strings.Join(args, " "), clean(board))
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  added ") + pink.Render(strconv.Itoa(num)) + " " + fresh.Text)
			return nil
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "general", "board that holds the task")

	return cmd
}

func edit() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <number> <text>",
		Short: "Change the text of a task",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			text := strings.Join(args[1:], " ")

			hit, err := change(numbers(args[:1]), func(t *task) { t.Text = text })
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
		Short: "Move tasks to another board",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			board := clean(args[len(args)-1])

			hit, err := change(numbers(args[:len(args)-1]), func(t *task) { t.Board = board })
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
		Short: "Remove tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := remove(numbers(args))
			if err != nil {
				return err
			}

			for _, t := range hit {
				fmt.Println(faint.Render("  removed ") + t.Text)
			}
			return nil
		},
	}
}

func purge() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all finished tasks",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			var count int

			err := update(func(b *bbolt.Bucket) error {
				all, err := load(b)
				if err != nil {
					return err
				}

				for _, t := range all {
					if t.State != closed {
						continue
					}
					if err := b.Delete(key(t.ID)); err != nil {
						return fmt.Errorf("failed to delete task: %w", err)
					}
					count++
				}

				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  cleared ") + strconv.Itoa(count) + faint.Render(" finished tasks"))
			return nil
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [board]",
		Short: "List tasks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				return paint(args[0])
			}
			return paint("")
		},
	}
}
