/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

const (
	para = iota
	bullet
	entry
	note
	plain
)

type block struct {
	Kind  int
	Text  string
	Level int
	Parts []string
	Date  string
}

type section struct {
	Title  string
	Blocks []block
	Hidden bool
}

type doc struct {
	Name     string
	Title    string
	Email    string
	Phone    string
	Place    string
	Links    []string
	Look     string
	Sections []section
}

type span struct {
	Text string
	Bold bool
	Ital bool
}

type word struct {
	Text string
	Bold bool
	Ital bool
	Glue bool
}

var (
	fence  = regexp.MustCompile("^\\s{0,3}(```|~~~)")
	item   = regexp.MustCompile(`^(\s*)([-*+•])\s+(.*)$`)
	label  = regexp.MustCompile(`^([^:*\[\]]{1,40}):\s+(.*)$`)
	stamp  = regexp.MustCompile(`(?i)\b(19|20)\d{2}\b|\b(present|current|now|sekarang|ongoing)\b`)
	phone  = regexp.MustCompile(`^\+?[\d\s().-]{7,}$`)
	pair   = regexp.MustCompile(`^([A-Za-z_]+):\s*(.*)$`)
	listed = regexp.MustCompile(`^\s*-\s+(.*)$`)
)

func ruled(text string) bool {
	text = strings.ReplaceAll(strings.TrimSpace(text), " ", "")
	if len(text) < 3 || strings.Trim(text, string(text[0])) != "" {
		return false
	}
	return strings.ContainsRune("-*_", rune(text[0]))
}

func skilly(title string) bool {
	low := strings.ToLower(title)
	for _, key := range []string{"skill", "keahlian", "kemampuan", "competenc", "technolog"} {
		if strings.Contains(low, key) {
			return true
		}
	}
	return false
}

func split(text string) (string, []string) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	var head, chunks, cur []string
	var open, started bool

	for _, line := range lines {
		if fence.MatchString(line) {
			open = !open
		}

		if !open && strings.HasPrefix(line, "## ") {
			if started {
				chunks = append(chunks, strings.Join(cur, "\n"))
			}
			started = true
			cur = []string{line}
			continue
		}

		if started {
			cur = append(cur, line)
		} else {
			head = append(head, line)
		}
	}

	if started {
		chunks = append(chunks, strings.Join(cur, "\n"))
	}

	return strings.Join(head, "\n"), chunks
}

func assign(out *doc, key, value string) {
	value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), `"'`))
	if value == "" {
		return
	}

	switch strings.ToLower(key) {
	case "name":
		out.Name = value
	case "title", "headline", "role":
		out.Title = value
	case "template", "look", "style":
		out.Look = strings.ToLower(value)
	case "email":
		out.Email = value
	case "phone":
		out.Phone = value
	case "location", "city", "address":
		out.Place = value
	case "links", "linkedin", "github", "website", "portfolio", "url":
		for _, part := range strings.Split(value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out.Links = append(out.Links, part)
			}
		}
	}
}

func front(head string, out *doc) {
	lines := strings.Split(head, "\n")
	idx := 0

	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}

	if idx < len(lines) && strings.TrimSpace(lines[idx]) == "---" {
		end := -1

		for pos := idx + 1; pos < len(lines); pos++ {
			if strings.TrimSpace(lines[pos]) == "---" {
				end = pos
				break
			}
		}

		if end > 0 {
			var key string

			for _, line := range lines[idx+1 : end] {
				if found := pair.FindStringSubmatch(line); found != nil {
					key = found[1]
					assign(out, key, found[2])
					continue
				}

				if found := listed.FindStringSubmatch(line); found != nil && key != "" {
					assign(out, key, found[1])
				}
			}

			idx = end + 1
		}
	}

	for _, line := range lines[min(idx, len(lines)):] {
		text := strings.TrimSpace(line)

		switch {
		case text == "" || ruled(text):
		case strings.HasPrefix(text, "# "):
			if out.Name == "" {
				out.Name = strings.TrimSpace(text[2:])
			}
		default:
			for _, part := range strings.Split(text, "|") {
				classify(out, strings.TrimSpace(part))
			}
		}
	}
}

func classify(out *doc, part string) {
	switch {
	case part == "":
	case strings.Contains(part, "@") && !strings.Contains(part, " ") && out.Email == "":
		out.Email = part
	case phone.MatchString(part) && out.Phone == "":
		out.Phone = part
	case !strings.Contains(part, " ") && strings.Contains(part, "."):
		out.Links = append(out.Links, part)
	case out.Place == "":
		out.Place = part
	case out.Title == "":
		out.Title = part
	}
}

func entries(text string) block {
	var parts []string

	for _, part := range strings.Split(text, "|") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}

	out := block{Kind: entry}

	if len(parts) > 1 && stamp.MatchString(parts[len(parts)-1]) && len(parts[len(parts)-1]) <= 40 {
		out.Date = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	out.Parts = parts

	return out
}

func blocks(title string, lines []string) []block {
	var out []block
	var open bool

	flat := skilly(title)

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t\r")
		text := strings.TrimSpace(line)

		switch {
		case text == "":
			open = false
		case ruled(text):
			open = false
		case strings.HasPrefix(text, "#### "):
			out = append(out, block{Kind: note, Text: strings.TrimSpace(text[5:])})
			open = false
		case strings.HasPrefix(text, "### "):
			out = append(out, entries(strings.TrimSpace(text[4:])))
			open = false
		case item.MatchString(line):
			found := item.FindStringSubmatch(line)
			body := strings.TrimSpace(found[3])
			kind := bullet

			if flat {
				kind = plain
				if parts := label.FindStringSubmatch(body); parts != nil && !strings.HasPrefix(body, "**") {
					body = "**" + parts[1] + ":** " + parts[2]
				}
			}

			out = append(out, block{Kind: kind, Text: body, Level: min(len(found[1])/2, 2)})
			open = false
		case len(out) > 0 && open && (out[len(out)-1].Kind == para || out[len(out)-1].Kind == bullet || out[len(out)-1].Kind == plain):
			last := &out[len(out)-1]
			last.Text += " " + text
		case len(out) > 0 && !open && unicode.IsSpace(rune(line[0])) && (out[len(out)-1].Kind == bullet || out[len(out)-1].Kind == plain):
			out[len(out)-1].Text += " " + text
			open = true
		default:
			out = append(out, block{Kind: para, Text: text})
			open = true
		}
	}

	return out
}

func parse(text string) (doc, error) {
	head, chunks := split(text)

	var out doc

	front(head, &out)

	for _, chunk := range chunks {
		lines := strings.Split(chunk, "\n")
		title := strings.TrimSpace(strings.TrimPrefix(lines[0], "## "))

		out.Sections = append(out.Sections, section{Title: title, Blocks: blocks(title, lines[1:])})
	}

	if out.Name == "" {
		return out, errors.New("the resume has no name, add 'name:' to the front matter")
	}

	return out, nil
}

func (d doc) shown() []section {
	var out []section

	for _, sec := range d.Sections {
		if !sec.Hidden {
			out = append(out, sec)
		}
	}

	return out
}

func (d doc) contact() []string {
	var out []string

	for _, part := range []string{d.Email, d.Phone, d.Place} {
		if part != "" {
			out = append(out, part)
		}
	}

	return append(out, d.Links...)
}

func pick(text string, at int) (string, int, bool) {
	runes := []rune(text)

	end := -1
	for pos := at + 1; pos < len(runes)-1; pos++ {
		if runes[pos] == ']' && runes[pos+1] == '(' {
			end = pos
			break
		}
	}
	if end < 0 {
		return "", 0, false
	}

	for pos := end + 2; pos < len(runes); pos++ {
		if runes[pos] == ')' {
			name := string(runes[at+1 : end])
			link := string(runes[end+2 : pos])

			if name == link || name == "" {
				return link, pos, true
			}

			return name + " (" + link + ")", pos, true
		}
	}

	return "", 0, false
}

func spans(text string) []span {
	runes := []rune(text)

	var out []span
	var buf []rune

	bold, ital := false, false

	flush := func() {
		if len(buf) > 0 {
			out = append(out, span{Text: string(buf), Bold: bold, Ital: ital})
			buf = nil
		}
	}

	space := func(at int) bool {
		return at < 0 || at >= len(runes) || unicode.IsSpace(runes[at])
	}

	for idx := 0; idx < len(runes); idx++ {
		char := runes[idx]

		switch {
		case char == '\\' && idx+1 < len(runes):
			idx++
			buf = append(buf, runes[idx])
		case char == '`':
		case char == '*' && idx+1 < len(runes) && runes[idx+1] == '*':
			if (!bold && !space(idx+2)) || (bold && !space(idx-1)) {
				flush()
				bold = !bold
				idx++
			} else {
				buf = append(buf, char)
			}
		case char == '*':
			if (!ital && !space(idx+1)) || (ital && !space(idx-1)) {
				flush()
				ital = !ital
			} else {
				buf = append(buf, char)
			}
		case char == '[':
			if shown, end, ok := pick(text, idx); ok {
				buf = append(buf, []rune(shown)...)
				idx = end
			} else {
				buf = append(buf, char)
			}
		default:
			buf = append(buf, char)
		}
	}

	flush()

	return out
}

func flat(text string) string {
	var out strings.Builder

	for _, part := range spans(text) {
		out.WriteString(part.Text)
	}

	return out.String()
}

func words(parts []span, force bool) []word {
	var out []word

	tail := true

	for _, part := range parts {
		fields := strings.Fields(part.Text)
		lead := part.Text != "" && unicode.IsSpace([]rune(part.Text)[0])

		for idx, text := range fields {
			out = append(out, word{
				Text: text,
				Bold: part.Bold || force,
				Ital: part.Ital,
				Glue: idx == 0 && !lead && !tail,
			})
		}

		if len(fields) == 0 {
			tail = true
			continue
		}

		last := []rune(part.Text)
		tail = unicode.IsSpace(last[len(last)-1])
	}

	if len(out) > 0 {
		out[0].Glue = false
	}

	return out
}

func fill(list []word, first, rest float64, size func(word) float64, gap float64) [][]word {
	var lines [][]word
	var cur []word

	used := 0.0
	limit := first

	for _, item := range list {
		wide := size(item)
		add := gap

		if item.Glue || len(cur) == 0 {
			add = 0
		}

		if len(cur) > 0 && used+add+wide > limit {
			lines = append(lines, cur)
			cur, used, limit = nil, 0, rest
			add = 0
			item.Glue = false
		}

		cur = append(cur, item)
		used += add + wide
	}

	if len(cur) > 0 {
		lines = append(lines, cur)
	}

	return lines
}
