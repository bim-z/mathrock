package calendar

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const (
	viewing = iota
	forming
	asking
)

var labels = []string{"  title > ", "  time > ", "  place > ", "  date > "}

var hints = []string{
	"what is happening",
	"14:00 or 14:00-15:30, empty for all day",
	"optional",
	"today, tomorrow, mon, +3, 2026-12-31 or 31/12",
}

type model struct {
	events []event
	day    time.Time
	index  int
	side   bool
	mode   int
	input  textinput.Model
	stage  int
	target uint64
	values []string
	fault  string
	width  int
	height int
}

func field(prompt, text string) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.CharLimit = 200
	in.SetValue(text)
	in.Focus()
	return in
}

func slice(lines []string, at, size int) []string {
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

func (m model) daily() []int {
	stamp := m.day.Format(layout)

	var out []int
	for i, ev := range m.events {
		if ev.Day == stamp {
			out = append(out, i)
		}
	}

	return out
}

func (m model) chosen() (event, bool) {
	list := m.daily()
	if len(list) == 0 {
		return event{}, false
	}

	idx := m.index
	if idx >= len(list) {
		idx = len(list) - 1
	}

	return m.events[list[idx]], true
}

func (m model) land(to time.Time) model {
	m.day = to
	m.index = 0
	m.side = m.side && len(m.daily()) > 0
	return m
}

func (m model) slide(delta int) model {
	size := len(m.daily())

	m.index += delta
	if m.index >= size {
		m.index = size - 1
	}
	if m.index < 0 {
		m.index = 0
	}

	return m
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

	m.events = all
	return m
}

func (m model) compose(target uint64, values []string) model {
	m.mode = forming
	m.target = target
	m.values = values
	m.stage = 0
	m.input = field(labels[0], values[0])
	return m
}

func (m model) retry(stage int, text string) model {
	m.fault = text
	m.stage = stage
	m.input = field(labels[stage], m.values[stage])
	return m
}

func (m model) commit() model {
	if m.values[0] == "" {
		return m.retry(0, "title is empty")
	}

	date, err := when(m.values[3], time.Now())
	if err != nil {
		return m.retry(3, err.Error())
	}

	begin, end, err := span(m.values[1])
	if err != nil {
		return m.retry(1, err.Error())
	}

	var id uint64

	if m.target == 0 {
		ev, _, err := insert(event{
			Title: m.values[0],
			Day:   date.Format(layout),
			Begin: begin,
			End:   end,
			Place: m.values[2],
		})
		if err != nil {
			m.mode = viewing
			m.fault = err.Error()
			return m
		}
		id = ev.ID
	} else {
		id = m.target
		_, err := change(ident(id), func(ev *event) {
			ev.Title = m.values[0]
			ev.Day = date.Format(layout)
			ev.Begin, ev.End = begin, end
			ev.Place = m.values[2]
		})
		if err != nil {
			m.mode = viewing
			m.fault = err.Error()
			return m
		}
	}

	m.mode = viewing
	m = m.reload(nil).land(date)

	for n, i := range m.daily() {
		if m.events[i].ID == id {
			m.index = n
		}
	}

	return m
}

func (m model) browse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.fault = ""

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		m = m.land(m.day.AddDate(0, 0, -1))
	case "l", "right":
		m = m.land(m.day.AddDate(0, 0, 1))
	case "k", "up":
		if m.side {
			m = m.slide(-1)
		} else {
			m = m.land(m.day.AddDate(0, 0, -7))
		}
	case "j", "down":
		if m.side {
			m = m.slide(1)
		} else {
			m = m.land(m.day.AddDate(0, 0, 7))
		}
	case "[", "p":
		m = m.land(shift(m.day, -1))
	case "]", "n":
		m = m.land(shift(m.day, 1))
	case "t":
		m = m.land(midnight(time.Now()))
	case "tab":
		if len(m.daily()) > 0 {
			m.side = !m.side
		}
	case "a":
		return m.compose(0, []string{"", "", "", m.day.Format(layout)}), textinput.Blink
	case "e":
		if ev, ok := m.chosen(); ok {
			return m.compose(ev.ID, []string{ev.Title, slot(ev), ev.Place, ev.Day}), textinput.Blink
		}
	case "d", "x":
		if _, ok := m.chosen(); ok {
			m.mode = asking
		}
	}

	return m, nil
}

func (m model) form(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.fault = ""

	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = viewing
		return m, nil
	case "enter":
		m.values[m.stage] = strings.TrimSpace(m.input.Value())

		if m.stage < len(labels)-1 {
			m.stage++
			m.input = field(labels[m.stage], m.values[m.stage])
			return m, textinput.Blink
		}

		return m.commit(), nil
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

	ev, ok := m.chosen()
	if !ok {
		return m, nil
	}

	_, err := remove(ident(ev.ID))
	m = m.reload(err)

	return m.land(m.day), nil
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
		case forming:
			return m.form(msg)
		case asking:
			return m.confirm(msg)
		}
		return m.browse(msg)
	}

	if m.mode == forming {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) panel() string {
	lines := []string{heading(m.day), ""}
	list := m.daily()
	width := len(strconv.Itoa(len(m.events)))

	var wide int
	if m.width > 0 {
		wide = m.width - 52 - width
		if wide < 12 {
			wide = 12
		}
	}

	if len(list) == 0 {
		lines = append(lines, faint.Render("no events"))
	}

	var at int
	for n, i := range list {
		lead := "  "
		if n == m.index {
			lead = faint.Render("> ")
			if m.side {
				lead = pink.Render("> ")
			}
			at = len(lines)
		}
		lines = append(lines, row(lead, i+1, width, wide, m.events[i]))
	}

	if m.height > 0 {
		lines = slice(lines, at, m.height-4)
	}

	return strings.Join(lines, "\n")
}

func (m model) View() string {
	left := grid(m.day, m.day, marks(m.events), time.Now())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "    ", m.panel())

	var foot string

	switch m.mode {
	case forming:
		note := faint.Render(hints[m.stage])
		if m.fault != "" {
			note = red.Render(m.fault)
		}
		foot = m.input.View() + "\n  " + note
	case asking:
		ev, _ := m.chosen()
		foot = faint.Render("  remove ") + ev.Title + faint.Render("? y/n")
	default:
		foot = faint.Render("  h/l day  j/k week  [ ] month  t today  tab events  a add  e edit  d remove  q quit")
		if m.fault != "" {
			foot = red.Render("  " + m.fault)
		}
	}

	return "\n" + body + "\n\n" + foot
}

func tui() *cobra.Command {
	return &cobra.Command{
		Use:   "tui [date]",
		Short: "Open the interactive interface",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			_, err = tea.NewProgram(model{events: all, day: at}, tea.WithAltScreen()).Run()
			return err
		},
	}
}
