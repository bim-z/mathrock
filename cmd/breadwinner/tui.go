/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const side = 26

type editedMsg struct{ err error }

type viewer struct {
	path   string
	paper  string
	doc    doc
	cursor int
	top    int
	lines  []string
	marks  []int
	pages  int
	list   []string
	panel  bool
	note   string
	fault  string
	width  int
	height int
}

func editor() []string {
	for _, key := range []string{"VISUAL", "EDITOR"} {
		if found := strings.Fields(os.Getenv(key)); len(found) > 0 {
			return found
		}
	}

	if runtime.GOOS == "windows" {
		return []string{"notepad"}
	}

	for _, name := range []string{"nano", "vim", "vi"} {
		if _, err := exec.LookPath(name); err == nil {
			return []string{name}
		}
	}

	return []string{"vi"}
}

func stem(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

func newviewer(path, paper, look string) (viewer, error) {
	d, raw, err := load(path)
	if err != nil {
		return viewer{}, err
	}

	if err := restyle(&d, look); err != nil {
		return viewer{}, err
	}

	out := viewer{path: path, paper: paper, doc: d, width: 100, height: 30}
	out.refresh(raw)

	return out, nil
}

func (v *viewer) room() int {
	if v.width < 70 {
		return max(v.width-4, 30)
	}
	return min(v.width-side-6, 86)
}

func (v *viewer) rows() int {
	return max(v.height-6, 3)
}

func (v *viewer) refresh(raw string) {
	v.lines, v.marks = preview(v.doc, v.room())

	_, pages, _, err := rendered(v.doc, v.paper)
	if err == nil {
		v.pages = pages
	}

	if raw != "" {
		v.list = issues(v.doc, raw, v.pages)
	} else {
		v.list = issues(v.doc, "", v.pages)
	}

	v.top = min(v.top, max(len(v.lines)-v.rows(), 0))
}

func (v *viewer) follow() {
	if v.cursor < len(v.marks) && v.marks[v.cursor] >= 0 {
		v.top = min(max(v.marks[v.cursor]-1, 0), max(len(v.lines)-v.rows(), 0))
	}
}

func (v viewer) Init() tea.Cmd {
	return nil
}

func (v viewer) reload() viewer {
	fresh, raw, err := load(v.path)
	if err != nil {
		v.fault = err.Error()
		return v
	}

	hidden := map[string]bool{}

	for _, sec := range v.doc.Sections {
		if sec.Hidden {
			hidden[sec.Title] = true
		}
	}

	for at := range fresh.Sections {
		fresh.Sections[at].Hidden = hidden[fresh.Sections[at].Title]
	}

	fresh.Look = v.doc.Look
	v.doc = fresh
	v.cursor = min(v.cursor, max(len(fresh.Sections)-1, 0))
	v.refresh(raw)
	v.follow()
	v.fault = ""
	v.note = "reloaded"

	return v
}

func (v viewer) save(ext string) viewer {
	target := stem(v.path) + ext

	pages, notes, err := export(v.doc, target, v.paper)
	if err != nil {
		v.fault = err.Error()
		return v
	}

	v.fault = ""
	v.note = "saved " + filepath.Base(target)

	if pages > 0 {
		v.note += "  " + plural(pages, "page")
	}

	if len(notes) > 0 {
		v.note += "  " + notes[0]
	}

	return v
}

func (v viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height
		v.refresh("")
		v.follow()
		return v, nil
	case editedMsg:
		if msg.err != nil {
			v.fault = msg.err.Error()
			return v, nil
		}
		return v.reload(), nil
	case tea.KeyMsg:
		return v.press(msg)
	}

	return v, nil
}

func (v viewer) press(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	v.note, v.fault = "", ""

	last := len(v.doc.Sections) - 1

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return v, tea.Quit
	case "up", "k":
		v.cursor = max(v.cursor-1, 0)
		v.panel = false
		v.follow()
	case "down", "j":
		v.cursor = max(0, min(v.cursor+1, last))
		v.panel = false
		v.follow()
	case "g", "home":
		v.cursor = 0
		v.follow()
	case "G", "end":
		v.cursor = max(last, 0)
		v.follow()
	case "pgdown", "ctrl+d", "f":
		v.top = min(v.top+v.rows()/2, max(len(v.lines)-v.rows(), 0))
	case "pgup", "ctrl+u", "b":
		v.top = max(v.top-v.rows()/2, 0)
	case " ", "space", "enter":
		if last >= 0 {
			v.doc.Sections[v.cursor].Hidden = !v.doc.Sections[v.cursor].Hidden
			v.refresh("")
			v.follow()
		}
	case "K", "shift+up":
		if v.cursor > 0 {
			v.doc.Sections[v.cursor], v.doc.Sections[v.cursor-1] = v.doc.Sections[v.cursor-1], v.doc.Sections[v.cursor]
			v.cursor--
			v.refresh("")
			v.follow()
		}
	case "J", "shift+down":
		if v.cursor < last {
			v.doc.Sections[v.cursor], v.doc.Sections[v.cursor+1] = v.doc.Sections[v.cursor+1], v.doc.Sections[v.cursor]
			v.cursor++
			v.refresh("")
			v.follow()
		}
	case "p":
		if v.paper == "a4" {
			v.paper = "letter"
		} else {
			v.paper = "a4"
		}
		v.refresh("")
	case "l":
		if v.doc.Look == "modern" {
			v.doc.Look = "classic"
		} else {
			v.doc.Look = "modern"
		}
		v.refresh("")
	case "c":
		v.panel = !v.panel
	case "r":
		return v.reload(), nil
	case "e":
		cmd := editor()
		return v, tea.ExecProcess(exec.Command(cmd[0], append(cmd[1:], v.path)...), func(err error) tea.Msg {
			return editedMsg{err: err}
		})
	case "s":
		return v.save(".pdf"), nil
	case "x":
		return v.save(".docx"), nil
	case "t":
		return v.save(".txt"), nil
	}

	return v, nil
}

func window(lines []string, at, size int) []string {
	if size <= 0 || len(lines) <= size {
		return lines
	}

	from := min(max(at-size/2, 0), len(lines)-size)

	return lines[from : from+size]
}

func (v viewer) sidebar() []string {
	out := []string{faint.Render(padded("SECTIONS", side)), ""}

	var rows []string

	for at, sec := range v.doc.Sections {
		name := sec.Title
		if runes(name) > side-4 {
			name = string([]rune(name)[:side-5]) + "…"
		}

		switch {
		case at == v.cursor && sec.Hidden:
			rows = append(rows, pink.Render(padded("> "+name+" (off)", side)))
		case at == v.cursor:
			rows = append(rows, pink.Render(padded("> "+name, side)))
		case sec.Hidden:
			rows = append(rows, faint.Render(padded("  "+name+" (off)", side)))
		default:
			rows = append(rows, padded("  "+name, side))
		}
	}

	return append(out, window(rows, v.cursor, v.rows()-2)...)
}

func padded(text string, wide int) string {
	return text + repeat(" ", wide-runes(text))
}

func (v viewer) body() []string {
	if v.panel {
		return strings.Split(strings.TrimRight(report(v.list), "\n"), "\n")
	}

	end := min(v.top+v.rows(), len(v.lines))

	return v.lines[min(v.top, end):end]
}

func (v viewer) View() string {
	var out strings.Builder

	status := fmt.Sprintf("%s  ·  %s  ·  %s  ·  %s", v.doc.Look, strings.ToUpper(v.paper), plural(v.pages, "page"), plural(len(v.list), "check"))

	if v.width < 70 && v.cursor < len(v.doc.Sections) {
		status += "  ·  " + v.doc.Sections[v.cursor].Title
	}

	out.WriteString("\n  " + lined.Render("resume") + "  " + faint.Render(filepath.Base(v.path)+"  ·  "+status) + "\n\n")

	right := v.body()

	var left []string
	if v.width >= 70 {
		left = v.sidebar()
	}

	for at := 0; at < v.rows(); at++ {
		line := ""

		if v.width >= 70 {
			cell := repeat(" ", side)
			if at < len(left) {
				cell = left[at]
			}
			line = cell + "  "
		}

		if at < len(right) {
			line += right[at]
		}

		out.WriteString("  " + line + "\n")
	}

	out.WriteString("\n")

	switch {
	case v.fault != "":
		out.WriteString("  " + red.Render(v.fault) + "\n")
	case v.note != "":
		out.WriteString("  " + green.Render(v.note) + "\n")
	default:
		out.WriteString("\n")
	}

	out.WriteString("  " + faint.Render("↑↓ section  ·  space show/hide  ·  K J move  ·  e edit  ·  c checks  ·  l template  ·  p paper  ·  s pdf  x docx  t txt  ·  q quit") + "\n")

	return out.String()
}

func play(path, paper, look string) error {
	model, err := newviewer(path, paper, look)
	if err != nil {
		return err
	}

	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()

	return err
}
