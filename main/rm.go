package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
	"github.com/mathrock-xyz/starducc/main/db/model"
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

	// Perform soft delete on the file record.
	// The file must not be locked to be deleted.
	result := tx.
		Where("name = ? AND user_id = ? AND locked = ?", name, userid, false).
		Delete(&model.File{})

	if result.Error != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to delete file record",
		)
	}

	if result.RowsAffected == 0 {
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found or is currently locked",
		)
	}

	tx.Commit()

	// return 200 OK success message
	return ctx.JSON(http.StatusOK,
		echo.Map{
			"message": "file deleted (soft-deleted)",
		})
}
