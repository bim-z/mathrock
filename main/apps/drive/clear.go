package drive

import (
	"errors"
	"net/http"

	"github.com/bim-z/mathrock/main/system/auth"
	"github.com/labstack/echo/v4"
)

func Clear(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.FormValue("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest, "file name required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File      model.File `gorm:"embedded"`
		VersionID uint       `gorm:"column:version_id"`
		Hash      string     `gorm:"column:version_hash"`
	}

	// get the latest version record (we only need the file ID, latest version ID, and the hash)
	err = tx.Table("files").
		Select("files.id, file_versions.id AS version_id, versions.hash AS version_hash").
		Joins("INNER JOIN file_versions ON versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ?", name, userid).
		Order("versions.version DESC").
		Limit(1).
		Scan(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound,
				"file not found or has no versions",
			)
		}
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"database query failed to find latest version",
		)
	}

	latestid := result.File.ID
	latestverid := result.VersionID
	latesthash := result.Hash

	if err = tx.Unscoped().
		Where("file_id = ? AND id != ?", latestid, latestverid).
		Delete(&model.Version{}).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to delete old versions",
		)
	}

	if err = tx.Model(&model.Version{}).
		Where("id = ?", latestverid).
		Update("version", 1).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to reset version number",
		)
	}

	// update the main file record's hash to the latest version's hash (which is the correct hash, not the version ID)
	if err = tx.Model(&model.File{}).
		Where("id = ?", latestid).
		Update("hash", latesthash).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to update file record hash",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message":     "file history reset successfully",
		"new_version": 1,
	})
}
