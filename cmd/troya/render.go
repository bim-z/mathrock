/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package troya

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	probing "github.com/prometheus-community/pro-bing"
)

func header(host, addr string) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render("PONG")
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("(" + addr + ")")
	return fmt.Sprintf("%s %s %s\n", title, host, dim)
}

func row(key, val string) string {
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(10).Render(key)
	value := lipgloss.NewStyle().Bold(true).Render(val)
	return label + value
}

func summary(stats *probing.Statistics) string {
	tone := lipgloss.Color("42")
	if stats.PacketLoss > 0 {
		tone = lipgloss.Color("214")
	}
	if stats.PacketLoss >= 50 {
		tone = lipgloss.Color("196")
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(tone).Render(stats.Addr + " statistics"),
		"",
		row("sent", fmt.Sprint(stats.PacketsSent)),
		row("received", fmt.Sprint(stats.PacketsRecv)),
		row("loss", fmt.Sprintf("%.1f%%", stats.PacketLoss)),
		row("min", stats.MinRtt.Round(10*time.Microsecond).String()),
		row("avg", stats.AvgRtt.Round(10*time.Microsecond).String()),
		row("max", stats.MaxRtt.Round(10*time.Microsecond).String()),
		row("stddev", stats.StdDevRtt.Round(10*time.Microsecond).String()),
	}

	if graph := spark(stats.Rtts); graph != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(tone).Render(graph))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tone).
		Padding(0, 2).
		Render(strings.Join(lines, "\n"))
}

func spark(rtts []time.Duration) string {
	if len(rtts) == 0 {
		return ""
	}

	rtts = rtts[max(0, len(rtts)-40):]
	bars := []rune("▁▂▃▄▅▆▇█")
	low, high := rtts[0], rtts[0]
	for _, rtt := range rtts {
		low = min(low, rtt)
		high = max(high, rtt)
	}

	var out strings.Builder
	for _, rtt := range rtts {
		idx := 0
		if high > low {
			idx = int(float64(rtt-low) / float64(high-low) * float64(len(bars)-1))
		}
		out.WriteRune(bars[idx])
	}
	return out.String()
}
