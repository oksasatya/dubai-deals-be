package database

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"mail-service/database/migrations"
	"os"
	"sync"
)

var (
	db   *gorm.DB
	once sync.Once
)

// InitDB initializes the database connection
func InitDB() (*gorm.DB, error) {
	var err error

	once.Do(func() {
		host := os.Getenv("DB_HOST")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		port := os.Getenv("DB_PORT")

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai", host, user, password, dbname, port)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	})

	// run Migration
	if err := migrations.Migrate(db); err != nil {
		err = fmt.Errorf("migration failed: %v", err)
		return nil, err
	}

	return db, err
}
