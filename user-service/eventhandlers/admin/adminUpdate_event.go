package admin

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// HandleAdminUpdateEvent Run Consumer for AdminUpdateEvent
func HandleAdminUpdateEvent(ctx context.Context, payload []byte, correlationID string, adminService service.AdminService) {
	var req models.AdminUpdateEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("Invalid event data: %v", err)
		return
	}

	logrus.Infof("[admin-service] Processing AdminUpdate | AdminID: %s", req.ID)
	adminService.HandleUpdateAdmin(ctx, payload, correlationID)
}
