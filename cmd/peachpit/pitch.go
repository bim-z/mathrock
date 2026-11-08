/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type options struct {
	Theme string
	Image string
	Page  int
	Wide  int
	Yes   bool
}

func flags(cmd *cobra.Command, opt *options) {
	cmd.Flags().StringVarP(&opt.Theme, "theme", "t", "", "Theme: auto, dark, light, dracula, pink, notty or ascii")
	cmd.Flags().StringVarP(&opt.Image, "image", "i", auto, "Image protocol: auto, kitty, sixel, iterm or off")
	cmd.Flags().IntVarP(&opt.Page, "page", "p", 1, "Slide to start from")
}

func present(path string, opt options) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return errors.New("show needs an interactive terminal, use print instead")
	}

	d, err := load(path)
	if err != nil {
		return err
	}

	if opt.Page < 1 || opt.Page > len(d.Pages) {
		return fmt.Errorf("slide %d does not exist, the deck has %d slides", opt.Page, len(d.Pages))
	}

	theme := opt.Theme
	if theme == "" {
		theme = d.Theme
	}

	style, err := resolve(theme)
	if err != nil {
		return err
	}

	proto, err := detect(opt.Image)
	if err != nil {
		return err
	}

	cx, cy := cells()

	start := model{
		deck:   d,
		path:   path,
		proto:  proto,
		engine: forge(style),
		index:  opt.Page - 1,
		cx:     cx,
		cy:     cy,
		cache:  map[string]frame{},
	}

	_, err = tea.NewProgram(start, tea.WithAltScreen()).Run()

	return err
}

func fresh() *cobra.Command {
	var opt options

	cmd := &cobra.Command{
		Use:   "new <file>",
		Short: "Create a starter deck",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if _, err := os.Stat(args[0]); err == nil && !opt.Yes {
				return errors.New("the file already exists, use --yes to overwrite it")
			}

			if err := os.WriteFile(args[0], []byte(starter), 0o644); err != nil {
				return err
			}

			fmt.Println("\n  " + green.Render("created") + " " + args[0] + faint.Render("  run 'pitch "+args[0]+"' to present it") + "\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&opt.Yes, "yes", false, "Overwrite an existing file")

	return cmd
}

func printing() *cobra.Command {
	var opt options

	cmd := &cobra.Command{
		Use:   "print <file> [number]",
		Short: "Print the rendered slides without the interface",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := load(args[0])
			if err != nil {
				return err
			}

			only := 0
			if len(args) == 2 {
				only, err = strconv.Atoi(args[1])
				if err != nil || only < 1 || only > len(d.Pages) {
					return fmt.Errorf("slide %q does not exist, the deck has %d slides", args[1], len(d.Pages))
				}
			}

			theme := opt.Theme
			if theme == "" {
				theme = d.Theme
			}

			style, err := resolve(theme)
			if err != nil {
				return err
			}

			if opt.Theme == "" && !term.IsTerminal(int(os.Stdout.Fd())) {
				style = "notty"
			}

			wide := opt.Wide
			if wide == 0 {
				wide = width()
			}

			return printout(d, style, wide, only)
		},
	}

	cmd.Flags().StringVarP(&opt.Theme, "theme", "t", "", "Theme: auto, dark, light, dracula, pink, notty or ascii")
	cmd.Flags().IntVarP(&opt.Wide, "width", "w", 0, "Output width, defaults to the terminal width")

	return cmd
}

func outline() *cobra.Command {
	return &cobra.Command{
		Use:   "ls <file>",
		Short: "List the slides of a deck",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := load(args[0])
			if err != nil {
				return err
			}

			listout(d)

			return nil
		},
	}
}

func info() *cobra.Command {
	var opt options

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show which image protocol this terminal supports",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return infoout(opt.Image)
		},
	}

	cmd.Flags().StringVarP(&opt.Image, "image", "i", auto, "Image protocol: auto, kitty, sixel, iterm or off")

	return cmd
}

func Peachpit() *cobra.Command {
	var opt options

	root := &cobra.Command{
		Use:          "peachpit [file]",
		Short:        "Present markdown in the terminal",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return present(args[0], opt)
		},
	}

	flags(root, &opt)

	root.AddCommand(show(), fresh(), printing(), outline(), info())

	return root
}
