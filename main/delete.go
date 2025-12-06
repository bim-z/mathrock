package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
	"github.com/mathrock-xyz/starducc/main/db/model"
)

func delete(ctx echo.Context) error {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"name is required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	result := tx.Unscoped().
		Where("name = ? AND user_id = ? AND locked = ?", name, userid, false).
		Delete(&model.File{})

	if result.Error != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed",
		)
	}

	if result.RowsAffected == 0 {
		return ctx.JSON(http.StatusNotFound, echo.Map{
			"error": "file not found",
		})
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file deleted",
	})
}
