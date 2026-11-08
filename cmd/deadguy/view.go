/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package deadguy

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pink  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

func glyph(name string) string {
	if name == dark {
		return pink.Render("●") + " " + dark
	}
	return pink.Render("○") + " " + light
}

func rows(now state) [][2]string {
	if now.Apps == now.System {
		return [][2]string{{"mode", glyph(now.Apps)}}
	}

	return [][2]string{{"apps", glyph(now.Apps)}, {"system", glyph(now.System)}}
}

func show(now state, via string) {
	fmt.Println()

	for _, row := range rows(now) {
		fmt.Println("  " + faint.Render(fmt.Sprintf("%-8s", row[0])) + row[1])
	}

	fmt.Println("  " + faint.Render(fmt.Sprintf("%-8s", "via")) + faint.Render(via))
	fmt.Println()
}

func echo(before, after state) {
	fmt.Println()

	if before == after {
		for _, row := range rows(after) {
			fmt.Println("  " + faint.Render(fmt.Sprintf("%-8s", row[0])) + row[1] + faint.Render("  already set"))
		}

		fmt.Println()

		return
	}

	merged := before.Apps == before.System && after.Apps == after.System

	pairs := [][3]string{{"apps", before.Apps, after.Apps}, {"system", before.System, after.System}}
	if merged {
		pairs = pairs[:1]
		pairs[0][0] = "mode"
	}

	for _, pair := range pairs {
		line := "  " + faint.Render(fmt.Sprintf("%-8s", pair[0]))

		if pair[1] == pair[2] {
			line += glyph(pair[2]) + faint.Render("  unchanged")
		} else {
			line += faint.Render(pair[1]) + "  " + pink.Render("→") + "  " + glyph(pair[2])
		}

		fmt.Println(line)
	}

	fmt.Println()
}
