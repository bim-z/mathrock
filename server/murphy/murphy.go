/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/resend/resend-go/v4"
)

type metadata struct {
	ID   string
	Name string
	Url  string
	Key  string
}

type message struct {
	Space    string
	Password string
	Emails   []string
	Meta     []metadata
}

func main() {}

func sign(db *sql.DB, client *s3.PresignClient) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		var files []string

		if err := c.Bind(&files); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to parse request body")
		}

		var metas []metadata

		for _, name := range files {
			uuid, err := uuid.NewV7()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate unique ID")
			}

			id := uuid.String()

			key := fmt.Sprintf("chunks/%s", id)

			presign, err := client.PresignPutObject(c.Request().Context(), &s3.PutObjectInput{
				Bucket: aws.String("main"),
				Key:    aws.String(key),
			}, func(opts *s3.PresignOptions) {
				opts.Expires = 15 * time.Minute
			})

			if err != nil {
				continue
			}

			metas = append(metas, metadata{
				ID:   id,
				Name: name,
				Url:  presign.URL,
				Key:  key,
			})
		}

		return c.JSON(http.StatusOK, metas)
	}
}

func up(db *sql.DB, client *s3.Client, smtp *resend.Client) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		var msg message
		if err := c.Bind(&msg); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body format")
		}

		ctx := c.Request().Context()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to start database transaction")
		}
		defer tx.Rollback()

		spaceID, err := gonanoid.New()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate space ID")
		}

		if _, err = sq.Insert("spaces").Columns("id", "name", "user_id", "password").
			Values(spaceID, msg.Space, current(c), msg.Password).RunWith(tx).ExecContext(ctx); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save space to database")
		}

		for _, meta := range msg.Meta {
			if _, err = client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String("main"),
				Key:    aws.String(meta.Key),
			}); err != nil {
				var nf *types.NotFound
				if errors.As(err, &nf) {
					return echo.NewHTTPError(http.StatusBadRequest, "File object not found in S3. Please upload it first.")
				}
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify uploaded file in S3")
			}

			if _, err = sq.Insert("files").Columns("id", "name", "space_id").
				Values(meta.ID, meta.Name, spaceID).RunWith(tx).ExecContext(ctx); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save file records")
			}
		}

		if err := tx.Commit(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to commit database transaction")
		}

		if len(msg.Emails) > 0 {
			params := &resend.SendEmailRequest{
				From:    "Acme <onboarding@resend.dev>",
				To:      msg.Emails,
				Html:    "<strong>hello world</strong>",
				Subject: "Hello from Golang",
			}

			if _, err = smtp.Emails.Send(params); err != nil {
				return
			}
		}

		return c.NoContent(http.StatusOK)
	}
}

// list all space
func ls(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		rows, err := sq.Select("id", "name", "password", "expired_date").From("spaces").Where(sq.Eq{"user_id": current(c)}).
			RunWith(db).QueryContext(c.Request().Context())

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch spaces from database")
		}
		defer rows.Close()

		type spaceModel struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Password    string `json:"password"`
			ExpiredDate string `json:"expiredDate"`
		}

		var spaces []spaceModel

		for rows.Next() {
			var s spaceModel
			if err = rows.Scan(&s.ID, &s.Name, &s.Password, &s.ExpiredDate); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse space data")
			}
			spaces = append(spaces, s)
		}

		return c.JSON(http.StatusOK, spaces)
	}
}

// list all file in one space
func show(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		id := c.Param("id")

		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing space ID")
		}

		rows, err := sq.Select("id", "name").From("files").Where(sq.And{sq.Eq{"space_id": id}, sq.Eq{"user_id": current(c)}}).
			RunWith(db).QueryContext(c.Request().Context())

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch files from database")
		}
		defer rows.Close()

		type fileModel struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		var files []fileModel

		for rows.Next() {
			var f fileModel
			if err := rows.Scan(&f.ID, &f.Name); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse file data")
			}
			files = append(files, f)
		}

		return c.JSON(http.StatusOK, files)
	}
}

func view(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		id := c.Param("id")

		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing space ID")
		}

		var spaceName, spacePassword, spaceExpiredDate string

		if err := sq.Select("name", "expired_date", "password").From("spaces").Where(sq.And{sq.Eq{"id": id}, sq.Eq{"user_id": current(c)}}).
			RunWith(db).QueryRowContext(c.Request().Context()).Scan(&spaceName, &spaceExpiredDate, &spacePassword); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "Space not found")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch space details")
		}

		if spacePassword != "" {
			password := c.QueryParam("password")

			if password == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Password is required to view this space")
			}

			if spacePassword != password {
				return echo.NewHTTPError(http.StatusUnauthorized, "Incorrect password")
			}
		}

		rows, err := sq.Select("name").From("files").Where(sq.And{sq.Eq{"space_id": id}, sq.Eq{"user_id": current(c)}}).RunWith(db).
			QueryContext(c.Request().Context())

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch space files")
		}
		defer rows.Close()

		var files []string

		for rows.Next() {
			var file string
			if err := rows.Scan(&file); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse file list")
			}
			files = append(files, file)
		}

		return c.JSON(http.StatusOK, files)
	}
}

func get(db *sql.DB, client *s3.PresignClient) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		spaceID, fileID := c.Param("spaceid"), c.Param("fileid")

		if spaceID == "" || fileID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing space ID or file ID")
		}

		var spacePassword, fileName, fileKey string

		if err = sq.Select("spaces.password", "files.name", "files.key").From("files").
			InnerJoin("spaces on spaces.id = files.space_id").
			Where(sq.And{sq.Eq{"files.id": fileID}, sq.Eq{"files.space_id": spaceID}, sq.Eq{"files.user_id": current(c)}}).
			RunWith(db).QueryRowContext(c.Request().Context()).Scan(&spacePassword, &fileName, &fileKey); err != nil {

			if errors.Is(err, sql.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "File not found")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch file details")
		}

		presign, err := client.PresignGetObject(c.Request().Context(), &s3.GetObjectInput{
			Bucket: aws.String("main"),
			Key:    aws.String(fileKey),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = 1 * time.Hour
		})

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate file download link")
		}

		return c.String(http.StatusOK, presign.URL)
	}
}

func download(db *sql.DB, client *s3.PresignClient) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		id := c.Param("id")

		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing space ID")
		}

		var password string

		if err = sq.Select("password").From("spaces").Where(sq.And{sq.Eq{"id": id}, sq.Eq{"user_id": current(c)}}).
			RunWith(db).QueryRowContext(c.Request().Context()).Scan(&password); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "Space not found")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch space details")
		}

		if password != "" {
			queryPassword := c.QueryParam("password")

			if queryPassword == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Password is required to download this space")
			}

			if password != queryPassword {
				return echo.NewHTTPError(http.StatusUnauthorized, "Incorrect password")
			}
		}

		rows, err := sq.Select("name", "key").From("files").Where(sq.And{sq.Eq{"space_id": id}, sq.Eq{"user_id": current(c)}}).
			RunWith(db).QueryContext(c.Request().Context())

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch files to download")
		}
		defer rows.Close()

		type response struct {
			Name string
			Key  string
		}

		var responses []response

		for rows.Next() {
			var req response
			if err = rows.Scan(&req.Name, &req.Key); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse download data")
			}
			responses = append(responses, req)
		}

		type responseItem struct {
			FileName string `json:"file_name"`
			URL      string `json:"url"`
		}

		var items []responseItem

		for _, res := range responses {
			presign, err := client.PresignGetObject(c.Request().Context(), &s3.GetObjectInput{
				Bucket: aws.String("main"),
				Key:    aws.String(res.Key),
			}, func(opts *s3.PresignOptions) {
				opts.Expires = 1 * time.Hour
			})

			if err == nil {
				items = append(items, responseItem{
					FileName: res.Name,
					URL:      presign.URL,
				})
			}
		}

		return c.JSON(http.StatusOK, items)
	}
}

func drop(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		var spaces []string

		if err = c.Bind(&spaces); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body format")
		}

		if _, err = sq.Delete("spaces").Where(sq.And{sq.Eq{"name": spaces}, sq.Eq{"user_id": current(c)}}).
			RunWith(db).ExecContext(c.Request().Context()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete spaces")
		}

		return c.NoContent(http.StatusOK)
	}
}

func rm(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		query := struct {
			Space string   `query:"space"`
			Files []string `query:"file"`
		}{}

		if err = c.Bind(&query); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid query parameters")
		}

		if _, err = sq.Delete("files").Where(sq.And{sq.Eq{"space_id": query.Space}, sq.Eq{"name": query.Files}, sq.Eq{"user_id": current(c)}}).
			RunWith(db).ExecContext(c.Request().Context()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete files")
		}

		return c.NoContent(http.StatusOK)
	}
}

func clean(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) (err error) {
		if _, err = sq.Delete("spaces").Where(sq.Eq{"user_id": current(c)}).
			RunWith(db).ExecContext(c.Request().Context()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to clean all spaces")
		}

		return c.NoContent(http.StatusOK)
	}
}
