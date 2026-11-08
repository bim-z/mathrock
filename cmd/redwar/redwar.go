/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package redwar

import (
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var redwar = &cobra.Command{
	Use:           "redwar [file...]",
	Short:         "Copy piped input or files to the clipboard",
	Example:       "  git diff | redwar\n  redwar main.go go.mod\n  make 2>&1 | redwar -t | less",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		tee, _ := cmd.Flags().GetBool("tee")
		quiet, _ := cmd.Flags().GetBool("quiet")
		strip, _ := cmd.Flags().GetBool("strip")
		remote, _ := cmd.Flags().GetBool("osc")
		data, err := gather(args)
		if err != nil {
			return err
		}
		if strip {
			data = bytes.TrimRight(data, "\r\n")
		}
		if len(data) == 0 {
			return errors.New("nothing to copy")
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return errors.New("input looks binary")
		}
		if tee {
			os.Stdout.Write(data)
		}
		via, err := send(data, remote)
		if err != nil {
			return err
		}
		if !quiet {
			report(data, via)
		}
		return nil
	},
}

func init() {
	redwar.Flags().BoolP("tee", "t", false, "pass input through to stdout")
	redwar.Flags().BoolP("quiet", "q", false, "do not print a summary")
	redwar.Flags().BoolP("strip", "n", false, "drop trailing newlines")
	redwar.Flags().BoolP("osc", "o", false, "force OSC 52 escape sequence")
}

func gather(args []string) ([]byte, error) {
	if len(args) == 0 {
		file, err := os.Stdin.Stat()
		if err != nil {
			return nil, err
		}
		if file.Mode()&os.ModeCharDevice != 0 {
			return nil, errors.New("no input, pipe something in or pass files")
		}
		return io.ReadAll(os.Stdin)
	}
	var buf bytes.Buffer
	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	return buf.Bytes(), nil
}
