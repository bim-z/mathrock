package model

type Hash struct {
	Value string
	Files []File `gorm:"foreignKey:HashID"`
}
