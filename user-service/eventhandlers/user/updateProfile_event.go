package user

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// UpdateProfileUser Run Consumer for UserUpdateEvent
func UpdateProfileUser(ctx context.Context, payload []byte, correlationID string, userService service.UserService) {
	var req models.UpdateProfileEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[user-service] Processing Update Profile | Email: %s", req.Email)
	userService.HandleUpdateProfile(ctx, payload, correlationID)
}
