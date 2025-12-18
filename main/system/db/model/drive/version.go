package drive

import "gorm.io/gorm"

type Version struct {
	gorm.Model
	Ver    int
	Hash   string
	Size   int64
	FileID string
}

func (Version) TableName() string {
	return "drive.versions"
}
