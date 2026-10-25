package dustbox

import "github.com/spf13/cobra"

func add() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Args:  cobra.ExactArgs(1),
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
		},
	}
}
