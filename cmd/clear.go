package main

import "github.com/spf13/cobra"

var clear = &cobra.Command{
	Use:   "clear",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
