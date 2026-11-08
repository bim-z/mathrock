//go:build linux

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
	accessed := time.Unix(int64(sys.Atim.Sec), int64(sys.Atim.Nsec))
	changed := time.Unix(int64(sys.Ctim.Sec), int64(sys.Ctim.Nsec))
	return accessed, changed
}
