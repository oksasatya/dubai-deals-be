package migrations

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"mail-service/database/seeder"
	"os"
)

func Migrate(db *gorm.DB) error {
	migrations := []*gormigrate.Migration{}

	m := gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              255,
		UseTransaction:            true,
		ValidateUnknownMigrations: true,
	}, migrations)

	autoMigrate := os.Getenv("AUTO_MIGRATE")
	autoDrop := os.Getenv("AUTO_DROP")

	if autoDrop == "true" && autoMigrate == "true" {
		logrus.Println("Running AutoDrop (Rollback all migrations) and AutoMigrate...")

		for i := len(migrations) - 1; i >= 0; i-- {
			if err := m.RollbackMigration(migrations[i]); err != nil {
				return fmt.Errorf("rollback migration %s failed: %v", migrations[i].ID, err)
			}
		}

		if err := m.Migrate(); err != nil {
			return fmt.Errorf("migration failed after drop: %v", err)
		}
		logrus.Println("Running Seeders...")
		seeder.SeedAll(db)
		logrus.Println("AutoMigrate and Seeders completed.")
	} else if autoDrop == "true" {
		logrus.Println("Running AutoDrop (Rollback all migrations)...")

		for i := len(migrations) - 1; i >= 0; i-- {
			if err := m.RollbackMigration(migrations[i]); err != nil {
				return fmt.Errorf("rollback migration %s failed: %v", migrations[i].ID, err)
			}
		}
	} else if autoMigrate == "true" {
		logrus.Println("Running AutoMigrate...")
		if err := m.Migrate(); err != nil {
			return fmt.Errorf("migration failed: %v", err)
		}

		logrus.Println("Running Seeders...")
		seeder.SeedAll(db)
		logrus.Println("AutoMigrate and Seeders completed.")
	} else {
		logrus.Println("Skipping AutoMigrate and AutoDrop.")
	}

	return nil

}
