package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
)

func history(ctx echo.Context) (err error) {
	fileName := ctx.FormValue("name")
	if fileName == "" {
		return
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	fl := new(model.File)
	if err = tx.Preload("versions").
		Where("name = ? AND user_id = ?").
		First(&fl).Error; err != nil {
		return
	}

	return ctx.JSON(http.StatusOK, fl.Versions)
}
