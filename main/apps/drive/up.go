package drive

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/bim-z/mathrock/main/system/auth"
	"github.com/bim-z/mathrock/main/system/box"
	"github.com/bim-z/mathrock/main/system/db"
	"github.com/bim-z/mathrock/main/system/db/model/drive"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func up(ctx echo.Context) error {
	userid := auth.UserId(ctx)

	file, err := ctx.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}

	descriptor, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to open file",
		)
	}
	defer descriptor.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, descriptor); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to read file data",
		)
	}

	if _, err = descriptor.Seek(0, io.SeekStart); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to reset file reader",
		)
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	if err = box.Box.Put(hash, descriptor); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	record := drive.File{
		ID:     uuid.NewString(),
		UserID: userid,
		Name:   file.Filename,
		Hash:   hash,
		Size:   file.Size,
	}

	if err := db.DB.Create(&record).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to save file record to database",
		)
	}

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file uploaded",
		"hash":    hash,
	})
}
