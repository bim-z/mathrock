package drive

import (
	"github.com/spf13/cobra"
)

var Drive = &cobra.Command{
	Use:   "drive",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
