package models

import (
	"time"
)

// Event  struct is used for event data
type Event struct {
	EventType     string      `json:"event_type"`
	CorrelationID string      `json:"correlation_id"`
	Timestamp     time.Time   `json:"timestamp"`
	Payload       interface{} `json:"payload"`
}

// UserRegisteredEvent struct is used for user registration event
type UserRegisteredEvent struct {
	Email    string `json:"email,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Address  string `json:"address"`
	Phone    string `json:"phone"`
	Age      int    `json:"age"`
}

// UserLoginEvent UserLoginSuccessEvent User Login Success Event
type UserLoginEvent struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UserOAuthEvent User Register via OAuth
type UserOAuthEvent struct {
	GoogleID string `json:"google_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
}

// UserOAuthSuccessEvent User Register OAuth Success
type UserOAuthSuccessEvent struct {
	Email string `json:"email"`
}

type GetUserProfileEvent struct {
	ID      string `json:"id"`
	Name    string `json:"name" `
	Email   string `json:"email" `
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Age     int    `json:"age" `
	Avatar  string `json:"avatar"`
}

// UpdateProfileEvent struct is used for user/admin update event
type UpdateProfileEvent struct {
	ID       string `json:"id"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Address  string `json:"address,omitempty"`
	Age      int    `json:"age,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

// UserLogoutEvent struct is used for user logout event
type UserLogoutEvent struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// AdminCreateEvent struct is used for admin creation event
type AdminCreateEvent struct {
	SuperAdminID string `json:"super_admin_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	AvatarBase64 string `json:"avatar_base64,omitempty"`
	AvatarName   string `json:"avatar_name,omitempty"`
	AvatarType   string `json:"avatar_type,omitempty"`
	Role         string `json:"role"`
}

// AdminUpdateEvent struct is used for admin update event
type AdminUpdateEvent struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	AvatarBase64 string `json:"avatar_base64,omitempty"`
	AvatarName   string `json:"avatar_name,omitempty"`
	AvatarType   string `json:"avatar_type,omitempty"`
}

// GetAllAdminsEvent struct is used for get all admins event
type GetAllAdminsEvent struct {
	AdminId   string `json:"admin_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}
