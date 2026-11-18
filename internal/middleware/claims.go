package middleware

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// UserIDFromContext returns the user ID from the claims Auth/OptionalAuth
// stored under "user"; false when the request is unauthenticated.
func UserIDFromContext(c *echo.Context) (string, bool) {
	userClaims := c.Get("user")
	if userClaims == nil {
		return "", false
	}

	switch v := userClaims.(type) {
	case jwt.MapClaims:
		userID, exists := v["user_id"]
		if !exists {
			return "", false
		}
		userIDStr, ok := userID.(string)
		if !ok {
			return "", false
		}
		return userIDStr, true
	case *jwt.Token:
		claims, ok := v.Claims.(jwt.MapClaims)
		if !ok {
			return "", false
		}
		userID, exists := claims["user_id"]
		if !exists {
			return "", false
		}
		userIDStr, ok := userID.(string)
		if !ok {
			return "", false
		}
		return userIDStr, true
	case map[string]any:
		userID, exists := v["user_id"]
		if !exists {
			return "", false
		}
		userIDStr, ok := userID.(string)
		if !ok {
			return "", false
		}
		return userIDStr, true
	}
	return "", false
}
