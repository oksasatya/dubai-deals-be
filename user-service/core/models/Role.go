package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Enum Role USER
const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleAdmin      = "ADMIN"
	RoleUser       = "USER"
)

// Role Model struct
type Role struct {
	ID   primitive.ObjectID `json:"id" bson:"_id"`
	Name string             `json:"name" bson:"name"`
}
