package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/main/auth"
	"github.com/mathrock-xyz/starducc/main/db"
	"github.com/mathrock-xyz/starducc/main/db/model"
	"gorm.io/gorm"
)

func revert(ctx echo.Context) (err error) {
	name, verstr := ctx.Param("name"), ctx.Param("version")

	version, err := strconv.Atoi(verstr)
	if err != nil || name == "" || version < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file name or version number")
	}

	id := auth.UserId(ctx)

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File      model.File `gorm:"embedded"`
		VersionID string     `gorm:"column:version_id"`
		Hash      string     `gorm:"column:version_hash"`
	}

	err = tx.Table("files").
		Select("files.id, file_versions.id AS version_id, file_versions.hash AS version_hash").
		Joins("INNER JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND file_versions.version = ?", name, id, version).
		Limit(1).
		Scan(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("File %s version %d not found.", name, version))
		}
		return err
	}

	targetid := result.File.ID
	targetverid := result.VersionID
	targethash := result.Hash

	if err = tx.Unscoped().Where("file_id = ? AND id != ?", targetid, targetverid).
		Delete(&model.Version{}).Error; err != nil {
		return err
	}

	if err = tx.Model(&model.Version{}).
		Where("id = ?", targetverid).
		Update("version", 1).Error; err != nil {
		return err
	}

	if err = tx.Model(&model.File{}).
		Where("id = ?", targetid).
		Update("hash", targethash).Error; err != nil {
		return err
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message":     fmt.Sprintf("File '%s' successfully reverted to version %d, now set as version 1", name, version),
		"new_version": 1,
	})
}
