//go:build windows

/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func cells() (int, int) {
	return 8, 16
}

func query() bool {
	input := int(os.Stdin.Fd())
	if !term.IsTerminal(input) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}

	state, err := term.MakeRaw(input)
	if err != nil {
		return false
	}
	defer func() { _ = term.Restore(input, state) }()

	if _, err := os.Stdout.WriteString("\x1b[c"); err != nil {
		return false
	}

	handle := windows.Handle(input)

	var reply []byte
	limit := time.Now().Add(300 * time.Millisecond)

	for time.Now().Before(limit) {
		left := uint32(time.Until(limit) / time.Millisecond)

		event, err := windows.WaitForSingleObject(handle, left)
		if err != nil || event != windows.WAIT_OBJECT_0 {
			break
		}

		chunk := make([]byte, 64)

		var count uint32
		if err := windows.ReadFile(handle, chunk, &count, nil); err != nil || count == 0 {
			break
		}

		reply = append(reply, chunk[:count]...)
		if bytes.Contains(reply, []byte("\x1b[?")) && bytes.HasSuffix(reply, []byte("c")) {
			break
		}
	}

	start := bytes.Index(reply, []byte("\x1b[?"))
	if start < 0 {
		return false
	}

	body := strings.TrimSuffix(string(reply[start+3:]), "c")
	for _, attr := range strings.Split(body, ";") {
		if attr == "4" {
			return true
		}
	}

	return false
}

func detect(mode string) (string, error) {
	switch mode {
	case "", auto:
	case off, kitty, sixel, iterm:
		return mode, nil
	default:
		return "", fmt.Errorf("unknown image mode %q, choose one of: auto, kitty, sixel, iterm, off", mode)
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return off, nil
	}

	kind := os.Getenv("TERM")
	program := os.Getenv("TERM_PROGRAM")

	if os.Getenv("TMUX") != "" || strings.HasPrefix(kind, "screen") {
		return off, nil
	}

	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "" || kind == "xterm-kitty" || kind == "xterm-ghostty" || program == "ghostty":
		return kitty, nil
	case program == "iTerm.app" || program == "WezTerm" || program == "mintty" || os.Getenv("LC_TERMINAL") == "iTerm2":
		return iterm, nil
	case strings.Contains(kind, "sixel") || kind == "foot" || kind == "foot-extra" || kind == "mlterm":
		return sixel, nil
	}

	if query() {
		return sixel, nil
	}

	return off, nil
}
