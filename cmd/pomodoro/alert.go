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
	"os/exec"
	"runtime"
	"strings"
)

const toast = "Add-Type -AssemblyName System.Windows.Forms, System.Drawing; $n = New-Object System.Windows.Forms.NotifyIcon; $n.Icon = [System.Drawing.SystemIcons]::Information; $n.Visible = $true; $n.ShowBalloonTip(5000, 'pomo', '%s', [System.Windows.Forms.ToolTipIcon]::Info); Start-Sleep -Seconds 6; $n.Dispose()"

func spawn(name string, args ...string) {
	path, err := exec.LookPath(name)
	if err != nil {
		return
	}
	cmd := exec.Command(path, args...)
	if cmd.Start() == nil {
		go cmd.Wait()
	}
}

func alert(msg string) {
	notify(msg)
	chime()
}

func notify(msg string) {
	switch runtime.GOOS {
	case "darwin":
		spawn("osascript", "-e", fmt.Sprintf("display notification %q with title \"pomo\"", msg))
	case "windows":
		safe := strings.ReplaceAll(msg, "'", "''")
		spawn("powershell", "-NoProfile", "-Command", fmt.Sprintf(toast, safe))
	default:
		spawn("notify-send", "pomo", msg)
	}
}

func chime() {
	fmt.Print("\a")
	switch runtime.GOOS {
	case "darwin":
		spawn("afplay", "/System/Library/Sounds/Glass.aiff")
	case "windows":
		spawn("powershell", "-NoProfile", "-Command", "[console]::beep(880,250); [console]::beep(660,250)")
	default:
		if _, err := exec.LookPath("paplay"); err == nil {
			spawn("paplay", "/usr/share/sounds/freedesktop/stereo/complete.oga")
			return
		}
		spawn("canberra-gtk-play", "-i", "complete")
	}
}
