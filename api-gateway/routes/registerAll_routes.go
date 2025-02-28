package routes

import (
	"api-gateway/config"
	"api-gateway/webResponse"
	"github.com/labstack/echo/v4"
	"messaging"
)

type RegisterRoutes struct {
	Echo *echo.Echo
	Cfg  *config.RateLimitConfig
	RMQ  *messaging.RabbitMQConnection
	Res  *webResponse.ResponseHandler
}

type NewRegisterRoutes interface {
	RegisterAllRoutes()
}

func (r *RegisterRoutes) RegisterAllRoutes() {
	UserRoutes(r.Echo, r.Cfg, r.RMQ, r.Res)
	AdminRoutes(r.Echo, r.Cfg, r.RMQ, r.Res)
}
