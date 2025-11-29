package main

import "github.com/spf13/cobra"

var auth = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate user",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
