package main

import (
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
)

func unlock(ctx echo.Context) (err error) {
	fileName := ctx.FormValue("name")
	if fileName == "" {
		return
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	userID := auth.UserId(ctx)

	if err = tx.Where("name = ? AND user_id = ?", fileName, userID).
		Set("locked", false).Error; err != nil {
		return
	}

	tx.Commit()
	return
}
