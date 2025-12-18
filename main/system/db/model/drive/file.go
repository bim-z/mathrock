package drive

import "gorm.io/gorm"

type File struct {
	gorm.Model
	ID       string
	Name     string `gorm:"uniqueIndex:idx_user_file_name"`
	Hash     string
	Size     int64
	Locked   bool
	Versions []Version
	UserID   string `gorm:"uniqueIndex:idx_user_file_name"`
}

func (File) TableName() string {
	return "drive.file"
}
