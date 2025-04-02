package models

import (
	"github.com/go-playground/validator/v10"
)

// AdminCreateRequest adalah model untuk request pembuatan admin
type AdminCreateRequest struct {
	SuperAdminID string `json:"super_admin_id" form:"super_admin_id" validate:"required"`
	Username     string `json:"username" form:"username" validate:"required,min=3,max=50"`
	Email        string `json:"email" form:"email" validate:"required,email"`
	Password     string `json:"password" form:"password" validate:"required,min=6"`
	AvatarBase64 string `json:"avatar_base64,omitempty" form:"-"`
	AvatarName   string `json:"avatar_name,omitempty" form:"-"`
	AvatarType   string `json:"avatar_type,omitempty" form:"-"`
}

// Validate memvalidasi request pembuatan admin
func (r *AdminCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// AdminUpdateRequest adalah model untuk request pembaruan admin
type AdminUpdateRequest struct {
	ID           string `json:"id" form:"id" validate:"required"`
	Username     string `json:"username" form:"username" validate:"omitempty,min=3,max=50"`
	Email        string `json:"email" form:"email" validate:"omitempty,email"`
	Password     string `json:"password" form:"password" validate:"omitempty,min=6"`
	AvatarBase64 string `json:"avatar_base64,omitempty" form:"-"`
	AvatarName   string `json:"avatar_name,omitempty" form:"-"`
	AvatarType   string `json:"avatar_type,omitempty" form:"-"`
}

// Validate memvalidasi request pembaruan admin
func (r *AdminUpdateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

type GetAllAdminRequest struct {
	AdminID   string `json:"admin_id" validate:"required"`
	Role      string `json:"role"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Age       int    `json:"age"`
	AvatarUrl string `json:"avatar_url"`
}
