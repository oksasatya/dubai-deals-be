package seeder

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	"user-service/core/models"
)

// RoleSeeder is a function to seed roles
func RoleSeeder(db *mongo.Database) {
	roleCollection := db.Collection("roles")

	roles := []any{
		models.Role{
			ID:   primitive.NewObjectID(),
			Name: models.RoleSuperAdmin,
		},

		models.Role{
			ID:   primitive.NewObjectID(),
			Name: models.RoleAdmin,
		},

		models.Role{
			ID:   primitive.NewObjectID(),
			Name: models.RoleUser,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := roleCollection.InsertMany(ctx, roles)
	if err != nil {
		logrus.Fatalf("Seed roles failed: %v", err)
		return
	}

	logrus.Println("Seed roles success")
}
