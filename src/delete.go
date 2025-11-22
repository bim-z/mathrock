package main

import (
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
)

func delete(ctx echo.Context) (err error) {
	fileName := ctx.FormValue("name")
	if fileName == "" {
		return
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	userID := auth.UserId(ctx)

	if err = db.DB.
		Unscoped().
		Where("name = ? AND user_id = ?", fileName, userID).
		Error; err != nil {
		return
	}

	tx.Commit()
	return
}
