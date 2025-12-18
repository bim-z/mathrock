package drive

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func delete(ctx echo.Context) error {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest, "name is required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	// Perform HARD DELETE (permanent deletion) using Unscoped().
	// The file must not be locked.
	result := tx.Unscoped().
		Where("name = ? AND user_id = ? AND locked = ?", name, userid, false).
		Delete(&model.File{})

	if result.Error != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to permanently delete file record",
		)
	}

	if result.RowsAffected == 0 {
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found or is currently locked",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file permanently deleted",
	})
}
