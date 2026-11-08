/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package bannedbook

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	strong = lipgloss.NewStyle().Bold(true)
	faint  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func days(from, to time.Time) int {
	return int(midnight(to).Sub(midnight(from)).Hours() / 24)
}

func age(entry item, now time.Time) string {
	count := days(entry.Added, now)
	if count < 1 {
		return "today"
	}
	return strconv.Itoa(count) + "d"
}

func when(entry item, now time.Time) string {
	label := entry.Due.Format("Mon 2 Jan")
	if entry.Clock {
		label += " " + entry.Due.Format("15:04")
	}

	gap := days(now, entry.Due)

	switch {
	case gap < 0:
		return red.Render(label + "  overdue " + strconv.Itoa(-gap) + "d")
	case gap == 0:
		return pink.Render(label + "  today")
	case gap == 1:
		return label + faint.Render("  tomorrow")
	}

	return label + faint.Render("  in "+strconv.Itoa(gap)+"d")
}

func (b *book) row(entry item, number string, now time.Time) string {
	line := "  " + faint.Render(number) + " " + entry.Text

	switch entry.List {
	case actions:
		if entry.Context != "" {
			line += "  " + pink.Render(entry.Context)
		}
		if parent := b.find(entry.Parent); parent != nil {
			line += faint.Render("  " + parent.Text)
		}
	case waiting:
		line += faint.Render("  " + entry.Who + "  " + age(entry, now))
		if parent := b.find(entry.Parent); parent != nil {
			line += faint.Render("  " + parent.Text)
		}
	case calendar:
		line += "  " + when(entry, now)
		if parent := b.find(entry.Parent); parent != nil {
			line += faint.Render("  " + parent.Text)
		}
	case projects:
		if kids := b.kids(entry.ID); len(kids) > 0 {
			line += faint.Render("  next: " + kids[0].Text)
		} else {
			line += "  " + red.Render("no next action")
		}
	case inbox:
		line += faint.Render("  " + age(entry, now))
	}

	return line
}

func (b *book) board(only string) string {
	now := time.Now()
	order := b.numbered()
	width := len(strconv.Itoa(len(order)))

	var out strings.Builder
	var shown int

	if only == finished {
		return b.history()
	}

	for _, name := range lists {
		if only != "" && only != name {
			continue
		}

		var rows []string

		for at, id := range order {
			entry := b.find(id)
			if entry.List == name {
				rows = append(rows, b.row(*entry, fmt.Sprintf("%*d.", width, at+1), now))
			}
		}

		if len(rows) == 0 && only == "" {
			continue
		}

		shown++

		out.WriteString("\n  " + strong.Render(titles[name]) + " " + faint.Render(strconv.Itoa(len(rows))) + "\n")

		if len(rows) == 0 {
			out.WriteString("  " + faint.Render("empty") + "\n")
		}

		for _, line := range rows {
			out.WriteString(line + "\n")
		}
	}

	if shown == 0 {
		return "\n  " + faint.Render("nothing here yet, capture something with: gtd add \"call the dentist\"") + "\n"
	}

	return out.String() + "\n"
}

func (b *book) history() string {
	var list []item

	for _, entry := range b.Items {
		if entry.List == finished {
			list = append(list, entry)
		}
	}

	sort.SliceStable(list, func(x, y int) bool { return list[x].Closed.After(list[y].Closed) })

	if len(list) == 0 {
		return "\n  " + faint.Render("nothing is done yet") + "\n"
	}

	var out strings.Builder

	out.WriteString("\n  " + strong.Render(titles[finished]) + " " + faint.Render(strconv.Itoa(len(list))) + "\n")

	for at, entry := range list {
		if at == 20 {
			out.WriteString("  " + faint.Render(fmt.Sprintf("and %d more", len(list)-20)) + "\n")
			break
		}
		out.WriteString("  " + green.Render("•") + " " + entry.Text + faint.Render("  "+entry.Closed.Format("2 Jan")) + "\n")
	}

	return out.String() + "\n"
}

func (b *book) review() string {
	now := time.Now()

	var out strings.Builder
	var clean = true

	line := func(mark, text string) {
		clean = false
		out.WriteString("  " + mark + " " + text + "\n")
	}

	out.WriteString("\n  " + strong.Render("Weekly review") + "\n\n")

	var count int
	for _, entry := range b.Items {
		if entry.List == inbox {
			count++
		}
	}
	if count > 0 {
		line(pink.Render("•"), fmt.Sprintf("%d in the inbox, run 'gtd clarify' to process them", count))
	}

	for _, entry := range b.Items {
		switch entry.List {
		case projects:
			if len(b.kids(entry.ID)) == 0 {
				line(red.Render("•"), "project without a next action: "+entry.Text)
			}
		case waiting:
			if days(entry.Added, now) >= 7 {
				line(pink.Render("•"), fmt.Sprintf("waiting on %s for %dd: %s", entry.Who, days(entry.Added, now), entry.Text))
			}
		case calendar:
			if gap := days(now, entry.Due); gap < 0 {
				line(red.Render("•"), "overdue: "+entry.Text)
			} else if gap <= 7 {
				line(faint.Render("•"), entry.Text+"  "+when(entry, now))
			}
		}
	}

	if clean {
		out.WriteString("  " + green.Render("everything is in order") + "\n")
	}

	return out.String() + "\n"
}
