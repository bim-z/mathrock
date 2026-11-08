/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package algernon

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	head = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dim  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	good = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warn = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

func clean(name string) (string, error) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "@")
	if name == "" || strings.ContainsAny(name, "/\\ \t") || strings.Contains(name, "..") {
		return "", errors.New("invalid username")
	}

	return name, nil
}

func safe(name string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"|?*`, r) {
			return '_'
		}

		return r
	}, name)
}

func human(size int) string {
	switch {
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/(1<<10))
	}

	return fmt.Sprintf("%d B", size)
}

func choose(names []string) ([]site, error) {
	if len(names) == 0 {
		return sites, nil
	}

	var list []site

	for _, name := range names {
		found := false

		for _, item := range sites {
			if strings.EqualFold(item.slug, strings.TrimSpace(name)) {
				list = append(list, item)
				found = true
			}
		}

		if !found {
			return nil, fmt.Errorf("unknown site %q, run: avatar sites", name)
		}
	}

	return list, nil
}

func get() *cobra.Command {
	var out string
	var only []string
	var workers int
	var every bool

	command := &cobra.Command{
		Use:   "get <username>",
		Short: "Download the avatar of a username from every supported site",
		Long:  "Download the avatar of a username from every supported site.\n\nInstagram, X, TikTok, Twitch, Threads, Snapchat, Spotify and Dribbble go through unavatar.io,\nwhich allows 50 requests per day per IP. Set UNAVATAR_KEY to use an API key.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := clean(args[0])
			if err != nil {
				return err
			}

			list, err := choose(only)
			if err != nil {
				return err
			}

			dir := filepath.Join(out, safe(user))
			total := len(list)
			live := term.IsTerminal(int(os.Stderr.Fd()))
			start := time.Now()

			tick := func(done int) {
				if live {
					fmt.Fprint(os.Stderr, "\r"+dim.Render(fmt.Sprintf("Checking %d/%d", done, total)))
				}
			}

			results := sweep(cmd.Context(), list, user, dir, workers, tick)

			if live {
				fmt.Fprint(os.Stderr, "\r\x1b[2K")
			}

			col := lipgloss.NewStyle().Width(14)
			saved, absent, failed := 0, 0, 0

			fmt.Println(head.Render(user), dim.Render(strconv.Itoa(total)+" sites"))
			fmt.Println()

			for _, item := range results {
				switch {
				case item.err == nil:
					saved++
					fmt.Println(" ", good.Render(col.Render(item.site.slug)), item.path, dim.Render(human(item.size)))
				case errors.Is(item.err, missing):
					absent++

					if every {
						fmt.Println(" ", dim.Render(col.Render(item.site.slug)), dim.Render("missing"))
					}
				default:
					failed++

					if every {
						fmt.Println(" ", warn.Render(col.Render(item.site.slug)), dim.Render(item.err.Error()))
					}
				}
			}

			if saved == 0 {
				_ = os.Remove(dir)
				return fmt.Errorf("no avatar found for %s", user)
			}

			fmt.Println()
			fmt.Println(head.Render(strconv.Itoa(saved)), "avatars saved to", dir)
			fmt.Println(dim.Render(fmt.Sprintf("%d missing, %d failed, %s", absent, failed, time.Since(start).Round(100*time.Millisecond))))

			if failed > 0 && !every {
				fmt.Println(dim.Render("Use --all to see why a site failed"))
			}

			return nil
		},
	}

	command.Flags().StringVarP(&out, "output", "o", "avatars", "Folder to save avatars in")
	command.Flags().StringSliceVarP(&only, "site", "s", nil, "Only these sites, comma separated")
	command.Flags().IntVarP(&workers, "workers", "w", 8, "Sites to check at the same time")
	command.Flags().BoolVarP(&every, "all", "a", false, "Show every site, including missing and failed")

	return command
}

func catalog() *cobra.Command {
	return &cobra.Command{
		Use:   "sites",
		Short: "List the supported sites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows := make([][]string, len(sites))

			for i, item := range sites {
				host := ""
				if parsed, err := url.Parse(item.link); err == nil {
					host = parsed.Host
				}

				rows[i] = []string{item.slug, item.name, host}
			}

			even := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
			odd := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)

			grid := table.New().
				Border(lipgloss.RoundedBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("Slug", "Site", "Source").
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

			fmt.Println(grid.String())
			fmt.Println(dim.Render(strconv.Itoa(len(sites)) + " sites"))

			return nil
		},
	}
}

// func main() {
// 	root := &cobra.Command{
// 		Use:          "avatar",
// 		Short:        "Download avatars from many sites with just a username",
// 		SilenceUsage: true,
// 	}

// 	root.AddCommand(get(), catalog())

// 	if err := root.Execute(); err != nil {
// 		os.Exit(1)
// 	}
// }
