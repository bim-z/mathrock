/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package victor

import (
	"fmt"
	"os"
	"time"
)

type meta struct {
	Owner, Group, Attrs        string
	Blocks                     int64
	Inode, Links, Device       uint64
	Created, Accessed, Changed time.Time
}

type info struct {
	Path, Kind, Target string
	Size               int64
	Mode               os.FileMode
	Modified           time.Time
	meta
}

func inspect(path string, follow bool) (info, error) {
	stat := os.Lstat
	if follow {
		stat = os.Stat
	}
	file, err := stat(path)
	if err != nil {
		return info{}, err
	}
	sys, ok := sysinfo(path, file)
	if !ok {
		return info{}, fmt.Errorf("unsupported platform")
	}
	data := info{
		Path:     path,
		Kind:     kind(file.Mode()),
		Size:     file.Size(),
		Mode:     file.Mode(),
		Modified: file.ModTime(),
		meta:     sys,
	}
	if file.Mode()&os.ModeSymlink != 0 {
		data.Target, _ = os.Readlink(path)
	}
	return data, nil
}

func kind(mode os.FileMode) string {
	switch {
	case mode&os.ModeSymlink != 0:
		return "symlink"
	case mode.IsDir():
		return "directory"
	case mode&os.ModeSocket != 0:
		return "socket"
	case mode&os.ModeNamedPipe != 0:
		return "pipe"
	case mode&os.ModeCharDevice != 0:
		return "char device"
	case mode&os.ModeDevice != 0:
		return "block device"
	}
	return "file"
}

func size(count int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	val := float64(count)
	pos := 0
	for val >= 1024 && pos < len(units)-1 {
		val /= 1024
		pos++
	}
	if pos == 0 {
		return fmt.Sprintf("%d B", count)
	}
	return fmt.Sprintf("%.1f %s", val, units[pos])
}

func octal(mode os.FileMode) string {
	val := uint32(mode.Perm())
	if mode&os.ModeSetuid != 0 {
		val |= 04000
	}
	if mode&os.ModeSetgid != 0 {
		val |= 02000
	}
	if mode&os.ModeSticky != 0 {
		val |= 01000
	}
	return fmt.Sprintf("%04o", val)
}

func stamp(when time.Time) string {
	return when.Format("2006-01-02 15:04:05")
}

func ago(when time.Time) string {
	span := time.Since(when)
	switch {
	case span < time.Minute:
		return "just now"
	case span < time.Hour:
		return plural(int(span.Minutes()), "minute")
	case span < 24*time.Hour:
		return plural(int(span.Hours()), "hour")
	case span < 30*24*time.Hour:
		return plural(int(span.Hours()/24), "day")
	case span < 365*24*time.Hour:
		return plural(int(span.Hours()/24/30), "month")
	}
	return plural(int(span.Hours()/24/365), "year")
}

func plural(count int, unit string) string {
	if count != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s ago", count, unit)
}
