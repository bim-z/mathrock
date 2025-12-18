package drive

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var rm = &cobra.Command{
	Use:   "rm [name]",
	Short: "Soft-deletes a file from the remote server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		// Construct the DELETE request URL using the original argument 'name'
		req, err := http.NewRequest("DELETE", "http://app.starducc.mathrock.xyz/rm/"+name, nil)
		if err != nil {
			// return error if request creation fails
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
			msg, err := parse(res.Body) // assume parse() function exists
			if err != nil {
				return fmt.Errorf("server returned status %d, but failed to parse error message: %w", res.StatusCode, err)
			}

			// return the error message parsed from the server response
			return fmt.Errorf("deletion failed (Status %d): %s", res.StatusCode, msg)
		}

		log.Info("Success", "action", fmt.Sprintf("file '%s' deleted", name))
		return
	},
}
