package model

import "gorm.io/gorm"

type FileVersion struct {
	gorm.Model
	Version int
	Hash    string
	Size    int64
	FileID  string
}
