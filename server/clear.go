package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mathrock-xyz/starducc/src/auth"
	"github.com/mathrock-xyz/starducc/src/db"
	"github.com/mathrock-xyz/starducc/src/db/model"
)

func clear(ctx echo.Context) (err error) {
	id := auth.UserId(ctx)

	tx := db.DB.Begin()
	defer tx.Rollback()

	var result struct {
		File      model.File `gorm:"embedded"`
		VersionID string     `gorm:"column:version_id"`
		Hash      string     `gorm:"column:version_hash"`
	}

	fileName := ctx.FormValue("name")

	// 1. Ambil File ID dan Versi Terbaru (ID, Hash, Version) dalam 1 Query
	err = tx.Table("files").
		Select("files.id, file_versions.id AS version_id, file_versions.hash AS version_hash").
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id").
		Where("files.name = ? AND files.user_id = ?", fileName, id).
		Order("file_versions.version DESC").
		Limit(1).
		Scan(&result).Error

	if err != nil {
		// Handle error atau jika file tidak ditemukan
		return echo.NewHTTPError(http.StatusNotFound, "file not found or no versions exist")
	}

	// Data versi terbaru
	latestFileID := result.File.ID
	latestVersionID := result.VersionID
	latestHash := result.Hash

	// 2. Hapus SEMUA versi file, kecuali versi terbaru (latestVersionID)
	err = tx.Where("file_id = ? AND id != ?", latestFileID, latestVersionID).
		Delete(&model.FileVersion{}).Error

	if err != nil {
		return err
	}

	// 3. Update versi terbaru (yang tersisa) menjadi Version 1
	err = tx.Model(&model.FileVersion{}).
		Where("id = ?", latestVersionID).
		Updates(map[string]interface{}{
			"version": 1,
			// (Optional) Anda mungkin ingin mengupdate kolom created_at/updated_at di sini
		}).Error

	if err != nil {
		return err
	}

	// 4. (Optional) Pastikan tabel files juga diupdate (walaupun di save method sebelumnya sudah diupdate)
	err = tx.Model(&model.File{}).
		Where("id = ?", latestFileID).
		Update("hash", latestHash).Error

	if err != nil {
		return err
	}

	tx.Commit()

	return ctx.JSON(http.StatusOK, echo.Map{
		"message":     "file history reset successfully",
		"new_version": 1,
	})
}
