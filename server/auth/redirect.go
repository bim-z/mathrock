package auth

import (
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"
)

func Redirect(ctx echo.Context) (err error) {
	gothic.BeginAuthHandler(ctx.Response(), ctx.Request())
	return
}
