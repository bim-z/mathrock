package main

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
	"github.com/mathrock-xyz/starducc/server/auth"
	"github.com/mathrock-xyz/starducc/server/db"
	"github.com/mathrock-xyz/starducc/server/db/model"
	"github.com/mathrock-xyz/starducc/server/rock"
	"github.com/mathrock-xyz/starducc/server/storage"
	"github.com/redis/go-redis/v9"
)

func save(ctx echo.Context) error {
	userID := auth.UserId(ctx)
	if userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "please login before updating")
	}

	header, err := ctx.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}

	descriptor, err := header.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer descriptor.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, descriptor); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read file")
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	if _, err := descriptor.Seek(0, 0); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to reset file reader")
	}

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File        `gorm:"embedded"`
		Version model.FileVersion `gorm:"embedded"`
	}

	if err = tx.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ? AND files.locked = ?", header.Filename, userID, false).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}

	if result.Version.Hash == hash {
		return echo.NewHTTPError(http.StatusBadRequest, "no changes detected")
	}

	_, err = rock.Rock.Get(context.Background(), hash).Result()
	if err == redis.Nil {
		_, err := storage.Box.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("bucket_name")),
			Key:    aws.String(hash),
			Body:   descriptor,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to upload file")
		}

		if err = rock.Rock.Set(context.Background(), hash, "1", 0).Err(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to store metadata")
		}
	} else if err != nil && !errors.Is(err, redis.Nil) {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check file status")
	}

	nextVersion := result.Version.Version + 1

	newVersion := model.FileVersion{
		FileID:  result.File.ID,
		Version: nextVersion,
		Hash:    hash,
		Size:    header.Size,
	}

	if err := tx.Create(&newVersion).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save new file version")
	}

	if err := tx.Model(&model.File{}).Where("id = ?", result.File.ID).Update("hash", hash).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update file record")
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file updated",
		"version": nextVersion,
		"hash":    hash,
	})
}
