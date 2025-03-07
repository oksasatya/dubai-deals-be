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
	HandleCreateAdmin(ctx context.Context, eventData []byte, correlationID string)
	HandleGetAllAdmins(ctx context.Context, eventData []byte, correlationID string)
	HandleUpdateAdmin(ctx context.Context, eventData []byte, correlationID string)
	HandleDeleteAdmin(ctx context.Context, eventData []byte, correlationID string)
	HandleGetAdminProfile(ctx context.Context, eventData []byte, correlationID string)
	HandleGetAdminActivityLog(ctx context.Context, eventData []byte, correlationID string)
}

type adminService struct {
	userRepo    repository.UserRepo
	rmq         *messaging.RabbitMQConnection
	sendMessage *api.SendingMessage
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
	if err != nil || superAdmin == nil {
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Super admin not found")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	// Check if super admin is super admin
	superAdminRole, err := c.userRepo.GetRoleNameByID(ctx, superAdmin.RoleID)
	if err != nil || superAdminRole != models.RoleSuperAdmin {
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Super admin not authorized")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	roleID, err := c.userRepo.GetRoleIDByName(ctx, models.RoleAdmin)
	if err != nil {
		roleID = primitive.NewObjectID()
		newRole := models.Role{
			ID:   roleID,
			Name: models.RoleAdmin,
		}

		_, err = c.userRepo.SaveRole(ctx, &newRole)
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
		RoleID:    roleID,
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
		Role:         models.RoleAdmin,
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

	// get role id from db
	roleID, err := c.userRepo.GetRoleIDByName(ctx, models.RoleAdmin)
	if err != nil {
		logrus.Errorf("[admin-service] Failed to get Role ID for Admins: %v", err)
		_ = c.sendMessage.SendingToMessage("GetAdminFailed", correlationID, "Failed to get role ID")
		return
	}

	// get all user role admin
	admins, err := c.userRepo.FindUsersByRole(ctx, roleID)
	if err != nil {
		logrus.Errorf("[admin-service] Failed to get admins: %v", err)
		_ = c.sendMessage.SendingToMessage("GetAdminFailed", correlationID, "Failed to get admins")
		return
	}

	// Konversi hasil query ke response
	var adminResponses []models.GetAllAdminsEvent
	for _, admin := range admins {
		adminResponses = append(adminResponses, models.GetAllAdminsEvent{
			AdminId:   admin.ID.Hex(),
			Username:  admin.Username,
			Email:     admin.Email,
			AvatarURL: admin.Avatar,
			Role:      models.RoleAdmin,
		})
	}

	// send response to message broker
	responseData, err := json.Marshal(adminResponses)
	if err != nil {
		logrus.Errorf("[admin-service] Failed to marshal response data: %v", err)
		_ = c.sendMessage.SendingToMessage("GetAdminFailed", correlationID, "Failed to encode response")
		return
	}

	if err := c.sendMessage.SendingToMessage("GetAdminSuccess", correlationID, responseData); err != nil {
		logrus.Errorf("[admin-service] Failed to publish GetAdminSuccess: %v", err)
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

func NewAdminService(userRepo repository.UserRepo, rmq *messaging.RabbitMQConnection, sendMessage *api.SendingMessage) AdminService {
	return &adminService{
		userRepo:    userRepo,
		rmq:         rmq,
		sendMessage: sendMessage,
	}
}
