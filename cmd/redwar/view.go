/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package redwar

import (
	"bytes"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	rend = lipgloss.NewRenderer(os.Stderr)
	good = rend.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))
	bad  = rend.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	dim  = rend.NewStyle().Foreground(lipgloss.Color("243"))
)

func size(count int) string {
	units := []string{"B", "KB", "MB", "GB"}
	val := float64(count)
	pos := 0
	for val >= 1024 && pos < len(units)-1 {
		val /= 1024
		pos++
	}
	if pos == 0 {
		return fmt.Sprintf("%d B", count)
	}
	return fmt.Sprintf("%.1f %s", val, units[pos])
}

func report(data []byte, via string) {
	lines := bytes.Count(data, []byte("\n"))
	if data[len(data)-1] != '\n' {
		lines++
	}
	unit := "lines"
	if lines == 1 {
		unit = "line"
	}
	note := fmt.Sprintf("%s  %d %s  via %s", size(len(data)), lines, unit, via)
	fmt.Fprintln(os.Stderr, good.Render("copied")+"  "+dim.Render(note))
}
