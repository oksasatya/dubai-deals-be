package seeder

import (
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func SeedAll(db *mongo.Database) {
	RoleSeeder(db)
	time.Sleep(3 * time.Second)
	SeedUsers(db)
	logrus.Println("Seed all success")
}
