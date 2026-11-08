/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package victor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	accent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	label  = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Width(10)
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	fail   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	frame  = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 2)
	paint = map[rune]lipgloss.Style{
		'r': lipgloss.NewStyle().Foreground(lipgloss.Color("221")),
		'w': lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
		'x': lipgloss.NewStyle().Foreground(lipgloss.Color("114")),
	}
)

func row(key, val, note string) string {
	line := label.Render(key) + val
	if note != "" {
		line += "  " + dim.Render(note)
	}
	return line
}

func perms(mode os.FileMode) string {
	var buf strings.Builder
	for pos, char := range "rwxrwxrwx" {
		if mode.Perm()&(1<<(8-uint(pos))) != 0 {
			buf.WriteString(paint[char].Render(string(char)))
		} else {
			buf.WriteString(dim.Render("-"))
		}
	}
	return buf.String()
}

func render(data info) string {
	rows := []string{accent.Render(data.Path) + "  " + dim.Render(data.Kind), ""}
	if data.Target != "" {
		rows = append(rows, row("target", data.Target, ""), "")
	}
	rows = append(rows, row("size", size(data.Size), fmt.Sprintf("%d bytes", data.Size)))
	if data.Attrs != "" {
		rows = append(rows, row("attrs", data.Attrs, ""))
	} else {
		rows = append(rows, row("mode", perms(data.Mode), octal(data.Mode)))
	}
	if data.Owner != "" {
		rows = append(rows, row("owner", data.Owner, ""), row("group", data.Group, ""))
	}
	rows = append(rows, "")
	stamps := []struct {
		key  string
		when time.Time
	}{
		{"created", data.Created},
		{"modified", data.Modified},
		{"accessed", data.Accessed},
		{"changed", data.Changed},
	}
	for _, item := range stamps {
		if !item.when.IsZero() {
			rows = append(rows, row(item.key, stamp(item.when), ago(item.when)))
		}
	}
	counts := []struct {
		key string
		val uint64
	}{
		{"inode", data.Inode},
		{"links", data.Links},
		{"blocks", uint64(data.Blocks)},
		{"device", data.Device},
	}
	rows = append(rows, "")
	for _, item := range counts {
		if item.val != 0 {
			rows = append(rows, row(item.key, fmt.Sprintf("%d", item.val), ""))
		}
	}
	return frame.Render(strings.TrimRight(strings.Join(rows, "\n"), "\n"))
}
