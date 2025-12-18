package drive

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func revert(ctx echo.Context) (err error) {
	name, verstr := ctx.Param("name"), ctx.Param("version")
	version, err := strconv.Atoi(verstr)
	if err != nil || name == "" || version < 1 {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"invalid file name or version number",
		)
	}

	id := auth.UserId(ctx)

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File      model.File `gorm:"embedded"`
		VersionID string     `gorm:"column:version_id"`
		Hash      string     `gorm:"column:version_hash"`
	}

	// fetch the specific file version record
	err = tx.Table("files").
		Select("files.id, file_versions.id AS version_id, file_versions.hash AS version_hash").
		Joins("INNER JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND file_versions.version = ?", name, id, version).
		Limit(1).
		Scan(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound,
				fmt.Sprintf("File %s version %d not found.", name, version),
			)
		}
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"database query failed",
		)
	}

	targetid := result.File.ID
	targetverid := result.VersionID
	targethash := result.Hash

	// delete all versions of the file except the target version (soft delete in GORM)
	if err = tx.Unscoped().Where("file_id = ? AND id != ?", targetid, targetverid).
		Delete(&model.Version{}).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to delete old file versions",
		)
	}

	// set the target version number to 1
	if err = tx.Model(&model.Version{}).
		Where("id = ?", targetverid).
		Update("version", 1).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to update version number",
		)
	}

	// update the main file record's hash to the target version's hash
	if err = tx.Model(&model.File{}).
		Where("id = ?", targetid).
		Update("hash", targethash).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to update file record hash",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message":     fmt.Sprintf("File '%s' successfully reverted to version %d, now set as version 1", name, version),
		"new_version": 1,
	})
}
