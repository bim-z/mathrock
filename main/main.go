package main

import (
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
)

func main() {
	mux := echo.New()

	// auth
	mux.GET("/auth/redirect", auth.Redirect)
	mux.GET("/auth/callback", auth.Callback)

	api := mux.Group("/api", auth.Auth)

	mux.Start(":8000")
}
