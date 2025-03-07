package middleware

import (
	"api-gateway/utils"
	"api-gateway/webResponse"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"net/http"
)

// RoleMiddleware function to check user role
func RoleMiddleware(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get("user").(*utils.JWTCustomClaims)
			if !ok || claims == nil {
				logrus.Warn("Invalid claims type or missing token claims")
				return webResponse.ResponseJson(c, http.StatusUnauthorized, nil, "Unauthorized")
			}

			logrus.Infof("User Role: %s, Allowed Roles: %v", claims.Role, allowedRoles)

			for _, role := range allowedRoles {
				if role == claims.Role {
					return next(c)
				}
			}

			logrus.Warn("User role is not allowed: " + claims.Role)
			return webResponse.ResponseJson(c, http.StatusForbidden, nil, "You don't have permission to access this route")
		}
	}
}
