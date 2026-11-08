//go:build darwin

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
	"strings"
)

const split = false

func engine() string {
	return "macOS appearance"
}

func current() (state, error) {
	name := light

	out, err := shell("defaults", "read", "-g", "AppleInterfaceStyle")
	if err == nil && strings.EqualFold(out, "dark") {
		name = dark
	}

	return state{Apps: name, System: name}, nil
}

func assign(name, _ string) error {
	flag := "false"
	if name == dark {
		flag = "true"
	}

	script := `tell application "System Events" to tell appearance preferences to set dark mode to ` + flag

	if _, err := shell("osascript", "-e", script); err != nil {
		return fmt.Errorf("%w, allow your terminal to control System Events in System Settings, Privacy and Security, Automation", err)
	}

	return nil
}
