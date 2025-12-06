package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var restore = &cobra.Command{
	Use:   "restore",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return fmt.Errorf("This command takes one argument")
		}

		name := args[0]
		if name == "" {
			return fmt.Errorf("This command takes one argument")
		}

		req, _ := http.NewRequest("GET", "http://app.starducc.mathrock.xyz/restore/"+name, nil)

		token, err := bearer()
		if err != nil {
			return
		}

		req.Header.Set("Authorization", "Bearer "+token)

		request := new(http.Client)

		res, _ := request.Do(req)
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			msg, err := parse(res.Body)
			if err != nil {
				return err
			}

			return fmt.Errorf(msg)
		}

		file, err := os.Create(name)
		if err != nil {
			return
		}

		if _, err = io.Copy(file, res.Body); err != nil {
			return
		}

		log.Info("Succes")
		return
	},
}
