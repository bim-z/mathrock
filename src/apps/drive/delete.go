package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var delete = &cobra.Command{
	Use:     "delete [name]",
	Aliases: []string{"del"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Permanently deletes a file from the remote server",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		file, err := os.Stat(name)
		if err != nil {
			return fmt.Errorf("failed to get local file info: %w", err)
		}

		if file.IsDir() {
			return fmt.Errorf("this command cannot accept folder")
		}

		req, err := http.NewRequest("DELETE", "http://app.starducc.mathrock.xyz/delete/"+name, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		token, err := bearer()
		if err != nil {
			return fmt.Errorf("failed to get authentication token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		request := new(http.Client)

		res, err := request.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			msg, err := parse(res.Body)
			if err != nil {
				return fmt.Errorf("server returned status %d, but failed to parse error message: %w", res.StatusCode, err)
			}

			// return the error message parsed from the server response
			return fmt.Errorf("deletion failed (Status %d): %s", res.StatusCode, msg)
		}

		log.Info("Success", "action", fmt.Sprintf("file '%s' permanently deleted", name))
		return
	},
}
