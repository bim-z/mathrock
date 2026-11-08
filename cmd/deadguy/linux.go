//go:build linux

/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package deadguy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const split = false

type backend struct {
	Name  string
	Read  func() (string, error)
	Write func(string) error
}

func quote(text string) string {
	return strings.Trim(strings.TrimSpace(text), `'"`)
}

func installed(theme string) bool {
	if theme == "Adwaita" || theme == "Adwaita-dark" {
		return true
	}

	home, _ := os.UserHomeDir()

	for _, root := range []string{
		filepath.Join(home, ".themes"),
		filepath.Join(home, ".local", "share", "themes"),
		"/usr/share/themes",
		"/usr/local/share/themes",
	} {
		if info, err := os.Stat(filepath.Join(root, theme)); err == nil && info.IsDir() {
			return true
		}
	}

	return false
}

func variant(theme, name string) (string, bool) {
	base := theme
	if strings.HasSuffix(strings.ToLower(base), "-dark") {
		base = base[:len(base)-len("-dark")]
	}

	if name == light {
		return base, installed(base)
	}

	for _, suffix := range []string{"-dark", "-Dark"} {
		if installed(base + suffix) {
			return base + suffix, true
		}
	}

	return base + "-dark", false
}

func gsettings(schema, key string) (string, error) {
	out, err := shell("gsettings", "get", schema, key)
	return quote(out), err
}

func themed(name string, get func() (string, error), set func(string) error) backend {
	return backend{
		Name: name,
		Read: func() (string, error) {
			theme, err := get()
			if err != nil {
				return "", err
			}
			if strings.Contains(strings.ToLower(theme), "dark") {
				return dark, nil
			}
			return light, nil
		},
		Write: func(target string) error {
			theme, err := get()
			if err != nil {
				return err
			}

			next, ok := variant(theme, target)
			if !ok {
				return fmt.Errorf("no %s variant of the theme %s is installed", target, theme)
			}

			if next == theme {
				return nil
			}

			return set(next)
		},
	}
}

func gnome() backend {
	const schema = "org.gnome.desktop.interface"

	look := func() (string, error) {
		return gsettings(schema, "gtk-theme")
	}

	return backend{
		Name: "GNOME",
		Read: func() (string, error) {
			scheme, err := gsettings(schema, "color-scheme")
			if err == nil {
				if strings.Contains(scheme, "dark") {
					return dark, nil
				}
				if scheme == "prefer-light" {
					return light, nil
				}
			}

			theme, err := look()
			if err != nil {
				return "", err
			}
			if strings.Contains(strings.ToLower(theme), "dark") {
				return dark, nil
			}

			return light, nil
		},
		Write: func(target string) error {
			value := "default"
			if target == dark {
				value = "prefer-dark"
			}

			_, err := shell("gsettings", "set", schema, "color-scheme", value)

			theme, terr := look()
			if terr != nil {
				return err
			}

			next, ok := variant(theme, target)
			if !ok || next == theme {
				return err
			}

			if _, serr := shell("gsettings", "set", schema, "gtk-theme", next); serr != nil && err != nil {
				return serr
			}

			return nil
		},
	}
}

func plasma() backend {
	read := func() (string, error) {
		for _, tool := range []string{"kreadconfig6", "kreadconfig5"} {
			out, err := shell(tool, "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme")
			if err == nil {
				return out, nil
			}
		}
		return "", errors.New("neither kreadconfig6 nor kreadconfig5 was found")
	}

	return backend{
		Name: "KDE Plasma",
		Read: func() (string, error) {
			scheme, err := read()
			if err != nil {
				return "", err
			}
			if strings.Contains(strings.ToLower(scheme), "dark") {
				return dark, nil
			}
			return light, nil
		},
		Write: func(target string) error {
			scheme, global := "BreezeLight", "org.kde.breeze.desktop"
			if target == dark {
				scheme, global = "BreezeDark", "org.kde.breezedark.desktop"
			}

			if _, err := shell("plasma-apply-colorscheme", scheme); err == nil {
				return nil
			}

			if _, err := shell("lookandfeeltool", "-a", global); err != nil {
				return errors.New("neither plasma-apply-colorscheme nor lookandfeeltool could apply the change")
			}

			return nil
		},
	}
}

func xfce() backend {
	get := func() (string, error) {
		return shell("xfconf-query", "-c", "xsettings", "-p", "/Net/ThemeName")
	}
	set := func(theme string) error {
		_, err := shell("xfconf-query", "-c", "xsettings", "-p", "/Net/ThemeName", "-s", theme)
		return err
	}

	return themed("Xfce", get, set)
}

func schema(label, path string) backend {
	get := func() (string, error) {
		return gsettings(path, "gtk-theme")
	}
	set := func(theme string) error {
		_, err := shell("gsettings", "set", path, "gtk-theme", theme)
		return err
	}

	return themed(label, get, set)
}

func pick() (backend, error) {
	desk := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	if desk == "" {
		desk = strings.ToLower(os.Getenv("DESKTOP_SESSION"))
	}

	switch {
	case desk == "":
		return backend{}, errors.New("no desktop environment was detected, XDG_CURRENT_DESKTOP is empty")
	case strings.Contains(desk, "kde") || strings.Contains(desk, "plasma"):
		return plasma(), nil
	case strings.Contains(desk, "xfce"):
		return xfce(), nil
	case strings.Contains(desk, "mate"):
		return schema("MATE", "org.mate.interface"), nil
	case strings.Contains(desk, "cinnamon"):
		return schema("Cinnamon", "org.cinnamon.desktop.interface"), nil
	}

	if _, err := exec.LookPath("gsettings"); err != nil {
		return backend{}, errors.New("this desktop is not supported, gsettings was not found")
	}

	return gnome(), nil
}

func engine() string {
	picked, err := pick()
	if err != nil {
		return "unsupported"
	}
	return picked.Name
}

func current() (state, error) {
	picked, err := pick()
	if err != nil {
		return state{}, err
	}

	name, err := picked.Read()
	if err != nil {
		return state{}, err
	}

	return state{Apps: name, System: name}, nil
}

func assign(name, _ string) error {
	picked, err := pick()
	if err != nil {
		return err
	}

	return picked.Write(name)
}
