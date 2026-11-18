package blog

import (
	"context"
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

// AdminChecker reports whether userID is currently a super admin. It must
// read the database: the is_super_admin JWT claim outlives a demotion.
// Implemented in core (internal/di) on top of the user service.
type AdminChecker interface {
	IsSuperAdmin(ctx context.Context, userID string) bool
}

// isSuperAdmin is nil-safe: no checker or no user means not an admin.
func isSuperAdmin(ctx context.Context, admins AdminChecker, userID string) bool {
	if admins == nil || userID == "" {
		return false
	}
	return admins.IsSuperAdmin(ctx, userID)
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

// respondError maps well-known blog errors to their HTTP status and falls
// back to 500 for anything else.
func respondError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound),
		errors.Is(err, ErrPostNotFound),
		errors.Is(err, ErrCommentNotFound),
		errors.Is(err, ErrTagNotFound):
		return response.NotFound(c, message, err)
	case errors.Is(err, apperrors.ErrNotAuthor),
		errors.Is(err, ErrCommentNotOwned),
		errors.Is(err, ErrPostNotOwned):
		return response.Forbidden(c, message)
	case errors.Is(err, apperrors.ErrContentSpamRejected):
		return respondSpamRejected(c, "Content rejected by anti-spam policy", "text", err)
	case errors.Is(err, ErrEmptyPostID),
		errors.Is(err, apperrors.ErrDateRangeTooLarge):
		return response.BadRequest(c, message, err)
	default:
		return response.InternalServerError(c, message, err)
	}
}
