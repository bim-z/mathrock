package main

import "github.com/spf13/cobra"

var ls = &cobra.Command{
	Use:   "ls",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) (err error) {},
}
