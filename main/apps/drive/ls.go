package drive

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
	"github.com/mathrock-xyz/starducc/main/db/model"
	"gorm.io/gorm"
)

func ls(ctx echo.Context) (err error) {
	userid, files := auth.UserId(ctx), []model.File{}

	if err = db.DB.Where("user_id = ?", userid).Find(&files).Error; err != nil {
		// Typically, listing files might not return ErrRecordNotFound unless gorm
		// explicitly returns it for zero results, but generally, finding zero results
		// is not considered an error in GORM (err will be nil).
		// We only handle serious database errors here.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			// return 500 internal server error for database query failures
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to fetch file list",
			)
		}
	}

	// return the list of files. If the list is empty, it returns an empty array ([]).
	return ctx.JSON(http.StatusOK, files)
}
