package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
)

func history(ctx echo.Context) (err error) {
	name := ctx.FormValue("name")
	if name == "" {
		return
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	file := new(model.File)
	if err = tx.Where("name = ? AND user_id = ?").
		Preload("versions").
		First(&file).Error; err != nil {
		return
	}

	return ctx.JSON(http.StatusOK, file.Versions)
}
