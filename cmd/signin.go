package main

import (
	"fmt"
	"net/http"
	"os/user"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/cmd/rest"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var signin = &cobra.Command{
	Use:   "login",
	Short: "Log in to your account",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if _, err = rest.Client.R().Get("http://starducc.mathrock.xyz"); err != nil {
			return
		}

		serv := &http.Server{
			Addr: "localhost:8000",
		}

		route := echo.New()

		errorf := ""

		route.GET("/", func(c echo.Context) (err error) {
			token := c.QueryParam("token")
			if token == "" {
				errorf = "token is empty"
				return serv.Close()
			}

			usr, err := user.Current()
			if err != nil {
				errorf = err.Error()
				return serv.Close()
			}

			if err = keyring.Set("mathrock", usr.Name, token); err != nil {
				errorf = err.Error()
				return serv.Close()
			}

			return serv.Close()
		})

		serv.Handler = route
		if err = serv.ListenAndServe(); err != nil {
			return
		}

		if errorf != "" {
			return fmt.Errorf(errorf)
		}

		return
	},
}
