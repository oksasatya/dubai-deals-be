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
	logrus.Infof("Sending AdminRegistered event | Correlation ID: %s | Payload: %+v", correlationID, requestBody)
	err = h.SendMessage.SendingToMessage("AdminRegistered", correlationID, requestBody)
	if err != nil {
		return err
	}

	// event data
	eventData, _ := json.Marshal(requestBody)
	err = h.SendMessage.SendingToMessage("AdminRegistered", correlationID, eventData)

	logrus.Infof("Waiting for response from AdminRegistered event | Correlation ID: %s", correlationID)
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
