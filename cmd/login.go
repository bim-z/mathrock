package main

import "github.com/spf13/cobra"

var login = &cobra.Command{
	Use:   "login",
	Short: "Log in to your account",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
