package main

import (
	_ "github.com/bim-z/mathrock/main/system"
	"github.com/labstack/echo/v4"
)

func main() {
	mux := echo.New()

	mux.Group("/drive")

	mux.Start(":8000")
}
