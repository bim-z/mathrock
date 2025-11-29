package main

import (
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
	"github.com/mathrock-xyz/starducc/src/storage"
	"gorm.io/gorm"
)

func cp(ctx echo.Context) (err error) {
	userID := auth.UserId(ctx)

	fileName := ctx.FormValue("name")
	if fileName == "" {
		return
	}

	// TODO : ubah jadi integer
	fileVersion := ctx.FormValue("version")

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File        `gorm:"embedded"`
		Version model.FileVersion `gorm:"embedded"`
	}

	if err = db.DB.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash, file_versions.size").
		Joins("INNER JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND file_versions.version = ?", fileName, userID, fileVersion).
		Limit(1).
		Scan(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "File or specific version not found")
		}
		return err
	}

	object, err := storage.Box.GetObject(ctx.Request().Context(), &s3.GetObjectInput{
		Key: &result.Version.Hash,
	})

	return ctx.Stream(7, "", object.Body)
}
