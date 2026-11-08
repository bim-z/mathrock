/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package track

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

var (
	projects = []byte("projects")
	points   = []byte("points")
	meta     = []byte("meta")
	active   = []byte("active")
)

type project struct {
	Name string    `json:"name"`
	Born time.Time `json:"born"`
}

type point struct {
	ID      uint64    `json:"id"`
	Project string    `json:"project"`
	Text    string    `json:"text"`
	Born    time.Time `json:"born"`
}

func key(id uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	return buf
}

func slug(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func clean(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "", errors.New("project name is empty")
	}
	return name, nil
}

func open() (*bbolt.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bbolt.Open(filepath.Join(dir, "track.db"), 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, name := range [][]byte{projects, points, meta} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	return db, nil
}

func update(fn func(*bbolt.Tx) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(fn)
}

func view(fn func(*bbolt.Tx) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(fn)
}

func lookup(tx *bbolt.Tx, name string) (project, error) {
	raw := tx.Bucket(projects).Get([]byte(slug(name)))
	if raw == nil {
		return project{}, fmt.Errorf("project %q does not exist, create it with 'track new %s'", name, name)
	}

	var out project
	if err := json.Unmarshal(raw, &out); err != nil {
		return project{}, fmt.Errorf("failed to decode project: %w", err)
	}

	return out, nil
}

func every(tx *bbolt.Tx) ([]project, error) {
	var out []project

	err := tx.Bucket(projects).ForEach(func(_, raw []byte) error {
		var item project
		if err := json.Unmarshal(raw, &item); err != nil {
			return fmt.Errorf("failed to decode project: %w", err)
		}
		out = append(out, item)
		return nil
	})

	sort.Slice(out, func(i, j int) bool { return slug(out[i].Name) < slug(out[j].Name) })

	return out, err
}

func entries(tx *bbolt.Tx) ([]point, error) {
	var out []point

	err := tx.Bucket(points).ForEach(func(_, raw []byte) error {
		var item point
		if err := json.Unmarshal(raw, &item); err != nil {
			return fmt.Errorf("failed to decode checkpoint: %w", err)
		}
		out = append(out, item)
		return nil
	})

	return out, err
}

func order(items []point) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Born.Equal(items[j].Born) {
			return items[i].ID < items[j].ID
		}
		return items[i].Born.Before(items[j].Born)
	})
}

func collect(tx *bbolt.Tx, proj project) ([]point, error) {
	all, err := entries(tx)
	if err != nil {
		return nil, err
	}

	var out []point

	for _, item := range all {
		if item.Project == slug(proj.Name) {
			out = append(out, item)
		}
	}

	order(out)

	return out, nil
}

func store(tx *bbolt.Tx, item point) error {
	raw, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to encode checkpoint: %w", err)
	}

	if err := tx.Bucket(points).Put(key(item.ID), raw); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

func keep(tx *bbolt.Tx, item project) error {
	raw, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to encode project: %w", err)
	}

	if err := tx.Bucket(projects).Put([]byte(slug(item.Name)), raw); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

func chosen(tx *bbolt.Tx, name string) (project, error) {
	if strings.TrimSpace(name) != "" {
		return lookup(tx, name)
	}

	if current := tx.Bucket(meta).Get(active); current != nil {
		if item, err := lookup(tx, string(current)); err == nil {
			return item, nil
		}
	}

	all, err := every(tx)
	if err != nil {
		return project{}, err
	}

	switch len(all) {
	case 0:
		return project{}, errors.New("there are no projects yet, create one with 'track new <name>'")
	case 1:
		return all[0], nil
	}

	return project{}, errors.New("no project is selected, pick one with 'track use <name>' or --project")
}
