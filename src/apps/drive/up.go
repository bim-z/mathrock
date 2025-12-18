package drive

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

var up = &cobra.Command{
	Use:   "up [name]",
	Short: "Uploads a file to the remote server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		name := args[0]

		// check file existence and type
		file, err := os.Stat(name)
		if err != nil {
			// return error if file cannot be stated (e.g., file not found or permission issue)
			return fmt.Errorf("failed to get file info: %w", err)
		}

		if file.IsDir() {
			// return error if the argument points to a directory
			return fmt.Errorf("this command cannot accept folder")
		}

		descriptor, err := os.Open(name)
		if err != nil {
			// return error if file cannot be opened
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer descriptor.Close()

		var data bytes.Buffer
		writer := multipart.NewWriter(&data)
		defer writer.Close()

		// create the form field for the file content
		fw, err := writer.CreateFormFile("file", file.Name()) // Use CreateFormFile for files
		if err != nil {
			return fmt.Errorf("failed to create form file writer: %w", err)
		}

		// copy file content into the form writer
		if _, err = io.Copy(fw, descriptor); err != nil {
			return fmt.Errorf("failed to copy file data to request body: %w", err)
		}

		// note: must close writer BEFORE creating the request to finalize boundary
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
			// Handle non-200 status codes (server error response)
			msg, err := parse(res.Body)
			if err != nil {
				return fmt.Errorf("server returned status %d, but failed to parse error message: %w", res.StatusCode, err)
			}

			// return the error message parsed from the server response
			return fmt.Errorf("upload failed (Status %d): %s", res.StatusCode, msg)
		}

		log.Info("Success", "file", file.Name())
		return nil
	},
}
