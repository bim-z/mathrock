package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get("Authorization")

		if header == "" {
			echo.NewHTTPError(http.StatusUnauthorized, "")
		}

		parts := strings.Split(header, " ")

		if len(parts) != 2 || parts[0] != "Bearer" {
			echo.NewHTTPError(http.StatusUnauthorized, "invalid auth header format")
		}

		token := parts[1]

		id, err := verifyToken(token)

		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}

		c.Set("user_id", id)

		return next(c)
	}
}
