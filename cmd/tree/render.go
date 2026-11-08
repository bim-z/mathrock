/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

var (
	rooted   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	folder   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	runnable = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	symlink  = lipgloss.NewStyle().Foreground(lipgloss.Color("37"))
	faint    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	fault    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	branches = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func dress(node *tree.Tree) *tree.Tree {
	return node.
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(branches.PaddingRight(1))
}

func human(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func plural(count int, one, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", count, many)
}

func footer(sum *counts) string {
	text := plural(sum.dirs, "directory", "directories")
	if !opts.dirs {
		text += ", " + plural(sum.files, "file", "files")
	}
	if opts.size && !opts.dirs {
		text += ", " + human(sum.bytes)
	}
	return faint.Render(text)
}

func render(path string) string {
	sum := &counts{}
	node := dress(tree.Root(rooted.Render(path)))
	if kids := branch(path, 1, sum); len(kids) > 0 {
		node.Child(kids...)
	}
	return node.String() + "\n\n" + footer(sum)
}
