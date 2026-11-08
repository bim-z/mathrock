/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package track

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	title  = lipgloss.NewStyle().Bold(true).Underline(true)
	strong = lipgloss.NewStyle().Bold(true)
	faint  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func width() int {
	size, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || size < 30 {
		return 80
	}
	return size
}

func plural(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func relative(t, now time.Time) string {
	days := int(midnight(now).Sub(midnight(t)).Hours()/24 + 0.5)

	switch {
	case days <= 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 7:
		return strconv.Itoa(days) + "d ago"
	case t.Year() == now.Year():
		return t.Format("02 Jan")
	}

	return t.Format("02 Jan 2006")
}

func wrap(text string, wide int) []string {
	var lines []string
	var line []rune

	for _, word := range strings.Fields(text) {
		runes := []rune(word)

		for len(runes) > wide {
			if len(line) > 0 {
				lines = append(lines, string(line))
				line = nil
			}
			lines = append(lines, string(runes[:wide]))
			runes = runes[wide:]
		}

		switch {
		case len(line) == 0:
			line = runes
		case len(line)+1+len(runes) <= wide:
			line = append(append(line, ' '), runes...)
		default:
			lines = append(lines, string(line))
			line = runes
		}
	}

	if len(line) > 0 || len(lines) == 0 {
		lines = append(lines, string(line))
	}

	return lines
}

func build(items []point, from, cursor, wide int, now time.Time) ([]string, int) {
	total := len(items)
	numw := len(strconv.Itoa(total))
	pad := strings.Repeat(" ", numw)
	link := "  " + pad + "  " + faint.Render("│")
	bold := total - 1

	if cursor >= 0 {
		bold = cursor
	}

	var lines []string
	var at int

	for idx := from; idx < total; idx++ {
		item := items[idx]
		when := relative(item.Born, now)
		room := max(16, wide-2-numw-2-1-2-utf8.RuneCountInString(when)-2)
		rows := wrap(item.Text, room)

		if idx == cursor {
			at = len(lines)
		}

		for pos, row := range rows {
			lead := "  " + pad + "  " + faint.Render("│") + "  "
			if pos == 0 {
				mark := "  "
				if idx == cursor {
					mark = pink.Render("> ")
				}
				lead = mark + faint.Render(fmt.Sprintf("%*d", numw, idx+1)) + "  " + pink.Render("●") + "  "
			}

			if idx == bold {
				row = strong.Render(row)
			}

			line := lead + row
			if pos == len(rows)-1 {
				line += "  " + faint.Render(when)
			}

			lines = append(lines, line)
		}

		if idx < total-1 {
			lines = append(lines, link)
		} else {
			lines = append(lines, "  "+pad+"  "+pink.Render("▼"))
		}
	}

	return lines, at
}

func timeline(proj project, items []point, last, wide int, now time.Time) string {
	total := len(items)
	head := "  " + title.Render(proj.Name)

	if total == 0 {
		return "\n" + head + "\n\n  " + faint.Render("no checkpoints yet, add one with 'track add'") + "\n"
	}

	info := plural(total, "checkpoint") + " · since " + relative(items[0].Born, now)
	pad := strings.Repeat(" ", len(strconv.Itoa(total)))

	var lines []string

	from := 0
	if last > 0 && last < total {
		from = total - last
		lines = append(lines, "  "+pad+"  "+faint.Render("┆ "+strconv.Itoa(from)+" earlier"), "  "+pad+"  "+faint.Render("│"))
	}

	body, _ := build(items, from, -1, wide, now)
	lines = append(lines, body...)

	return "\n" + head + "  " + faint.Render(info) + "\n\n" + strings.Join(lines, "\n") + "\n"
}

func summary(now time.Time, names []project, counts map[string]int, latest map[string]time.Time, current string) string {
	if len(names) == 0 {
		return "\n  " + faint.Render("no projects yet, create one with 'track new <name>'") + "\n"
	}

	wide := 0
	for _, item := range names {
		wide = max(wide, utf8.RuneCountInString(item.Name))
	}

	var lines []string

	for _, item := range names {
		lead := "    "
		if slug(item.Name) == current {
			lead = "  " + pink.Render(">") + " "
		}

		count := counts[slug(item.Name)]
		last := "no activity"
		if count > 0 {
			last = relative(latest[slug(item.Name)], now)
		}

		name := fmt.Sprintf("%-*s", wide, item.Name)
		num := fmt.Sprintf("%-14s", plural(count, "checkpoint"))

		lines = append(lines, lead+name+"  "+faint.Render(num+"  "+last))
	}

	return "\n" + strings.Join(lines, "\n") + "\n"
}
