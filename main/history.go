package main

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
	"github.com/mathrock-xyz/starducc/main/db/model"
	"gorm.io/gorm"
)

func history(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"file name is required",
		)
	}

	file := new(model.File)

	// query the file record and preload its associated versions
	if err = db.DB.Where("name = ? AND user_id = ?", name, userid).
		Preload("versions").
		First(&file).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound,
				"file not found or does not belong to user",
			)
		}
		// return 500 internal server error for other database errors
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"database query failed",
		)
	}

	// return the file versions (history) as JSON
	return ctx.JSON(http.StatusOK, file.Versions)
}
