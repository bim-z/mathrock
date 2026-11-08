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

var stopcmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running cycle",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := send("stop"); err != nil {
			return err
		}
		fmt.Println(good.Render("stopped"))
		return nil
	},
}
