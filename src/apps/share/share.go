package share

import "github.com/spf13/cobra"

var Share = &cobra.Command{
	Use:   "share",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		cmd.AddCommand(create)
		return
	},
}
