package note

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	title = lipgloss.NewStyle().Bold(true).Underline(true)
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blue  = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	gold  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	frame = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)
)

func cols() int {
	wide, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || wide <= 0 {
		return 80
	}
	return wide
}

func fit(width int) int {
	wide := width - 24
	if wide < 20 {
		return 20
	}
	return wide
}

func clip(text string, wide int) string {
	runes := []rune(text)
	if len(runes) <= wide {
		return text
	}
	return string(runes[:wide-1]) + "…"
}

func first(body string) (string, int) {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	return strings.TrimSpace(lines[0]), len(lines) - 1
}

func count(num int, word string) string {
	if num != 1 {
		word += "s"
	}
	return strconv.Itoa(num) + " " + word
}

func age(n note) string {
	days := int(time.Since(n.Born).Hours() / 24)
	if days < 1 {
		return ""
	}
	return strconv.Itoa(days) + "d"
}

func item(n note, wide int) string {
	line, more := first(n.Body)

	out := blue.Render("●") + " " + clip(line, wide)
	if more > 0 {
		out += " " + faint.Render(fmt.Sprintf("(+%d more)", more))
	}
	if n.Star {
		out += " " + gold.Render("★")
	}
	if span := age(n); span != "" {
		out += " " + faint.Render(span)
	}

	return out
}

func build(all []note, at int, keep func(note) bool, wide int) ([]string, int) {
	width := len(strconv.Itoa(len(all)))
	total := map[string]int{}

	for _, n := range all {
		if keep == nil || keep(n) {
			total[n.Board]++
		}
	}

	var lines []string
	var cursor int
	var last string

	for i, n := range all {
		if keep != nil && !keep(n) {
			continue
		}

		if n.Board != last {
			if last != "" {
				lines = append(lines, "")
			}
			badge := fmt.Sprintf("[%d]", total[n.Board])
			lines = append(lines, "  "+title.Render(n.Board)+" "+faint.Render(badge))
			last = n.Board
		}

		lead := "  "
		if i == at {
			lead = pink.Render("> ")
			cursor = len(lines)
		}

		num := faint.Render(fmt.Sprintf("%*d.", width, i+1))
		lines = append(lines, lead+"  "+num+" "+item(n, wide))
	}

	return lines, cursor
}

func stats(all []note) string {
	if len(all) == 0 {
		return ""
	}

	boards := map[string]bool{}
	var starred int

	for _, n := range all {
		boards[n.Board] = true
		if n.Star {
			starred++
		}
	}

	line := count(len(all), "note") + " · " + count(len(boards), "board") + " · " + strconv.Itoa(starred) + " starred"

	return "  " + faint.Render(line)
}

func detail(n note, num, width int) string {
	size := width - 8
	if size > 72 {
		size = 72
	}
	if size < 20 {
		size = 20
	}

	head := "  " + title.Render(n.Board) + faint.Render(fmt.Sprintf("  #%d  %s", num, n.Born.Format("02 Jan 2006 15:04")))
	if n.Star {
		head += " " + gold.Render("★")
	}

	return head + "\n\n" + frame.Width(size).Render(n.Body)
}

func paint(keep func(note) bool, none string) error {
	all, err := fetch()
	if err != nil {
		return err
	}

	if len(all) == 0 {
		fmt.Println(faint.Render("  no notes yet, add one with 'note add'"))
		return nil
	}

	lines, _ := build(all, -1, keep, fit(cols()))
	if len(lines) == 0 {
		fmt.Println(faint.Render("  " + none))
		return nil
	}

	fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
	fmt.Println(stats(all) + "\n")

	return nil
}
