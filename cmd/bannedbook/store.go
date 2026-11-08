/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package bannedbook

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	inbox     = "inbox"
	actions   = "next"
	waiting   = "waiting"
	calendar  = "calendar"
	projects  = "project"
	someday   = "someday"
	reference = "reference"
	finished  = "done"
)

var lists = []string{inbox, actions, waiting, calendar, projects, someday, reference}

var titles = map[string]string{
	inbox:     "Inbox",
	actions:   "Next Actions",
	waiting:   "Waiting For",
	calendar:  "Calendar",
	projects:  "Projects",
	someday:   "Someday/Maybe",
	reference: "Reference",
	finished:  "Done",
}

var aliases = map[string]string{
	"inbox": inbox, "in": inbox,
	"next": actions, "actions": actions, "action": actions,
	"waiting": waiting, "wait": waiting, "delegated": waiting,
	"calendar": calendar, "cal": calendar, "agenda": calendar,
	"project": projects, "projects": projects,
	"someday": someday, "maybe": someday,
	"reference": reference, "ref": reference,
	"done": finished, "finished": finished,
}

type item struct {
	ID      int       `json:"id"`
	Text    string    `json:"text"`
	List    string    `json:"list"`
	Parent  int       `json:"parent,omitempty"`
	Who     string    `json:"who,omitempty"`
	Context string    `json:"context,omitempty"`
	Due     time.Time `json:"due,omitempty"`
	Clock   bool      `json:"clock,omitempty"`
	Added   time.Time `json:"added"`
	Closed  time.Time `json:"closed,omitempty"`
}

type book struct {
	Last  int    `json:"last"`
	Items []item `json:"items"`
	path  string
}

func locate() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, "mathrock", "gtd.json"), nil
}

func open() (*book, error) {
	path, err := locate()
	if err != nil {
		return nil, err
	}

	out := &book{path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, out); err != nil {
		return nil, fmt.Errorf("%s is not valid: %w", path, err)
	}

	return out, nil
}

func (b *book) save() error {
	if err := os.MkdirAll(filepath.Dir(b.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	temp := b.path + ".tmp"

	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(temp, b.path)
}

func (b *book) find(id int) *item {
	for at := range b.Items {
		if b.Items[at].ID == id {
			return &b.Items[at]
		}
	}
	return nil
}

func (b *book) create(entry item) *item {
	b.Last++
	entry.ID = b.Last
	entry.Added = time.Now()
	b.Items = append(b.Items, entry)

	return &b.Items[len(b.Items)-1]
}

func (b *book) drop(id int) {
	kept := b.Items[:0]

	for _, entry := range b.Items {
		if entry.ID != id && entry.Parent != id {
			kept = append(kept, entry)
		}
	}

	b.Items = kept
}

func active(entry item) bool {
	return entry.List != finished
}

func (b *book) kids(id int) []item {
	var out []item

	for _, entry := range b.Items {
		if entry.Parent == id && active(entry) && entry.List != projects {
			out = append(out, entry)
		}
	}

	return out
}

func (b *book) numbered() []int {
	var out []int

	for _, name := range lists {
		var group []item

		for _, entry := range b.Items {
			if entry.List == name {
				group = append(group, entry)
			}
		}

		sort.SliceStable(group, func(x, y int) bool {
			if name == calendar {
				return group[x].Due.Before(group[y].Due)
			}
			return group[x].ID < group[y].ID
		})

		for _, entry := range group {
			out = append(out, entry.ID)
		}
	}

	return out
}

func (b *book) pick(number int) (*item, error) {
	order := b.numbered()

	if number < 1 || number > len(order) {
		return nil, fmt.Errorf("no item numbered %d, run 'gtd' to see the numbers", number)
	}

	return b.find(order[number-1]), nil
}

func listof(name string) (string, error) {
	if found, ok := aliases[strings.ToLower(strings.TrimSpace(name))]; ok {
		return found, nil
	}

	return "", fmt.Errorf("unknown list %q, use inbox, next, waiting, calendar, projects, someday, reference or done", name)
}
