package models

import (
	"github.com/go-playground/validator/v10"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3"`
	Password string `json:"password" validate:"required,min=6"`
	Address  string `json:"address" validate:"required"`
	Phone    string `json:"phone" validate:"required"`
	Age      int    `json:"age" validate:"required,gt=0"`
	Role     string `json:"role" `
}

func (r *RegisterRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// LoginRequest Request for login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

func (l *LoginRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(l)
}

// OAuthUserRequest Request for Register OAuth
type OAuthUserRequest struct {
	GoogleID string `json:"google_id" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required"`
}

// UserProfileRequest Request for Get and Update Profile
type UserProfileRequest struct {
	ID       string `json:"id" validate:"required"`
	Username string `json:"username" `
	Email    string `json:"email" `
	Address  string `json:"address"`
	Phone    string `json:"phone"`
	Age      int    `json:"age" `
}

func (u *UserProfileRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

type UpdateProfileRequest struct {
	ID           string `json:"id" validate:"required"`
	Username     string `json:"username" validate:"omitempty,min=3,max=50"`
	Email        string `json:"email" validate:"omitempty,email"`
	Address      string `json:"address" validate:"omitempty"`
	Phone        string `json:"phone" validate:"omitempty"`
	Age          int    `json:"age" validate:"omitempty,gt=0"`
	AvatarBase64 string `json:"avatar_base64,omitempty" form:"-"`
	AvatarName   string `json:"avatar_name,omitempty" form:"-"`
	AvatarType   string `json:"avatar_type,omitempty" form:"-"`
}

func (u *UpdateProfileRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// LogoutRequest Request for Logout
type LogoutRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Token  string `json:"token" validate:"required"`
}
