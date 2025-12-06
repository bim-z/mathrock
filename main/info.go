package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
)

func info(ctx echo.Context) (err error) {
	name, id := ctx.Param("name"), auth.UserId()
	if name == "" {
		return fmt.Errorf("")
	}
}
