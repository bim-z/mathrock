package main

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/server/auth"
	"github.com/mathrock-xyz/starducc/server/db"
	"github.com/mathrock-xyz/starducc/server/db/model"
	"github.com/mathrock-xyz/starducc/server/storage"
)

func undo(ctx echo.Context) error {
	userId := auth.UserId(ctx)
	if userId == "" {
		return ctx.JSON(http.StatusUnauthorized, echo.Map{
			"error": "please login before undoing",
		})
	}

	header, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "file is required",
		})
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File        `gorm:"embedded"`
		Version model.FileVersion `gorm:"embedded"`
	}

	if err = tx.Table("files").
		Select("files.id as id, files.name, files.locked, files.user_id, files.hash, file_versions.id as version_id, file_versions.version, file_versions.hash as version_hash").
		Joins("left join file_versions on file_versions.file_id = files.id").
		Where("files.name = ? and files.user_id = ? and files.locked = ?", header.Filename, userId, false).
		Order("file_versions.version desc").
		Limit(1).
		Scan(&result).Error; err != nil {
		return ctx.JSON(http.StatusNotFound, echo.Map{
			"error": "file not found",
		})
	}

	if result.Version.Version <= 1 {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "no previous version available",
		})
	}

	var prev model.FileVersion
	if err = tx.Where("file_id = ? and version = ?", result.File.ID, result.Version.Version-1).
		First(&prev).Error; err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "failed to load previous version",
		})
	}

	object, err := storage.Box.GetObject(ctx.Request().Context(), &s3.GetObjectInput{
		Bucket: nil,
		Key:    &prev.Hash,
	})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "failed to retrieve file data",
		})
	}

	return ctx.Stream(http.StatusOK, "application/octet-stream", object.Body)
}
