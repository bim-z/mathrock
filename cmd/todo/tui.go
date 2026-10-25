package todo

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const (
	viewing = iota
	adding
	editing
	asking
)

type model struct {
	tasks  []task
	cursor int
	mode   int
	input  textinput.Model
	fault  string
	height int
}

func field(text string) textinput.Model {
	in := textinput.New()
	in.Prompt = "  > "
	in.CharLimit = 200
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

func (m model) reload(err error) model {
	if err != nil {
		m.fault = err.Error()
		return m
	}

	all, err := fetch()
	if err != nil {
		m.fault = err.Error()
		return m
	}

	m.tasks = all
	if m.cursor >= len(all) {
		m.cursor = len(all) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m
}

func (m model) apply(fn func(*task)) model {
	if len(m.tasks) == 0 {
		return m
	}

	_, err := change(ident(m.tasks[m.cursor].ID), fn)
	return m.reload(err)
}

func (m model) create(text string) model {
	board := "general"
	if len(m.tasks) > 0 {
		board = m.tasks[m.cursor].Board
	}

	_, num, err := insert(text, board)
	m = m.reload(err)
	if err == nil && num > 0 && num <= len(m.tasks) {
		m.cursor = num - 1
	}

	return m
}

func (m model) rewrite(text string) model {
	_, err := change(ident(m.tasks[m.cursor].ID), func(t *task) { t.Text = text })
	return m.reload(err)
}

func (m model) browse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.fault = ""

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.tasks) > 0 {
			m.cursor = len(m.tasks) - 1
		}
	case " ", "space", "enter":
		return m.apply(finish), nil
	case "b":
		return m.apply(progress), nil
	case "s":
		return m.apply(pin), nil
	case "a":
		m.mode = adding
		m.input = field("")
		return m, textinput.Blink
	case "e":
		if len(m.tasks) > 0 {
			m.mode = editing
			m.input = field(m.tasks[m.cursor].Text)
			return m, textinput.Blink
		}
	case "d", "x":
		if len(m.tasks) > 0 {
			m.mode = asking
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
		if mode == adding {
			return m.create(text), nil
		}
		return m.rewrite(text), nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m model) confirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = viewing

	if msg.String() != "y" {
		return m, nil
	}

	_, err := remove(ident(m.tasks[m.cursor].ID))
	return m.reload(err), nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case adding, editing:
			return m.form(msg)
		case asking:
			return m.confirm(msg)
		}
		return m.browse(msg)
	}

	if m.mode == adding || m.mode == editing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	lines, at := build(m.tasks, m.cursor, "")
	if len(lines) == 0 {
		lines = []string{faint.Render("  no tasks yet")}
	}
	lines = window(lines, at, m.height-7)

	var foot string

	switch m.mode {
	case adding, editing:
		foot = m.input.View()
	case asking:
		foot = faint.Render("  remove ") + m.tasks[m.cursor].Text + faint.Render("? y/n")
	default:
		foot = faint.Render("  j/k move  space done  b begin  s star  a add  e edit  d remove  q quit")
	}

	if m.fault != "" {
		foot = red.Render("  " + m.fault)
	}

	return "\n" + strings.Join(lines, "\n") + "\n\n" + stats(m.tasks) + "\n\n" + foot
}

func tui() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive interface",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			all, err := fetch()
			if err != nil {
				return err
			}

			_, err = tea.NewProgram(model{tasks: all}, tea.WithAltScreen()).Run()
			return err
		},
	}
}
