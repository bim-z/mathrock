package calendar

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	title = lipgloss.NewStyle().Bold(true).Underline(true)
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blue  = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	focus = lipgloss.NewStyle().Bold(true).Reverse(true)
)

func clip(text string, wide int) string {
	runes := []rune(text)
	if wide < 2 || len(runes) <= wide {
		return text
	}
	return string(runes[:wide-1]) + "…"
}

func relative(t time.Time) string {
	diff := int(math.Round(midnight(t).Sub(midnight(time.Now())).Hours() / 24))

	switch {
	case diff == 0:
		return "today"
	case diff == 1:
		return "tomorrow"
	case diff == -1:
		return "yesterday"
	case diff > 1:
		return fmt.Sprintf("in %d days", diff)
	default:
		return fmt.Sprintf("%d days ago", -diff)
	}
}

func heading(t time.Time) string {
	return title.Render(t.Format("Monday, 02 Jan 2006")) + " " + faint.Render(relative(t))
}

func marks(all []event) map[string]int {
	out := map[string]int{}
	for _, ev := range all {
		out[ev.Day]++
	}
	return out
}

func grid(shown, cursor time.Time, dots map[string]int, now time.Time) string {
	first := time.Date(shown.Year(), shown.Month(), 1, 0, 0, 0, 0, time.Local)
	lead := (int(first.Weekday()) + 6) % 7
	total := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()
	picked := cursor.Format(layout)
	current := now.Format(layout)

	lines := []string{
		"  " + title.Render(first.Format("January 2006")),
		"",
		"  " + faint.Render("Mo  Tu  We  Th  Fr  Sa  Su"),
	}

	var cells []string
	for i := 0; i < lead; i++ {
		cells = append(cells, "    ")
	}

	for num := 1; num <= total; num++ {
		stamp := time.Date(first.Year(), first.Month(), num, 0, 0, 0, 0, time.Local).Format(layout)
		text := fmt.Sprintf("%2d", num)

		mark := " "
		if dots[stamp] > 0 {
			mark = blue.Render("•")
		}

		switch {
		case stamp == picked:
			text = focus.Render(text)
		case stamp == current:
			text = pink.Bold(true).Render(text)
		}

		cells = append(cells, text+mark+" ")
	}

	for i := 0; i < len(cells); i += 7 {
		end := i + 7
		if end > len(cells) {
			end = len(cells)
		}
		lines = append(lines, "  "+strings.Join(cells[i:end], ""))
	}

	return strings.Join(lines, "\n")
}

func row(lead string, num, width, wide int, ev event) string {
	text := ev.Title

	var spot string
	if ev.Place != "" {
		spot = "@ " + ev.Place
	}

	if wide > 0 {
		text = clip(text, wide)
		room := wide - len([]rune(text)) - 1
		if room >= 6 {
			spot = clip(spot, room)
		} else {
			spot = ""
		}
	}

	out := lead + faint.Render(fmt.Sprintf("%*d.", width, num)) + " " + pink.Render(fmt.Sprintf("%-11s", label(ev))) + " " + text
	if spot != "" {
		out += " " + faint.Render(spot)
	}

	return out
}

func listing(all []event, keep func(event) bool) []string {
	width := len(strconv.Itoa(len(all)))

	var lines []string
	var last string

	for i, ev := range all {
		if !keep(ev) {
			continue
		}

		if ev.Day != last {
			if last != "" {
				lines = append(lines, "")
			}
			lines = append(lines, "  "+heading(parse(ev.Day)))
			last = ev.Day
		}

		lines = append(lines, row("    ", i+1, width, 0, ev))
	}

	return lines
}

func block(at time.Time, all []event, empty bool) []string {
	stamp := at.Format(layout)
	width := len(strconv.Itoa(len(all)))

	lines := []string{"  " + heading(at)}
	var found bool

	for i, ev := range all {
		if ev.Day == stamp {
			lines = append(lines, row("    ", i+1, width, 0, ev))
			found = true
		}
	}

	if !found {
		if !empty {
			return nil
		}
		lines = append(lines, "    "+faint.Render("no events"))
	}

	return lines
}

func upcoming(all []event, days int) []string {
	today := midnight(time.Now())
	from := today.Format(layout)
	until := today.AddDate(0, 0, days).Format(layout)

	return listing(all, func(ev event) bool { return ev.Day >= from && ev.Day < until })
}

func dash() error {
	all, err := fetch()
	if err != nil {
		return err
	}

	now := time.Now()

	fmt.Println("\n" + grid(now, time.Time{}, marks(all), now) + "\n")

	lines := upcoming(all, 7)
	if len(lines) == 0 {
		fmt.Println(faint.Render("  nothing planned for the next 7 days") + "\n")
		return nil
	}

	fmt.Println(faint.Render("  next 7 days") + "\n")
	fmt.Println(strings.Join(lines, "\n") + "\n")

	return nil
}

func echo(hit []event) error {
	all, err := fetch()
	if err != nil {
		return err
	}

	width := len(strconv.Itoa(len(all)))

	for _, ev := range hit {
		for i, cur := range all {
			if cur.ID == ev.ID {
				fmt.Println(row("  ", i+1, width, 0, cur) + faint.Render("  "+parse(cur.Day).Format("Mon, 02 Jan")))
			}
		}
	}

	return nil
}
