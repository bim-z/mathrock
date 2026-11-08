/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/markbates/goth/gothic"
	"github.com/spf13/viper"
)

func auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) (err error) {
		header := ctx.Request().Header.Get("Authorization")
		if header == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is required")
		}

		if !strings.HasPrefix(header, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format. Use 'Bearer <token>'")
		}

		tokeString := header[len("Bearer "):]
		token, err := verifyToken(tokeString)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
		}

		claims := token.Claims.(jwt.MapClaims)
		ctx.Set("email", claims["email"])

		return next(ctx)
	}
}

func current(ctx *echo.Context) string {
	return ctx.Get("id").(string)
}

func begin(c *echo.Context) (err error) {
	q := c.Request().URL.Query()
	q.Add("provider", "github")
	c.Request().URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Response(), c.Request())
	return
}

func callback(db *sql.DB, log *log.Logger) echo.HandlerFunc {
	return func(ctx *echo.Context) (err error) {
		user, err := gothic.CompleteUserAuth(ctx.Response(), ctx.Request())

		if err != nil {
			return echo.NewHTTPError(http.StatusConflict, "")
		}

		if _, err := sq.Insert("users").Columns("id", "name", "email").
			Values(uuid.NewString, user.Name, user.Email).RunWith(db).
			ExecContext(ctx.Request().Context()); err != nil {
			return fmt.Errorf("")
		}

		token, err := generateToken(user.Email, log)
		if err != nil {
			return err
		}

		return ctx.Redirect(http.StatusOK, fmt.Sprintf("http://localhost:7669/%s", token))
	}
}

func verifyToken(t string) (token *jwt.Token, err error) {
	token, err = jwt.Parse(t, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method")
		}
		return []byte(viper.GetString("jwt_secret")), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("Invalid token")
	}

	return
}

func generateToken(email string, log *log.Logger) (string, error) {
	secret := viper.GetString("jwt_secret")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(secret))
}
