/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package murphy

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"github.com/alitto/pond/v2"
	"github.com/bim-z/mathrock/cmd/auth"
	logger "github.com/bim-z/murphy/log"
	"github.com/charlievieth/fastwalk"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"resty.dev/v3"
)

type metadata struct {
	ID   string
	Name string
	Url  string
	Key  string
	Path string
}

type message struct {
	Space    string
	Password string
	Emails   []string
	Meta     []metadata
}

func main() {
	murphy := &cobra.Command{
		Use: "murphy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	user, err := user.Current()
	if err != nil {
		return
	}

	murphy.AddGroup(auth)

	logger := logger.Initiate()

	rest := resty.New()

	murphy.AddCommand(signin(user, logger), signout(user), whoami(user, logger), up(rest, user, logger), drop(rest, user), rm(rest, user), clean(rest, user))

	_ = fang.Execute(murphy.Context(), murphy)
}

func up(restClient *resty.Client, user *user.User, log *log.Logger) (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use:  "up [name] [object...]",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			spaceName := args[0]
			password, _ := cmd.Flags().GetString("password")
			emails, _ := cmd.Flags().GetStringSlice("emails")

			var mu sync.Mutex
			var localFiles []metadata

			process := func(filePath string) error {
				desc, err := os.Open(filePath)
				if err != nil {
					log.Printf("Gagal membuka file %s: %v\n", filePath, err)
					return nil
				}
				defer desc.Close()

				absPath, err := filepath.Abs(desc.Name())
				if err != nil {
					return err
				}

				mu.Lock()
				localFiles = append(localFiles, metadata{
					Name: filepath.Base(absPath),
					Path: absPath,
				})
				mu.Unlock()

				return nil
			}

			for _, path := range args[1:] {
				stat, err := os.Stat(path)
				if err != nil {
					return err
				}

				if stat.IsDir() {
					conf := fastwalk.Config{Follow: false}
					sym, _ := cmd.Flags().GetBool("sym")
					if sym {
						conf.Follow = true
					}

					recursive, _ := cmd.Flags().GetBool("recursive")

					err = fastwalk.Walk(&conf, path, func(walkPath string, d os.DirEntry, err error) error {
						if err != nil {
							return nil
						}

						if d.IsDir() {
							if !recursive && walkPath != path {
								return fastwalk.SkipDir
							}
							return nil
						}

						return process(walkPath)
					})
					if err != nil {
						return err
					}
				} else {
					if err := process(path); err != nil {
						return err
					}
				}
			}

			if len(localFiles) == 0 {
				return fmt.Errorf("tidak ada file yang berhasil diproses")
			}

			token, err := getToken(user)
			if err != nil {
				return err
			}

			var fileNames []string
			for _, f := range localFiles {
				fileNames = append(fileNames, f.Name)
			}

			var signedMetas []metadata

			fmt.Println("Meminta akses upload ke server...")
			if _, err = restClient.R().SetBody(fileNames).
				SetResult(&signedMetas).SetAuthToken(token).
				Post("https://api.backend-anda.com/v1/files/sign"); err != nil {
				return fmt.Errorf("gagal komunikasi ke backend untuk sign URL: %v", err)
			}

			for i := range signedMetas {
				signedMetas[i].Path = localFiles[i].Path
			}

			pool := pond.NewPool(5)
			var hasError int32

			for _, m := range signedMetas {
				m := m

				pool.Submit(func() {
					if atomic.LoadInt32(&hasError) == 1 {
						return
					}

					file, err := os.Open(m.Path)
					if err != nil {
						atomic.StoreInt32(&hasError, 1)
						log.Printf("Gagal membaca file lokal %s: %v\n", m.Name, err)
						return
					}
					defer file.Close()

					fileStat, _ := file.Stat()

					req, err := http.NewRequest("PUT", m.Url, file)
					if err != nil {
						atomic.StoreInt32(&hasError, 1)
						return
					}

					req.Header.Set("Content-Type", "application/octet-stream")
					req.ContentLength = fileStat.Size()

					client := &http.Client{}
					s3Resp, err := client.Do(req)
					if err != nil || s3Resp.StatusCode >= 300 {
						atomic.StoreInt32(&hasError, 1)
						log.Printf("Gagal upload file %s ke S3. Status: %d\n", m.Name, s3Resp.StatusCode)
						return
					}
					defer s3Resp.Body.Close()

					fmt.Printf("✓ %s terupload\n", m.Name)
				})
			}

			pool.StopAndWait()

			if atomic.LoadInt32(&hasError) == 1 {
				return fmt.Errorf("upload dihentikan karena terjadi kesalahan pada salah satu file")
			}

			finalMsg := message{
				Space:    spaceName,
				Password: password,
				Emails:   emails,
				Meta:     signedMetas,
			}

			if _, err := restClient.R().SetTimeout(10 * time.Second).SetBody(finalMsg).SetAuthToken(token).
				Post("https://api.backend-anda.com/v1/files/up"); err != nil {
				return fmt.Errorf("gagal komit metadata file: %v", err)
			}

			return
		},
	}

	return cmd
}

func drop(rest *resty.Client, user *user.User) *cobra.Command {
	return &cobra.Command{
		Use:   "drop [space...]",
		Args:  cobra.MinimumNArgs(1),
		Short: "",
		RunE: func(cmd *cobra.Command, spaces []string) (err error) {
			token, err := auth.GetToken(user)
			if err != nil {
				return
			}

			if _, err = rest.R().SetTimeout(10 * time.Second).
				SetAuthToken(token).SetBody(spaces).Delete(""); err != nil {
				return
			}

			return
		},
	}
}

func rm(rest *resty.Client, user *user.User) *cobra.Command {
	return &cobra.Command{
		Use:   "rm [space] [file...]",
		Args:  cobra.MinimumNArgs(2),
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			token, err := getToken(user)
			if err != nil {
				return
			}

			type req struct {
				Space string
				Files []string
			}

			if _, err = rest.R().SetTimeout(10 * time.Second).SetAuthToken(token).SetBody(&req{
				Space: args[0],
				Files: args[1:],
			}).Delete(""); err != nil {
				return
			}

			return
		},
	}
}

func clean(rest *resty.Client, user *user.User) *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var ok bool

			if err = huh.NewConfirm().Title("").Description("").Value(&ok).Run(); err != nil {
				return
			}

			token, err := getToken(user)
			if err != nil {
				return
			}

			if ok {
				rest.R().SetTimeout(10 * time.Second).SetAuthToken(token).Delete("")
			}

			return
		},
	}
}
