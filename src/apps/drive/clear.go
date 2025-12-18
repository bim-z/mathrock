package drive

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var clear = &cobra.Command{
	Use:   "clear [name]",
	Short: "removes all previous versions of a file, resetting the current version to 1",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		req, err := http.NewRequest("POST", "http://app.starducc.mathrock.xyz/clear/"+name, nil)
		if err != nil {
			return fmt.Errorf("failed to create http request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
			return fmt.Errorf("clear history failed (status %d): %s", res.StatusCode, msg)
		}

		log.Info("Success", "action", fmt.Sprintf("file '%s' history cleared and current version reset to 1", name))
		return nil
	},
}
