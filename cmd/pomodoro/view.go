/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	accent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	good   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))
	warn   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("221"))
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

var tint = map[string]lipgloss.Style{
	"focus": accent,
	"grace": warn,
	"short": good,
	"long":  good,
}

var label = map[string]string{
	"focus": "focus",
	"grace": "locking",
	"short": "short break",
	"long":  "long break",
}

func clock(span time.Duration) string {
	secs := int(span.Round(time.Second).Seconds())
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

func bar(st *state) string {
	width := 20
	filled := 0
	if st.Total > 0 {
		filled = int(float64(width) * (1 - float64(st.remain())/float64(st.Total)))
	}
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return tint[st.Phase].Render(strings.Repeat("━", filled)) + dim.Render(strings.Repeat("─", width-filled))
}

func line(st *state) string {
	parts := []string{tint[st.Phase].Width(12).Render(label[st.Phase])}
	if st.Phase == "focus" {
		parts = append(parts, dim.Render(fmt.Sprintf("%d/%d", st.Round, st.Cycle)))
	}
	parts = append(parts, clock(st.remain()), bar(st))
	if st.Paused {
		parts = append(parts, dim.Render("paused"))
	}
	return strings.Join(parts, "  ")
}

func draw(st *state) {
	fmt.Print("\r\x1b[K" + line(st))
}
