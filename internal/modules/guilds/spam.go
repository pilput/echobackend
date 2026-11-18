package guilds

import (
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/antispam"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

// rejectSpamText applies the shared anti-spam text rules to guild names and
// descriptions before any database access.
func rejectSpamText(texts ...string) error {
	if reason := antispam.TextRejectReason(texts...); reason != "" {
		return apperrors.NewSpamRejection(apperrors.ErrContentSpamRejected, reason)
	}
	return nil
}

// optionalText treats an absent optional field as empty text, which never
// trips any anti-spam rule.
func optionalText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// respondSpamRejected answers an anti-spam rejection with 422 and its reason.
func respondSpamRejected(c *echo.Context, message, field string, err error) error {
	reason := err.Error()
	if rejection, ok := errors.AsType[*apperrors.SpamRejection](err); ok {
		reason = rejection.Reason
	}
	return response.SpamRejected(c, message, field, reason)
}
