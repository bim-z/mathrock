package todo

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	title = lipgloss.NewStyle().Bold(true).Underline(true)
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	amber = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	gold  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func age(t task) string {
	if t.State == closed {
		return ""
	}

	days := int(time.Since(t.Born).Hours() / 24)
	if days < 1 {
		return ""
	}

	return strconv.Itoa(days) + "d"
}

func item(t task) string {
	var glyph, text string

	switch t.State {
	case closed:
		glyph, text = green.Render("✓"), faint.Render(t.Text)
	case begun:
		glyph, text = amber.Render("◐"), t.Text
	default:
		glyph, text = pink.Render("☐"), t.Text
	}

	out := glyph + " " + text
	if t.Star {
		out += " " + gold.Render("★")
	}
	if span := age(t); span != "" {
		out += " " + faint.Render(span)
	}

	return out
}

func build(all []task, at int, only string) ([]string, int) {
	width := len(strconv.Itoa(len(all)))
	total := map[string]int{}
	finished := map[string]int{}

	for _, t := range all {
		total[t.Board]++
		if t.State == closed {
			finished[t.Board]++
		}
	}

	var lines []string
	var cursor int
	var last string

	for i, t := range all {
		if only != "" && t.Board != only {
			continue
		}

		if t.Board != last {
			if last != "" {
				lines = append(lines, "")
			}
			count := fmt.Sprintf("[%d/%d]", finished[t.Board], total[t.Board])
			lines = append(lines, "  "+title.Render(t.Board)+" "+faint.Render(count))
			last = t.Board
		}

		lead := "  "
		if i == at {
			lead = pink.Render("> ")
			cursor = len(lines)
		}

		num := faint.Render(fmt.Sprintf("%*d.", width, i+1))
		lines = append(lines, lead+"  "+num+" "+item(t))
	}

	return lines, cursor
}

func stats(all []task) string {
	if len(all) == 0 {
		return ""
	}

	var finished, active, waiting int

	for _, t := range all {
		switch t.State {
		case closed:
			finished++
		case begun:
			active++
		default:
			waiting++
		}
	}

	percent := fmt.Sprintf("%d%% of all tasks complete", finished*100/len(all))
	counts := fmt.Sprintf("%d done · %d in progress · %d pending", finished, active, waiting)

	return "  " + faint.Render(percent) + "\n  " + faint.Render(counts)
}

func paint(only string) error {
	all, err := fetch()
	if err != nil {
		return err
	}

	if len(all) == 0 {
		fmt.Println(faint.Render("  no tasks yet, add one with 'todo add'"))
		return nil
	}

	lines, _ := build(all, -1, only)
	if len(lines) == 0 {
		fmt.Println(faint.Render("  no tasks on board " + only))
		return nil
	}

	fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
	fmt.Println(stats(all) + "\n")

	return nil
}
