package share

import "github.com/spf13/cobra"

var delete = &cobra.Command{
	Use:   "del",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
