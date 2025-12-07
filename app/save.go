package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var save = &cobra.Command{
	Use:   "save [name]",
	Short: "Saves a file and creates a new version on the remote server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		// Check file existence and type
		file, err := os.Stat(name)
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		if file.IsDir() {
			return fmt.Errorf("this command cannot accept folder")
		}

		descriptor, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer descriptor.Close()

		var data bytes.Buffer
		writer := multipart.NewWriter(&data)
		defer writer.Close()

		fw, err := writer.CreateFormFile("file", file.Name()) // Use CreateFormFile for file content
		if err != nil {
			return fmt.Errorf("failed to create form file writer: %w", err)
		}

		if _, err = io.Copy(fw, descriptor); err != nil {
			return fmt.Errorf("failed to copy file data to request body: %w", err)
		}

		// close the writer
		writer.Close()

		req, err := http.NewRequest("POST", "http://app.starducc.mathrock.xyz", &data)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		req.Header.Set("Content-Type", writer.FormDataContentType()) // Set the correct Content-Type

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
			return fmt.Errorf("save failed (Status %d): %s", res.StatusCode, msg)
		}

		log.Info("Success", "action", "save")
		return
	},
}
