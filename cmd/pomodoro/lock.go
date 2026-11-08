/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"errors"
	"os/exec"
	"runtime"
)

func lock() error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pmset", "displaysleepnow").Run()
	case "windows":
		return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
	}
	tools := [][]string{
		{"loginctl", "lock-session"},
		{"xdg-screensaver", "lock"},
		{"gnome-screensaver-command", "-l"},
	}
	for _, args := range tools {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		if exec.Command(args[0], args[1:]...).Run() == nil {
			return nil
		}
	}
	return errors.New("no lock command found")
}
