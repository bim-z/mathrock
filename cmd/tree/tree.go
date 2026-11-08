/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var opts struct {
	all    bool
	dirs   bool
	size   bool
	level  int
	ignore []string
}

var root = &cobra.Command{
	Use:   "tree [directory]",
	Short: "Display a directory as a beautiful tree",
	Args:  cobra.MaximumNArgs(1),
	RunE:  run,
}

func init() {
	flags := root.Flags()
	flags.BoolVarP(&opts.all, "all", "a", false, "Include hidden files and directories")
	flags.BoolVarP(&opts.dirs, "dirs", "d", false, "List directories only")
	flags.BoolVarP(&opts.size, "size", "s", false, "Print the size of each file")
	flags.IntVarP(&opts.level, "level", "L", 0, "Maximum depth of the tree, 0 for unlimited")
	flags.StringSliceVarP(&opts.ignore, "ignore", "I", nil, "Skip names matching a glob pattern")
}

func execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if opts.level < 0 {
		return errors.New("level must not be negative")
	}

	cmd.SilenceUsage = true

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory: " + path)
	}

	fmt.Println(render(path))
	return nil
}
