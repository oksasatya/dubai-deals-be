package webResponse

import (
	"api-gateway/config"
	"api-gateway/utils"
	"encoding/base64"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"messaging"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Meta struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Status  string `json:"status"`
}

type ResponseHandler struct {
	RMQ *messaging.RabbitMQConnection
}

// NewResponseHandler creates a new instance of ResponseHandler
func NewResponseHandler(rmq *messaging.RabbitMQConnection) *ResponseHandler {
	if rmq == nil {
		logrus.Fatal("NewResponseHandler: RabbitMQ connection is nil!")
	}
	return &ResponseHandler{
		RMQ: rmq,
	}
}

type Response struct {
	Meta Meta        `json:"meta"`
	Data interface{} `json:"data"`
}

// ResponseJson is a utility function that sends a JSON response to the client
func ResponseJson(c echo.Context, status int, payload interface{}, message string) error {
	response := Response{
		Meta: Meta{
			Message: message,
			Code:    status,
			Status:  getStatusText(status),
		},
		Data: payload,
	}
	return c.JSON(status, response)
}

func getStatusText(status int) string {
	if status >= 200 && status < 300 {
		return "success"
	} else if status >= 400 && status < 500 {
		return "fail"
	} else {
		return "error"
	}
}

// HandleEventResponse handles event-based response waiting and processing
func (h *ResponseHandler) HandleEventResponse(c echo.Context, generateToken bool, statusCode int, timeout time.Duration, message string, eventName ...string) error {
	if h == nil {
		logrus.Fatal("HandleEventResponse: ResponseHandler is nil!")
		return ResponseJson(c, http.StatusInternalServerError, nil, "Internal Server Error: ResponseHandler is nil")
	}

	if h.RMQ == nil {
		logrus.Fatal("HandleEventResponse: RabbitMQ connection is nil!")
		return ResponseJson(c, http.StatusInternalServerError, nil, "Internal Server Error: RabbitMQ connection is nil")
	}

	responseEvent, err := messaging.WaitForEvent(h.RMQ, timeout, "api-gateway", eventName...)
	if err != nil {
		logrus.Errorf("Event timeout while waiting for: %s", eventName)
		return ResponseJson(c, http.StatusGatewayTimeout, nil, "Request timed out waiting for response")
	}

	logrus.Infof("Received event: %s | CorrelationID: %s", eventName, responseEvent.CorrelationID)
	ctx := c.Request().Context()

	var jsonResponse any
	if payloadStr, ok := responseEvent.Payload.(string); ok {
		decodedPayload, err := base64.StdEncoding.DecodeString(payloadStr)
		if err == nil {
			logrus.Infof("[ResponseHandler] Decoded payload from Base64: %s", string(decodedPayload))
			payloadStr = string(decodedPayload)
		} else {
			logrus.Warnf("[ResponseHandler] Payload is not Base64-encoded, using raw string")
		}

		if json.Valid([]byte(payloadStr)) {
			if err := json.Unmarshal([]byte(payloadStr), &jsonResponse); err != nil {
				logrus.Errorf("[ResponseHandler] Failed to parse event payload: %v", err)
				return ResponseJson(c, http.StatusInternalServerError, nil, "Failed to parse response")
			}
		} else {
			logrus.Warnf("[ResponseHandler] Payload is not valid JSON, returning as string")
			return ResponseJson(c, http.StatusOK, payloadStr, message)
		}
	} else if payloadMap, ok := responseEvent.Payload.(map[string]interface{}); ok {
		jsonResponse = payloadMap
	} else {
		logrus.Errorf("[ResponseHandler] Unexpected payload type: %T", responseEvent.Payload)
		return ResponseJson(c, http.StatusInternalServerError, nil, "Unexpected event payload format")
	}

	ttlHoursStr := os.Getenv("JWT_EXPIRATION_TIME")
	ttlHours := 72
	if ttlHoursStr != "" {
		var err error
		ttlHours, err = strconv.Atoi(ttlHoursStr)
		if err != nil {
			logrus.Errorf("Invalid JWT_EXPIRATION_TIME value, using default: %v", err)
		}
	}

	for _, expectedEvent := range eventName {
		if responseEvent.EventType == expectedEvent {
			if jsonResponseArray, ok := jsonResponse.([]interface{}); ok {
				return ResponseJson(c, statusCode, jsonResponseArray, message)
			}

			if jsonResponseMap, ok := jsonResponse.(map[string]interface{}); ok {
				delete(jsonResponseMap, "password")
				jsonResponse = jsonResponseMap
			}

			// handle logout
			if expectedEvent == "UserLogoutSuccess" {
				logrus.Infof("processing logout event ...")
				authHeader := c.Request().Header.Get("Authorization")
				if authHeader != "" {
					token := strings.TrimPrefix(authHeader, "Bearer ")
					if token == "" {
						logrus.Error("No token found")
						return ResponseJson(c, http.StatusUnauthorized, nil, "Invalid Token")
					}

					redisErr := config.BlacklistToken(token, time.Duration(ttlHours)*time.Hour)
					if redisErr != nil {
						logrus.Errorf("Failed to blacklist token: %v", redisErr)
						return ResponseJson(c, http.StatusInternalServerError, nil, "Failed to blacklist token")
					}
					logrus.Infof("Blacklisted token in redis: %s", token)
					return ResponseJson(c, http.StatusOK, jsonResponse, message)
				}
			}

			//logrus.Infof("Received expected event: %s", expectedEvent)
			if generateToken {
				jsonResponseMap, ok := jsonResponse.(map[string]interface{})
				if !ok {
					logrus.Errorf("Cannot generate token: response is not a JSON object")
					return ResponseJson(c, http.StatusInternalServerError, nil, "Invalid response format for token generation")
				}
				userID, _ := jsonResponseMap["id"].(string)
				userEmail, _ := jsonResponseMap["email"].(string)
				userRole, _ := jsonResponseMap["role"].(string)
				token, err := utils.GenerateToken(userID, userEmail, userRole)
				if err != nil {
					logrus.Error("Failed to generate token")
					return ResponseJson(c, http.StatusInternalServerError, nil, "Failed to generate token")
				}

				// Redis client
				storeToken := config.StoreTokenInRedis(ctx, token, userID, userRole, ttlHours)
				if storeToken != nil {
					logrus.Errorf("Failed to store token in Redis: %v", storeToken)
					return ResponseJson(c, http.StatusInternalServerError, nil, "Failed to store token in Redis")
				}

				logrus.Infof("Token stored in Redis: %s", token)

				jsonResponseMap["token"] = token
				jsonResponse = jsonResponseMap
			}

			return ResponseJson(c, statusCode, jsonResponse, message)
		}
	}

	logrus.Errorf("Unexpected event received: %s", responseEvent.EventType)
	return ResponseJson(c, http.StatusInternalServerError, nil, "Unexpected event received")
}
