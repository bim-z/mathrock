/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type spot struct {
	Row  int
	Col  int
	Rows int
	Wide int
	High int
	Pic  *picture
}

type frame struct {
	Lines []string
	Spots []spot
}

type ref struct {
	Alt string
	Src string
}

var (
	figure = regexp.MustCompile(`^\s*!\[([^\]]*)\]\(\s*<?([^)>]+?)>?(?:\s+"[^"]*")?\s*\)\s*$`)
	token  = regexp.MustCompile(`MRIMG(\d+)X`)
	ansi   = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	tail   = regexp.MustCompile(`(?:\x1b\[[0-9;]*m|\s)+$`)
)

func scan(page string) (string, []ref) {
	var refs []ref
	var out []string
	var open bool

	for _, line := range strings.Split(page, "\n") {
		if fence.MatchString(line) {
			open = !open
		}

		if !open {
			if found := figure.FindStringSubmatch(line); found != nil {
				out = append(out, "", fmt.Sprintf("MRIMG%dX", len(refs)), "")
				refs = append(refs, ref{Alt: found[1], Src: strings.TrimSpace(found[2])})
				continue
			}
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n"), refs
}

func blank(line string) bool {
	return strings.TrimSpace(ansi.ReplaceAllString(line, "")) == ""
}

func trim(lines []string) []string {
	for len(lines) > 0 && blank(lines[0]) {
		lines = lines[1:]
	}

	for len(lines) > 0 && blank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}

	return lines
}

func squeeze(lines []string) []string {
	var out []string

	for _, line := range lines {
		if blank(line) && len(out) > 0 && blank(out[len(out)-1]) {
			continue
		}
		out = append(out, line)
	}

	return trim(out)
}

func layout(page string, width, height int, base, proto string, cx, cy int, engine func(string, int) (string, error)) (frame, error) {
	marked, refs := scan(page)

	limit := min(width-4, 100)
	limit = max(limit, 20)

	text, err := engine(marked, limit)
	if err != nil {
		return frame{}, err
	}

	lines := trim(strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n"))

	for i, line := range lines {
		if blank(line) {
			lines[i] = ""
			continue
		}

		if cut := tail.FindStringIndex(line); cut != nil {
			lines[i] = line[:cut[0]]
			if strings.Contains(line[cut[0]:], "\x1b") {
				lines[i] += "\x1b[0m"
			}
		}
	}

	pad := strings.Repeat(" ", max(0, (width-limit)/2))

	for i, line := range lines {
		if !blank(line) {
			lines[i] = pad + line
		}
	}

	marks := map[int]int{}
	var order []int

	for i, line := range lines {
		if found := token.FindStringSubmatch(ansi.ReplaceAllString(line, "")); found != nil {
			if n, err := strconv.Atoi(found[1]); err == nil && n < len(refs) {
				marks[i] = n
				order = append(order, i)
			}
		}
	}

	sort.Ints(order)

	if len(order) == 0 {
		return frame{Lines: lines}, nil
	}

	if proto == off {
		out := make([]string, 0, len(lines))

		for i, line := range lines {
			if at, ok := marks[i]; ok {
				label := strings.TrimSpace(refs[at].Alt)
				if label == "" {
					label = refs[at].Src
				}
				out = append(out, pad+"\x1b[90m[image] "+label+"\x1b[0m")
				continue
			}
			out = append(out, line)
		}

		return frame{Lines: out}, nil
	}

	pics := map[int]*picture{}
	notes := map[int]string{}

	for _, at := range order {
		item := refs[marks[at]]
		pic, err := source(base, item.Src)
		if err != nil {
			notes[at] = pad + "\x1b[90mmissing image " + item.Src + " (" + err.Error() + ")\x1b[0m"
			continue
		}
		pics[at] = pic
	}

	room := height - 3 - (len(lines) - len(pics))
	share := 0
	if len(pics) > 0 {
		share = room / len(pics)
	}

	var out []string
	var spots []spot

	for i, line := range lines {
		if _, ok := marks[i]; !ok {
			out = append(out, line)
			continue
		}

		if note, ok := notes[i]; ok {
			out = append(out, note)
			continue
		}

		if share < 3 {
			out = append(out, "")
			continue
		}

		pic := pics[i]
		cols, rows, wide, high := fit(pic, width-4, share, cx, cy)

		spots = append(spots, spot{
			Row:  len(out) + 1,
			Col:  max(0, (width-cols)/2),
			Rows: rows,
			Wide: wide,
			High: high,
			Pic:  pic,
		})

		for n := 0; n < rows; n++ {
			out = append(out, "")
		}
	}

	return frame{Lines: out, Spots: spots}, nil
}

func (f frame) shown(room, top int) ([]string, []spot) {
	room = max(room, 0)
	top = max(0, min(top, max(len(f.Lines)-room, 0)))

	lines := f.Lines[top:]
	if len(lines) > room {
		lines = lines[:room]
	}

	var spots []spot
	for _, item := range f.Spots {
		first := item.Row - 1

		if first >= top && first+item.Rows <= top+room {
			item.Row -= top
			spots = append(spots, item)
		}
	}

	return lines, spots
}
