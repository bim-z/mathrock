/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package charmer

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func clean(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	if !strings.Contains(target, "://") {
		return "https://" + target
	}

	return target
}

func spawn(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() { _ = cmd.Wait() }()

	return nil
}

func launch(target string) error {
	switch runtime.GOOS {
	case "windows":
		return spawn("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		return spawn("open", target)
	}

	for _, name := range []string{"xdg-open", "termux-open-url"} {
		if path, err := exec.LookPath(name); err == nil {
			return spawn(path, target)
		}
	}

	return errors.New("no browser launcher found, install xdg-utils")
}
