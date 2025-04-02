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

// HandleCreateAdmin is a function to handle create admin
func (c *adminService) HandleCreateAdmin(ctx context.Context, eventData []byte, correlationID string) {
	if c.sendMessage == nil {
		logrus.Fatalf("Failed to initialize SendingMessage")
		return
	}

	// Unmarshal event JSON ke struct `AdminCreateRequest`
	var req models.AdminCreateEvent
	if err := json.Unmarshal(eventData, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Format data tidak valid")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	// Check if super admin is exist
	superAdmin, err := c.userRepo.FindUserByID(ctx, req.SuperAdminID)
	if err != nil {
		logrus.Errorf("Failed to find super admin: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Super admin tidak ditemukan")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	if superAdmin == nil {
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Super admin tidak ditemukan")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	// Check if super admin is super admin
	superAdminRole, err := c.userRepo.GetRoleNameByID(ctx, superAdmin.RoleID)
	if err != nil {
		logrus.Errorf("Failed to get role name: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Gagal mendapatkan role super admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	if superAdminRole != models.RoleSuperAdmin {
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Super admin tidak memiliki otorisasi")
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
		if err != nil {
			logrus.Errorf("Failed to save role: %v", err)
			errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Gagal menyimpan role admin")
			if errorResponse != nil {
				logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
			}
			return
		}
	}

	// hash Password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		logrus.Errorf("Failed to hash password: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Gagal meng-hash password")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminCreateFailed: %v", errorResponse)
		}
		return
	}

	// handle save avatar
	var avatarURL string
	if req.AvatarBase64 != "" {
		avatarURL, err = utils.UploadAvatarFromBase64(req.AvatarBase64, req.Username, req.AvatarType)
		if err != nil {
			logrus.Errorf("Failed to upload avatar: %v", err)
			errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Gagal menyimpan avatar")
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
		errorResponse := c.sendMessage.SendingToMessage("AdminCreateFailed", correlationID, "Gagal menyimpan admin")
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
		ActivityTimestamp: primitive.NewDateTimeFromTime(time.Now().UTC()),
	}

	_, err = c.userRepo.SaveToActivityLog(ctx, &SaveActivityLog)
	if err != nil {
		logrus.Errorf("Failed to save user activity log: %v", err)
	}

	// send success response
	successResponse := c.sendMessage.SendingToMessage("AdminCreateSuccess", correlationID, map[string]interface{}{
		"id":       newAdmin.ID.Hex(),
		"username": newAdmin.Username,
		"email":    newAdmin.Email,
		"avatar":   avatarURL,
		"role":     models.RoleAdmin,
	})

	if successResponse != nil {
		logrus.Errorf("Failed to publish AdminCreateSuccess: %v", successResponse)
	}
}

func (c *adminService) HandleUpdateAdmin(ctx context.Context, eventData []byte, correlationID string) {
	if c.sendMessage == nil {
		logrus.Fatalf("Failed to initialize SendingMessage")
		return
	}

	// Unmarshal event JSON ke struct `AdminUpdateRequest`
	var req models.AdminUpdateEvent
	if err := json.Unmarshal(eventData, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Format data tidak valid")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	// find admin by ID
	_, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		logrus.Errorf("Invalid admin ID: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "ID admin tidak valid")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	admin, err := c.userRepo.FindUserByID(ctx, req.ID)
	if err != nil {
		logrus.Errorf("Failed to find admin: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Admin tidak ditemukan")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	if admin == nil {
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Admin tidak ditemukan")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	// Check if user is admin
	adminRole, err := c.userRepo.GetRoleNameByID(ctx, admin.RoleID)
	if err != nil {
		logrus.Errorf("Failed to get role name: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Gagal mendapatkan role admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	if adminRole != models.RoleAdmin {
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "User bukan admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	// replace admin data with new data
	var updatedAdmin models.User

	// Create updated admin object
	updatedAdmin = models.User{
		ID:        admin.ID,
		Email:     admin.Email,
		Username:  admin.Username,
		Password:  admin.Password,
		RoleID:    admin.RoleID,
		Avatar:    admin.Avatar,
		CreatedAt: admin.CreatedAt,
		UpdatedAt: time.Now(),
	}

	// Update data admin
	if req.Username != "" {
		updatedAdmin.Username = req.Username
	}

	if req.Email != "" {
		updatedAdmin.Email = req.Email
	}

	if req.Password != "" {
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			logrus.Errorf("Failed to hash password: %v", err)
			errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Gagal meng-hash password")
			if errorResponse != nil {
				logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
			}
			return
		}
		updatedAdmin.Password = hashedPassword
	}

	// save old avatar path
	oldAvatarPath := updatedAdmin.Avatar

	// handle save avatar
	if req.AvatarBase64 != "" {
		avatarURL, err := utils.UploadAvatarFromBase64(req.AvatarBase64, updatedAdmin.Username, req.AvatarType)
		if err != nil {
			logrus.Errorf("Failed to upload avatar: %v", err)
			errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Gagal menyimpan avatar")
			if errorResponse != nil {
				logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
			}
			return
		}
		updatedAdmin.Avatar = avatarURL
	}

	// Update admin di database
	_, err = c.userRepo.UpdateUser(ctx, &updatedAdmin)
	if err != nil {
		logrus.Errorf("Failed to update admin: %v", err)
		errorResponse := c.sendMessage.SendingToMessage("AdminUpdateFailed", correlationID, "Gagal memperbarui admin")
		if errorResponse != nil {
			logrus.Errorf("Failed to publish AdminUpdateFailed: %v", errorResponse)
		}
		return
	}

	// delete old avatar if exist
	if oldAvatarPath != "" && oldAvatarPath != updatedAdmin.Avatar {
		err = utils.DeleteOldAvatar(oldAvatarPath)
		if err != nil {
			logrus.Warnf("Failed to delete old avatar: %v", err)
		}
	}

	// save to userActivityLog
	SaveActivityLog := models.UserActivityLog{
		ID:                primitive.NewObjectID(),
		UserID:            updatedAdmin.ID,
		ActivityType:      "Update Admin",
		ActivityTimestamp: primitive.NewDateTimeFromTime(time.Now().UTC()),
	}

	_, err = c.userRepo.SaveToActivityLog(ctx, &SaveActivityLog)
	if err != nil {
		logrus.Errorf("Failed to save user activity log: %v", err)
	}

	// send success response
	successResponse := c.sendMessage.SendingToMessage("AdminUpdateSuccess", correlationID, map[string]interface{}{
		"id":       updatedAdmin.ID.Hex(),
		"username": updatedAdmin.Username,
		"email":    updatedAdmin.Email,
		"avatar":   updatedAdmin.Avatar,
		"role":     models.RoleAdmin,
	})

	if successResponse != nil {
		logrus.Errorf("Failed to publish AdminUpdateSuccess: %v", successResponse)
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

	// create response
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
