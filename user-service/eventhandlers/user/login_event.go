package user

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// HandleLoginEvent Run Consumer for UserLoginEvent
func HandleLoginEvent(ctx context.Context, payload []byte, correlationID string, userService service.UserService) {
	var req models.UserLoginEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[user-service] Processing UserLogin | Email: %s", req.Email)
	userService.HandleUserLogin(ctx, payload, correlationID)
}
