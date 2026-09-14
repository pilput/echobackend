package handler

import (
	"context"
	"errors"
	"strconv"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/service"
	"echobackend/pkg/response"

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

// isSuperAdmin reports whether userID is currently a super admin. It always
// reads the database: the is_super_admin JWT claim stays in the token until it
// expires, so trusting it would keep a demoted admin privileged.
func isSuperAdmin(ctx context.Context, userService service.UserService, userID string) bool {
	if userService == nil || userID == "" {
		return false
	}
	user, err := userService.GetAdminByID(ctx, userID, false)
	if err != nil || user.IsSuperAdmin == nil {
		return false
	}
	return *user.IsSuperAdmin
}

// respondError maps well-known domain errors to their HTTP status and falls
// back to 500 for anything else.
func respondError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound),
		errors.Is(err, apperrors.ErrPostNotFound),
		errors.Is(err, apperrors.ErrCommentNotFound),
		errors.Is(err, apperrors.ErrTagNotFound),
		errors.Is(err, apperrors.ErrNotificationNotFound):
		return response.NotFound(c, message, err)
	case errors.Is(err, apperrors.ErrNotAuthor),
		errors.Is(err, apperrors.ErrCommentNotOwned),
		errors.Is(err, apperrors.ErrPostNotOwned):
		return response.Forbidden(c, message)
	case errors.Is(err, apperrors.ErrUserExists),
		errors.Is(err, apperrors.ErrAlreadyFollowing):
		return response.Conflict(c, message, err.Error())
	case errors.Is(err, apperrors.ErrCannotFollowSelf),
		errors.Is(err, apperrors.ErrNotFollowing),
		errors.Is(err, apperrors.ErrEmptyPostID),
		errors.Is(err, apperrors.ErrDateRangeTooLarge):
		return response.BadRequest(c, message, err)
	default:
		return response.InternalServerError(c, message, err)
	}
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
