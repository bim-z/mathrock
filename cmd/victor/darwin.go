//go:build darwin

/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package victor

import (
	"syscall"
	"time"
)

func times(sys *syscall.Stat_t) (time.Time, time.Time) {
	accessed := time.Unix(int64(sys.Atimespec.Sec), int64(sys.Atimespec.Nsec))
	changed := time.Unix(int64(sys.Ctimespec.Sec), int64(sys.Ctimespec.Nsec))
	return accessed, changed
}
