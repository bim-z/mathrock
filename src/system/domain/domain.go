package domain

import "github.com/spf13/cobra"

var Domain = &cobra.Command{
	Use:   "domain",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.AddCommand(add)
		return cmd.Help()
	},
}
