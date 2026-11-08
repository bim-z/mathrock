/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package heck

import (
	"bufio"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type stats struct {
	lines   int
	words   int
	chars   int
	bytes   int
	longest int
}

func (s stats) merge(other stats) stats {
	return stats{
		lines:   s.lines + other.lines,
		words:   s.words + other.words,
		chars:   s.chars + other.chars,
		bytes:   s.bytes + other.bytes,
		longest: max(s.longest, other.longest),
	}
}

func space(char rune) bool {
	switch char {
	case 0xA0, 0x2007, 0x202F:
		return false
	}
	return unicode.IsSpace(char)
}

func scan(src io.Reader) (stats, error) {
	var out stats
	reader := bufio.NewReaderSize(src, 64*1024)
	inword := false
	width := 0

	for {
		char, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}

		out.bytes += size
		if char != utf8.RuneError || size != 1 {
			out.chars++
		}

		switch {
		case char == '\n':
			out.lines++
			out.longest = max(out.longest, width)
			width = 0
			inword = false
		case char == '\r' || char == '\f':
			out.longest = max(out.longest, width)
			width = 0
			inword = false
		case char == '\t':
			width = (width/8 + 1) * 8
			inword = false
		case char == ' ':
			width++
			inword = false
		case space(char):
			inword = false
		default:
			width += runewidth.RuneWidth(char)
			if !inword {
				out.words++
				inword = true
			}
		}
	}

	out.longest = max(out.longest, width)
	return out, nil
}
