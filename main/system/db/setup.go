package db

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Setup() (err error) {
	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  os.Getenv("DB_DSN"),
		PreferSimpleProtocol: true,
	}))

	return
}
