package middleware

import (
	"api-gateway/config"
	"api-gateway/utils"
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

var Ctx = context.Background()

// JWTMiddleware function to check JWT token
func JWTMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				logrus.Warn("[JWT Middleware] Missing Authorization header")
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Missing token"})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				logrus.Warn("[JWT Middleware] Invalid token format")
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Invalid token format"})
			}

			if config.IsBlacklistedToken(tokenString) {
				logrus.Warn("[JWT Middleware] Token has been blacklisted")
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Token has been blacklisted"})
			}

			claims := &utils.JWTCustomClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			})

			if err != nil {
				logrus.Errorf("[JWT Middleware] Error parsing token: %v", err)
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
			}

			if !token.Valid {
				logrus.Warn("[JWT Middleware] Token is not valid")
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
			}

			logrus.Infof("[JWT Middleware] Token Parsed: UserID=%s, Role=%s, Email=%s, Endpoint=%s",
				claims.UserID, claims.Role, claims.Email, c.Path())

			if claims.UserID == "" {
				logrus.Errorf("[JWT Middleware] Token does not contain UserID!")
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Token does not contain UserID"})
			}

			c.Set("user", claims)

			return next(c)
		}
	}
}
