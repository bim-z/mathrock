package drive

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var info = &cobra.Command{
	Use:   "info [name]",
	Short: "Shows detailed information about a file",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		// use url parameter: /info/:name
		req, err := http.NewRequest("GET", fmt.Sprintf("http://app.starducc.mathrock.xyz/info/%s", name), nil)
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
			return fmt.Errorf("failed to retrieve file info (status %d): %s", res.StatusCode, msg)
		}

		// attempt to decode the response body into the file info struct
		info := struct {
			Name           string `json:"name"`
			Hash           string `json:"hash"`
			Size           int64  `json:"size"`
			CurrentVersion int    `json:"current_version"`
		}{}

		if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
			// return error if decoding the server response fails
			return fmt.Errorf("failed to parse server response: %w", err)
		}

		return
	},
}
