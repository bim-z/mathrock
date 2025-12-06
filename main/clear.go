package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/server/auth"
	"github.com/mathrock-xyz/starducc/server/db"
	"github.com/mathrock-xyz/starducc/server/db/model"
)

func clear(ctx echo.Context) (err error) {
	id := auth.UserId(ctx)
	fileName := ctx.FormValue("name")
	if fileName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "file name required")
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File      model.File `gorm:"embedded"`
		VersionID string     `gorm:"column:version_id"`
		Hash      string     `gorm:"column:version_hash"`
	}

	// get latest version
	err = tx.Table("files").
		Select("files.id, file_versions.id AS version_id, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ?", fileName, id).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error
	if err != nil || result.VersionID == "" {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}

	latestFileID := result.File.ID
	latestVersionID := result.VersionID
	latestHash := result.Hash

	// delete old versions (keep latest)
	if err = tx.Where("file_id = ? AND id != ?", latestFileID, latestVersionID).
		Delete(&model.FileVersion{}).Error; err != nil {
		return
	}

	// reset latest version to 1
	if err = tx.Model(&model.FileVersion{}).
		Where("id = ?", latestVersionID).
		Update("version", 1).Error; err != nil {
		return
	}

	// update main file metadata
	if err = tx.Model(&model.File{}).
		Where("id = ?", latestFileID).
		Update("hash", latestHash).Error; err != nil {
		return
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message":     "file history reset successfully",
		"new_version": 1,
	})
}
