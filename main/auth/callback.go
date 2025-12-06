package auth

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"
	"github.com/mathrock-xyz/starducc/server/db"
	"github.com/mathrock-xyz/starducc/server/db/model"
)

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

	token, err := createToken(user.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	url := fmt.Sprintf("http://localhost:8000?token=%s&email=%s", token, user.Email)
	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}
