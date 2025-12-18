package drive

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
)

func undo(ctx echo.Context) (err error) {
	userid, name := auth.UserId(ctx), ctx.Param("name")
	if name == "" {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"file name is required",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File    `gorm:"embedded"`
		Version model.Version `gorm:"embedded"`
	}

	// query for the latest version of the unlocked file
	if err = tx.Table("files").
		Select("files.id as id, files.name, files.locked, files.user_id, files.hash, file_versions.id as version_id, file_versions.version, file_versions.hash as version_hash").
		Joins("left join file_versions on file_versions.file_id = files.id").
		Where("files.name = ? and files.user_id = ? and files.locked = ?", name, userid, false).
		Order("file_versions.version desc").
		Limit(1).
		Scan(&result).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found",
		)
	}

	if result.Version.Ver <= 1 {
		// return 400 bad request if there's no version to undo (only version 1 exists)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"no previous version available",
		)
	}

	var prev model.Version
	// query the previous version (current_version - 1)
	if err = tx.Where("file_id = ? and version = ?", result.File.ID, result.Version.Ver-1).
		First(&prev).Error; err != nil {
		// internal error if unable to load the previous version record
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to load previous version",
		)
	}

	// fetch the object from S3 storage using the previous version's hash
	object, err := storage.Box.GetObject(ctx.Request().Context(), &s3.GetObjectInput{
		Bucket: aws.String("default"),
		Key:    &prev.Hash,
	})
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to retrieve file data",
		)
	}

	// stream the previous file content back to the client
	return ctx.Stream(http.StatusOK, "application/octet-stream", object.Body)
}
