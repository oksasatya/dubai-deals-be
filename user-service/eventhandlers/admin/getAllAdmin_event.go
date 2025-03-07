package admin

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"user-service/core/models"
	"user-service/core/service"
)

// GetAllAdminEvent for Run Consumer for GetAllAdminEvent
func GetAllAdminEvent(ctx context.Context, payload []byte, correlationID string, adminService service.AdminService) {
	logrus.Infof("[admin-service] Received payload: %s", string(payload))

	if adminService == nil {
		logrus.Fatal("[admin-service] AdminService is nil inside GetAllAdminEvent!")
	}

	var req models.GetAllAdminsEvent
	if err := json.Unmarshal(payload, &req); err != nil {
		logrus.Errorf("[admin-service] Invalid JSON payload: %v", err)
		return
	}

	logrus.Infof("[admin-service] Parsed payload: %+v", req)

	// Kirim payload yang sudah valid ke service
	adminService.HandleGetAllAdmins(ctx, payload, correlationID)
}
