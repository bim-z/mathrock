/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package chmod

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	title = lipgloss.NewStyle().Bold(true).Underline(true)
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	amber = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

var owners = [3]string{"owner", "group", "others"}

func labels(dir bool) [3]string {
	if dir {
		return [3]string{"list", "modify", "enter"}
	}
	return [3]string{"read", "write", "run"}
}

func kind(dir bool) string {
	if dir {
		return "folder"
	}
	return "file"
}

func sym(bits uint32) string {
	out := []byte("---------")

	for idx := 0; idx < 9; idx++ {
		if bits&(1<<uint(8-idx)) != 0 {
			out[idx] = "rwx"[idx%3]
		}
	}

	flag := func(at int, bit uint32, low, high byte) {
		if bits&bit == 0 {
			return
		}
		if out[at] == '-' {
			out[at] = high
			return
		}
		out[at] = low
	}

	flag(2, setuid, 's', 'S')
	flag(5, setgid, 's', 'S')
	flag(8, sticky, 't', 'T')

	return string(out)
}

func oct(bits uint32) string {
	if bits&0o7000 != 0 {
		return fmt.Sprintf("%04o", bits)
	}
	return fmt.Sprintf("%03o", bits)
}

func center(text string, wide int) string {
	gap := wide - utf8.RuneCountInString(text)
	if gap <= 0 {
		return text
	}

	left := gap / 2

	return strings.Repeat(" ", left) + text + strings.Repeat(" ", gap-left)
}

func join(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}

	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func trim(name string, wide int) string {
	runes := []rune(name)
	if len(runes) <= wide {
		return name
	}

	return "…" + string(runes[len(runes)-wide+1:])
}

func simple(err error) string {
	var inner *fs.PathError
	if errors.As(err, &inner) {
		return inner.Err.Error()
	}
	return err.Error()
}

func grid(bits, orig uint32, dir bool, row, col int, live bool) string {
	names := labels(dir)
	head := strings.Repeat(" ", 10)

	for _, name := range names {
		head += faint.Render(center(name, 8))
	}

	lines := []string{head, ""}

	for idx, who := range owners {
		line := "  " + faint.Render(fmt.Sprintf("%-8s", who))

		for pos := 0; pos < 3; pos++ {
			bit := uint32(1) << uint(8-(idx*3+pos))
			on := bits&bit != 0
			style, glyph := faint, "·"

			if on {
				style, glyph = pink, "✓"
			}
			if live && on != (orig&bit != 0) {
				style = amber.Bold(true)
			}
			if live && idx == row && pos == col {
				style = style.Reverse(true)
			}

			line += "  " + style.Render(" "+glyph+" ") + "   "
		}

		lines = append(lines, strings.TrimRight(line, " "))
	}

	return strings.Join(lines, "\n")
}

func speak(bits uint32, dir bool) []string {
	names := labels(dir)
	lines := make([]string, 0, 3)

	for idx, who := range owners {
		val := bits >> uint(6-3*idx) & 7

		var acts []string

		for pos, bit := range []uint32{4, 2, 1} {
			if val&bit != 0 {
				acts = append(acts, names[pos])
			}
		}

		text := "cannot do anything"
		if len(acts) > 0 {
			text = "can " + join(acts)
		}

		lines = append(lines, "  "+faint.Render(fmt.Sprintf("%-7s", who))+text)
	}

	return lines
}

func notes(bits uint32, dir bool) []string {
	var out []string

	switch {
	case bits&every == every:
		out = append(out, "everyone can do everything, avoid this unless you are sure")
	case bits&2 != 0:
		out = append(out, "everyone can change this, which is risky")
	}

	if !dir && bits&0o400 == 0 {
		out = append(out, "you cannot read it yourself")
	}
	if dir && bits&0o100 == 0 {
		out = append(out, "you cannot open this folder yourself")
	}

	var special []string

	if bits&setuid != 0 {
		special = append(special, "setuid")
	}
	if bits&setgid != 0 {
		special = append(special, "setgid")
	}
	if bits&sticky != 0 {
		special = append(special, "sticky")
	}
	if len(special) > 0 {
		out = append(out, "special bits are set: "+join(special))
	}

	return out
}

func explain(name string, dir bool, bits uint32) {
	head := "  " + faint.Render(oct(bits)+"  "+sym(bits))

	fmt.Println()
	if name != "" {
		fmt.Println("  " + title.Render(name) + " " + faint.Render(kind(dir)))
		fmt.Println()
	}
	fmt.Println(head)
	fmt.Println()
	fmt.Println(grid(bits, bits, dir, 0, 0, false))
	fmt.Println()
	fmt.Println(strings.Join(speak(bits, dir), "\n"))

	for _, note := range notes(bits, dir) {
		fmt.Println("\n  " + red.Render(note))
	}

	fmt.Println()
}

func report(items []result, dry bool) error {
	wide := 0

	for _, item := range items {
		wide = max(wide, utf8.RuneCountInString(shown(item)))
	}
	wide = min(wide, 48)

	var changed, same, failed int

	fmt.Println()

	for _, item := range items {
		name := fmt.Sprintf("%-*s", wide, trim(shown(item), wide))
		line := "  " + name + "  "

		switch {
		case item.Fault != nil:
			failed++
			line += red.Render(simple(item.Fault))
		case item.Old == item.New:
			same++
			line += faint.Render(sym(item.Old) + "  unchanged")
		default:
			changed++
			line += faint.Render(sym(item.Old)) + "  " + pink.Render("→") + "  " + sym(item.New) + "  " + faint.Render(oct(item.New))
		}

		fmt.Println(line)
	}

	verb := "changed"
	if dry {
		verb = "would change"
	}

	parts := []string{fmt.Sprintf("%d %s", changed, verb), fmt.Sprintf("%d unchanged", same)}
	if failed > 0 {
		parts = append(parts, red.Render(fmt.Sprintf("%d failed", failed)))
	}

	fmt.Println()
	fmt.Println("  " + faint.Render(strings.Join(parts[:2], " · ")) + tail(parts[2:]))

	if dry {
		fmt.Println("  " + amber.Render("dry run, nothing was modified"))
	}

	fmt.Println()

	if failed > 0 {
		return fmt.Errorf("%d of %d items failed", failed, len(items))
	}

	return nil
}

func tail(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return faint.Render(" · ") + parts[0]
}

func shown(item result) string {
	if item.Dir {
		return item.Path + "/"
	}
	return item.Path
}
