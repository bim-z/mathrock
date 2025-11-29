package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/mathrock-xyz/starducc/cmd/rest"
	"github.com/spf13/cobra"
)

var up = &cobra.Command{
	Use:   "up",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return fmt.Errorf("This command takes one argument")
		}

		name := args[0]
		if name == "" {
			return fmt.Errorf("This command takes one argument")
		}

		file, err := os.Stat(name)
		if err != nil {
			return
		}

		if file.IsDir() {
			return fmt.Errorf("This command cannot accept folder")
		}

		descriptor, err := os.Open(name)
		defer descriptor.Close()

		if err != nil {
			return
		}

		res, err := rest.
			Client.
			R().
			SetFileReader("file", file.Name(), descriptor).
			Post("/")

		if err != nil {
			return
		}

		response, err := rest.Parse(res)
		if err != nil {
			return
		}

		if response.Status == "error" {
			return fmt.Errorf(response.Message)
		}

		log.Info("Succes")
		return
	},
}
