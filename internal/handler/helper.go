package handler

import (
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/middleware"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

// GetUserIDFromClaims returns the user ID that Auth/OptionalAuth stored on the
// context. It delegates to middleware.UserIDFromContext.
func GetUserIDFromClaims(c *echo.Context) (string, bool) {
	return middleware.UserIDFromContext(c)
}

// respondSpamRejected answers an anti-spam rejection with the shared 422
// envelope, attaching the rule's reason to field so the UI can show it.
func respondSpamRejected(c *echo.Context, message, field string, err error) error {
	reason := err.Error()
	if rejection, ok := errors.AsType[*apperrors.SpamRejection](err); ok {
		reason = rejection.Reason
	}
	return response.SpamRejected(c, message, field, reason)
}

// MsgInvalidUserID is the 400 message for a :id path segment that is not a
// UUID; checked before any DB call so Postgres never sees a malformed uuid.
const MsgInvalidUserID = "Invalid user ID"

// MsgInvalidNotificationID is the 400 message for a notification :id path
// segment that is not a UUID.
const MsgInvalidNotificationID = "Invalid notification ID"

// respondInvalidUserID answers a malformed user :id path parameter with 400.
func respondInvalidUserID(c *echo.Context) error {
	return response.BadRequest(c, MsgInvalidUserID, apperrors.ErrInvalidUserID)
}

// respondError maps well-known domain errors to their HTTP status and falls
// back to 500 for anything else.
func respondError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound),
		errors.Is(err, apperrors.ErrNotificationNotFound):
		return response.NotFound(c, message, err)
	case errors.Is(err, apperrors.ErrNotAuthor):
		return response.Forbidden(c, message)
	case errors.Is(err, apperrors.ErrContentSpamRejected):
		return respondSpamRejected(c, "Content rejected by anti-spam policy", "text", err)
	case errors.Is(err, apperrors.ErrUserExists),
		errors.Is(err, apperrors.ErrAlreadyFollowing):
		return response.Conflict(c, message, err.Error())
	case errors.Is(err, apperrors.ErrCannotFollowSelf),
		errors.Is(err, apperrors.ErrNotFollowing),
		errors.Is(err, apperrors.ErrDateRangeTooLarge):
		return response.BadRequest(c, message, err)
	default:
		return response.InternalServerError(c, message, err)
	}
}

// ParsePaginationParams delegates to response.ParsePaginationParams.
func ParsePaginationParams(c *echo.Context, defaultLimit int) (limit, offset int) {
	return response.ParsePaginationParams(c, defaultLimit)
}
