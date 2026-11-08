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
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss/tree"
)

type counts struct {
	dirs  int
	files int
	bytes int64
}

func skip(item os.DirEntry) bool {
	name := item.Name()
	if !opts.all && strings.HasPrefix(name, ".") {
		return true
	}
	if opts.dirs && !item.IsDir() {
		return true
	}
	for _, pattern := range opts.ignore {
		if match, _ := filepath.Match(pattern, name); match {
			return true
		}
	}
	return false
}

func order(items []os.DirEntry) {
	slices.SortStableFunc(items, func(a, b os.DirEntry) int {
		if a.IsDir() != b.IsDir() {
			if a.IsDir() {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})
}

func reason(err error) string {
	if errors.Is(err, fs.ErrPermission) {
		return "[permission denied]"
	}
	return "[unreadable]"
}

func leaf(path string, item os.DirEntry, sum *counts) string {
	sum.files++

	name := item.Name()
	info, err := item.Info()
	if err != nil {
		return name
	}

	label := name
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, _ := os.Readlink(path)
		label = symlink.Render(name) + faint.Render(" -> "+target)
	case info.Mode()&0o111 != 0:
		label = runnable.Render(name)
	}

	if opts.size && info.Mode().IsRegular() {
		sum.bytes += info.Size()
		label += faint.Render(" " + human(info.Size()))
	}

	return label
}

func branch(path string, level int, sum *counts) []any {
	items, err := os.ReadDir(path)
	if err != nil {
		return []any{fault.Render(reason(err))}
	}

	order(items)

	out := make([]any, 0, len(items))
	for _, item := range items {
		if skip(item) {
			continue
		}

		full := filepath.Join(path, item.Name())
		if !item.IsDir() {
			out = append(out, leaf(full, item, sum))
			continue
		}

		sum.dirs++
		node := dress(tree.Root(folder.Render(item.Name() + "/")))
		if opts.level == 0 || level < opts.level {
			if kids := branch(full, level+1, sum); len(kids) > 0 {
				node.Child(kids...)
			}
		}
		out = append(out, node)
	}

	return out
}
