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
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var themes = []string{"dark", "light", "dracula", "pink", "notty", "ascii"}

func resolve(name string) (string, error) {
	if name == "" || name == auto {
		if lipgloss.HasDarkBackground() {
			return "dark", nil
		}
		return "light", nil
	}

	for _, item := range themes {
		if item == name {
			return name, nil
		}
	}

	return "", fmt.Errorf("unknown theme %q, choose one of: auto, %s", name, strings.Join(themes, ", "))
}

func forge(style string) func(string, int) (string, error) {
	cache := map[int]*glamour.TermRenderer{}

	return func(text string, width int) (string, error) {
		rend, ok := cache[width]
		if !ok {
			var err error

			rend, err = glamour.NewTermRenderer(
				glamour.WithStandardStyle(style),
				glamour.WithWordWrap(width),
			)
			if err != nil {
				return "", err
			}

			cache[width] = rend
		}

		return rend.Render(text)
	}
}
