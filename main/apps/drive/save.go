package drive

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func save(ctx echo.Context) error {
	userid := auth.UserId(ctx)

	header, err := ctx.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"file is required",
		)
	}

	descriptor, err := header.Open()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to open file",
		)
	}
	defer descriptor.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, descriptor); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to read file",
		)
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	if _, err := descriptor.Seek(0, 0); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to reset file reader",
		)
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File    `gorm:"embedded"`
		Version model.Version `gorm:"embedded"`
	}

	// query for the last version of the file
	if err = tx.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND files.locked = ?", header.Filename, userid, false).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error; err != nil {
		// return 404 if the file record is not found
		return echo.NewHTTPError(
			http.StatusNotFound,
			"file not found",
		)
	}

	if result.Version.Hash == hash {
		// return 400 if file hash is the same (no changes)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"no changes detected",
		)
	}

	// check if the file object already exists in storage cache (redis)
	_, err = rock.Rock.Get(context.Background(), hash).Result()
	if err == redis.Nil {
		// if not exists, upload the object to S3
		_, err := storage.Box.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("bucket_name")),
			Key:    aws.String(hash),
			Body:   descriptor,
		})
		if err != nil {
			// internal error during S3 upload
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to upload file",
			)
		}

		// store file hash metadata in redis
		if err = rock.Rock.Set(context.Background(), hash, "1", 0).Err(); err != nil {
			// internal error when saving metadata to redis
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to store metadata",
			)
		}
	} else if err != nil && !errors.Is(err, redis.Nil) {
		// internal error during redis status check
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to check file status",
		)
	}

	nextver := result.Version.Ver + 1

	newver := model.Version{
		FileID: result.File.ID,
		Ver:    nextver,
		Hash:   hash,
		Size:   header.Size,
	}

	// create new file version record
	if err := tx.Create(&newver).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to save new file version",
		)
	}

	// update the main file record with the new hash
	if err := tx.Model(&model.File{}).
		Where("id = ?", result.File.ID).
		Update("hash", hash).Error; err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to update file record",
		)
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file updated",
		"version": nextver,
		"hash":    hash,
	})
}
