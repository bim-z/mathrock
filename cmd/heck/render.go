/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package heck

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type entry struct {
	name string
	data stats
}

type column struct {
	head string
	show bool
	text func(stats) string
}

func columns() []column {
	picked := opts.lines || opts.words || opts.chars || opts.bytes || opts.longest

	return []column{
		{"lines", opts.lines || !picked, func(s stats) string { return strconv.Itoa(s.lines) }},
		{"words", opts.words || !picked, func(s stats) string { return strconv.Itoa(s.words) }},
		{"chars", opts.chars, func(s stats) string { return strconv.Itoa(s.chars) }},
		{"bytes", opts.bytes || !picked, func(s stats) string { return strconv.Itoa(s.bytes) }},
		{"longest", opts.longest, func(s stats) string { return strconv.Itoa(s.longest) }},
		{"time", opts.clock, func(s stats) string { return format(reading(s.words)) }},
	}
}

func reading(words int) time.Duration {
	return time.Duration(float64(words) / float64(opts.speed) * float64(time.Minute))
}

func format(span time.Duration) string {
	if span <= 0 {
		return "0s"
	}
	if span < time.Second {
		return "<1s"
	}

	span = span.Round(time.Second)
	hours := int(span.Hours())
	mins := int(span.Minutes()) % 60
	secs := int(span.Seconds()) % 60

	switch {
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func cells(name string, data stats, shown []column) []string {
	out := []string{name}
	for _, col := range shown {
		out = append(out, col.text(data))
	}
	return out
}

func render(rows []entry, total bool) string {
	shown := make([]column, 0, 6)
	for _, col := range columns() {
		if col.show {
			shown = append(shown, col)
		}
	}

	heads := []string{"file"}
	for _, col := range shown {
		heads = append(heads, col.head)
	}

	grid := make([][]string, 0, len(rows)+1)
	sum := stats{}
	for _, row := range rows {
		name := row.name
		if name == "-" {
			name = "stdin"
		}
		grid = append(grid, cells(name, row.data, shown))
		sum = sum.merge(row.data)
	}
	if total {
		grid = append(grid, cells("total", sum, shown))
	}

	last := len(grid) - 1
	base := lipgloss.NewStyle().Padding(0, 1)
	align := func(style lipgloss.Style, col int) lipgloss.Style {
		if col == 0 {
			return style.Align(lipgloss.Left)
		}
		return style.Align(lipgloss.Right)
	}

	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers(heads...).
		Rows(grid...).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return align(base.Bold(true).Foreground(lipgloss.Color("212")), col)
			case total && row == last:
				return align(base.Bold(true).Foreground(lipgloss.Color("42")), col)
			}
			return align(base, col)
		}).
		String()
}
