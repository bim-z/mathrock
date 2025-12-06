package main

import "github.com/spf13/cobra"

var info = &cobra.Command{
	Use:   "info",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) (err error) {},
}
