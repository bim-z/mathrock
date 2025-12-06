package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
)

func lock(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest, "file name is required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	// Update the 'locked' status to true
	result := tx.Table("files").
		Where("name = ? AND user_id = ?", name, userid).
		Update("locked", true)

	if result.Error != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to lock file in database",
		)
	}

	// Check if any row was affected
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(
			http.StatusNotFound, "file not found",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file successfully locked",
	})
}
