package user

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// GetProfileUser Run Consumer for UserLoginEvent
func GetProfileUser(ctx context.Context, payload []byte, correlationID string, userService service.UserService) {
	var req models.GetUserProfileEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[user-service] Processing Get Profile | Email: %s", req.Email)
	userService.HandleGetProfile(ctx, payload, correlationID)
}
