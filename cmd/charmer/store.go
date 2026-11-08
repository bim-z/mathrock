/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package charmer

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

const bucket = "links"

var db *bbolt.DB

type Link struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Description string `json:"description"`
	Key         []byte `json:"-"`
}

func connect() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	dir = filepath.Join(dir, "piw")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	db, err = bbolt.Open(filepath.Join(dir, "piw.db"), 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}

	return db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
}

func fetch() ([]Link, error) {
	var all []Link

	err := db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(bucket)).ForEach(func(k, v []byte) error {
			var link Link
			if err := json.Unmarshal(v, &link); err != nil {
				return err
			}

			link.Key = append([]byte(nil), k...)
			all = append(all, link)

			return nil
		})
	})

	return all, err
}

func find(seq int) (Link, error) {
	all, err := fetch()
	if err != nil {
		return Link{}, err
	}

	if seq < 1 || seq > len(all) {
		return Link{}, fmt.Errorf("link number %d not found", seq)
	}

	return all[seq-1], nil
}

func insert(link Link) error {
	link.Name = strings.TrimSpace(link.Name)
	link.Target = clean(link.Target)
	link.Description = strings.TrimSpace(link.Description)

	if link.Name == "" || link.Target == "" {
		return errors.New("name and url are required")
	}
	if link.Description == "" {
		link.Description = "-"
	}

	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))

		id, err := b.NextSequence()
		if err != nil {
			return err
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, id)

		data, err := json.Marshal(link)
		if err != nil {
			return err
		}

		return b.Put(key, data)
	})
}

func remove(key []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Delete(key)
	})
}
