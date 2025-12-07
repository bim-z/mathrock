package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var cp = &cobra.Command{
	Use:   "cp [name] [version]",
	Short: "Copies a specific file version",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name, version := args[0], args[1]

		req, err := http.NewRequest("GET", fmt.Sprintf("http://app.starducc.mathrock.xyz/cp/%s/%s", name, version), nil)
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
			return fmt.Errorf("copy failed (Status %d): %s", res.StatusCode, msg)
		}

		// create the local file to write the content
		file, err := os.Create(name)
		if err != nil {
			// return error if local file creation fails
			return fmt.Errorf("failed to create local file '%s': %w", name, err)
		}
		defer file.Close()

		if _, err = io.Copy(file, res.Body); err != nil {
			return fmt.Errorf("failed to save file content locally: %w", err)
		}

		log.Info("Success", "action", fmt.Sprintf("file '%s' version %s copied", name, version))
		return
	},
}
