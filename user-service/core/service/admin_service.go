package service

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"messaging"
	"time"
	"user-service/api"
	"user-service/core/models"
	"user-service/core/repository"
	"user-service/utils"
)

type AdminService interface {
	HandleAdminLogin(ctx context.Context, eventData []byte, correlationID string)
	HandleCreateAdmin(ctx context.Context, eventData []byte, correlationID string)
	HandleGetAllAdmins(ctx context.Context, eventData []byte, correlationID string)
	HandleUpdateAdmin(ctx context.Context, eventData []byte, correlationID string)
	HandleDeleteAdmin(ctx context.Context, eventData []byte, correlationID string)
	HandleGetAdminProfile(ctx context.Context, eventData []byte, correlationID string)
	HandleGetAdminActivityLog(ctx context.Context, eventData []byte, correlationID string)
	HandleAdminLogout(ctx context.Context, eventData []byte, correlationID string)
}

type adminService struct {
	userRepo    repository.UserRepo
	rmq         *messaging.RabbitMQConnection
	sendMessage *api.SendingMessage
}

// HandleAdminLogin is a function to handle admin login
func (c *adminService) HandleAdminLogin(ctx context.Context, eventData []byte, correlationID string) {
	if c.sendMessage == nil {
		logrus.Fatalf("Failed to initialize SendingMessage")
		return
	}
	// Unmarshal event JSON ke struct `AdminLoginEvent`
	var req models.UserLoginEvent
	if err := json.Unmarshal(eventData, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	// Cek if admin already registered
	existingAdmin, err := c.userRepo.FindUserByEmail(ctx, req.Email)
	if existingAdmin == nil {
		errorResponse := c.sendMessage.SendingToMessage("AdminLoginFailed", correlationID, "Admin not found")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminLoginFailed: %v", errorResponse)
		}
		return
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, existingAdmin.Password) {
		errorResponse := c.sendMessage.SendingToMessage("AdminLoginFailed", correlationID, "Invalid password")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminLoginFailed: %v", errorResponse)
		}
		return
	}

	// save to userActivityLog
	SaveActivityLog := models.UserActivityLog{
		ID:                primitive.NewObjectID(),
		UserID:            existingAdmin.ID,
		ActivityType:      "Login",
		ActivityTimestamp: primitive.DateTime(time.Now().Unix()),
	}

	_, err = c.userRepo.SaveToActivityLog(ctx, &SaveActivityLog)
	if err != nil {
		logrus.Errorf("Failed to save user activity log: %v", err)
	}
	successResponse := c.sendMessage.SendingToMessage("AdminLoginSuccess", correlationID, models.UserLoginEvent{
		ID:    existingAdmin.ID.Hex(),
		Email: existingAdmin.Email,
		Role:  existingAdmin.Role,
	})

	if successResponse != nil {
		logrus.Errorf("failed to publish Admin Login Success %v", successResponse)
	}
}

// HandleCreateAdmin is a function to handle admin creation
func (c *adminService) HandleCreateAdmin(ctx context.Context, eventData []byte, correlationID string) {
	if c.sendMessage == nil {
		logrus.Fatalf("Failed to initialize SendingMessage")
		return
	}

	// Unmarshal event JSON ke struct `AdminCreateEvent`
	var req models.AdminCreateEvent
	if err := json.Unmarshal(eventData, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	// Check if super admin is exist
	superAdmin, err := c.userRepo.FindUserByID(ctx, req.SuperAdminID)
	if superAdmin == nil || superAdmin.Role != models.RoleSuperAdmin || err != nil {
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Super Admin not found")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	// hash Password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		logrus.Errorf("Failed to hash password: %v", err)
		return
	}

	// handle save avatar
	var avatarURL string
	if len(req.AvatarData) > 0 {
		avatarURL, err = utils.SaveAvatar(req.AvatarData, req.Username)
		if err != nil {
			errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Failed to save avatar")
			if errorResponse != nil {
				logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
			}
			return
		}
	}

	// save new admin
	newAdmin := models.User{
		ID:        primitive.NewObjectID(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		Role:      models.RoleAdmin,
		Avatar:    avatarURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// save user
	_, err = c.userRepo.SaveUser(ctx, &newAdmin)
	if err != nil {
		logrus.Errorf("Failed to save new admin: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Failed to save admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	// save to userActivityLog
	SaveActivityLog := models.UserActivityLog{
		ID:                primitive.NewObjectID(),
		UserID:            newAdmin.ID,
		ActivityType:      "Create Admin",
		ActivityTimestamp: primitive.DateTime(time.Now().Unix()),
	}

	_, err = c.userRepo.SaveToActivityLog(ctx, &SaveActivityLog)
	if err != nil {
		logrus.Errorf("Failed to save user activity log: %v", err)
	}

	// send success response
	successResponse := c.sendMessage.SendingToMessage("AdminCreateSuccess", correlationID, models.AdminCreateEvent{
		SuperAdminID: superAdmin.ID.Hex(),
		Username:     newAdmin.Username,
		Email:        newAdmin.Email,
		AvatarName:   avatarURL,
	})

	if successResponse != nil {
		logrus.Errorf("Failed to publish AdminCreateSuccess: %v", successResponse)
	}
}

// HandleGetAllAdmins is a function to handle get admins
func (c *adminService) HandleGetAllAdmins(ctx context.Context, eventData []byte, correlationID string) {
	if c.sendMessage == nil {
		logrus.Fatalf("Failed to initialize SendingMessage")
		return
	}

	// get all admins
	admins, err := c.userRepo.FindUserByRole(ctx, models.RoleAdmin)
	if err != nil {
		logrus.Errorf("Failed to get admins: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("GetAdminsFailed", correlationID, "Failed to get admins")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish GetAdminsFailed: %v", errorResponse)
		}
		return
	}

	// send success response
	successResponse := c.sendMessage.SendingToMessage("GetAdminsSuccess", correlationID, admins)
	if successResponse != nil {
		logrus.Errorf("Failed to publish GetAdminsSuccess: %v", successResponse)
	}
}

func (c *adminService) HandleUpdateAdmin(ctx context.Context, eventData []byte, correlationID string) {
	if c.sendMessage == nil {
		logrus.Fatalf("Failed to initialize SendingMessage")
		return
	}

	// Unmarshal event JSON ke struct `AdminUpdateEvent`
	var req models.AdminUpdateEvent
	if err := json.Unmarshal(eventData, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	// check if admin is exist
	existingAdmin, err := c.userRepo.FindUserByID(ctx, req.ID)
	if existingAdmin == nil || err != nil {
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Admin not found")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	newEmail := req.Email
	newUsername := req.Username
	newPassword := req.Password

	switch {
	case newEmail != "":
		existingAdmin.Email = newEmail
	case newUsername != "":
		existingAdmin.Username = newUsername
	case newPassword != "":
		hashedPassword, err := utils.HashPassword(newPassword)
		if err != nil {
			logrus.Errorf("Failed to hash password: %v", err)
			return
		}
		existingAdmin.Password = hashedPassword
	}

	// handle save avatar
	newAvatarUrl := existingAdmin.Avatar
	if len(req.AvatarData) > 0 {
		uploadedAvatarURL, err := utils.SaveAvatar(req.AvatarData, existingAdmin.Username)
		if err != nil {
			errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Failed to save avatar")
			if errorResponse != nil {
				logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
			}
			return
		}
		newAvatarUrl = uploadedAvatarURL
	}

	// update admin
	_, err = c.userRepo.SaveUser(ctx, existingAdmin)
	if err != nil {
		logrus.Errorf("Failed to update admin: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Failed to update admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	updatedAdmin := models.User{
		ID:        existingAdmin.ID,
		Username:  newUsername,
		Email:     newEmail,
		Password:  newPassword,
		Avatar:    newAvatarUrl,
		UpdatedAt: time.Now(),
	}

	// update user
	_, err = c.userRepo.UpdateUser(ctx, &updatedAdmin)
	if err != nil {
		logrus.Errorf("Failed to update admin: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Failed to update admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	// save to userActivityLog
	SaveActivityLog := models.UserActivityLog{
		ID:                primitive.NewObjectID(),
		UserID:            existingAdmin.ID,
		ActivityType:      "Update Admin Data",
		ActivityTimestamp: primitive.DateTime(time.Now().Unix()),
	}

	_, err = c.userRepo.SaveToActivityLog(ctx, &SaveActivityLog)
	if err != nil {
		logrus.Errorf("Failed to save user activity log: %v", err)
	}

	// send success response
	successResponse := c.sendMessage.SendingToMessage("AdminUpdateSuccess", correlationID, updatedAdmin)
	if successResponse != nil {
		logrus.Errorf("Failed to publish AdminUpdateSuccess: %v", successResponse)
	}
}

func (c *adminService) HandleDeleteAdmin(ctx context.Context, eventData []byte, correlationID string) {
	//TODO implement me
	panic("implement me")
}

func (c *adminService) HandleGetAdminProfile(ctx context.Context, eventData []byte, correlationID string) {
	//TODO implement me
	panic("implement me")
}

func (c *adminService) HandleGetAdminActivityLog(ctx context.Context, eventData []byte, correlationID string) {
	//TODO implement me
	panic("implement me")
}

func (c *adminService) HandleAdminLogout(ctx context.Context, eventData []byte, correlationID string) {
	//TODO implement me
	panic("implement me")
}

func NewAdminService(userRepo repository.UserRepo, rmq *messaging.RabbitMQConnection, sendMessage *api.SendingMessage) AdminService {
	return &adminService{
		userRepo:    userRepo,
		rmq:         rmq,
		sendMessage: sendMessage,
	}
}
