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
func (h *ResponseHandler) HandleEventResponse(
	c echo.Context,
	generateToken bool,
	statusCode int,
	timeout time.Duration,
	successMessage string,
	eventName ...string,
) error {
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
		logrus.Errorf("Event timeout while waiting for: %v", eventName)
		return ResponseJson(c, http.StatusGatewayTimeout, nil, "Request timed out waiting for response")
	}

	logrus.Infof("Received event: %s | CorrelationID: %s", responseEvent.EventType, responseEvent.CorrelationID)

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
			var parsedJson any
			if err := json.Unmarshal([]byte(payloadStr), &parsedJson); err != nil {
				logrus.Errorf("[ResponseHandler] Failed to parse event payload: %v", err)
				return ResponseJson(c, http.StatusInternalServerError, nil, "Failed to parse response")
			} else {
				logrus.Warnf("[ResponseHandler] Payload is not valid JSON, using as error message")
				jsonResponse = map[string]interface{}{"message": payloadStr}
			}

			jsonResponse = parsedJson
		} else {
			logrus.Warnf("[ResponseHandler] Payload is not valid JSON, returning as string")
			jsonResponse = []map[string]interface{}{}
		}

	} else if payloadMap, ok := responseEvent.Payload.(map[string]interface{}); ok {
		jsonResponse = payloadMap
	} else {
		logrus.Errorf("[ResponseHandler] Unexpected payload type: %T", responseEvent.Payload)
		return ResponseJson(c, http.StatusInternalServerError, nil, "Unexpected event payload format")
	}

	var failureEvents, successEvents []string
	for _, event := range eventName {
		if strings.Contains(strings.ToLower(event), "failed") {
			failureEvents = append(failureEvents, event)
		} else {
			successEvents = append(successEvents, event)
		}
	}

	for _, failureEvent := range failureEvents {
		if responseEvent.EventType == failureEvent {
			logrus.Warnf("🔥 Handling FAILED Event: %s | Payload: %+v", responseEvent.EventType, responseEvent.Payload)

			var jsonResponse map[string]interface{}

			// **Pastikan payload berupa map[string]interface{}**
			if payloadMap, ok := responseEvent.Payload.(map[string]interface{}); ok {
				jsonResponse = payloadMap
			} else if payloadStr, ok := responseEvent.Payload.(string); ok {
				jsonResponse = map[string]interface{}{"message": payloadStr}
			} else {
				jsonResponse = map[string]interface{}{"message": "Unknown error occurred"}
			}

			return ResponseJson(c, http.StatusUnauthorized, jsonResponse, "Error")
		}
	}

	for _, successEvent := range successEvents {
		if responseEvent.EventType == successEvent {
			logrus.Infof("Processing Success Event: %s", successEvent)

			if jsonResponseArray, ok := jsonResponse.([]interface{}); ok {
				return ResponseJson(c, statusCode, jsonResponseArray, successMessage)
			}

			if jsonResponseMap, ok := jsonResponse.(map[string]interface{}); ok {
				delete(jsonResponseMap, "password")
				jsonResponse = jsonResponseMap
			}

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

				storeToken := config.StoreTokenInRedis(c.Request().Context(), token, userID, userRole, 72)
				if storeToken != nil {
					logrus.Errorf("Failed to store token in Redis: %v", storeToken)
					return ResponseJson(c, http.StatusInternalServerError, nil, "Failed to store token in Redis")
				}

				logrus.Infof("Token stored in Redis: %s", token)

				jsonResponseMap["token"] = token
				jsonResponse = jsonResponseMap
			}

			return ResponseJson(c, statusCode, jsonResponse, successMessage)
		}
	}

	logrus.Errorf("Unexpected event received: %s", responseEvent.EventType)
	return ResponseJson(c, http.StatusInternalServerError, nil, "Unexpected event received")
}
