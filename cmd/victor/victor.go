/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package victor

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var victor = &cobra.Command{
	Use:           "victor [file...]",
	Short:         "Show file status in a clean layout",
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		follow, _ := cmd.Flags().GetBool("follow")
		failed := false
		for idx, path := range args {
			if idx > 0 {
				fmt.Println()
			}
			data, err := inspect(path, follow)
			if err != nil {
				failed = true
				fmt.Println(fail.Render(err.Error()))
				continue
			}
			fmt.Println(render(data))
		}
		if failed {
			return errors.New("some files failed")
		}

		cmd.Flags().BoolP("follow", "L", false, "follow symlinks")
		return nil
	},
}
