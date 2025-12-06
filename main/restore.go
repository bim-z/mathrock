package main

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
	"github.com/mathrock-xyz/starducc/main/db/model"
	"github.com/mathrock-xyz/starducc/main/storage"
)

func restore(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"file name is required",
		)
	}

	var result struct {
		File    model.File    `gorm:"embedded"`
		Version model.Version `gorm:"embedded"`
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	// query for the latest version of the file
	if err = tx.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ?", name, userid).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found",
		)
	}

	// fetch the object from S3 storage using the file hash
	object, err := storage.
		Box.GetObject(
		ctx.Request().Context(),
		&s3.GetObjectInput{
			Key: &result.Version.Hash,
		})
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to retrieve file from storage",
		)
	}

	// note: we don't defer object.Body.Close() here because ctx.Stream closes it automatically
	// stream the file content back to the client
	return ctx.Stream(http.StatusOK, "application/octet-stream", object.Body)
}
