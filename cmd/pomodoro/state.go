/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type state struct {
	Phase  string        `json:"phase"`
	Round  int           `json:"round"`
	Cycle  int           `json:"cycle"`
	Done   int           `json:"done"`
	Total  time.Duration `json:"total"`
	Left   time.Duration `json:"left"`
	End    time.Time     `json:"end"`
	Paused bool          `json:"paused"`
	Beat   time.Time     `json:"beat"`
}

var dir = home()

func home() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	path := filepath.Join(base, "mathrock")
	os.MkdirAll(path, 0o755)
	return path
}

func slot(ext string) string {
	return filepath.Join(dir, "pomo."+ext)
}

func (st *state) remain() time.Duration {
	left := st.Left
	if !st.Paused {
		left = time.Until(st.End)
	}
	if left < 0 {
		return 0
	}
	return left
}

func save(st *state) {
	data, err := json.Marshal(st)
	if err != nil {
		return
	}
	temp := slot("tmp")
	if os.WriteFile(temp, data, 0o644) != nil {
		return
	}
	os.Rename(temp, slot("json"))
}

func load() (*state, error) {
	data, err := os.ReadFile(slot("json"))
	if err != nil {
		return nil, err
	}
	st := &state{}
	if err := json.Unmarshal(data, st); err != nil {
		return nil, err
	}
	return st, nil
}

func alive(st *state) bool {
	return time.Since(st.Beat) < 3*time.Second
}

func send(word string) error {
	st, err := load()
	if err != nil || !alive(st) {
		return errors.New("no session running")
	}
	return os.WriteFile(slot("sig"), []byte(word), 0o644)
}

func recv() string {
	data, err := os.ReadFile(slot("sig"))
	if err != nil {
		return ""
	}
	os.Remove(slot("sig"))
	return strings.TrimSpace(string(data))
}

func wipe() {
	os.Remove(slot("json"))
}
