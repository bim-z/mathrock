package main

import "github.com/spf13/cobra"

var save = &cobra.Command{
	Use:   "save",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
	},
}
