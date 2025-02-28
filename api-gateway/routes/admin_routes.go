package routes

import (
	"api-gateway/config"
	"api-gateway/handler"
	"api-gateway/middleware"
	"api-gateway/webResponse"
	"github.com/labstack/echo/v4"
	"messaging"
	modelsUser "user-service/core/models"
)

// AdminRoutes register admin routes
func AdminRoutes(e *echo.Echo, cfg *config.RateLimitConfig, rmq *messaging.RabbitMQConnection, res *webResponse.ResponseHandler) {
	adminHandler := handler.NewAdminHandler(cfg, rmq, res)
	e.Static("/uploads", "uploads")
	r := e.Group("/api/admins")
	//r.POST("/login", adminHandler.Login)
	// protected routes
	r.Use(middleware.JWTMiddleware())
	r.POST("/create-admin", adminHandler.CreateAdmin, middleware.RoleMiddleware(modelsUser.RoleSuperAdmin))
	r.PUT("/update-admin/:id", adminHandler.UpdateAdmin, middleware.RoleMiddleware(modelsUser.RoleSuperAdmin, modelsUser.RoleAdmin))
}
