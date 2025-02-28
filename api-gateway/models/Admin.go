package models

import (
	"github.com/go-playground/validator/v10"
)

// AdminCreateRequest struct
type AdminCreateRequest struct {
	SuperAdminID string `json:"super_admin_id" validate:"required"`
	Username     string `json:"username" validate:"required,min=3"`
	Email        string `json:"email" validate:"required,email"`
	Password     string `json:"password" validate:"required,min=6"`
	AvatarData   []byte `json:"avatar_data,omitempty"`
	AvatarName   string `json:"avatar_name,omitempty"`
}

// Validate func
func (a *AdminCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(a)
}

// AdminUpdateRequest struct
type AdminUpdateRequest struct {
	ID         string `json:"id" validate:"required"`
	Username   string `json:"username,omitempty"`
	Email      string `json:"email,omitempty" validate:"omitempty,email"`
	Password   string `json:"password,omitempty" validate:"omitempty,min=6"`
	AvatarData []byte `json:"avatar_data,omitempty"`
	AvatarName string `json:"avatar_name,omitempty"`
	AvatarType string `json:"avatar_type,omitempty"`
}

// Validate function for AdminUpdateRequest
func (a *AdminUpdateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(a)
}
