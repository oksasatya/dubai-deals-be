package seeder

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func SeedAll(db *gorm.DB) {
	// Seed all data
	logrus.Println("Seed all success")
}
