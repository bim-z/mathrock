package main

import (
	"context"
	"fmt"
	"io"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
	"github.com/mathrock-xyz/starducc/src/rock"
	"github.com/redis/go-redis/v9"
)

func up(ctx echo.Context) (err error) {
	id := auth.UserId(ctx)

	hash := ctx.FormValue("hash")

	val, err := rock.Rock.Get(context.Background(), "mykey").Result()

	if err == redis.Nil {
		file, err := ctx.FormFile("file")

		if err != nil {
			return err
		}

		tx := db.DB.Begin()
		defer tx.Rollback()

		if err := tx.Create(&model.File{
			UserID: id,
			Name:   file.Filename,
		}).Error; err != nil {
			_ = tx.Rollback()
			// TODO : return error message
		}

	} else if err != nil {
		return fmt.Errorf("")
	}

}
