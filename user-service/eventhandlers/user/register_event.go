package user

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// HandleRegisterEvent Run Consumer for UserRegisteredEvent
func HandleRegisterEvent(ctx context.Context, payload []byte, correlationID string, userService service.UserService) {
	var req models.UserRegisteredEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[user-service] Processing UserRegistered | Email: %s", req.Email)
	userService.HandleUserRegistered(ctx, payload, correlationID)
}
