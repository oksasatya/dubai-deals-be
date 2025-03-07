package admin

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// HandleCreateAdmin Run Consumer for AdminCreateEvent
func HandleCreateAdmin(ctx context.Context, payload []byte, correlationID string, adminService service.AdminService) {
	var req models.AdminCreateEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[admin-service] Processing AdminCreate | Username: %s", req.Username)
	adminService.HandleCreateAdmin(ctx, payload, correlationID)
}
