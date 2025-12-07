package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var history = &cobra.Command{
	Use:   "history [name]",
	Short: "shows the version history for a specific file",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		// use url parameter: /history/:name
		req, err := http.NewRequest("GET", fmt.Sprintf("http://app.starducc.mathrock.xyz/history/%s", name), nil)
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
			return fmt.Errorf("failed to retrieve history (status %d): %s", res.StatusCode, msg)
		}

		// attempt to decode the response body into the version struct
		versions := []struct {
			ID        uint   `json:"id"`
			Version   int    `json:"version"`
			Hash      string `json:"hash"`
			Size      int64  `json:"size"`
			CreatedAt string `json:"created_at"`
		}{}

		if err := json.NewDecoder(res.Body).Decode(&versions); err != nil {
			return fmt.Errorf("failed to parse server response: %w", err)
		}

		return
	},
}
