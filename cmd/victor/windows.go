//go:build windows

/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package victor

import (
	"os"
	"strings"
	"syscall"
	"time"
)

func sysinfo(path string, file os.FileInfo) (meta, bool) {
	data, ok := file.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return meta{}, false
	}
	out := meta{
		Created:  time.Unix(0, data.CreationTime.Nanoseconds()),
		Accessed: time.Unix(0, data.LastAccessTime.Nanoseconds()),
		Attrs:    attrs(data.FileAttributes),
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return out, true
	}
	share := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	handle, err := syscall.CreateFile(name, 0, share, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return out, true
	}
	defer syscall.CloseHandle(handle)
	var detail syscall.ByHandleFileInformation
	if syscall.GetFileInformationByHandle(handle, &detail) == nil {
		out.Inode = uint64(detail.FileIndexHigh)<<32 | uint64(detail.FileIndexLow)
		out.Links = uint64(detail.NumberOfLinks)
		out.Device = uint64(detail.VolumeSerialNumber)
	}
	return out, true
}

func attrs(bits uint32) string {
	flags := []struct {
		bit  uint32
		name string
	}{
		{syscall.FILE_ATTRIBUTE_READONLY, "readonly"},
		{syscall.FILE_ATTRIBUTE_HIDDEN, "hidden"},
		{syscall.FILE_ATTRIBUTE_SYSTEM, "system"},
		{syscall.FILE_ATTRIBUTE_ARCHIVE, "archive"},
	}
	var names []string
	for _, flag := range flags {
		if bits&flag.bit != 0 {
			names = append(names, flag.name)
		}
	}
	if len(names) == 0 {
		return "normal"
	}
	return strings.Join(names, " ")
}
