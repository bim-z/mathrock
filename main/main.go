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

	route.POST("/up", up)     // Upload (assuming multipart/form-data POST)
	route.POST("/save", save) // Save new version (assuming multipart/form-data POST)

	route.DELETE("/rm/:name", rm)         // Soft Delete (uses URL parameter)
	route.DELETE("/delete/:name", delete) // Permanent Delete (uses URL parameter and Unscoped)
	route.POST("/clear", clear)           // Clear all file versions except the latest (uses FormValue, typically POST)
	route.POST("/restore/:name", restore) // Restore soft-deleted file (uses URL parameter, typically POST)

	route.GET("/cp/:name/:version", cp)          // Copy/Download a specific version (GET is appropriate)
	route.POST("/revert/:name/:version", revert) // Revert file structure to a specific version (POST is appropriate for modification)
	route.GET("/undo/:name", undo)               // Undo: download previous version (GET is appropriate, although server-side logic might be complex)

	route.POST("/lock", lock)     // Lock file
	route.POST("/unlock", unlock) // Unlock file

	route.GET("/ls", ls)                 // List all files
	route.GET("/info/:name", info)       // Get detailed file info (uses URL parameter)
	route.GET("/history/:name", history) // Get file history (uses URL parameter)
	route.GET("/latest/:name", latest)   // Get the latest file content (uses URL parameter)

	log.Fatal(route.Start)
}
