package seeder

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	"user-service/core/models"
	"user-service/utils"
)

func SeedUsers(db *mongo.Database) {
	userCollection := db.Collection("users")
	roleCollection := db.Collection("roles")

	var roles []models.Role
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Cek jumlah roles sebelum fetching data
	count, err := roleCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		logrus.Fatalf("Error counting roles: %v", err)
	}
	logrus.Infof("Total roles found in DB: %d", count)

	cursor, err := roleCollection.Find(ctx, bson.M{"name": bson.M{"$exists": true}})

	if err != nil {
		logrus.Fatalf("Error fetching roles: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &roles); err != nil {
		logrus.Fatalf("Fetching roles failed: %v", err)
	}

	// Log isi roles setelah fetch
	logrus.Infof("Roles fetched from DB: %+v", roles)

	roleMap := make(map[string]primitive.ObjectID)
	for _, role := range roles {
		logrus.Infof("Mapping Role: %s -> %s", role.Name, role.ID.Hex()) // Debugging
		roleMap[role.Name] = role.ID
	}

	// Log hasil mapping
	logrus.Infof("Final Role Map: %+v", roleMap)

	// Pastikan roleMap tidak kosong
	if len(roleMap) == 0 {
		logrus.Fatal("No roles found in database! Seeder might have failed.")
		return
	}

	var users []any
	password, _ := utils.HashPassword("test12345")
	imageUrl := "https://picsum.photos/200/300"
	for i := 1; i <= 15; i++ {
		gofakeit.Seed(0)
		roleName := gofakeit.RandomString([]string{models.RoleAdmin, models.RoleUser})

		roleId, exist := roleMap[roleName]
		if !exist {
			logrus.Fatalf("Role %s not found in roles collection. Available roles: %+v", roleName, roleMap)
		}

		user := models.User{
			Username: gofakeit.Name(),
			Email:    gofakeit.Email(),
			Address:  gofakeit.Address().Address,
			Age:      gofakeit.Number(18, 60),
			Phone:    gofakeit.Phone(),
			Password: password,
			GoogleID: gofakeit.UUID(),
			Avatar:   imageUrl,
			RoleID:   roleId,
		}
		users = append(users, user)
	}

	roleSuperAdmin, exist := roleMap[models.RoleSuperAdmin]
	if !exist {
		logrus.Fatalf("Role %s not found", models.RoleSuperAdmin)
		return
	}

	superAdmin := models.User{
		Username: "superadmin",
		Email:    "superAdmin@example.net",
		Address:  gofakeit.Address().Address,
		Age:      gofakeit.Number(18, 60),
		Phone:    gofakeit.Phone(),
		Password: password,
		GoogleID: gofakeit.UUID(),
		Avatar:   imageUrl,
		RoleID:   roleSuperAdmin,
	}
	users = append(users, superAdmin)

	_, err = userCollection.InsertMany(ctx, users, options.InsertMany().SetOrdered(false))
	if err != nil {
		logrus.Fatalf("Seed users failed: %v", err)
		return
	}

	logrus.Println("Seed users success")
}
