package repository

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	"user-service/core/models"
)

type UserRepo interface {
	SaveUser(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error)
	UpdateUser(ctx context.Context, user *models.User) (*mongo.UpdateResult, error)
	SaveToActivityLog(ctx context.Context, activity *models.UserActivityLog) (*mongo.InsertOneResult, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	FindUserByID(ctx context.Context, id string) (*models.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	GetUserByRole(userID primitive.ObjectID, db *mongo.Database) (models.User, models.Role, error)
	GetRoleNameByID(ctx context.Context, roleID primitive.ObjectID) (string, error)
	GetRoleIDByName(ctx context.Context, roleName string) (primitive.ObjectID, error)
	SaveRole(ctx context.Context, role *models.Role) (*mongo.InsertOneResult, error)
	FindUsersByRole(ctx context.Context, roleID primitive.ObjectID) ([]models.User, error)
}

type userRepo struct {
	db *mongo.Database
}

func (r *userRepo) FindUsersByRole(ctx context.Context, roleID primitive.ObjectID) ([]models.User, error) {
	var users []models.User
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := r.db.Collection("users").Find(ctx, bson.M{"role_id": roleID})
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepo) SaveRole(ctx context.Context, role *models.Role) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.db.Collection("roles").InsertOne(ctx, role)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *userRepo) FindByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	var user models.User
	err := r.db.Collection("users").FindOne(ctx, bson.M{"google_id": googleID}).Decode(&user)
	if err != nil {
		if errors.Is(mongo.ErrNoDocuments, err) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) SaveUser(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.db.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, user *models.User) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(user.ID.Hex())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %v", err)
	}

	result, err := r.db.Collection("users").UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": user})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *userRepo) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.db.Collection("users").FindOne(ctx, map[string]string{"email": email}).Decode(&user)
	if errors.Is(mongo.ErrNoDocuments, err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %v", err)
	}

	err = r.db.Collection("users").FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) SaveToActivityLog(ctx context.Context, activity *models.UserActivityLog) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.db.Collection("userActivityLog").InsertOne(ctx, activity)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *userRepo) GetUserByRole(userID primitive.ObjectID, db *mongo.Database) (models.User, models.Role, error) {
	userCollection := db.Collection("users")
	roleCollection := db.Collection("roles")

	var user models.User
	err := userCollection.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return models.User{}, models.Role{}, err
	}

	var role models.Role
	err = roleCollection.FindOne(context.TODO(), bson.M{"_id": user.RoleID}).Decode(&role)
	if err != nil {
		return user, models.Role{}, err
	}

	return user, role, nil
}

func (r *userRepo) GetRoleNameByID(ctx context.Context, roleID primitive.ObjectID) (string, error) {
	roleCollection := r.db.Collection("roles")

	var role models.Role
	err := roleCollection.FindOne(ctx, bson.M{"_id": roleID}).Decode(&role)
	if err != nil {
		return "", err
	}
	return role.Name, nil
}

func (r *userRepo) GetRoleIDByName(ctx context.Context, roleName string) (primitive.ObjectID, error) {
	roleCollection := r.db.Collection("roles")
	var role models.Role
	err := roleCollection.FindOne(ctx, bson.M{"name": roleName}).Decode(&role)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return role.ID, nil
}

func NewUserRepo(db *mongo.Database) UserRepo {
	return &userRepo{db: db}
}
