package handler

import (
	"api-gateway/config"
	"api-gateway/models"
	"api-gateway/utils"
	"api-gateway/webResponse"
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"messaging"
	"net/http"
	"user-service/api"
	modelsUser "user-service/core/models"
)

type AdminHandler struct {
	Config          *config.RateLimitConfig
	RMQ             *messaging.RabbitMQConnection
	SendMessage     *api.SendingMessage
	ResponseHandler *webResponse.ResponseHandler
}

func NewAdminHandler(cfg *config.RateLimitConfig, rmq *messaging.RabbitMQConnection, res *webResponse.ResponseHandler) *AdminHandler {
	return &AdminHandler{
		Config:          cfg,
		RMQ:             rmq,
		ResponseHandler: res,
		SendMessage:     api.NewSendingMessage(rmq),
	}
}

// CreateAdmin handles admin registration event-driven
func (h *AdminHandler) CreateAdmin(c echo.Context) error {
	// Rate Limit
	err := config.CheckRateLimit(c)
	if err != nil {
		return err
	}

	// get token from header role superadmin
	claims, ok := c.Get("role").(*utils.JWTCustomClaims)
	if !ok || claims == nil || claims.Role != modelsUser.RoleSuperAdmin {
		logrus.Errorf("Invalid Claims type or Role is not SuperAdmin")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Unauthorized")
	}

	requestBody := models.AdminCreateRequest{
		SuperAdminID: claims.ID,
		Username:     c.FormValue("username"),
		Email:        c.FormValue("email"),
		Password:     c.FormValue("password"),
	}

	// check if avatar data is not empty
	file, fileHeader, err := c.Request().FormFile("avatar")
	if err == nil {
		avatarData := make([]byte, fileHeader.Size)
		_, err = file.Read(avatarData)
		if err != nil {
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Failed to read avatar data")
		}

		requestBody.AvatarData = avatarData
		requestBody.AvatarName = fileHeader.Filename
	}

	// bind & validate request
	if err := requestBody.Validate(); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			formatterErrors := utils.FormatValidationError(&requestBody, validationErrors)
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, formatterErrors)
		}
		return webResponse.ResponseJson(c, http.StatusBadRequest, nil, err.Error())
	}

	// Generate Correlation ID
	correlationID := utils.GenerateCorrelationID()
	logrus.Infof("Sending AdminCreate event | Correlation ID: %s | Payload: %+v", correlationID, requestBody)
	err = h.SendMessage.SendingToMessage("AdminCreate", correlationID, requestBody)
	if err != nil {
		return err
	}

	// event data
	eventData, _ := json.Marshal(requestBody)
	err = h.SendMessage.SendingToMessage("AdminCreate", correlationID, eventData)

	logrus.Infof("Waiting for response from AdminCreate event | Correlation ID: %s", correlationID)
	return h.ResponseHandler.HandleEventResponse(
		c,
		false,
		http.StatusCreated,
		h.Config.RequestTimeout,
		"Admin registered successfully",
		"AdminRegisteredSuccess",
		"AdminRegisteredFailed",
	)
}

// UpdateAdmin handles admin update event-driven
func (h *AdminHandler) UpdateAdmin(c echo.Context) error {
	if err := config.CheckRateLimit(c); err != nil {
		return err
	}

	// get token from header role superadmin and admin
	claims, ok := c.Get("role").(*utils.JWTCustomClaims)
	if !ok || claims == nil || claims.Role != modelsUser.RoleSuperAdmin && claims.Role != modelsUser.RoleAdmin {
		logrus.Errorf("Invalid Claims type or Role is not SuperAdmin or Admin")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Unauthorized")
	}

	requestBody := models.AdminUpdateRequest{
		ID:       c.FormValue("id"),
		Username: c.FormValue("username"),
		Email:    c.FormValue("email"),
		Password: c.FormValue("password"),
	}

	// check if avatar data is not empty
	file, fileHeader, err := c.Request().FormFile("avatar")
	if err == nil {
		avatarData := make([]byte, fileHeader.Size)
		_, err = file.Read(avatarData)
		if err != nil {
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Failed to read avatar data")
		}

		requestBody.AvatarData = avatarData
		requestBody.AvatarName = fileHeader.Filename
	}

	// bind & validate request
	if err := requestBody.Validate(); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			formatterErrors := utils.FormatValidationError(&requestBody, validationErrors)
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, formatterErrors)
		}
		return webResponse.ResponseJson(c, http.StatusBadRequest, nil, err.Error())
	}

	// Generate Correlation ID
	correlationID := utils.GenerateCorrelationID()
	logrus.Infof("Sending AdminUpdated event | Correlation ID: %s | Payload: %+v", correlationID, requestBody)

	eventData, _ := json.Marshal(requestBody)
	err = h.SendMessage.SendingToMessage("AdminUpdated", correlationID, eventData)
	if err != nil {
		return err
	}

	logrus.Infof("Waiting for response from AdminUpdated event | Correlation ID: %s", correlationID)

	return h.ResponseHandler.HandleEventResponse(
		c,
		false,
		http.StatusAccepted,
		h.Config.RequestTimeout,
		"Admin updated successfully",
		"AdminUpdatedSuccess",
		"AdminUpdatedFailed",
	)
}

// GetAllAdmin handles get all admin event-driven
func (h *AdminHandler) GetAllAdmin(c echo.Context) error {
	if err := config.CheckRateLimit(c); err != nil {
		return err
	}

	claims, ok := c.Get("user").(*utils.JWTCustomClaims)
	if !ok || claims == nil {
		logrus.Errorf("[API-Gateway] Failed to get JWT claims from context")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Unauthorized")
	}

	logrus.Infof("[API-Gateway] Token Claims: UserID=%s, Role=%s, Email=%s", claims.UserID, claims.Role, claims.Email)

	if claims.Role != modelsUser.RoleAdmin && claims.Role != modelsUser.RoleSuperAdmin {
		logrus.Warn("[API-Gateway] Unauthorized role attempted to access get-admin API")
		return webResponse.ResponseJson(c, http.StatusForbidden, nil, "Forbidden")
	}

	payload := models.GetAllAdminRequest{
		Role: claims.Role,
	}

	err := h.SendMessage.SendingToMessage("GetAllAdmin", utils.GenerateCorrelationID(), payload)

	if err != nil {
		logrus.Errorf("[API-Gateway] Failed to send GetAllAdmin event: %v", err)
		return err
	}

	return h.ResponseHandler.HandleEventResponse(
		c, false, http.StatusOK, h.Config.RequestTimeout,
		"Get all admin successfully", "GetAdminSuccess", "GetAdminFailed",
	)
}
