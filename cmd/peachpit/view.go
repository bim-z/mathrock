/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	title = lipgloss.NewStyle().Bold(true).Underline(true)
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

const starter = `---
title: Hello Pitch
author: Your Name
theme: auto
---

# Hello Pitch

A presentation rendered right in your terminal.

---

## How it works

- Separate slides with a line of three dashes
- Write plain markdown
- Press l and h to move between slides
- Press tab for the overview

---

## Code

~~~go
func main() {
	fmt.Println("hello")
}
~~~

---

## Images

Put an image on its own line:

~~~
![caption](path/to/image.png)
~~~

Terminals with Kitty, iTerm2 or Sixel support will show it.

---

# Thank you
`

func width() int {
	size, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || size < 20 {
		return 80
	}
	return size
}

func listout(d deck) {
	total := len(d.Pages)
	size := len(strconv.Itoa(total))

	fmt.Println("\n  " + title.Render(d.Title) + " " + faint.Render(fmt.Sprintf("%d slides", total)) + "\n")

	for i, page := range d.Pages {
		fmt.Println("  " + faint.Render(fmt.Sprintf("%*d.", size, i+1)) + " " + heading(page))
	}

	fmt.Println()
}

func printout(d deck, style string, wide, only int) error {
	engine := forge(style)

	for i, page := range d.Pages {
		if only > 0 && i+1 != only {
			continue
		}

		made, err := layout(page, wide, 0, d.Base, off, 8, 16, engine)
		if err != nil {
			return err
		}

		fmt.Println("\n  " + faint.Render(fmt.Sprintf("%d/%d", i+1, len(d.Pages))))
		fmt.Println(strings.Join(made.Lines, "\n"))
	}

	fmt.Println()

	return nil
}

func infoout(mode string) error {
	proto, err := detect(mode)
	if err != nil {
		return err
	}

	cx, cy := cells()
	cols, _, _ := term.GetSize(int(os.Stdout.Fd()))

	name := os.Getenv("TERM_PROGRAM")
	if name == "" {
		name = os.Getenv("TERM")
	}

	row := func(key, value string) {
		fmt.Println("  " + faint.Render(fmt.Sprintf("%-9s", key)) + " " + value)
	}

	paint := green
	if proto == off {
		paint = red
	}

	fmt.Println()
	row("images", paint.Render(proto))
	row("terminal", name)
	row("cell", fmt.Sprintf("%dx%d px", cx, cy))
	row("columns", strconv.Itoa(cols))
	fmt.Println()

	return nil
}
