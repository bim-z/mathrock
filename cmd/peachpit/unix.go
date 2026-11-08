//go:build unix

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

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func cells() (int, int) {
	size, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || size.Col == 0 || size.Row == 0 || size.Xpixel == 0 || size.Ypixel == 0 {
		return 8, 16
	}

	return max(1, int(size.Xpixel)/int(size.Col)), max(1, int(size.Ypixel)/int(size.Row))
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

	var reply []byte
	limit := time.Now().Add(300 * time.Millisecond)

	for time.Now().Before(limit) {
		left := int(time.Until(limit) / time.Millisecond)
		fds := []unix.PollFd{{Fd: int32(input), Events: unix.POLLIN}}

		ready, err := unix.Poll(fds, left)
		if err == unix.EINTR {
			continue
		}
		if err != nil || ready == 0 {
			break
		}

		chunk := make([]byte, 64)
		count, err := unix.Read(input, chunk)
		if err != nil || count <= 0 {
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
