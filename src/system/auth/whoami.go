package auth

import (
	"encoding/json"
	"os/user"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var Whoami = &cobra.Command{
	Use:     "whoami",
	GroupID: "auth",
	Short:   "Show current logged-in user",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		usr, err := user.Current()
		if err != nil {
			return
		}

		val, err := keyring.Get("starducc", usr.Name)
		if err != nil {
			return
		}

		cur := new(current)
		if err = json.Unmarshal([]byte(val), cur); err != nil {
			return
		}

		log.Info("name: ", cur.Email)
		return
	},
}
