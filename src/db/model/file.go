package model

type File struct {
	ID     uint
	Name   string `gorm:"uniqueIndex:idx_user_file_name"`
	UserID string `gorm:"uniqueIndex:idx_user_file_name"`
}
