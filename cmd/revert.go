package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var revert = &cobra.Command{
	Use:   "",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 2 {
			return fmt.Errorf("This command takes one argument")
		}

		name, version := args[0], args[1]
		if name == "" && version == "" {
			return fmt.Errorf("This command takes one argument")
		}

		req, _ := http.NewRequest("GET", fmt.Sprintf("http://app.starducc.mathrock.xyz/cp/%s/%s", name, version), nil)

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
