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

func undo(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "file name is required",
		})
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File    `gorm:"embedded"`
		Version model.Version `gorm:"embedded"`
	}

	if err = tx.Table("files").
		Select("files.id as id, files.name, files.locked, files.user_id, files.hash, versions.id as version_id, versions.version, versions.hash as version_hash").
		Joins("left join file_versions on file_versions.file_id = files.id").
		Where("files.name = ? and files.user_id = ? and files.locked = ?", name, userid, false).
		Order("file_versions.version desc").
		Limit(1).
		Scan(&result).Error; err != nil {
		return ctx.JSON(http.StatusNotFound, echo.Map{
			"error": "file not found",
		})
	}

	if result.Version.Ver <= 1 {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "no previous version available",
		})
	}

	var prev model.Version
	if err = tx.Where("file_id = ? and version = ?", result.File.ID, result.Version.Ver-1).
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
