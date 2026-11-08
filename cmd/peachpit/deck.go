/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type deck struct {
	Title  string
	Author string
	Theme  string
	Pages  []string
	Base   string
}

var (
	divider = regexp.MustCompile(`^-{3,}\s*$`)
	entry   = regexp.MustCompile(`^([A-Za-z_]+):\s*(.*)$`)
	fence   = regexp.MustCompile("^\\s{0,3}(```|~~~)")
	hashes  = regexp.MustCompile(`^#+\s*`)
)

func front(lines []string) (map[string]string, bool) {
	meta := map[string]string{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		found := entry.FindStringSubmatch(line)
		if found == nil {
			return nil, false
		}

		meta[strings.ToLower(found[1])] = strings.Trim(strings.TrimSpace(found[2]), `"'`)
	}

	return meta, len(meta) > 0
}

func parse(text, base string) deck {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := deck{Base: base}
	start := 0

	if len(lines) > 0 && divider.MatchString(lines[0]) {
		for i := 1; i < len(lines); i++ {
			if !divider.MatchString(lines[i]) {
				continue
			}

			if meta, ok := front(lines[1:i]); ok {
				out.Title = meta["title"]
				out.Author = meta["author"]
				out.Theme = meta["theme"]
				start = i + 1
			}

			break
		}
	}

	var page []string
	var open bool

	flush := func() {
		body := strings.Trim(strings.Join(page, "\n"), "\n")
		if strings.TrimSpace(body) != "" {
			out.Pages = append(out.Pages, body)
		}
		page = nil
	}

	for _, line := range lines[start:] {
		if fence.MatchString(line) {
			open = !open
		}

		if !open && divider.MatchString(line) {
			flush()
			continue
		}

		page = append(page, line)
	}

	flush()

	return out
}

func heading(page string) string {
	var open bool

	for _, line := range strings.Split(page, "\n") {
		if fence.MatchString(line) {
			open = !open
			continue
		}

		text := strings.TrimSpace(line)
		if open || text == "" || strings.HasPrefix(text, "![") {
			continue
		}

		text = strings.TrimSpace(hashes.ReplaceAllString(text, ""))
		text = strings.TrimLeft(text, "-*+> ")
		text = strings.NewReplacer("**", "", "`", "", "__", "").Replace(text)

		if text != "" {
			return text
		}
	}

	return "untitled"
}

func load(path string) (deck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return deck{}, err
	}

	out := parse(string(data), filepath.Dir(path))
	if len(out.Pages) == 0 {
		return deck{}, errors.New("the file contains no slides")
	}

	if out.Title == "" {
		name := filepath.Base(path)
		out.Title = strings.TrimSuffix(name, filepath.Ext(name))
	}

	return out, nil
}
