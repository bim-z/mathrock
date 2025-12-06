package main

import (
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "star",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.AddCommand(auth)
		return cmd.Help()
	},
}
