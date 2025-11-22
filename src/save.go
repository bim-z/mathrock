package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
	"github.com/mathrock-xyz/starducc/src/rock"
	"github.com/mathrock-xyz/starducc/src/storage"
	"github.com/redis/go-redis/v9"
)

func save(ctx echo.Context) error {
	userID := auth.UserId(ctx)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return err
	}

	descriptor, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer descriptor.Close()

	// Hashing dulu
	hasher := sha256.New()
	if _, err := io.Copy(hasher, descriptor); err != nil {
		return err
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	descriptor.Seek(0, 0)

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File    model.File        `gorm:"embedded"`
		Version model.FileVersion `gorm:"embedded"`
	}

	// Pastikan hanya satu record hasil JOIN yang diambil, yaitu yang terbesar (terbaru)
	if err = tx.Table("files").
		Select("files.id AS id, files.name, files.user_id, files.hash, file_versions.id AS version_id, file_versions.version, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ?", fileHeader.Filename, userID).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}

	// Jika hash sama → tidak perlu buat versi baru
	if result.Version.Hash == hash {
		return echo.NewHTTPError(http.StatusBadRequest, "no changes")
	}

	// Upload hanya jika hash belum pernah tersimpan
	if _, err := rock.Rock.Get(ctx.Request().Context(), hash).Result(); err == redis.Nil {
		_, err := storage.Box.PutObject(ctx.Request().Context(), &s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("BUCKET_NAME")),
			Key:    aws.String(hash),
			Body:   descriptor,
		})
		if err != nil {
			return err
		}

		rock.Rock.Set(ctx.Request().Context(), hash, "1", 0)
	}

	// Hitung next version
	nextVersion := result.Version.Version + 1

	newVersion := model.FileVersion{
		FileID:  result.File.ID,
		Version: nextVersion,
		Hash:    hash,
		Size:    fileHeader.Size,
	}

	tx.Create(&newVersion)

	// Update file to latest hash
	tx.Model(&model.File{}).
		Where("id = ?", result.File.ID).
		Update("hash", hash)

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "updated",
		"version": nextVersion,
		"hash":    hash,
	})
}
