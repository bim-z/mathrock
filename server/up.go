package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
	"github.com/mathrock-xyz/starducc/src/rock"
	"github.com/mathrock-xyz/starducc/src/storage"
	"github.com/redis/go-redis/v9"
)

func up(ctx echo.Context) (err error) {
	id := auth.UserId(ctx)

	file, err := ctx.FormFile("file")
	if err != nil {
		return err
	}

	descriptor, err := file.Open()
	if err != nil {
		return err
	}

	defer descriptor.Close()

	hasher := sha256.New()

	if _, err = io.Copy(hasher, descriptor); err != nil {
		return
	}

	if _, err = descriptor.Seek(0, io.SeekStart); err != nil {
		return
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	_, err = rock.Rock.Get(context.Background(), hash).Result()
	if err == redis.Nil {
		if _, err := storage.Box.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("BUCKET_NAME")),
			Key:    aws.String(hash),
			Body:   descriptor,
		}); err != nil {
			return err
		}

		if err = rock.Rock.Set(context.Background(), hash, "1", 0).Err(); err != nil {
			return
		}
	} else if err != nil {
		return err
	}

	record := model.File{
		ID:     uuid.NewString(),
		UserID: id,
		Name:   file.Filename,
		Hash:   hash,
		Size:   file.Size,
	}

	if err := db.DB.Create(&record).Error; err != nil {
		return err
	}

	return ctx.JSON(200, echo.Map{
		"message": "upload ok",
		"hash":    hash,
	})
}
