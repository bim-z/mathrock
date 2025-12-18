package share

import "github.com/spf13/cobra"

var rm = &cobra.Command{
	Use:   "rm",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
