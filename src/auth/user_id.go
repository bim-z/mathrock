package auth

import "github.com/labstack/echo/v4"

func UserId(ctx echo.Context) string {
	return ctx.Get("user_id").(string)
}
