package drive

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:   "undo [file name]",
	Short: "Undo the file to its immediate previous version",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		// note: the undo handler on the server returns the file content if successful.
		req, err := http.NewRequest("GET", "http://app.starducc.mathrock.xyz/undo/"+name, nil)
		if err != nil {

			return fmt.Errorf("failed to create http request: %w", err)
		}

		token, err := bearer()
		if err != nil {

			return fmt.Errorf("failed to get authentication token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		request := new(http.Client)

		res, err := request.Do(req)
		if err != nil {

			return fmt.Errorf("http request failed: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {

			msg, err := parse(res.Body)
			if err != nil {
				// return error if parsing the error response body fails
				return fmt.Errorf("server returned status %d, but failed to parse error message: %w", res.StatusCode, err)
			}

			// return the error message parsed from the server response
			return fmt.Errorf("undo failed (status %d): %s", res.StatusCode, msg)
		}

		// at this point, the server sent back the file data (the previous version)

		// 1. check if the local file exists, then overwrite it
		// 2. if the file does not exist, create it.

		// create/overwrite the local file
		file, err := os.Create(name)
		if err != nil {
			return fmt.Errorf("failed to create local file '%s': %w", name, err)
		}
		defer file.Close()

		// copy data from the response body to the local file
		if _, err = io.Copy(file, res.Body); err != nil {
			// return error if copying file data fails
			return fmt.Errorf("failed to save file content locally: %w", err)
		}

		log.Info("Success", "action", fmt.Sprintf("file '%s' reverted to previous version", name))
		return
	},
}
