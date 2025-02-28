package middleware

import (
	"api-gateway/utils"
	"api-gateway/webResponse"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"net/http"
)

// RoleMiddleware function to check user role
func RoleMiddleware[T comparable](allowedRole ...T) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get("user").(*utils.JWTCustomClaims)
			if !ok || claims == nil {
				logrus.Warn("Invalid claims type or missing token claims")
				return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Unauthorized")
			}

			// Check if user role is allowed
			for _, role := range allowedRole {
				if role == role {
					return next(c)
				}
			}

			logrus.Warn("User role is not allowed")
			return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "You don't have permission to access this route")
		}
	}
}
