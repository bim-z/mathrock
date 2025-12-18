package share

import "github.com/spf13/cobra"

var get = &cobra.Command{
	Use:   "get",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
