package main

import (
	_ "github.com/bim-z/mathrock/main/system"
	"github.com/bim-z/mathrock/main/system/auth"
	"github.com/labstack/echo/v4"
)

func main() {
	mux := echo.New()

	// auth
	mux.GET("/auth/redirect", auth.Redirect)
	mux.GET("/auth/callback", auth.Callback)

	mux.Group("/drive")

	mux.Start(":8000")
}
