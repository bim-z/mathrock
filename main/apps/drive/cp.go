package drive

import (
	"errors"
	"net/http"

	"github.com/bim-z/mathrock/main/system/auth"
	"github.com/bim-z/mathrock/main/system/box"
	"github.com/bim-z/mathrock/main/system/db"
	"github.com/bim-z/mathrock/main/system/db/model/drive"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func cp(ctx echo.Context) (err error) {
	userid, name, ver := auth.UserId(ctx), ctx.Param("name"), ctx.Param("version")
	if name == "" || ver == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			// "file name and version are required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    drive.File    `gorm:"embedded"`
		Version drive.Version `gorm:"embedded"`
	}

	// query the specific file version record
	if err = db.DB.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash, file_versions.size").
		Joins("INNER JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND file_versions.version = ?", name, userid, ver).
		Limit(1).
		Scan(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound,
				"File or specific version not found",
			)
		}
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"database query failed",
		)
	}

	// fetch the object from S3 storage using the version hash
	data, err := box.Box.Get(result.Version.Hash)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to retrieve file from storage",
		)
	}

	// stream the file content back to the client
	return ctx.Stream(http.StatusOK, "application/octet-stream", data)
}
