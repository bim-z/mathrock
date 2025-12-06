package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/server/auth"
	"github.com/mathrock-xyz/starducc/server/db"
	"github.com/mathrock-xyz/starducc/server/db/model"
)

func rm(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"name is required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	result := tx.
		Where("name = ? AND user_id = ? AND locked = ?", name, userid, false).
		Delete(&model.File{})

	if result.Error != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"try again",
		)
	}

	if result.RowsAffected == 0 {
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK,
		echo.Map{
			"message": "file deleted",
		})
}
