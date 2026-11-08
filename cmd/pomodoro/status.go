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

	"github.com/spf13/cobra"
)

var statuscmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := load()
		if err != nil || !alive(st) {
			fmt.Println(dim.Render("idle"))
			return nil
		}
		fmt.Println(line(st) + "  " + dim.Render(fmt.Sprintf("%d done", st.Done)))
		return nil
	},
}
