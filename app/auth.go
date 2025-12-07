package main

import "github.com/spf13/cobra"

var auth = &cobra.Command{
	Use:   "auth [command]",
	Short: "Manage authentication settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.AddCommand(signin, whoami, signout)
		return cmd.Help()
	},
}
