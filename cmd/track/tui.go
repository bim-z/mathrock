/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package track

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
	"golang.org/x/term"
)

const (
	viewing = iota
	writing
	revising
	removing
	naming
	erasing
)

type model struct {
	names  []project
	index  int
	items  []point
	cursor int
	mode   int
	input  textinput.Model
	fault  string
	width  int
	height int
}

func field(prompt, text string) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.CharLimit = 300
	in.SetValue(text)
	in.Focus()
	return in
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

func (m model) load(name string, last bool) model {
	var names []project
	var items []point
	var proj project

	err := view(func(tx *bbolt.Tx) error {
		var err error

		names, err = every(tx)
		if err != nil || len(names) == 0 {
			return err
		}

		proj, err = chosen(tx, name)
		if err != nil {
			proj = names[0]
		}

		items, err = collect(tx, proj)

		return err
	})
	if err != nil {
		m.fault = err.Error()
		return m
	}

	m.names, m.items, m.index = names, items, 0

	for idx, item := range names {
		if slug(item.Name) == slug(proj.Name) {
			m.index = idx
		}
	}

	if last {
		m.cursor = len(items) - 1
	}
	m.cursor = max(0, min(m.cursor, len(items)-1))

	return m
}

func (m model) current() string {
	if len(m.names) == 0 {
		return ""
	}
	return m.names[m.index].Name
}

func (m model) jump(delta int) model {
	if len(m.names) < 2 {
		return m
	}

	target := m.names[(m.index+delta+len(m.names))%len(m.names)]

	err := update(func(tx *bbolt.Tx) error {
		return tx.Bucket(meta).Put(active, []byte(slug(target.Name)))
	})
	if err != nil {
		m.fault = err.Error()
		return m
	}

	return m.load(target.Name, true)
}

func (m model) insert(text string) model {
	err := update(func(tx *bbolt.Tx) error {
		proj, err := lookup(tx, m.current())
		if err != nil {
			return err
		}

		seq, err := tx.Bucket(points).NextSequence()
		if err != nil {
			return fmt.Errorf("failed to create checkpoint: %w", err)
		}

		return store(tx, point{ID: seq, Project: slug(proj.Name), Text: text, Born: time.Now()})
	})
	if err != nil {
		m.fault = err.Error()
		return m
	}

	return m.load(m.current(), true)
}

func (m model) rewrite(id uint64, text string) model {
	err := update(func(tx *bbolt.Tx) error {
		proj, err := lookup(tx, m.current())
		if err != nil {
			return err
		}

		items, err := collect(tx, proj)
		if err != nil {
			return err
		}

		for _, item := range items {
			if item.ID == id {
				item.Text = text
				return store(tx, item)
			}
		}

		return errors.New("the checkpoint no longer exists")
	})
	if err != nil {
		m.fault = err.Error()
	}

	return m.load(m.current(), false)
}

func (m model) delete(id uint64) model {
	err := update(func(tx *bbolt.Tx) error {
		return tx.Bucket(points).Delete(key(id))
	})
	if err != nil {
		m.fault = err.Error()
	}

	return m.load(m.current(), false)
}

func (m model) create(raw string) model {
	name, err := clean(raw)
	if err != nil {
		m.fault = err.Error()
		return m
	}

	err = update(func(tx *bbolt.Tx) error {
		if tx.Bucket(projects).Get([]byte(slug(name))) != nil {
			return fmt.Errorf("project %q already exists", name)
		}

		if err := keep(tx, project{Name: name, Born: time.Now()}); err != nil {
			return err
		}

		return tx.Bucket(meta).Put(active, []byte(slug(name)))
	})
	if err != nil {
		m.fault = err.Error()
		return m
	}

	return m.load(name, true)
}

func (m model) erase() model {
	name := m.current()

	err := update(func(tx *bbolt.Tx) error {
		proj, err := lookup(tx, name)
		if err != nil {
			return err
		}

		items, err := collect(tx, proj)
		if err != nil {
			return err
		}

		for _, item := range items {
			if err := tx.Bucket(points).Delete(key(item.ID)); err != nil {
				return err
			}
		}

		if err := tx.Bucket(projects).Delete([]byte(slug(proj.Name))); err != nil {
			return err
		}

		if raw := tx.Bucket(meta).Get(active); raw != nil && string(raw) == slug(proj.Name) {
			return tx.Bucket(meta).Delete(active)
		}

		return nil
	})
	if err != nil {
		m.fault = err.Error()
		return m
	}

	return m.load("", true)
}

func (m model) browse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.fault = ""

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = max(0, min(m.cursor+1, len(m.items)-1))
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(0, len(m.items)-1)
	case "right", "l", "tab", "]":
		return m.jump(1), nil
	case "left", "h", "shift+tab", "[":
		return m.jump(-1), nil
	case "a":
		if len(m.names) > 0 {
			m.mode = writing
			m.input = field("  > ", "")
			return m, textinput.Blink
		}
	case "e":
		if len(m.items) > 0 {
			m.mode = revising
			m.input = field("  > ", m.items[m.cursor].Text)
			return m, textinput.Blink
		}
	case "d", "x":
		if len(m.items) > 0 {
			m.mode = removing
		}
	case "n":
		m.mode = naming
		m.input = field("  project > ", "")
		return m, textinput.Blink
	case "D":
		if len(m.names) > 0 {
			m.mode = erasing
		}
	}

	return m, nil
}

func (m model) form(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = viewing
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.input.Value())
		mode := m.mode
		m.mode = viewing

		if text == "" {
			return m, nil
		}

		switch mode {
		case writing:
			return m.insert(text), nil
		case naming:
			return m.create(text), nil
		}

		return m.rewrite(m.items[m.cursor].ID, text), nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m model) confirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	mode := m.mode
	m.mode = viewing

	if msg.String() != "y" {
		return m, nil
	}

	if mode == erasing {
		return m.erase(), nil
	}

	return m.delete(m.items[m.cursor].ID), nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case writing, revising, naming:
			return m.form(msg)
		case removing, erasing:
			return m.confirm(msg)
		}
		return m.browse(msg)
	}

	if m.mode == writing || m.mode == revising || m.mode == naming {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) tabs() string {
	if len(m.names) == 0 {
		return "  " + faint.Render("track")
	}

	parts := make([]string, 0, len(m.names))
	size := 0

	for idx, item := range m.names {
		size += utf8.RuneCountInString(item.Name) + 5

		if idx == m.index {
			parts = append(parts, title.Render(item.Name))
		} else {
			parts = append(parts, faint.Render(item.Name))
		}
	}

	if m.width > 0 && size > m.width-4 {
		return "  " + title.Render(m.current()) + " " + faint.Render(strconv.Itoa(m.index+1)+"/"+strconv.Itoa(len(m.names)))
	}

	return "  " + strings.Join(parts, faint.Render("  ·  "))
}

func (m model) body() string {
	if len(m.names) == 0 {
		return "  " + faint.Render("no projects yet, press n to create one")
	}

	if len(m.items) == 0 {
		return "  " + faint.Render("no checkpoints yet, press a to add one")
	}

	wide := m.width
	if wide == 0 {
		wide = 80
	}

	lines, at := build(m.items, 0, m.cursor, wide, time.Now())

	return strings.Join(window(lines, at, m.height-9), "\n")
}

func (m model) View() string {
	var foot string

	switch m.mode {
	case writing, revising, naming:
		foot = m.input.View()
	case removing:
		foot = faint.Render("  remove ") + m.items[m.cursor].Text + faint.Render("? y/n")
	case erasing:
		foot = faint.Render("  drop ") + m.current() + faint.Render(" and "+plural(len(m.items), "checkpoint")+"? y/n")
	default:
		foot = faint.Render("  j/k move  a add  e edit  d remove  q quit") + "\n" + faint.Render("  h/l project  n new project  D drop project")
	}

	if m.fault != "" {
		foot = "  " + red.Render(m.fault)
	}

	info := ""
	if len(m.items) > 0 {
		info = "  " + faint.Render(plural(len(m.items), "checkpoint")+" · since "+relative(m.items[0].Born, time.Now()))
	}

	return "\n" + m.tabs() + "\n\n" + m.body() + "\n\n" + info + "\n\n" + foot + "\n"
}

func tui(opt *options) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive timeline",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
				return errors.New("the interactive mode needs a terminal, use 'track' to print the timeline")
			}

			start := model{}.load(opt.Project, true)
			if start.fault != "" {
				return errors.New(start.fault)
			}

			_, err := tea.NewProgram(start, tea.WithAltScreen()).Run()

			return err
		},
	}
}
