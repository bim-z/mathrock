/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const (
	playing = iota
	listing
)

type drawMsg struct {
	seq int
}

type model struct {
	deck   deck
	path   string
	proto  string
	engine func(string, int) (string, error)
	index  int
	cursor int
	mode   int
	jump   string
	fault  string
	width  int
	height int
	cx     int
	cy     int
	top    int
	seq    int
	cache  map[string]frame
}

func (m model) sweep() tea.Cmd {
	if m.proto != kitty {
		return nil
	}

	return func() tea.Msg {
		_, _ = os.Stdout.WriteString(wipe)
		return nil
	}
}

func window(lines []string, at, size int) []string {
	if size <= 0 || len(lines) <= size {
		return lines
	}

	from := at - size/2
	if from < 0 {
		from = 0
	}
	if from+size > len(lines) {
		from = len(lines) - size
	}

	return lines[from : from+size]
}

func (m model) frame() (frame, error) {
	key := fmt.Sprintf("%d:%d:%d:%d:%d:%s", m.index, m.width, m.height, m.cx, m.cy, m.proto)

	if hit, ok := m.cache[key]; ok {
		return hit, nil
	}

	made, err := layout(m.deck.Pages[m.index], m.width, m.height, m.deck.Base, m.proto, m.cx, m.cy, m.engine)
	if err != nil {
		return frame{}, err
	}

	m.cache[key] = made

	return made, nil
}

func (m model) refresh() (model, tea.Cmd) {
	if m.proto == off || m.mode != playing {
		return m, nil
	}

	m.seq++
	seq := m.seq

	tick := tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg {
		return drawMsg{seq: seq}
	})

	return m, tea.Sequence(m.sweep(), tea.ClearScreen, tick)
}

func (m model) paint() tea.Cmd {
	made, err := m.frame()
	if err != nil {
		return nil
	}

	_, spots := made.shown(m.height-3, m.top)
	proto := m.proto

	return func() tea.Msg {
		var out strings.Builder

		if proto == kitty {
			out.WriteString(wipe)
		}

		for _, item := range spots {
			data, err := encode(proto, item.Pic, item.Wide, item.High)
			if err != nil {
				continue
			}
			fmt.Fprintf(&out, "\x1b7\x1b[%d;%dH%s\x1b8", item.Row+1, item.Col+1, data)
		}

		_, _ = os.Stdout.WriteString(out.String())

		return nil
	}
}

func (m model) move(to int) (model, tea.Cmd) {
	m.jump = ""
	to = max(0, min(to, len(m.deck.Pages)-1))

	if to == m.index {
		return m, nil
	}

	m.index = to
	m.top = 0

	return m.refresh()
}

func (m model) scroll(dir int) (model, tea.Cmd) {
	made, err := m.frame()
	if err != nil {
		return m, nil
	}

	room := m.height - 3
	next := max(0, min(m.top+dir*max(room/2, 1), len(made.Lines)-room))

	if next == m.top {
		return m, nil
	}

	m.top = next

	return m.refresh()
}

func (m model) reload() (model, tea.Cmd) {
	next, err := load(m.path)
	if err != nil {
		m.fault = err.Error()
		return m, nil
	}

	forget()
	m.top = 0

	m.deck = next
	m.cache = map[string]frame{}
	m.index = min(m.index, len(next.Pages)-1)
	m.cursor = min(m.cursor, len(next.Pages)-1)

	return m.refresh()
}

func (m model) play(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		m.jump += key
		if len(m.jump) > 4 {
			m.jump = m.jump[1:]
		}
		return m, nil
	}

	m.fault = ""

	switch key {
	case "q", "ctrl+c":
		return m, tea.Sequence(m.sweep(), tea.Quit)
	case "esc":
		if m.jump != "" {
			m.jump = ""
			return m, nil
		}
		return m, tea.Sequence(m.sweep(), tea.Quit)
	case "enter":
		if m.jump != "" {
			num, _ := strconv.Atoi(m.jump)
			return m.move(num - 1)
		}
		return m.move(m.index + 1)
	case "l", "right", "n", "j", "down", "pgdown", " ", "space":
		return m.move(m.index + 1)
	case "h", "left", "p", "k", "up", "pgup", "backspace":
		return m.move(m.index - 1)
	case "g", "home":
		return m.move(0)
	case "G", "end":
		return m.move(len(m.deck.Pages) - 1)
	case "J", "ctrl+d":
		return m.scroll(1)
	case "K", "ctrl+u":
		return m.scroll(-1)
	case "r":
		m.jump = ""
		return m.refresh()
	case "R":
		m.jump = ""
		return m.reload()
	case "tab":
		m.jump = ""
		m.mode = listing
		m.cursor = m.index
		return m, tea.Sequence(m.sweep(), tea.ClearScreen)
	}

	m.jump = ""

	return m, nil
}

func (m model) browse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	last := len(m.deck.Pages) - 1

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "tab":
		m.mode = playing
		return m.refresh()
	case "enter":
		m.mode = playing
		m.index = m.cursor
		return m.refresh()
	case "j", "down":
		m.cursor = min(m.cursor+1, last)
	case "k", "up":
		m.cursor = max(m.cursor-1, 0)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = last
	}

	return m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.cx, m.cy = cells()
		return m.refresh()
	case drawMsg:
		if msg.seq == m.seq && m.mode == playing {
			return m, m.paint()
		}
		return m, nil
	case tea.KeyMsg:
		if m.mode == listing {
			return m.browse(msg)
		}
		return m.play(msg)
	}

	return m, nil
}

func (m model) footer() string {
	total := len(m.deck.Pages)
	size := 20
	if m.width < 60 {
		size = 10
	}

	filled := (m.index + 1) * size / total
	bar := pink.Render(strings.Repeat("━", filled)) + faint.Render(strings.Repeat("━", size-filled))
	left := "  " + faint.Render(fmt.Sprintf("%d/%d", m.index+1, total)) + "  " + bar

	if m.jump != "" {
		left += "  " + pink.Render(":"+m.jump)
	}
	if m.fault != "" {
		left += "  " + red.Render(m.fault)
	}

	hint := "h/l move  tab overview  q quit"

	if made, err := m.frame(); err == nil && len(made.Lines) > m.height-3 {
		hint = "J/K scroll  " + hint
	}

	right := faint.Render(hint) + "  "
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)

	if gap < 2 {
		return left
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m model) overview() string {
	total := len(m.deck.Pages)
	width := len(strconv.Itoa(total))
	lines := make([]string, 0, total)

	for i, page := range m.deck.Pages {
		lead := "  "
		text := heading(page)

		if i == m.cursor {
			lead = pink.Render("> ")
		}
		if i == m.index {
			text = text + " " + faint.Render("*")
		}

		lines = append(lines, lead+faint.Render(fmt.Sprintf("%*d.", width, i+1))+" "+text)
	}

	lines = window(lines, m.cursor, m.height-5)
	head := "  " + title.Render(m.deck.Title) + " " + faint.Render(fmt.Sprintf("%d slides", total))
	foot := faint.Render("  j/k move  enter open  tab back  q quit")

	return "\n" + head + "\n\n" + strings.Join(lines, "\n") + "\n\n" + foot
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.mode == listing {
		return m.overview()
	}

	room := m.height - 3
	body := []string{}

	made, err := m.frame()
	if err != nil {
		body = append(body, "  "+red.Render(err.Error()))
	} else {
		body, _ = made.shown(room, m.top)
	}

	body = append([]string(nil), body...)
	for len(body) < room {
		body = append(body, "")
	}

	return "\n" + strings.Join(body, "\n") + "\n\n" + m.footer()
}

func show() *cobra.Command {
	var opt options

	cmd := &cobra.Command{
		Use:   "show <file>",
		Short: "Present a deck in the interactive interface",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return present(args[0], opt)
		},
	}

	flags(cmd, &opt)

	return cmd
}
