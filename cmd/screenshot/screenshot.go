package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/spf13/cobra"
)

func main() {
	var output string

	var rootCmd = &cobra.Command{
		Use:   "webgrab",
		Short: "A CLI tool to fetch HTML and capture screenshots of websites",
		SilenceUsage: true, // Prevents printing usage when returning an error
	}

	var screenshotCmd = &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Capture a full-page screenshot of the specified website",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if output == "" {
				output = "screenshot.png"
			}
			return captureScreenshot(url, output)
		},
	}
	screenshotCmd.Flags().StringVarP(&output, "output", "o", "screenshot.png", "Path to save the screenshot")

	var htmlCmd = &cobra.Command{
		Use:   "html [url]",
		Short: "Download the HTML source code of the specified website",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if output == "" {
				output = "index.html"
			}
			return downloadHTML(url, output)
		},
	}
	htmlCmd.Flags().StringVarP(&output, "output", "o", "index.html", "Path to save the HTML file")

	rootCmd.AddCommand(screenshotCmd, htmlCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func captureScreenshot(url, outputPath string) error {
	// Initialize context for the headless browser
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set a 30-second timeout to prevent hanging
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte
	
	// Execute browser tasks: navigate to URL and take a full screenshot
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.FullScreenshot(&buf, 90), // 90 is the image quality
	})
	if err != nil {
		return fmt.Errorf("failed to capture screenshot for %s: %w", url, err)
	}

	// Save the image buffer to the local device
	if err := os.WriteFile(outputPath, buf, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot to %s: %w", outputPath, err)
	}

	fmt.Printf("Screenshot successfully saved to %s\n", outputPath)
	return nil
}

func downloadHTML(url, outputPath string) error {
	// Fetch the website
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch HTML from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code from %s: %d", url, resp.StatusCode)
	}

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	// Write the HTTP response body directly to the file
	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write HTML data to %s: %w", outputPath, err)
	}

	fmt.Printf("HTML successfully saved to %s\n", outputPath)
	return nil
}
