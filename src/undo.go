package main

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
	"github.com/mathrock-xyz/starducc/src/storage"
)

func undo(ctx echo.Context) (err error) {
	id := auth.UserId(ctx)

	file, err := ctx.FormFile("file")
	if err != nil {
		return
	}

	tx := db.DB.Begin()

	var result struct {
		File    model.File        `gorm:"embedded"`
		Version model.FileVersion `gorm:"embedded"`
	}

	if err = tx.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ?", file.Filename, id).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}

	if result.Version.Version == 1 {
		return
	}

	version := new(model.FileVersion)
	if err = tx.Where("file_id = ? AND version = ?", result.File.ID, result.Version.Version-1).
		First(&version).Error; err != nil {
		return
	}

	object, err := storage.Box.GetObject(ctx.Request().Context(), &s3.GetObjectInput{
		Key: &version.Hash,
	})

	return ctx.Stream(7, "", object.Body)
}
