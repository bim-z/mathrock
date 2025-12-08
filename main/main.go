package main

import (
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
)

func main() {
	route := echo.New()

	route.GET("/auth/redirect", auth.Redirect)
	route.GET("/auth/callback", auth.Callback)

	route.POST("/up", up)
	route.POST("/save", save)

	route.DELETE("/rm/:name", rm)
	route.DELETE("/delete/:name", delete)
	route.POST("/clear", clear)
	route.POST("/restore/:name", restore)

	route.GET("/cp/:name/:version", cp)
	route.PUT("/revert/:name/:version", revert)
	route.GET("/undo/:name", undo)

	route.POST("/lock", lock)
	route.POST("/unlock", unlock)

	route.GET("/ls", ls)
	route.GET("/info/:name", info)
	route.GET("/history/:name", history)
	route.GET("/latest/:name", latest)

	log.Fatal(route.Start)
}
