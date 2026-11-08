/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package chmod

import (
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

const (
	setuid = 0o4000
	setgid = 0o2000
	sticky = 0o1000
	every  = 0o777
)

var (
	octal  = regexp.MustCompile(`^[0-7]{3,4}$`)
	clause = regexp.MustCompile(`^[ugoa]*([-+=][rwxXstugo]*)+$`)
)

func word(text string, dir bool) (string, bool) {
	pick := func(file, folder string) (string, bool) {
		if dir {
			return folder, true
		}
		return file, true
	}

	switch text {
	case "private":
		return pick("600", "700")
	case "shared":
		return pick("640", "750")
	case "public":
		return pick("644", "755")
	case "run":
		return "u+x", true
	case "lock":
		return "a-w", true
	}

	return "", false
}

func valid(text string) bool {
	if _, ok := word(text, false); ok {
		return true
	}

	if octal.MatchString(text) {
		return true
	}

	for _, part := range strings.Split(text, ",") {
		if !clause.MatchString(part) {
			return false
		}
	}

	return true
}

func unix(mode fs.FileMode) uint32 {
	bits := uint32(mode.Perm())

	if mode&fs.ModeSetuid != 0 {
		bits |= setuid
	}
	if mode&fs.ModeSetgid != 0 {
		bits |= setgid
	}
	if mode&fs.ModeSticky != 0 {
		bits |= sticky
	}

	return bits
}

func native(bits uint32, old fs.FileMode) fs.FileMode {
	out := old &^ (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)
	out |= fs.FileMode(bits & every)

	if bits&setuid != 0 {
		out |= fs.ModeSetuid
	}
	if bits&setgid != 0 {
		out |= fs.ModeSetgid
	}
	if bits&sticky != 0 {
		out |= fs.ModeSticky
	}

	return out
}

func collect(spec string, who [3]bool, bits uint32, dir bool) uint32 {
	var add uint32

	for _, char := range spec {
		var val uint32

		switch char {
		case 'r':
			val = 4
		case 'w':
			val = 2
		case 'x':
			val = 1
		case 'X':
			if dir || bits&0o111 != 0 {
				val = 1
			}
		case 'u':
			val = bits >> 6 & 7
		case 'g':
			val = bits >> 3 & 7
		case 'o':
			val = bits & 7
		case 's':
			if who[0] {
				add |= setuid
			}
			if who[1] {
				add |= setgid
			}
			continue
		case 't':
			add |= sticky
			continue
		}

		for idx, on := range who {
			if on {
				add |= val << uint(6-3*idx)
			}
		}
	}

	return add
}

func symbol(part string, bits uint32, dir bool) (uint32, error) {
	if !clause.MatchString(part) {
		return 0, fmt.Errorf("invalid mode %q", part)
	}

	at := strings.IndexAny(part, "+-=")

	var who [3]bool

	for _, char := range part[:at] {
		switch char {
		case 'u':
			who[0] = true
		case 'g':
			who[1] = true
		case 'o':
			who[2] = true
		case 'a':
			who = [3]bool{true, true, true}
		}
	}

	if at == 0 {
		who = [3]bool{true, true, true}
	}

	var mask uint32

	for idx, on := range who {
		if on {
			mask |= 7 << uint(6-3*idx)
		}
	}

	rest := part[at:]

	for rest != "" {
		op := rest[0]
		rest = rest[1:]

		end := strings.IndexAny(rest, "+-=")
		if end < 0 {
			end = len(rest)
		}

		add := collect(rest[:end], who, bits, dir)
		rest = rest[end:]

		switch op {
		case '+':
			bits |= add
		case '-':
			bits &^= add
		default:
			bits = bits&^mask | add
		}
	}

	return bits, nil
}

func resolve(text string, bits uint32, dir bool) (uint32, error) {
	if swap, ok := word(text, dir); ok {
		text = swap
	}

	if octal.MatchString(text) {
		val, err := strconv.ParseUint(text, 8, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid mode %q", text)
		}

		if len(text) == 3 {
			return bits&^every | uint32(val), nil
		}

		return uint32(val), nil
	}

	for _, part := range strings.Split(text, ",") {
		var err error

		bits, err = symbol(part, bits, dir)
		if err != nil {
			return 0, fmt.Errorf("invalid mode %q, try something like 755, u+x or private", text)
		}
	}

	return bits, nil
}
