package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bim-z/mathrock/main/system/db"
	"github.com/bim-z/mathrock/main/system/db/model"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"
)

// Authentication middleware
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

		id, err := verifytoken(token)

		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}

		c.Set("user_id", id)

		return next(c)
	}
}

// Handle callback from the provider
func Callback(ctx echo.Context) (err error) {
	user, err := gothic.CompleteUserAuth(ctx.Response(), ctx.Request())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := db.DB.FirstOrCreate(&model.User{
		Name:  user.Name,
		Email: user.Email,
	}).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	token, err := createtoken(user.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	url := fmt.Sprintf("http://localhost:8000?token=%s&email=%s", token, user.Email)
	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

// Redirect to the provider
func Redirect(ctx echo.Context) (err error) {
	gothic.BeginAuthHandler(ctx.Response(), ctx.Request())
	return
}
