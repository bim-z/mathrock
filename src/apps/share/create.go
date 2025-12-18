package share

import (
	"github.com/spf13/cobra"
)

var create = &cobra.Command{
	Use:   "create",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
