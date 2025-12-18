package drive

import (
	"errors"
	"net/http"

	"github.com/bim-z/mathrock/main/system/auth"
	"github.com/bim-z/mathrock/main/system/db"
	"github.com/bim-z/mathrock/main/system/db/model/drive"
	"github.com/labstack/echo/v4"

	"gorm.io/gorm"
)

func Info(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest, "file name is required",
		)
	}

	file := new(drive.File)

	if err = db.DB.Where("name = ? AND user_id = ?", name, userid).
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Order("version DESC").Limit(1)
		}).
		First(&file).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound, "file not found",
			)
		}
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"database query failed",
		)
	}

	currver := 0
	if len(file.Versions) > 0 {
		currver = file.Versions[0].Ver
	}

	return ctx.JSON(http.StatusOK, echo.Map{
		"name":            file.Name,
		"hash":            file.Hash,
		"size":            file.Size,
		"current_version": currver,
	})
}
