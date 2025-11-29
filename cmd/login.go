package main

import "github.com/spf13/cobra"

var login = &cobra.Command{
	Use:   "login",
	Short: "",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
