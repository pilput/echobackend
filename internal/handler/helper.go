package handler

import (
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func GetUserIDFromClaims(c *echo.Context) (string, bool) {
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

// IsSuperAdminFromClaims reports whether the JWT claims attached to the
// request assert super-admin status. This is a fast-path check only: it
// mirrors the claim shortcut in AuthMiddleware.AuthAdmin, so a false/absent
// result does not mean the user isn't an admin — JWTs can be long-lived and
// a user's admin status can change after issuance. Callers that need an
// authoritative answer should fall back to a DB lookup when this returns
// false.
func IsSuperAdminFromClaims(c *echo.Context) bool {
	userClaims := c.Get("user")
	if userClaims == nil {
		return false
	}

	switch v := userClaims.(type) {
	case jwt.MapClaims:
		return superAdminClaimTrue(v)
	case *jwt.Token:
		claims, ok := v.Claims.(jwt.MapClaims)
		if !ok {
			return false
		}
		return superAdminClaimTrue(claims)
	case map[string]any:
		return superAdminClaimTrue(v)
	}
	return false
}

func superAdminClaimTrue(claims map[string]any) bool {
	isSuperAdminClaim, exists := claims["is_super_admin"]
	if !exists || isSuperAdminClaim == nil {
		return false
	}
	isSuperAdmin, ok := isSuperAdminClaim.(bool)
	return ok && isSuperAdmin
}

func ParsePaginationParams(c *echo.Context, defaultLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
