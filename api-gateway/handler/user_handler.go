package handler

import (
	"api-gateway/config"
	"api-gateway/models"
	"api-gateway/utils"
	"api-gateway/webResponse"
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"io"
	"messaging"
	"net/http"
	"strings"
	"user-service/api"
)

type UserHandler struct {
	Config          *config.RateLimitConfig
	RMQ             *messaging.RabbitMQConnection
	SendMessage     *api.SendingMessage
	ResponseHandler *webResponse.ResponseHandler
}

func NewUserHandler(cfg *config.RateLimitConfig, rmq *messaging.RabbitMQConnection, res *webResponse.ResponseHandler) *UserHandler {
	return &UserHandler{
		Config:          cfg,
		RMQ:             rmq,
		ResponseHandler: res,
		SendMessage:     api.NewSendingMessage(rmq),
	}
}

// Register handles user registration event-driven
func (h *UserHandler) Register(c echo.Context) error {
	// Rate Limit
	err := config.CheckRateLimit(c)
	if err != nil {
		return err
	}
	// bind & validate request
	var requestBody models.RegisterRequest
	if err := c.Bind(&requestBody); err != nil {
		return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Invalid request format")
	}

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
	logrus.Infof("Sending UserRegistered event | Correlation ID: %s | Payload: %+v", correlationID, requestBody)
	err = h.SendMessage.SendingToMessage("UserRegistered", correlationID, requestBody)
	if err != nil {
		return err
	}

	logrus.Infof("Waiting for UserRegisteredSuccess/UserRegisteredFailed response (Timeout: %v)", h.Config.RequestTimeout)
	return h.ResponseHandler.HandleEventResponse(
		c,
		false,
		http.StatusCreated,
		h.Config.RequestTimeout,
		"User registered successfully",
		"UserRegisteredSuccess",
		"UserRegisteredFailed",
	)

}

// Login handles user login event-driven
func (h *UserHandler) Login(c echo.Context) error {
	// Rate Limit
	err := config.CheckRateLimit(c)
	if err != nil {
		return err
	}
	// bind & validate request
	var requestBody models.LoginRequest
	if err := c.Bind(&requestBody); err != nil {
		return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Invalid request format")
	}
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

	err = h.SendMessage.SendingToMessage("UserLogin", correlationID, requestBody)
	if err != nil {
		return err
	}

	return h.ResponseHandler.HandleEventResponse(
		c,
		true,
		http.StatusAccepted,
		h.Config.RequestTimeout,
		"User login successfully",
		"UserLoginSuccess",
		"UserLoginFailed",
	)
}

// GetProfile handles user testing
func (h *UserHandler) GetProfile(c echo.Context) error {
	err := config.CheckRateLimit(c)
	if err != nil {
		return err
	}

	claims, ok := c.Get("user").(*utils.JWTCustomClaims)
	if !ok || claims == nil {
		logrus.Error("Invalid claims type or nil claims")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Invalid token claims")
	}

	if claims.UserID == "" {
		logrus.Error("UserID not found in claims")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "UserID not found in token")
	}

	requestBody := models.UserProfileRequest{
		ID: claims.UserID,
	}

	logrus.Infof("Processing GetProfile | UserID: %s", claims.UserID)

	// Generate Correlation ID
	correlationID := utils.GenerateCorrelationID()

	logrus.Infof("Sending GetProfile event | Correlation ID: %s | UserID: %s", correlationID, claims.UserID)

	err = h.SendMessage.SendingToMessage("GetProfile", correlationID, requestBody)
	if err != nil {
		logrus.Errorf("Failed to send GetProfile message: %v", err)
		return webResponse.ResponseJson(c, http.StatusInternalServerError, nil, "Failed to send GetProfile request")
	}

	logrus.Infof("Waiting for GetProfileSuccess/GetProfileFailed response | Timeout: %v", h.Config.RequestTimeout)

	return h.ResponseHandler.HandleEventResponse(
		c,
		false,
		http.StatusOK,
		h.Config.RequestTimeout,
		"Get Profile successfully",
		"GetProfileSuccess",
		"GetProfileFailed",
	)
}

// UpdateProfile handles user profile update event-driven
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	// rate limit
	err := config.CheckRateLimit(c)
	if err != nil {
		return err
	}

	claims, ok := c.Get("user").(*utils.JWTCustomClaims)
	if !ok || claims == nil {
		logrus.Error("Invalid claims type or nil claims")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Invalid token claims")
	}

	if claims.UserID == "" {
		logrus.Error("UserID not found in claims")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "UserID not found in token")
	}

	var requestBody models.UpdateProfileRequest

	contentType := c.Request().Header.Get("Content-Type")
	logrus.Infof("Request Content-Type: %s", contentType)

	if strings.Contains(contentType, "application/json") {
		if err := c.Bind(&requestBody); err != nil {
			logrus.Errorf("Failed to bind JSON request: %v", err)
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Invalid JSON format")
		}

		// Pastikan ID ada
		requestBody.ID = claims.UserID

		logrus.Infof("Received JSON update request for user ID: %s", requestBody.ID)
	} else if strings.Contains(contentType, "multipart/form-data") {
		if err := c.Bind(&requestBody); err != nil {
			logrus.Errorf("Failed to bind form request: %v", err)
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Invalid form format")
		}

		// Pastikan ID ada
		requestBody.ID = claims.UserID

		// Handle file upload avatar
		file, fileHeader, err := c.Request().FormFile("avatar")
		if err == nil && fileHeader != nil {
			if fileHeader.Size > 2*1024*1024 {
				return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Size of avatar file is too large (max 2MB)")
			}

			buffer := bytes.NewBuffer(nil)
			if _, err := io.Copy(buffer, file); err != nil {
				logrus.Errorf("Failed to read avatar file: %v", err)
				return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Failed to read avatar file")
			}

			// Encode ke base64
			base64Data := base64.StdEncoding.EncodeToString(buffer.Bytes())

			requestBody.AvatarBase64 = base64Data
			requestBody.AvatarName = fileHeader.Filename
			requestBody.AvatarType = fileHeader.Header.Get("Content-Type")

			logrus.Infof("Avatar file received: %s, size: %d bytes, type: %s",
				fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
		}
	} else {
		logrus.Error("Unsupported Content-Type")
		return webResponse.ResponseJson(c, http.StatusBadRequest, nil, "Unsupported Content-Type. Use application/json or multipart/form-data")
	}

	// Validasi request body
	if err := requestBody.Validate(); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			formatterErrors := utils.FormatValidationError(&requestBody, validationErrors)
			return webResponse.ResponseJson(c, http.StatusBadRequest, nil, formatterErrors)
		}
		return webResponse.ResponseJson(c, http.StatusBadRequest, nil, err.Error())
	}

	logRequestBody := requestBody
	if requestBody.AvatarBase64 != "" {
		logRequestBody.AvatarBase64 = "[base64 data]"
	}
	logrus.Infof("Update profile request data: %+v", logRequestBody)

	correlationID := utils.GenerateCorrelationID()
	logrus.Infof("Sending UpdateProfile event | Correlation ID: %s | UserID: %s", correlationID, claims.UserID)

	err = h.SendMessage.SendingToMessage("UpdateProfile", correlationID, requestBody)
	if err != nil {
		logrus.Errorf("Failed to send UpdateProfile message: %v", err)
		return webResponse.ResponseJson(c, http.StatusInternalServerError, nil, "Failed to send profile update request")
	}

	logrus.Infof("Waiting for UpdateProfileSuccess/UpdateProfileFailed response | Timeout: %v", h.Config.RequestTimeout)

	// Tunggu respons dari service
	return h.ResponseHandler.HandleEventResponse(
		c,
		false,
		http.StatusOK,
		h.Config.RequestTimeout,
		"Profile updated successfully",
		"UpdateProfileSuccess",
		"UpdateProfileFailed",
	)
}

// Logout handles user logout event-driven
func (h *UserHandler) Logout(c echo.Context) error {
	// rate limit
	if err := config.CheckRateLimit(c); err != nil {
		return err
	}

	userClaims := c.Get("user").(*utils.JWTCustomClaims)
	if userClaims == nil {
		logrus.Error("[Logout] Invalid token claims")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Unauthorized! : No user Id found")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		logrus.Error("[Logout] No Authorization header found")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Missing Authorization header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		logrus.Error("[Logout] No token found")
		return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Invalid Token")
	}

	// Generate Correlation ID
	correlationID := utils.GenerateCorrelationID()

	// send event logout to user-service
	logoutRequest := models.LogoutRequest{
		UserID: userClaims.UserID,
		Token:  token,
	}
	err := h.SendMessage.SendingToMessage("UserLogout", correlationID, logoutRequest)
	if err != nil {
		logrus.Errorf("[Logout] Failed to send message: %v", err)
		return webResponse.ResponseJson(c, http.StatusInternalServerError, nil, "Failed to process logout")
	}

	// wait for response from user-service
	return h.ResponseHandler.HandleEventResponse(
		c,
		false,
		http.StatusOK,
		h.Config.RequestTimeout,
		"User logged out successfully",
		"UserLogoutSuccess",
		"UserLogoutFailed",
	)
}
