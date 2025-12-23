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

func restore(ctx echo.Context) error {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"name is required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	// check if the file exists and is currently soft-deleted by the user
	var file drive.File

	// use Unscoped() to search for soft-deleted records (where deleted_at is not NULL).
	// also ensure it not currently locked.
	if err := tx.Unscoped().
		Where("name = ? AND user_id = ? AND deleted_at IS NOT NULL", name, userid).
		First(&file).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound, "file not found",
			)
		}
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"database query failed during file check",
		)
	}

	// perform the restoration (set deleted_at to NULL)
	result := tx.Unscoped().
		Model(&drive.File{}).
		Where("id = ?", file.ID).
		UpdateColumn("deleted_at", nil)

	if result.Error != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to restore file record",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file successfully restored",
		"name":    name,
	})
}
