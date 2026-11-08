/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package chmod

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type result struct {
	Path  string
	Dir   bool
	Old   uint32
	New   uint32
	Mode  fs.FileMode
	Fault error
}

type options struct {
	Recurse bool
	Dry     bool
}

func guard() error {
	if runtime.GOOS == "windows" {
		return errors.New("file permissions are not supported on Windows")
	}
	return nil
}

func entry(path string, info fs.FileInfo) result {
	return result{Path: path, Dir: info.IsDir(), Old: unix(info.Mode()), Mode: info.Mode()}
}

func gather(paths []string, recurse bool) []result {
	var out []result

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			out = append(out, result{Path: path, Fault: err})
			continue
		}

		out = append(out, entry(path, info))

		if !recurse || !info.IsDir() {
			continue
		}

		root := path

		_ = filepath.WalkDir(root, func(here string, item fs.DirEntry, err error) error {
			if err != nil {
				out = append(out, result{Path: here, Fault: err})
				return nil
			}

			if here == root || item.Type()&fs.ModeSymlink != 0 {
				return nil
			}

			stat, err := item.Info()
			if err != nil {
				out = append(out, result{Path: here, Fault: err})
				return nil
			}

			out = append(out, entry(here, stat))

			return nil
		})
	}

	return out
}

func apply(items []result, edit func(uint32, bool) (uint32, error), dry bool) []result {
	for idx := range items {
		item := &items[idx]
		if item.Fault != nil {
			continue
		}

		next, err := edit(item.Old, item.Dir)
		if err != nil {
			item.Fault = err
			continue
		}

		item.New = next
	}

	if dry {
		return items
	}

	change := func(idx int) {
		item := &items[idx]
		if item.Fault != nil || item.New == item.Old {
			return
		}

		if err := os.Chmod(item.Path, native(item.New, item.Mode)); err != nil {
			item.Fault = err
		}
	}

	closing := func(item result) bool {
		return item.Dir && item.New&0o100 == 0
	}

	for idx := range items {
		if !closing(items[idx]) {
			change(idx)
		}
	}

	for idx := len(items) - 1; idx >= 0; idx-- {
		if closing(items[idx]) {
			change(idx)
		}
	}

	return items
}

func change(spec string, paths []string, opt options) error {
	if _, err := resolve(spec, 0, false); err != nil {
		return err
	}

	edit := func(old uint32, dir bool) (uint32, error) {
		return resolve(spec, old, dir)
	}

	return report(apply(gather(paths, opt.Recurse), edit, opt.Dry), opt.Dry)
}

func edit(paths []string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return errors.New("the interactive mode needs a terminal, pass a mode such as 755 or u+x instead")
	}

	var first fs.FileInfo
	var deep bool

	for idx, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if len(paths) == 1 && valid(path) {
				return fmt.Errorf("%s looks like a mode, add a file name after it", path)
			}
			return fmt.Errorf("%s: %s", path, simple(err))
		}

		if idx == 0 {
			first = info
		}
		if info.IsDir() {
			deep = true
		}
	}

	start := model{
		paths: paths,
		bits:  unix(first.Mode()),
		orig:  unix(first.Mode()),
		dir:   first.IsDir(),
		deep:  false,
		can:   deep,
	}

	final, err := tea.NewProgram(start, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}

	end, ok := final.(model)
	if !ok || !end.done {
		fmt.Println("\n  " + faint.Render("nothing changed") + "\n")
		return nil
	}

	set := func(old uint32, _ bool) (uint32, error) {
		return old&^every | end.bits&every, nil
	}

	return report(apply(gather(paths, end.deep), set, false), false)
}

func explaining() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <mode|file>...",
		Short: "Explain a mode or the permissions of a file in plain English",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := guard(); err != nil {
				return err
			}

			var bad int

			for _, arg := range args {
				if info, err := os.Stat(arg); err == nil {
					explain(arg, info.IsDir(), unix(info.Mode()))
					continue
				}

				bits, err := resolve(arg, 0, false)
				if err != nil || !valid(arg) {
					bad++
					fmt.Println("\n  " + red.Render(arg+": no such file, and not a valid mode") + "\n")
					continue
				}

				explain("", false, bits)
			}

			if bad > 0 {
				return fmt.Errorf("%d of %d arguments could not be explained", bad, len(args))
			}

			return nil
		},
	}
}

func editing() *cobra.Command {
	return &cobra.Command{
		Use:   "tui <file>...",
		Short: "Open the interactive permission editor",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := guard(); err != nil {
				return err
			}
			return edit(args)
		},
	}
}

func Chmod() *cobra.Command {
	var opt options

	root := &cobra.Command{
		Use:          "chmod [mode] <file>...",
		Short:        "Change file permissions without the guesswork",
		SilenceUsage: true,
		Args:         cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := guard(); err != nil {
				return err
			}

			if len(args) == 0 {
				return cmd.Help()
			}

			if len(args) > 1 && valid(args[0]) {
				return change(args[0], args[1:], opt)
			}

			return edit(args)
		},
	}

	root.Flags().BoolVarP(&opt.Recurse, "recursive", "R", false, "Apply to folders and everything inside them")
	root.Flags().BoolVarP(&opt.Dry, "dry-run", "n", false, "Show what would change without changing anything")

	root.AddCommand(explaining(), editing())

	return root
}
