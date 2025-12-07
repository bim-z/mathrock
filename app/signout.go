package main

import (
	"os/user"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var signout = &cobra.Command{
	Use:   "logout",
	Short: "Log out from your account",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		usr, err := user.Current()
		if err != nil {
			return
		}

		return keyring.Delete("starducc", usr.Name)
	},
}
