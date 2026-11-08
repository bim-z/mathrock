/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package charmer

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	frame  = lipgloss.NewStyle().Padding(1, 0)
	indent = lipgloss.NewStyle().Padding(1, 2)
	head   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warn   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

type phase int

const (
	viewing phase = iota
	adding
	removing
)

type entry struct {
	seq  int
	link Link
}

func (e entry) Title() string {
	return fmt.Sprintf("%d  %s", e.seq, e.link.Name)
}

func (e entry) Description() string {
	if e.link.Description == "" || e.link.Description == "-" {
		return e.link.Target
	}

	return e.link.Target + "  " + e.link.Description
}

func (e entry) FilterValue() string {
	return e.Title() + " " + e.link.Target + " " + e.link.Description
}

type keys struct {
	visit key.Binding
	add   key.Binding
	erase key.Binding
}

type model struct {
	list   list.Model
	inputs []textinput.Model
	focus  int
	phase  phase
	notice string
	keys   keys
}

func items() ([]list.Item, error) {
	all, err := fetch()
	if err != nil {
		return nil, err
	}

	rows := make([]list.Item, len(all))
	for i, link := range all {
		rows[i] = entry{seq: i + 1, link: link}
	}

	return rows, nil
}

func build() (model, error) {
	rows, err := items()
	if err != nil {
		return model{}, err
	}

	bind := keys{
		visit: key.NewBinding(key.WithKeys("enter", "o"), key.WithHelp("enter", "open")),
		add:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		erase: key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
	}

	pane := list.New(rows, list.NewDefaultDelegate(), 0, 0)
	pane.Title = "piw"
	pane.Styles.Title = head
	pane.SetShowStatusBar(false)
	pane.SetStatusBarItemName("link", "links")
	pane.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{bind.visit, bind.add, bind.erase}
	}
	pane.AdditionalFullHelpKeys = pane.AdditionalShortHelpKeys

	holders := []string{"GitHub", "github.com", "Optional"}
	inputs := make([]textinput.Model, len(holders))
	for i, holder := range holders {
		input := textinput.New()
		input.Prompt = "> "
		input.PromptStyle = head
		input.Placeholder = holder
		input.CharLimit = 512
		input.Width = 48
		inputs[i] = input
	}

	return model{list: pane, inputs: inputs, keys: bind}, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		x, y := frame.GetFrameSize()
		m.list.SetSize(msg.Width-x, msg.Height-y)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.phase {
	case adding:
		return m.compose(msg)
	case removing:
		return m.confirm(msg)
	}

	return m.browse(msg)
}

func (m model) browse(msg tea.Msg) (tea.Model, tea.Cmd) {
	press, ok := msg.(tea.KeyMsg)
	if ok && m.list.FilterState() != list.Filtering {
		switch {
		case key.Matches(press, m.keys.visit):
			return m.visit()
		case key.Matches(press, m.keys.add):
			return m.start()
		case key.Matches(press, m.keys.erase):
			if m.list.SelectedItem() != nil {
				m.phase = removing
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m model) visit() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(entry)
	if !ok {
		return m, nil
	}

	if err := launch(item.link.Target); err != nil {
		cmd := m.list.NewStatusMessage(warn.Render(err.Error()))
		return m, cmd
	}

	cmd := m.list.NewStatusMessage(dim.Render("Opened " + item.link.Name))

	return m, cmd
}

func (m model) start() (tea.Model, tea.Cmd) {
	for i := range m.inputs {
		m.inputs[i].Reset()
		m.inputs[i].Blur()
	}

	m.phase = adding
	m.focus = 0
	m.notice = ""

	cmd := m.inputs[0].Focus()

	return m, cmd
}

func (m model) compose(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyMsg); ok {
		switch press.String() {
		case "esc":
			m.phase = viewing
			return m, nil
		case "tab", "down":
			return m.shift(1)
		case "shift+tab", "up":
			return m.shift(-1)
		case "enter":
			if m.focus < len(m.inputs)-1 {
				return m.shift(1)
			}
			return m.submit()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)

	return m, cmd
}

func (m model) shift(step int) (tea.Model, tea.Cmd) {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + step + len(m.inputs)) % len(m.inputs)

	cmd := m.inputs[m.focus].Focus()

	return m, cmd
}

func (m model) submit() (tea.Model, tea.Cmd) {
	link := Link{
		Name:        m.inputs[0].Value(),
		Target:      m.inputs[1].Value(),
		Description: m.inputs[2].Value(),
	}

	if err := insert(link); err != nil {
		m.notice = err.Error()
		return m, nil
	}

	return m.refresh(len(m.list.Items()), "Added "+link.Name)
}

func (m model) confirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	press, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	item, found := m.list.SelectedItem().(entry)
	if press.String() != "y" || !found {
		m.phase = viewing
		return m, nil
	}

	if err := remove(item.link.Key); err != nil {
		m.phase = viewing
		cmd := m.list.NewStatusMessage(warn.Render(err.Error()))
		return m, cmd
	}

	return m.refresh(item.seq-1, "Removed "+item.link.Name)
}

func (m model) refresh(index int, note string) (tea.Model, tea.Cmd) {
	m.phase = viewing

	rows, err := items()
	if err != nil {
		cmd := m.list.NewStatusMessage(warn.Render(err.Error()))
		return m, cmd
	}

	m.list.ResetFilter()
	load := m.list.SetItems(rows)
	if len(rows) > 0 {
		m.list.Select(min(index, len(rows)-1))
	}

	status := m.list.NewStatusMessage(dim.Render(note))

	return m, tea.Batch(load, status)
}

func (m model) form() string {
	labels := []string{"Name", "URL", "Description"}

	rows := []string{head.Render("New link"), ""}
	for i, label := range labels {
		rows = append(rows, dim.Render(label), m.inputs[i].View(), "")
	}

	rows = append(rows, dim.Render("tab switch   enter next   esc cancel"))
	if m.notice != "" {
		rows = append(rows, "", warn.Render(m.notice))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m model) ask() string {
	item, ok := m.list.SelectedItem().(entry)
	if !ok {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		head.Render("Delete link"),
		"",
		item.link.Name,
		dim.Render(item.link.Target),
		"",
		dim.Render("y delete   any other key cancel"),
	)
}

func (m model) View() string {
	switch m.phase {
	case adding:
		return indent.Render(m.form())
	case removing:
		return indent.Render(m.ask())
	}

	return frame.Render(m.list.View())
}

func run() error {
	m, err := build()
	if err != nil {
		return err
	}

	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()

	return err
}
