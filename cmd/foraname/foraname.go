package foraname

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var bucket = []byte("foraname")

var (
	title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#D1D1D1")).
		Padding(1, 2).
		Width(60)

	head = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	cell = lipgloss.NewStyle().Padding(0, 1)

	line = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	muted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	accent = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
)

func open() (*bbolt.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bbolt.Open(filepath.Join(dir, "foraname.db"), 0o600, &bbolt.Options{Timeout: time.Second})
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

func update(fn func(*bbolt.Tx) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(fn)
}

func view(fn func(*bbolt.Tx) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(fn)
}

func done(verb, name string) {
	fmt.Println(muted.Render(verb+" ") + accent.Render(name))
}

func Foraname() *cobra.Command {
	root := &cobra.Command{
		Use:          "foraname",
		Short:        "Keep name ideas and the projects they are meant for",
		SilenceUsage: true,
	}

	root.AddCommand(add(), rm(), rename(), edit(), drop(), show(), ls())

	return root
}

func add() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <desc>",
		Short: "Add a name idea",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			name, desc := args[0], args[1]

			err := update(func(tx *bbolt.Tx) error {
				b := tx.Bucket(bucket)
				if b.Get([]byte(name)) != nil {
					return fmt.Errorf("name %q already exists", name)
				}
				if err := b.Put([]byte(name), []byte(desc)); err != nil {
					return fmt.Errorf("failed to save name: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			done("added", name)
			return nil
		},
	}
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a name idea",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			err := update(func(tx *bbolt.Tx) error {
				b := tx.Bucket(bucket)
				if b.Get([]byte(name)) == nil {
					return fmt.Errorf("name %q not found", name)
				}
				if err := b.Delete([]byte(name)); err != nil {
					return fmt.Errorf("failed to delete name: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			done("removed", name)
			return nil
		},
	}
}

func rename() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <from> <to>",
		Short: "Rename a name idea",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			from, to := args[0], args[1]

			err := update(func(tx *bbolt.Tx) error {
				b := tx.Bucket(bucket)
				raw := b.Get([]byte(from))
				if raw == nil {
					return fmt.Errorf("name %q not found", from)
				}
				if b.Get([]byte(to)) != nil {
					return fmt.Errorf("name %q already exists", to)
				}
				if err := b.Put([]byte(to), bytes.Clone(raw)); err != nil {
					return fmt.Errorf("failed to save name: %w", err)
				}
				if err := b.Delete([]byte(from)); err != nil {
					return fmt.Errorf("failed to remove old name: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println(muted.Render("renamed ") + accent.Render(from) + muted.Render(" to ") + accent.Render(to))
			return nil
		},
	}
}

func edit() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name> <desc>",
		Short: "Edit the description of a name idea",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			name, desc := args[0], args[1]

			err := update(func(tx *bbolt.Tx) error {
				b := tx.Bucket(bucket)
				if b.Get([]byte(name)) == nil {
					return fmt.Errorf("name %q not found", name)
				}
				if err := b.Put([]byte(name), []byte(desc)); err != nil {
					return fmt.Errorf("failed to update description: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			done("updated", name)
			return nil
		},
	}
}

func drop() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "drop",
		Short: "Delete every name idea",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !yes {
				return errors.New("dropping everything requires the --yes flag")
			}

			err := update(func(tx *bbolt.Tx) error {
				if err := tx.DeleteBucket(bucket); err != nil {
					return fmt.Errorf("failed to drop bucket: %w", err)
				}
				if _, err := tx.CreateBucket(bucket); err != nil {
					return fmt.Errorf("failed to recreate bucket: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			done("dropped", "all names")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the deletion")

	return cmd
}

func show() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Display a name idea",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			var desc string
			err := view(func(tx *bbolt.Tx) error {
				raw := tx.Bucket(bucket).Get([]byte(name))
				if raw == nil {
					return fmt.Errorf("name %q not found", name)
				}
				desc = string(raw)
				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println("\n" + title.Render(name) + "\n")
			fmt.Println(box.Render(desc) + "\n")
			return nil
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all name ideas",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			var rows [][]string

			err := view(func(tx *bbolt.Tx) error {
				return tx.Bucket(bucket).ForEach(func(k, v []byte) error {
					rows = append(rows, []string{strconv.Itoa(len(rows) + 1), string(k), string(v)})
					return nil
				})
			})
			if err != nil {
				return fmt.Errorf("failed to read names: %w", err)
			}

			if len(rows) == 0 {
				fmt.Println(muted.Render("no names yet"))
				return nil
			}

			grid := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(line).
				Headers("No", "Name", "Description").
				Rows(rows...).
				StyleFunc(func(row, _ int) lipgloss.Style {
					if row == table.HeaderRow {
						return head
					}
					return cell
				})

			fmt.Println(grid.Render())
			fmt.Println("\n" + muted.Render("total ") + accent.Render(strconv.Itoa(len(rows))) + "\n")
			return nil
		},
	}
}
