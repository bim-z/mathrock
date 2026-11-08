/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	idle = iota
	asking
	picking
	choosing
)

type reply struct {
	Text  string
	Picks []int
	Yes   bool
}

type link struct {
	reply chan reply
	stop  chan struct{}
	once  sync.Once
	send  func(tea.Msg)
}

func (l *link) abort() {
	l.once.Do(func() { close(l.stop) })
}

type askMsg struct {
	query query
	tag   string
}

type pickMsg struct{ items []topic }

type moreMsg struct{ label string }

type endMsg struct{}

type formModel struct {
	link   *link
	mode   int
	query  query
	tag    string
	input  textinput.Model
	fault  string
	items  []topic
	marks  []bool
	cursor int
	label  string
	count  int
}

func (m formModel) Init() tea.Cmd {
	return nil
}

func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case askMsg:
		m.mode = asking
		m.query = msg.query
		m.tag = msg.tag
		m.fault = ""
		m.input = field()
		return m, textinput.Blink
	case pickMsg:
		m.mode = picking
		m.items = msg.items
		m.marks = make([]bool, len(msg.items))
		for at, item := range msg.items {
			m.marks[at] = item.Base
		}
		m.cursor = 0
		return m, nil
	case moreMsg:
		m.mode = choosing
		m.label = msg.label
		return m, nil
	case endMsg:
		m.mode = idle
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			m.link.abort()
			m.mode = idle
			return m, tea.Quit
		}

		switch m.mode {
		case asking:
			return m.typing(msg)
		case picking:
			return m.choose(msg)
		case choosing:
			return m.confirm(msg)
		}
	}

	return m, nil
}

func field() textinput.Model {
	in := textinput.New()
	in.Prompt = "  > "
	in.CharLimit = 400
	in.Focus()
	return in
}

func (m formModel) typing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "enter" {
		m.fault = ""

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	value := tidy(m.input.Value())

	if value == "" && m.query.Need {
		m.fault = "this field is required"
		return m, nil
	}

	if value != "" && m.query.Check != nil {
		if fault := m.query.Check(value); fault != "" {
			m.fault = fault
			return m, nil
		}
	}

	m.mode = idle
	m.link.reply <- reply{Text: value}

	if value == "" {
		return m, nil
	}

	return m, tea.Println("  " + faint.Render(m.tag+" · "+m.query.Ask) + "  " + value)
}

func (m formModel) choose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = min(m.cursor+1, len(m.items)-1)
	case " ", "space", "x":
		m.marks[m.cursor] = !m.marks[m.cursor]
		m.fault = ""
	case "enter":
		var picks []int
		var names []string

		for at, on := range m.marks {
			if on {
				picks = append(picks, at)
				names = append(names, m.items[at].Title)
			}
		}

		if len(picks) == 0 {
			m.fault = "pick at least one section"
			return m, nil
		}

		m.mode = idle
		m.link.reply <- reply{Picks: picks}

		return m, tea.Println("  " + faint.Render("Sections") + "  " + strings.Join(names, ", "))
	}

	return m, nil
}

func (m formModel) confirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = idle
		m.link.reply <- reply{Yes: true}
	case "n", "N":
		m.mode = idle
		m.link.reply <- reply{}
	}

	return m, nil
}

func (m formModel) View() string {
	switch m.mode {
	case asking:
		var out strings.Builder

		out.WriteString("\n  " + pink.Render(m.query.Ask) + faint.Render("  "+m.tag) + "\n")
		out.WriteString(m.input.View() + "\n")

		switch {
		case m.fault != "":
			out.WriteString("  " + red.Render(m.fault) + "\n")
		case m.query.Hint != "":
			out.WriteString("  " + faint.Render(m.query.Hint) + "\n")
		}

		return out.String()
	case picking:
		var out strings.Builder

		out.WriteString("\n  " + lined.Render("What goes on your resume?") + "\n\n")

		for at, item := range m.items {
			mark := "[ ]"
			if m.marks[at] {
				mark = "[x]"
			}

			line := mark + " " + item.Title + faint.Render("  "+item.Note)
			if at == m.cursor {
				line = pink.Render(mark+" "+item.Title) + faint.Render("  "+item.Note)
			}

			out.WriteString("  " + line + "\n")
		}

		if m.fault != "" {
			out.WriteString("\n  " + red.Render(m.fault) + "\n")
		}

		out.WriteString("\n  " + faint.Render("space toggle  ·  enter continue  ·  esc cancel") + "\n")

		return out.String()
	case choosing:
		return "\n  " + pink.Render(m.label) + faint.Render("  y / n") + "\n"
	}

	return ""
}

type console struct {
	link *link
}

func (c console) await() (reply, error) {
	select {
	case got := <-c.link.reply:
		return got, nil
	case <-c.link.stop:
		return reply{}, errAbort
	}
}

func (c console) ask(q query, tag string) (string, error) {
	c.link.send(askMsg{query: q, tag: tag})

	got, err := c.await()

	return got.Text, err
}

func (c console) pick(items []topic) ([]int, error) {
	c.link.send(pickMsg{items: items})

	got, err := c.await()

	return got.Picks, err
}

func (c console) more(label string) (bool, error) {
	c.link.send(moreMsg{label: label})

	got, err := c.await()

	return got.Yes, err
}

func collect() (string, error) {
	hub := &link{reply: make(chan reply, 1), stop: make(chan struct{})}
	prog := tea.NewProgram(formModel{link: hub})
	hub.send = prog.Send

	var text string
	var fail error

	done := make(chan struct{})

	go func() {
		defer close(done)
		text, fail = gather(console{link: hub})
		prog.Send(endMsg{})
	}()

	if _, err := prog.Run(); err != nil {
		hub.abort()
		<-done
		return "", err
	}

	hub.abort()
	<-done

	return text, fail
}
