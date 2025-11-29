package main

import "github.com/labstack/echo/v4"

func revert(ctx echo.Context) (err error) {
	fileName := ctx.FormValue("name")
	if fileName == "" {
		return
	}

	version := ctx.FormValue("version")
}
