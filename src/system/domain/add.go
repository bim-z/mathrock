package domain

import "github.com/spf13/cobra"

var add = &cobra.Command{
	Use:   "add",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
