/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package chmod

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	paths []string
	bits  uint32
	orig  uint32
	dir   bool
	row   int
	col   int
	deep  bool
	can   bool
	done  bool
}

func (m model) preset(name string) model {
	if next, err := resolve(name, m.bits, m.dir); err == nil {
		m.bits = next
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.done = true
		return m, tea.Quit
	case "left", "h":
		m.col = max(m.col-1, 0)
	case "right", "l":
		m.col = min(m.col+1, 2)
	case "up", "k":
		m.row = max(m.row-1, 0)
	case "down", "j":
		m.row = min(m.row+1, 2)
	case " ", "space", "x":
		m.bits ^= uint32(1) << uint(8-(m.row*3+m.col))
	case "1":
		return m.preset("private"), nil
	case "2":
		return m.preset("shared"), nil
	case "3":
		return m.preset("public"), nil
	case "u":
		m.bits = m.orig
	case "r":
		if m.can {
			m.deep = !m.deep
		}
	}

	return m, nil
}

func (m model) name() string {
	if len(m.paths) == 1 {
		return title.Render(m.paths[0]) + " " + faint.Render(kind(m.dir))
	}

	return title.Render(m.paths[0]) + " " + faint.Render("and "+plural(len(m.paths)-1)+" more")
}

func plural(count int) string {
	if count == 1 {
		return "1 item"
	}
	return strconv.Itoa(count) + " items"
}

func (m model) View() string {
	var out strings.Builder

	out.WriteString("\n  " + m.name() + "\n\n")
	out.WriteString(grid(m.bits, m.orig, m.dir, m.row, m.col, true) + "\n\n")

	line := "  " + pink.Render(oct(m.bits)) + "  " + sym(m.bits)
	if m.bits != m.orig {
		line += "  " + faint.Render("was "+oct(m.orig)+" "+sym(m.orig))
	}

	out.WriteString(line + "\n\n")
	out.WriteString(strings.Join(speak(m.bits, m.dir), "\n") + "\n")

	for _, note := range notes(m.bits, m.dir) {
		out.WriteString("\n  " + red.Render(note))
	}

	if m.deep {
		out.WriteString("\n\n  " + amber.Render("recursive, applies to everything inside"))
	}

	keys := "  1 private  2 shared  3 public  u undo"
	if m.can {
		keys += "  r recursive"
	}

	out.WriteString("\n\n" + faint.Render("  arrows move  space toggle  enter apply  q quit") + "\n")
	out.WriteString(faint.Render(keys) + "\n")

	return out.String()
}
