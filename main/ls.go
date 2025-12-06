package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
)

func ls(ctx echo.Context) (err error) {
	userID := auth.UserId(ctx)

	tx := db.DB.Begin()
	defer tx.Rollback()

	files := []model.File{}
	if err = tx.Where("user_id = ?", userID).Find(files).Error; err != nil {
		return
	}

	return ctx.JSON(http.StatusOK, files)
}
