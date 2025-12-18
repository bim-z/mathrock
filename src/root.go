package main

import (
	"github.com/bim-z/mathrock/src/apps/drive"
	"github.com/bim-z/mathrock/src/apps/share"
	"github.com/bim-z/mathrock/src/system/auth"
	"github.com/bim-z/mathrock/src/system/domain"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:     "mathrock",
	Aliases: []string{"mr", "math", "rock"},
	Short:   "Swiss army knife",
	RunE: func(cmd *cobra.Command, args []string) error {

		// authentication
		cmd.AddGroup(&cobra.Group{
			ID:    "auth",
			Title: "Manage authentication",
		})

		cmd.AddCommand(auth.Signin, auth.Signout, auth.Whoami)

		// domain
		cmd.AddCommand(domain.Domain)

		// drive
		cmd.AddCommand(drive.Drive)

		// share
		cmd.AddCommand(share.Share)

		return cmd.Help()
	},
}
