package main

import (
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "star",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.AddCommand(
			auth,
			up,
			save,
			rm,
			delete,
			cp,
			latest,
			lock,
			unlock,
			history,
			info,
			ls,
		)
		return cmd.Help()
	},
}
