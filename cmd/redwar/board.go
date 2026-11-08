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
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const script = "[Console]::InputEncoding = New-Object System.Text.UTF8Encoding $false; Set-Clipboard -Value ([Console]::In.ReadToEnd())"

func backends() [][]string {
	switch runtime.GOOS {
	case "darwin":
		return [][]string{{"pbcopy"}}
	case "windows":
		return [][]string{{"powershell", "-NoProfile", "-NonInteractive", "-Command", script}}
	}
	var list [][]string
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		list = append(list, []string{"wl-copy"})
	}
	return append(list,
		[]string{"xclip", "-selection", "clipboard"},
		[]string{"xsel", "--clipboard", "--input"},
		[]string{"wl-copy"},
		[]string{"termux-clipboard-set"},
		[]string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script},
	)
}

func send(data []byte, remote bool) (string, error) {
	if remote || os.Getenv("SSH_CONNECTION") != "" {
		return osc(data)
	}
	via, err := native(data)
	if err == nil {
		return via, nil
	}
	if via, fallback := osc(data); fallback == nil {
		return via, nil
	}
	return "", err
}

func native(data []byte) (string, error) {
	var last error
	for _, args := range backends() {
		path, err := exec.LookPath(args[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, args[1:]...)
		cmd.Stdin = bytes.NewReader(data)
		if err := cmd.Run(); err != nil {
			last = err
			continue
		}
		return args[0], nil
	}
	if last != nil {
		return "", last
	}
	return "", errors.New("no clipboard tool found")
}

func osc(data []byte) (string, error) {
	out := os.Stderr
	if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		defer tty.Close()
		out = tty
	}
	seq := fmt.Sprintf("\x1b]52;c;%s\x07", base64.StdEncoding.EncodeToString(data))
	if os.Getenv("TMUX") != "" {
		seq = "\x1bPtmux;\x1b" + seq + "\x1b\\"
	}
	_, err := fmt.Fprint(out, seq)
	return "osc52", err
}
