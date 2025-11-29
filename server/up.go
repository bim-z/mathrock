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
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/server/auth"
	"github.com/mathrock-xyz/starducc/server/db"
	"github.com/mathrock-xyz/starducc/server/db/model"
	"github.com/mathrock-xyz/starducc/server/rock"
	"github.com/mathrock-xyz/starducc/server/storage"
	"github.com/redis/go-redis/v9"
)

func up(ctx echo.Context) error {
	userId := auth.UserId(ctx)
	if userId == "" {
		return ctx.JSON(echo.ErrUnauthorized.Code, echo.Map{
			"error": "unauthorize",
		})
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSON(echo.ErrBadRequest.Code, echo.Map{
			"error": "file is required",
		})
	}

	descriptor, err := file.Open()
	if err != nil {
		return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
			"error": "failed to open file",
		})
	}
	defer descriptor.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, descriptor); err != nil {
		return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
			"error": "failed to read file data",
		})
	}

	if _, err = descriptor.Seek(0, io.SeekStart); err != nil {
		return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
			"error": "failed to reset file reader",
		})
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	_, err = rock.Rock.Get(context.Background(), hash).Result()
	if err == redis.Nil {
		_, err = storage.Box.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("bucket_name")),
			Key:    aws.String(hash),
			Body:   descriptor,
		})
		if err != nil {
			return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
				"error": "failed to upload file to storage",
			})
		}

		if err = rock.Rock.Set(context.Background(), hash, "1", 0).Err(); err != nil {
			return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
				"error": "failed to save file metadata",
			})
		}
	} else if err != nil && !errors.Is(err, redis.Nil) {
		return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
			"error": "unexpected error while checking file status",
		})
	}

	record := model.File{
		ID:     uuid.NewString(),
		UserID: userId,
		Name:   file.Filename,
		Hash:   hash,
		Size:   file.Size,
	}

	if err := db.DB.Create(&record).Error; err != nil {
		return ctx.JSON(echo.ErrInternalServerError.Code, echo.Map{
			"error": "failed to save file",
		})
	}

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "file uploaded",
		"hash":    hash,
	})
}
