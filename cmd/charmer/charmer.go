/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package charmer

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func number(arg string) (int, error) {
	seq, err := strconv.Atoi(arg)
	if err != nil || seq < 1 {
		return 0, errors.New("please provide a valid positive number")
	}

	return seq, nil
}

func width() int {
	cols, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}

	return cols
}

func add() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <url> [description...]",
		Short: "Add a new link",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			link := Link{
				Name:        args[0],
				Target:      args[1],
				Description: strings.Join(args[2:], " "),
			}

			if err := insert(link); err != nil {
				return err
			}

			fmt.Println("Added", head.Render(link.Name))

			return nil
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all saved links",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := fetch()
			if err != nil {
				return err
			}

			if len(all) == 0 {
				fmt.Println(dim.Render("No links yet. Add one with: piw add <name> <url>"))
				return nil
			}

			rows := make([][]string, len(all))
			for i, link := range all {
				rows[i] = []string{strconv.Itoa(i + 1), link.Name, link.Target, link.Description}
			}

			even := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
			odd := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)

			grid := table.New().
				Border(lipgloss.RoundedBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("No", "Name", "URL", "Description").
				Rows(rows...).
				StyleFunc(func(row, col int) lipgloss.Style {
					switch {
					case row == table.HeaderRow:
						return head.Padding(0, 1)
					case row%2 == 0:
						return even
					default:
						return odd
					}
				})

			out := grid.String()
			if cols := width(); cols > 0 && lipgloss.Width(out) > cols {
				out = grid.Width(cols).String()
			}

			fmt.Println(out)

			return nil
		},
	}
}

func show() *cobra.Command {
	return &cobra.Command{
		Use:     "show <number>",
		Aliases: []string{"open"},
		Short:   "Open a link in your browser by its number",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			seq, err := number(args[0])
			if err != nil {
				return err
			}

			link, err := find(seq)
			if err != nil {
				return err
			}

			fmt.Println("Opening", head.Render(link.Name), dim.Render(link.Target))

			return launch(link.Target)
		},
	}
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <number>",
		Short: "Remove a link by its number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			seq, err := number(args[0])
			if err != nil {
				return err
			}

			link, err := find(seq)
			if err != nil {
				return err
			}

			if err := remove(link.Key); err != nil {
				return err
			}

			fmt.Println("Removed", head.Render(link.Name))

			return nil
		},
	}
}

func tui() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive interface",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
}

// func main() {
// 	root := &cobra.Command{
// 		Use:               "piw",
// 		Short:             "Save links and open them from your terminal",
// 		SilenceUsage:      true,
// 		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return connect() },
// 	}

// 	root.AddCommand(add(), ls(), show(), rm(), tui())

// 	err := root.Execute()
// 	if db != nil {
// 		_ = db.Close()
// 	}

// 	if err != nil {
// 		os.Exit(1)
// 	}
// }
