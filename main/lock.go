package main

import (
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
)

func lock(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.FormValue("name")
	if name == "" {
		return
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	if err = tx.Where("name = ? AND user_id = ?", name, userid).
		Set("locked", true).Error; err != nil {
		return
	}

	tx.Commit()
	return
}
