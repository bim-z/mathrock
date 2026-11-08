/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package heck

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var opts struct {
	lines   bool
	words   bool
	chars   bool
	bytes   bool
	longest bool
	clock   bool
	speed   int
}

var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: false,
	Prefix:          "tally",
})

var root = &cobra.Command{
	Use:   "tally [file...]",
	Short: "Count lines, words, and bytes with beautiful output",
	Args:  cobra.ArbitraryArgs,
	RunE:  run,
}

func init() {
	flags := root.Flags()
	flags.BoolVarP(&opts.lines, "lines", "l", false, "Print the newline counts")
	flags.BoolVarP(&opts.words, "words", "w", false, "Print the word counts")
	flags.BoolVarP(&opts.chars, "chars", "m", false, "Print the character counts")
	flags.BoolVarP(&opts.bytes, "bytes", "c", false, "Print the byte counts")
	flags.BoolVarP(&opts.longest, "max-line-length", "L", false, "Print the maximum display width")
	flags.BoolVarP(&opts.clock, "time", "t", false, "Print the estimated reading time")
	flags.IntVar(&opts.speed, "speed", 200, "Reading speed in words per minute")
}

func execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if opts.speed <= 0 {
		return errors.New("speed must be greater than zero")
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	names := args
	if len(names) == 0 {
		names = []string{"-"}
	}

	rows := make([]entry, 0, len(names))
	failed := false
	for _, name := range names {
		data, err := load(name)
		if err != nil {
			logger.Error("cannot read file", "file", name, "error", err)
			failed = true
			continue
		}
		rows = append(rows, entry{name: name, data: data})
	}

	if len(rows) > 0 {
		fmt.Println(render(rows, len(names) > 1))
	}

	if failed {
		return errors.New("some files could not be read")
	}
	return nil
}

func load(name string) (stats, error) {
	if name == "-" {
		return scan(os.Stdin)
	}

	file, err := os.Open(name)
	if err != nil {
		return stats{}, err
	}
	defer file.Close()

	return scan(file)
}
