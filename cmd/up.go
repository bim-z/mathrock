package main

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var up = &cobra.Command{
	Use:   "up",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return fmt.Errorf("This command takes one argument")
		}

		name := args[0]

		if name == "" {
			return fmt.Errorf("This command takes one argument")
		}

		file, err := os.Stat(name)

		if err != nil {
			return
		}

		if file.IsDir() {
			return fmt.Errorf("This command cannot accept folder")
		}

		descriptor, err := os.Open(name)
		defer descriptor.Close()

		if err != nil {
			return
		}

		hasher := sha256.New()

		if _, err = io.Copy(hasher, descriptor); err != nil {
			return
		}

		req := client.NewRequest()

		reader, writer := io.Pipe()

		go func() {
			defer writer.Close()
			io.Copy(writer, descriptor)
		}()

		client.NewRequest().SetBody(reader)
	},
}
