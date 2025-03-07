package user

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// HandleLogoutEvent Run Consumer for UserLogoutEvent
func HandleLogoutEvent(ctx context.Context, payload []byte, correlationID string, userService service.UserService) {
	var req models.UserLogoutEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[user-service] Processing UserLogout | userID: %s", req.UserID)
	userService.HandleUserLogout(ctx, payload, correlationID)
}
