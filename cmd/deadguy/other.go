//go:build !windows && !darwin && !linux

/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package deadguy

import "errors"

const split = false

func engine() string {
	return "none"
}

func current() (state, error) {
	return state{}, errors.New("this operating system is not supported")
}

func assign(_, _ string) error {
	return errors.New("this operating system is not supported")
}
