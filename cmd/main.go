package main

import "github.com/spf13/cobra"

func main() {}

func mathrock() *cobra.Command {
	return &cobra.Command{
		Use:   "mathrock",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}
