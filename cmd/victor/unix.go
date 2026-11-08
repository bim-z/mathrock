//go:build linux || darwin

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
	"os/user"
	"strconv"
	"syscall"
)

func sysinfo(path string, file os.FileInfo) (meta, bool) {
	sys, ok := file.Sys().(*syscall.Stat_t)
	if !ok {
		return meta{}, false
	}
	accessed, changed := times(sys)
	return meta{
		Owner:    owner(sys.Uid),
		Group:    group(sys.Gid),
		Blocks:   int64(sys.Blocks),
		Inode:    uint64(sys.Ino),
		Links:    uint64(sys.Nlink),
		Device:   uint64(sys.Dev),
		Accessed: accessed,
		Changed:  changed,
	}, true
}

func owner(uid uint32) string {
	ident := strconv.FormatUint(uint64(uid), 10)
	acct, err := user.LookupId(ident)
	if err != nil {
		return ident
	}
	return acct.Username
}

func group(gid uint32) string {
	ident := strconv.FormatUint(uint64(gid), 10)
	grp, err := user.LookupGroupId(ident)
	if err != nil {
		return ident
	}
	return grp.Name
}
