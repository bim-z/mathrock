/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const relock = 15 * time.Second

type config struct {
	Focus, Short, Long, Grace time.Duration
	Cycle, Rounds             int
	Soft                      bool
}

var intr = make(chan os.Signal, 1)

func run(plan config) error {
	signal.Notify(intr, os.Interrupt, syscall.SIGTERM)
	os.Remove(slot("sig"))
	defer wipe()
	defer fmt.Println()
	st := &state{Cycle: plan.Cycle}
	for round := 1; plan.Rounds == 0 || round <= plan.Rounds; round++ {
		pos := (round-1)%plan.Cycle + 1
		st.Round = pos
		st.Done = round - 1
		if !phase(st, "focus", plan.Focus, nil) {
			return nil
		}
		st.Done = round
		alert(fmt.Sprintf("Session complete. Locking in %s", plan.Grace.Round(time.Second)))
		if !phase(st, "grace", plan.Grace, nil) {
			return nil
		}
		name, span := "short", plan.Short
		if pos == plan.Cycle {
			name, span = "long", plan.Long
		}
		var last time.Time
		warned := false
		hook := func() {
			if plan.Soft && !last.IsZero() {
				return
			}
			if time.Since(last) < relock {
				return
			}
			last = time.Now()
			if err := lock(); err != nil && !warned {
				warned = true
				notify("Could not lock the screen: " + err.Error())
			}
		}
		if !phase(st, name, span, hook) {
			return nil
		}
		alert("Break over. Back to focus.")
	}
	return nil
}

func phase(st *state, name string, span time.Duration, hook func()) bool {
	st.Phase = name
	st.Total = span
	st.Paused = false
	st.End = time.Now().Add(span)
	st.Beat = time.Now()
	save(st)
	beat := time.NewTicker(250 * time.Millisecond)
	defer beat.Stop()
	for {
		select {
		case <-intr:
			return false
		case <-beat.C:
		}
		switch recv() {
		case "stop":
			return false
		case "pause":
			if !st.Paused {
				st.Left = time.Until(st.End)
				st.Paused = true
			}
		case "resume":
			if st.Paused {
				st.End = time.Now().Add(st.Left)
				st.Paused = false
			}
		}
		st.Beat = time.Now()
		save(st)
		draw(st)
		if st.Paused {
			continue
		}
		if hook != nil {
			hook()
		}
		if st.remain() == 0 {
			return true
		}
	}
}
