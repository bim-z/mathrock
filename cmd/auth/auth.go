/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"syscall"
	"time"

	"charm.land/log/v2"
	"github.com/labstack/echo/v5"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var auth = &cobra.Group{
	Title: "authentication",
	ID:    "auth",
}

func signin(user *user.User, log *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "signin",
		GroupID: "auth",
		Short:   "Login via browser",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			message := make(chan string, 1)

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			e := echo.New()

			e.GET("/:token", func(c *echo.Context) (err error) {
				token := c.Param("token")

				if token == "" {
					message <- ""
				}

				keyring.Set("murphy", user.Name, token)

				cancel()

				return
			})

			if err := browser.OpenURL("http://localhost:hhh/"); err != nil {
				fmt.Println("Gagal membuka browser otomatis, silakan buka URL secara manual.")
			}

			sc := echo.StartConfig{
				Address:         ":1323",
				GracefulTimeout: 2 * time.Second,
			}

			if err := sc.Start(ctx, e); err != nil && err != http.ErrServerClosed {
				return err
			}

			select {
			case msg := <-message:
				if msg != "" {
					return fmt.Errorf("login gagal: %s", msg)
				}

				log.Info("")
			default:
				log.Info("")
			}

			return nil
		},
	}
}

func signout(user *user.User) *cobra.Command {
	return &cobra.Command{
		Use:     "signout",
		GroupID: "auth",
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return keyring.Delete("murphy", user.Name)
		},
	}
}

func whoami(user *user.User, log *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "whoami",
		GroupID: "auth",
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name, err := keyring.Get("murphy/user", user.Name)
			if err != nil {
				return
			}

			log.Info(name)
			return
		},
	}
}

func Token(user *user.User) (token string, err error) {
	token, err = keyring.Get("murphy/user", user.Name)
	if err != nil {
		return
	}

	return
}
