package license

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	strong = lipgloss.NewStyle().Bold(true)
	faint  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	amber  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func listing(term string) string {
	var out strings.Builder
	var count int

	term = strings.ToLower(strings.TrimSpace(term))

	out.WriteString("\n")

	for _, item := range catalog() {
		hay := strings.ToLower(item.Key + " " + item.ID + " " + item.Title + " " + item.kind())
		if term != "" && !strings.Contains(hay, term) {
			continue
		}

		count++

		out.WriteString("  " + pink.Render(fmt.Sprintf("%-20s", item.Key)) + fmt.Sprintf("%-46s", item.Title) + faint.Render(item.kind()) + "\n")
	}

	if count == 0 {
		return "\n  " + faint.Render("no license matches "+term) + "\n\n"
	}

	noun := "licenses"
	if count == 1 {
		noun = "license"
	}

	out.WriteString("\n  " + faint.Render(fmt.Sprintf("%d %s, add one with: license add <name>", count, noun)) + "\n\n")

	return out.String()
}

func column(title string, tags []string, style lipgloss.Style) []string {
	out := []string{strong.Render(title)}

	for _, tag := range tags {
		out = append(out, style.Render(humane(tag)))
	}

	return out
}

func summary(g grant) string {
	var out strings.Builder

	out.WriteString("\n  " + strong.Render(g.Title) + faint.Render("  "+g.ID+"  "+g.kind()) + "\n\n")

	for _, line := range wrap(g.Brief, 76) {
		out.WriteString("  " + line + "\n")
	}

	out.WriteString("\n")

	cols := [][]string{
		column("Can", g.Can, green),
		column("Must", g.Must, amber),
		column("Cannot", g.Not, red),
	}

	rows := 0
	for _, col := range cols {
		rows = max(rows, len(col))
	}

	for at := 0; at < rows; at++ {
		line := "  "

		for _, col := range cols {
			cell, width := "", 0

			if at < len(col) {
				cell = col[at]
				width = lipgloss.Width(cell)
			}

			line += cell + strings.Repeat(" ", max(30-width, 1))
		}

		out.WriteString(strings.TrimRight(line, " ") + "\n")
	}

	return out.String() + "\n"
}

func wrap(text string, wide int) []string {
	var lines []string
	var line string

	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= wide:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}

	if line != "" {
		lines = append(lines, line)
	}

	return lines
}
