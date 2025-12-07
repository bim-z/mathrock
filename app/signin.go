package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/user"

	"github.com/labstack/echo/v4"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var signin = &cobra.Command{
	Use:   "login",
	Short: "Log in to your account",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		/* if _, err = rest.Client.R().Get("http://starducc.mathrock.xyz"); err != nil {
			return
		} */

		serv := &http.Server{
			Addr: "localhost:8000",
		}

		route := echo.New()

		errorf := ""

		route.GET("/", func(c echo.Context) (err error) {
			token, email := c.QueryParam("token"), c.QueryParam("email")
			if token == "" && email == "" {
				errorf = "token is empty"
				return serv.Close()
			}

			usr, err := user.Current()
			if err != nil {
				errorf = err.Error()
				return serv.Close()
			}

			cur := &current{
				Email: email,
				Token: token,
			}

			val, err := json.Marshal(cur)
			if err != nil {
				return
			}

			if err = keyring.Set("starducc", usr.Name, string(val)); err != nil {
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
