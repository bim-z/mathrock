/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
)

const (
	plainfont = iota
	boldfont
	italfont
	bothfont
)

type run struct {
	Text string
	Font int
}

type mark struct {
	Rule  bool
	X     float64
	X2    float64
	Y     float64
	Size  float64
	Font  int
	Bytes []byte
	Gray  bool
	Dash  bool
	Space float64
}

type sheet struct {
	wide   float64
	high   float64
	margin float64
	y      float64
	pages  [][]mark
	lost   map[rune]bool
	look   string
	gray   bool
	dash   bool
	track  float64
}

var specials = map[rune]byte{
	0x20ac: 0x80, 0x2026: 0x85, 0x2018: 0x91, 0x2019: 0x92, 0x201c: 0x93,
	0x201d: 0x94, 0x2022: 0x95, 0x2013: 0x96, 0x2014: 0x97, 0x2122: 0x99,
}

func encode(char rune) (byte, bool) {
	switch {
	case char >= 0x20 && char <= 0x7e, char >= 0xa0 && char <= 0xff:
		return byte(char), true
	case char == '\t' || char == 0x2009 || char == 0x202f:
		return ' ', true
	}

	code, ok := specials[char]

	return code, ok
}

func advance(code byte, font int) float64 {
	if code < 32 {
		return 0
	}

	if font == boldfont || font == bothfont {
		return float64(heavy[code-32]) / 1000
	}

	return float64(regular[code-32]) / 1000
}

func fid(bold, ital bool) int {
	switch {
	case bold && ital:
		return bothfont
	case bold:
		return boldfont
	case ital:
		return italfont
	}
	return plainfont
}

func measure(text string, font int, size float64) float64 {
	total := 0.0

	for _, char := range text {
		code, ok := encode(char)
		if !ok {
			code = '?'
		}
		total += advance(code, font)
	}

	return total * size
}

func (s *sheet) fresh() {
	s.pages = append(s.pages, nil)
	s.y = s.margin
}

func (s *sheet) need(height float64) {
	if len(s.pages) == 0 || s.y+height > s.high-s.margin {
		s.fresh()
	}
}

func (s *sheet) escape(text string) []byte {
	var out []byte

	for _, char := range text {
		code, ok := encode(char)
		if !ok {
			s.lost[char] = true
			code = '?'
		}

		switch {
		case code == '(' || code == ')' || code == '\\':
			out = append(out, '\\', code)
		case code < 0x20 || code > 0x7e:
			out = append(out, []byte(fmt.Sprintf("\\%03o", code))...)
		default:
			out = append(out, code)
		}
	}

	return out
}

func (s *sheet) put(x float64, font int, size, lead float64, text string) {
	base := s.y + (lead-0.93*size)/2 + 0.72*size
	last := len(s.pages) - 1

	s.pages[last] = append(s.pages[last], mark{X: x, Y: base, Size: size, Font: font, Bytes: s.escape(text), Gray: s.gray, Space: s.track})
}

func (s *sheet) draw(x float64, size, lead float64, parts []run) {
	for _, part := range parts {
		s.put(x, part.Font, size, lead, part.Text)
		x += measure(part.Text, part.Font, size)
	}
}

func (s *sheet) line(y, x1, x2 float64) {
	last := len(s.pages) - 1
	s.pages[last] = append(s.pages[last], mark{Rule: true, X: x1, X2: x2, Y: y, Size: 0.6, Gray: s.gray, Dash: s.dash})
}

func width(parts []run, size float64) float64 {
	total := 0.0

	for _, part := range parts {
		total += measure(part.Text, part.Font, size)
	}

	return total
}

func runs(line []word) []run {
	var out []run

	for idx, item := range line {
		font := fid(item.Bold, item.Ital)

		if idx > 0 && !item.Glue {
			out[len(out)-1].Text += " "
		}

		if len(out) > 0 && out[len(out)-1].Font == font {
			out[len(out)-1].Text += item.Text
			continue
		}

		out = append(out, run{Text: item.Text, Font: font})
	}

	return out
}

func (s *sheet) wrap(list []word, size, first, rest float64) [][]run {
	size1 := func(item word) float64 {
		return measure(item.Text, fid(item.Bold, item.Ital), size)
	}

	var out [][]run

	for _, line := range fill(list, first, rest, size1, measure(" ", plainfont, size)) {
		out = append(out, runs(line))
	}

	return out
}

func (s *sheet) centered(parts []run, size, lead float64) {
	s.need(lead)
	s.draw((s.wide-width(parts, size))/2, size, lead, parts)
	s.y += lead
}

func (s *sheet) banner(d doc) bool {
	items := d.contact()

	longest := 0.0
	for _, item := range items {
		longest = max(longest, measure(item, plainfont, 9))
	}

	if measure(d.Name, boldfont, 18)+longest+24 > s.wide-2*s.margin {
		return false
	}

	s.need(24 + 13*float64(len(items)) + 20)

	top := s.y

	s.draw(s.margin, 18, 22, []run{{Text: d.Name, Font: boldfont}})
	s.y += 22

	if d.Title != "" {
		s.gray = true
		s.draw(s.margin, 10.5, 14, []run{{Text: d.Title, Font: plainfont}})
		s.gray = false
		s.y += 14
	}

	left := s.y
	s.y = top

	for _, item := range items {
		s.put(s.wide-s.margin-measure(item, plainfont, 9), plainfont, 9, 12.5, item)
		s.y += 12.5
	}

	s.y = max(s.y, left) + 6

	return true
}

func (s *sheet) header(d doc) {
	if s.look == "modern" && s.banner(d) {
		return
	}

	size := 20.0

	s.need(26)
	s.centered([]run{{Text: d.Name, Font: boldfont}}, size, 25)

	if d.Title != "" {
		s.centered([]run{{Text: d.Title, Font: plainfont}}, 11, 15)
	}

	items := d.contact()
	text := ""

	flush := func() {
		if text != "" {
			s.centered([]run{{Text: text, Font: plainfont}}, 9.5, 13)
			text = ""
		}
	}

	for _, part := range items {
		next := part
		if text != "" {
			next = text + "  |  " + part
		}

		if text != "" && measure(next, plainfont, 9.5) > s.wide-2*s.margin {
			flush()
			next = part
		}

		text = next
	}

	flush()

	s.y += 4
}

func (s *sheet) heading(title string) {
	if s.look == "modern" {
		s.y += 14
		s.need(12 + 8 + 28)

		s.draw(s.margin, 9.5, 12, []run{{Text: strings.ToUpper(title), Font: boldfont}})
		s.y += 12

		s.gray, s.dash = true, true
		s.line(s.y+3, s.margin, s.wide-s.margin)
		s.gray, s.dash = false, false
		s.y += 9

		return
	}

	s.y += 8
	s.need(14 + 6 + 28)

	text := strings.ToUpper(title)
	s.draw(s.margin, 11, 14, []run{{Text: text, Font: boldfont}})
	s.y += 14
	s.line(s.y+1, s.margin, s.wide-s.margin)
	s.y += 5
}

func (s *sheet) lines(list [][]run, x float64, size, lead float64, mark []run) {
	for idx, line := range list {
		s.need(lead)

		if idx == 0 && mark != nil {
			s.draw(x-11, size, lead, mark)
		}

		s.draw(x, size, lead, line)
		s.y += lead
	}
}

func (s *sheet) block(item block) {
	left := s.margin
	size := 10.0
	room := s.wide - 2*s.margin

	switch item.Kind {
	case para, plain:
		text := words(spans(item.Text), false)
		s.lines(s.wrap(text, size, room, room), left, size, 13, nil)
		s.y += 2
	case note:
		text := words(spans(item.Text), false)
		for idx := range text {
			text[idx].Ital = true
		}
		s.lines(s.wrap(text, size, room, room), left, size, 13, nil)
		s.y += 1
	case bullet:
		indent := 11.0 + float64(item.Level)*14
		text := words(spans(item.Text), false)
		glyph := []run{{Text: "•", Font: plainfont}}
		s.lines(s.wrap(text, size, room-indent-6, room-indent-6), left+6+indent, size, 13, glyph)
		s.y += 1
	case entry:
		if s.look == "modern" && len(item.Parts) > 0 {
			s.stack(item)
			return
		}

		s.y += 4
		s.need(14 + 13)

		var head []word

		for idx, part := range item.Parts {
			prefix := ""
			if idx > 0 {
				prefix = " | "
			}

			if idx == 0 {
				head = append(head, words(spans(part), true)...)
				continue
			}

			head = append(head, words(spans(prefix+part), false)...)
		}

		reserve := 0.0
		if item.Date != "" {
			reserve = measure(item.Date, plainfont, 10) + 14
		}

		list := s.wrap(head, 10.5, room-reserve, room-reserve)

		for idx, line := range list {
			s.need(14)
			s.draw(left, 10.5, 14, line)

			if idx == 0 && item.Date != "" {
				s.put(s.wide-s.margin-measure(item.Date, plainfont, 10), plainfont, 10, 14, item.Date)
			}

			s.y += 14
		}
	}
}

func (s *sheet) stack(item block) {
	room := s.wide - 2*s.margin

	s.y += 6
	s.need(13 + 12)

	reserve := 0.0
	if item.Date != "" {
		reserve = measure(item.Date, plainfont, 9) + 14
	}

	for idx, line := range s.wrap(words(spans(item.Parts[0]), true), 10, room-reserve, room-reserve) {
		s.need(13)
		s.draw(s.margin, 10, 13, line)

		if idx == 0 && item.Date != "" {
			s.gray = true
			s.put(s.wide-s.margin-measure(item.Date, plainfont, 9), plainfont, 9, 13, item.Date)
			s.gray = false
		}

		s.y += 13
	}

	if len(item.Parts) > 1 {
		text := words(spans(strings.Join(item.Parts[1:], " | ")), false)

		s.gray = true
		s.lines(s.wrap(text, 9.5, room, room), s.margin, 9.5, 12, nil)
		s.gray = false
	}

	s.y += 1
}

func layout(d doc, paper string) *sheet {
	sheetw, sheeth := 595.28, 841.89
	if paper == "letter" {
		sheetw, sheeth = 612, 792
	}

	out := &sheet{wide: sheetw, high: sheeth, margin: 54, lost: map[rune]bool{}, look: d.Look}

	out.header(d)

	for _, sec := range d.shown() {
		out.heading(sec.Title)

		for _, item := range sec.Blocks {
			out.block(item)
		}
	}

	return out
}

func hexed(text string) string {
	var out strings.Builder

	out.WriteString("<FEFF")

	for _, unit := range utf16.Encode([]rune(text)) {
		fmt.Fprintf(&out, "%04X", unit)
	}

	out.WriteString(">")

	return out.String()
}

func (s *sheet) bytes(d doc) ([]byte, error) {
	var buf bytes.Buffer
	var offsets []int

	object := func(body string) int {
		offsets = append(offsets, buf.Len())
		id := len(offsets)
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", id, body)
		return id
	}

	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	count := len(s.pages)
	first := 8

	object("<< /Type /Catalog /Pages 2 0 R /Lang (en-US) >>")

	var kids []string
	for idx := 0; idx < count; idx++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", first+idx*2))
	}

	object(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), count))
	object(fmt.Sprintf("<< /Title %s /Author %s /Creator (mathrock resume) /Producer (mathrock resume) >>", hexed(d.Name+" - Resume"), hexed(d.Name)))

	for _, face := range []string{"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique"} {
		object(fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>", face))
	}

	if len(offsets) != 7 {
		return nil, fmt.Errorf("unexpected object layout")
	}

	for idx, page := range s.pages {
		var content bytes.Buffer

		for _, item := range page {
			if item.Rule {
				style := ""
				if item.Gray {
					style += "0.6 G "
				}
				if item.Dash {
					style += "[1.2 2.4] 0 d "
				}

				fmt.Fprintf(&content, "%s%.2f w %.2f %.2f m %.2f %.2f l S [] 0 d 0 G\n", style, item.Size, item.X, s.high-item.Y, item.X2, s.high-item.Y)
				continue
			}

			if item.Gray {
				content.WriteString("0.38 g ")
			}

			fmt.Fprintf(&content, "BT /F%d %.2f Tf %.2f Tc %.2f %.2f Td (%s) Tj ET\n", item.Font+1, item.Size, item.Space, item.X, s.high-item.Y, item.Bytes)

			if item.Gray {
				content.WriteString("0 g\n")
			}
		}

		var packed bytes.Buffer

		zip := zlib.NewWriter(&packed)
		if _, err := zip.Write(content.Bytes()); err != nil {
			return nil, err
		}
		if err := zip.Close(); err != nil {
			return nil, err
		}

		object(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << /F1 4 0 R /F2 5 0 R /F3 6 0 R /F4 7 0 R >> >> /Contents %d 0 R >>",
			s.wide, s.high, first+idx*2+1,
		))

		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n<< /Length %d /Filter /FlateDecode >>\nstream\n", len(offsets), packed.Len())
		buf.Write(packed.Bytes())
		buf.WriteString("\nendstream\nendobj\n")
	}

	start := buf.Len()

	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)

	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}

	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R /Info 3 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, start)

	return buf.Bytes(), nil
}

func rendered(d doc, paper string) ([]byte, int, []string, error) {
	sh := layout(d, paper)

	data, err := sh.bytes(d)
	if err != nil {
		return nil, 0, nil, err
	}

	var notes []string

	if len(sh.lost) > 0 {
		var chars []string

		for char := range sh.lost {
			chars = append(chars, string(char))
		}

		sort.Strings(chars)
		notes = append(notes, fmt.Sprintf("these characters are not supported by the PDF font and became '?': %s", strings.Join(chars, " ")))
	}

	return data, len(sh.pages), notes, nil
}
