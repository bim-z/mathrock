package main

import "github.com/spf13/cobra"

var signout = &cobra.Command{
	Use:   "logout",
	Short: "Log out from your account",
	RunE:  func(cmd *cobra.Command, args []string) error {},
}
