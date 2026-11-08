//go:build windows

/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package deadguy

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	split = true
	where = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
)

func engine() string {
	return "Windows registry"
}

func look(key registry.Key, name string) string {
	val, _, err := key.GetIntegerValue(name)
	if err == nil && val == 0 {
		return dark
	}
	return light
}

func current() (state, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, where, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return state{Apps: light, System: light}, nil
		}
		return state{}, err
	}
	defer key.Close()

	return state{Apps: look(key, "AppsUseLightTheme"), System: look(key, "SystemUsesLightTheme")}, nil
}

func broadcast() {
	text, err := syscall.UTF16PtrFromString("ImmersiveColorSet")
	if err != nil {
		return
	}

	var out uintptr

	proc := syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW")
	_, _, _ = proc.Call(0xffff, 0x001a, 0, uintptr(unsafe.Pointer(text)), 0x2, 100, uintptr(unsafe.Pointer(&out)))
}

func assign(name, scope string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, where, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	val := uint32(1)
	if name == dark {
		val = 0
	}

	if scope != "system" {
		if err := key.SetDWordValue("AppsUseLightTheme", val); err != nil {
			return err
		}
	}

	if scope != "apps" {
		if err := key.SetDWordValue("SystemUsesLightTheme", val); err != nil {
			return err
		}
	}

	broadcast()

	return nil
}
