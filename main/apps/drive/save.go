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
	"github.com/labstack/echo/v4"
)

func save(ctx echo.Context) (err error) {
	header, err := ctx.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"file is required",
		)
	}

	descriptor, err := header.Open()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to open file",
		)
	}
	defer descriptor.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, descriptor); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to read file",
		)
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	if _, err := descriptor.Seek(0, 0); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to reset file reader",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    drive.File    `gorm:"embedded"`
		Version drive.Version `gorm:"embedded"`
	}

	userid := auth.UserId(ctx)

	// query for the last version of the file
	if err = tx.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND files.locked = ?", header.Filename, userid, false).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error; err != nil {
		// return 404 if the file record is not found
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found",
		)
	}

	if result.Version.Hash == hash {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"no changes detected",
		)
	}

	if err = box.Box.Put(hash, descriptor); err != nil {
		return
	}

	nextver := result.Version.Ver + 1

	newver := drive.Version{
		FileID: result.File.ID,
		Ver:    nextver,
		Hash:   hash,
		Size:   header.Size,
	}

	// create new file version record
	if err := tx.Create(&newver).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to save new file version",
		)
	}

	// update the main file record with the new hash
	if err := tx.Model(&drive.File{}).
		Where("id = ?", result.File.ID).
		Update("hash", hash).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to update file record",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file updated",
		"version": nextver,
		"hash":    hash,
	})
}
