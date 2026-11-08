/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"errors"
	"time"

	"github.com/spf13/cobra"
)

var startcmd = &cobra.Command{
	Use:   "start",
	Short: "Start a pomodoro cycle",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		focus, _ := flags.GetDuration("focus")
		short, _ := flags.GetDuration("short")
		long, _ := flags.GetDuration("long")
		grace, _ := flags.GetDuration("grace")
		cycle, _ := flags.GetInt("cycle")
		rounds, _ := flags.GetInt("rounds")
		soft, _ := flags.GetBool("soft")
		if cycle < 1 {
			return errors.New("cycle must be at least 1")
		}
		if focus <= 0 || short <= 0 || long <= 0 {
			return errors.New("durations must be positive")
		}
		if st, err := load(); err == nil && alive(st) {
			return errors.New("a session is already running")
		}
		return run(config{
			Focus:  focus,
			Short:  short,
			Long:   long,
			Grace:  grace,
			Cycle:  cycle,
			Rounds: rounds,
			Soft:   soft,
		})
	},
}

func init() {
	flags := startcmd.Flags()
	flags.Duration("focus", 25*time.Minute, "length of a focus session")
	flags.Duration("short", 5*time.Minute, "length of a short break")
	flags.Duration("long", 20*time.Minute, "length of the long break")
	flags.Duration("grace", 10*time.Second, "warning time before the screen locks")
	flags.Int("cycle", 4, "focus sessions before the long break")
	flags.Int("rounds", 0, "stop after this many sessions, 0 for endless")
	flags.Bool("soft", false, "lock once instead of locking again during breaks")
}
