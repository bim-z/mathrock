/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	strong = lipgloss.NewStyle().Bold(true)
	faint  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	lined  = lipgloss.NewStyle().Bold(true).Underline(true)
)

func columns() int {
	size, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || size < 30 {
		return 80
	}
	return size
}

func runes(text string) int {
	return utf8.RuneCountInString(text)
}

func repeat(char string, count int) string {
	if count < 0 {
		count = 0
	}
	return strings.Repeat(char, count)
}

func wrap(text string, wide int) []string {
	var lines []string
	var line []rune

	for _, word := range strings.Fields(text) {
		piece := []rune(word)

		for len(piece) > wide {
			if len(line) > 0 {
				lines = append(lines, string(line))
				line = nil
			}
			lines = append(lines, string(piece[:wide]))
			piece = piece[wide:]
		}

		switch {
		case len(line) == 0:
			line = piece
		case len(line)+1+len(piece) <= wide:
			line = append(append(line, ' '), piece...)
		default:
			lines = append(lines, string(line))
			line = piece
		}
	}

	if len(line) > 0 {
		lines = append(lines, string(line))
	}

	return lines
}

func centered(text string, wide int) string {
	return repeat(" ", (wide-runes(text))/2) + text
}

func labeled(text string) string {
	at := strings.Index(text, ":")
	if at < 0 {
		return text
	}
	return strong.Render(text[:at+1]) + text[at+1:]
}

func preview(d doc, wide int) ([]string, []int) {
	wide = max(wide, 30)

	var out []string

	out = append(out, strong.Render(centered(d.Name, wide)))

	if d.Title != "" {
		out = append(out, centered(d.Title, wide))
	}

	if items := d.contact(); len(items) > 0 {
		for _, line := range wrap(strings.Join(items, "  |  "), wide) {
			out = append(out, faint.Render(centered(line, wide)))
		}
	}

	marks := make([]int, len(d.Sections))

	for at, sec := range d.Sections {
		if sec.Hidden {
			marks[at] = -1
			continue
		}

		out = append(out, "")
		marks[at] = len(out)
		out = append(out, pink.Render(strings.ToUpper(sec.Title)), faint.Render(repeat("─", wide)))

		for _, item := range sec.Blocks {
			switch item.Kind {
			case para, note:
				for _, line := range wrap(flat(item.Text), wide) {
					out = append(out, line)
				}
			case plain:
				for at, line := range wrap(flat(item.Text), wide) {
					if at == 0 {
						line = labeled(line)
					}
					out = append(out, line)
				}
			case bullet:
				indent := repeat("  ", item.Level)
				for at, line := range wrap(flat(item.Text), wide-len(indent)-2) {
					lead := "• "
					if at > 0 {
						lead = "  "
					}
					out = append(out, indent+faint.Render(lead)+line)
				}
			case entry:
				out = append(out, headed(item, wide)...)
			}
		}
	}

	return out, marks
}

func headed(item block, wide int) []string {
	parts := make([]string, 0, len(item.Parts))

	for _, part := range item.Parts {
		parts = append(parts, flat(part))
	}

	head := strings.Join(parts, " | ")
	room := wide - runes(item.Date) - 2

	if item.Date == "" {
		lines := wrap(head, wide)
		for at := range lines {
			lines[at] = strong.Render(lines[at])
		}
		return lines
	}

	lines := wrap(head, max(room, 10))
	if len(lines) == 0 {
		lines = []string{""}
	}

	last := lines[len(lines)-1]
	pad := wide - runes(last) - runes(item.Date)

	for at := range lines {
		lines[at] = strong.Render(lines[at])
	}

	if pad >= 2 {
		lines[len(lines)-1] = strong.Render(last) + repeat(" ", pad) + faint.Render(item.Date)
		return lines
	}

	return append(lines, repeat(" ", max(wide-runes(item.Date), 0))+faint.Render(item.Date))
}

func report(list []string) string {
	if len(list) == 0 {
		return "  " + green.Render("ATS check") + faint.Render("  no issues found") + "\n"
	}

	var out strings.Builder

	out.WriteString("  " + pink.Render("ATS check") + faint.Render(fmt.Sprintf("  %d to review", len(list))) + "\n\n")

	for _, line := range list {
		wrapped := wrap(line, max(columns()-6, 30))
		for at, text := range wrapped {
			lead := "  • "
			if at > 0 {
				lead = "    "
			}
			out.WriteString(faint.Render(lead) + text + "\n")
		}
	}

	return out.String()
}
