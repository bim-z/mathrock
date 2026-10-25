package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var db *bbolt.DB

const bucket = "links"

type Link struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

func initDB() error {
	var err error
	db, err = bbolt.Open("piw.db", 0600, nil)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // Linux, Termux, FreeBSD, etc.
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func main() {
	if err := initDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var rootCmd = &cobra.Command{
		Use:   "piw",
		Short: "piw is a CLI tool to manage and open your saved links",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	var addCmd = &cobra.Command{
		Use:   "add [name] [target_url] [description...]",
		Short: "Add a new link",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			target := args[1]
			desc := "-"
			if len(args) > 2 {
				desc = strings.Join(args[2:], " ")
			}

			return db.Update(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(bucketName))

				id, _ := b.NextSequence()
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, id)

				link := Link{
					Name:        name,
					Target:      target,
					Description: desc,
				}

				data, err := json.Marshal(link)
				if err != nil {
					return err
				}

				fmt.Printf("Successfully added link: %s\n", name)
				return b.Put(key, data)
			})
		},
	}

	var lsCmd = &cobra.Command{
		Use:   "ls",
		Short: "List all saved links",
		RunE: func(cmd *cobra.Command, args []string) error {
			return db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(bucketName))
				c := b.Cursor()

				var rows [][]string
				seq := 1

				for k, v := c.First(); k != nil; k, v = c.Next() {
					var link Link
					if err := json.Unmarshal(v, &link); err != nil {
						return err
					}
					rows = append(rows, []string{
						strconv.Itoa(seq),
						link.Name,
						link.Target,
						link.Description,
					})
					seq++
				}

				if len(rows) == 0 {
					fmt.Println("No links found. Use 'piw add' to store one.")
					return nil
				}

				// Styling untuk tabel Lip Gloss
				headerStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("212")). // Warna pink/ungu muda
					Bold(true).
					Align(lipgloss.Center)

				evenRowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Abu-abu
				oddRowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))  // Putih pucat

				t := table.New().
					Border(lipgloss.RoundedBorder()).
					BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))). // Warna border ungu
					Headers("No", "Name", "Target", "Description").
					Rows(rows...).
					StyleFunc(func(row, col int) lipgloss.Style {
						// row 0 adalah header di dalam context StyleFunc table lipgloss terbaru
						if row == 0 {
							return headerStyle
						}
						// Selang-seling warna baris
						if row%2 == 0 {
							return evenRowStyle
						}
						return oddRowStyle
					})

				fmt.Println(t.Render())
				return nil
			})
		},
	}

	var showCmd = &cobra.Command{
		Use:   "show [number]",
		Short: "Open a link in your default browser by its sequence number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetSeq, err := strconv.Atoi(args[0])
			if err != nil || targetSeq < 1 {
				return fmt.Errorf("please provide a valid positive sequence number")
			}

			var targetURL string
			err = db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(bucketName))
				c := b.Cursor()

				seq := 1
				for k, v := c.First(); k != nil; k, v = c.Next() {
					if seq == targetSeq {
						var link Link
						if err := json.Unmarshal(v, &link); err != nil {
							return err
						}
						targetURL = link.Target
						return nil
					}
					seq++
				}
				return fmt.Errorf("link number %d not found", targetSeq)
			})

			if err != nil {
				return err
			}

			fmt.Printf("Opening %s in browser...\n", targetURL)
			return openBrowser(targetURL)
		},
	}

	var rmCmd = &cobra.Command{
		Use:   "rm [number]",
		Short: "Remove a link by its sequence number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetSeq, err := strconv.Atoi(args[0])
			if err != nil || targetSeq < 1 {
				return fmt.Errorf("please provide a valid positive sequence number")
			}

			return db.Update(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(bucketName))
				c := b.Cursor()

				seq := 1
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					if seq == targetSeq {
						if err := b.Delete(k); err != nil {
							return err
						}
						fmt.Printf("Successfully removed link number %d.\n", targetSeq)
						return nil
					}
					seq++
				}
				return fmt.Errorf("link number %d not found", targetSeq)
			})
		},
	}

	rootCmd.AddCommand(addCmd, lsCmd, showCmd, rmCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
