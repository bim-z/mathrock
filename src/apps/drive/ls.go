package drive

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var ls = &cobra.Command{
	Use:   "ls",
	Short: "lists all files belonging to the current user on the remote server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		req, err := http.NewRequest("GET", "http://app.starducc.mathrock.xyz/ls", nil)
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
				return fmt.Errorf("server returned status %d, but failed to parse error message: %w", res.StatusCode, err)
			}

			// return the error message parsed from the server response
			return fmt.Errorf("failed to retrieve file list (status %d): %s", res.StatusCode, msg)
		}

		files := []struct {
			Name string `json:"name"`
		}{}

		if err := json.NewDecoder(res.Body).Decode(&files); err != nil {
			// return error if decoding the server response fails
			return fmt.Errorf("failed to parse server response: %w", err)
		}

		return
	},
}
