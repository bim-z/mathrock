package note

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const (
	viewing = iota
	adding
	moving
	asking
	reading
)

type edited struct {
	id   uint64
	path string
	err  error
}

type model struct {
	notes  []note
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
	in.CharLimit = 500
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

	m.notes = all
	if m.cursor >= len(all) {
		m.cursor = len(all) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m
}

func (m model) follow(id uint64) model {
	for i, n := range m.notes {
		if n.ID == id {
			m.cursor = i
		}
	}
	return m
}

func (m model) apply(fn func(*note)) model {
	if len(m.notes) == 0 {
		return m
	}

	id := m.notes[m.cursor].ID
	_, err := change(ident(id), fn)

	return m.reload(err).follow(id)
}

func (m model) create(text string) model {
	board := "general"
	if len(m.notes) > 0 {
		board = m.notes[m.cursor].Board
	}

	fresh, _, err := insert(text, board)
	m = m.reload(err)
	if err == nil {
		m = m.follow(fresh.ID)
	}

	return m
}

func (m model) settle(msg edited) model {
	text, err := collect(msg.path)
	if msg.err != nil {
		m.fault = "editor failed: " + msg.err.Error()
		return m
	}
	if err != nil {
		m.fault = err.Error()
		return m
	}
	if text == "" {
		m.fault = "note is empty"
		return m
	}

	_, err = change(ident(msg.id), func(n *note) { n.Body = text })

	return m.reload(err).follow(msg.id)
}

func (m model) launch() (tea.Model, tea.Cmd) {
	if len(m.notes) == 0 {
		return m, nil
	}

	cur := m.notes[m.cursor]

	path, cmd, err := prepare(cur.Body)
	if err != nil {
		m.fault = err.Error()
		return m, nil
	}

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return edited{id: cur.ID, path: path, err: err}
	})
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
		if m.cursor < len(m.notes)-1 {
			m.cursor++
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.notes) > 0 {
			m.cursor = len(m.notes) - 1
		}
	case "enter", " ", "space":
		if len(m.notes) > 0 {
			m.mode = reading
		}
	case "s":
		return m.apply(pin), nil
	case "a":
		m.mode = adding
		m.input = field("  > ", "")
		return m, textinput.Blink
	case "m":
		if len(m.notes) > 0 {
			m.mode = moving
			m.input = field("  board > ", m.notes[m.cursor].Board)
			return m, textinput.Blink
		}
	case "e":
		return m.launch()
	case "d", "x":
		if len(m.notes) > 0 {
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

		board := clean(text)
		return m.apply(func(n *note) { n.Board = board }), nil
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

	_, err := remove(ident(m.notes[m.cursor].ID))

	return m.reload(err), nil
}

func (m model) read(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "enter", "backspace":
		m.mode = viewing
	case "e":
		return m.launch()
	case "ctrl+c":
		return m, tea.Quit
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
		return m, nil
	case edited:
		return m.settle(msg), nil
	case tea.KeyMsg:
		switch m.mode {
		case adding, moving:
			return m.form(msg)
		case asking:
			return m.confirm(msg)
		case reading:
			return m.read(msg)
		}
		return m.browse(msg)
	}

	if m.mode == adding || m.mode == moving {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.mode == reading && len(m.notes) > 0 {
		foot := faint.Render("  esc back  e edit")
		if m.fault != "" {
			foot = red.Render("  " + m.fault)
		}

		return "\n" + detail(m.notes[m.cursor], m.cursor+1, m.width) + "\n\n" + foot
	}

	lines, at := build(m.notes, m.cursor, nil, fit(m.width))
	if len(lines) == 0 {
		lines = []string{faint.Render("  no notes yet")}
	}
	lines = window(lines, at, m.height-6)

	var foot string

	switch m.mode {
	case adding, moving:
		foot = m.input.View()
	case asking:
		line, _ := first(m.notes[m.cursor].Body)
		foot = faint.Render("  remove ") + clip(line, fit(m.width)) + faint.Render("? y/n")
	default:
		foot = faint.Render("  j/k move  enter open  a add  e edit  s star  m move  d remove  q quit")
	}

	if m.fault != "" {
		foot = red.Render("  " + m.fault)
	}

	return "\n" + strings.Join(lines, "\n") + "\n\n" + stats(m.notes) + "\n\n" + foot
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

			_, err = tea.NewProgram(model{notes: all, width: cols()}, tea.WithAltScreen()).Run()
			return err
		},
	}
}
