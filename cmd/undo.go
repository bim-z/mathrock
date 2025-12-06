package main

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:   "undo",
	Short: "",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]
		if name == "" {
			return fmt.Errorf("")
		}

		req, _ := http.NewRequest("PUT", "http://app.starducc.mathrock.xyz/undo/"+name, nil)

		token, err := bearer()
		if err != nil {
			return
		}

		req.Header.Set("Authorization", "Bearer "+token)

		request := new(http.Client)

		res, _ := request.Do(req)
		if res.StatusCode != http.StatusOK {
			msg, err := parse(res.Body)
			if err != nil {
				return err
			}

			return fmt.Errorf(msg)
		}

		log.Info("Succes")
		return
	},
}
