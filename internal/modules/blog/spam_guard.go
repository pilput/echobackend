package blog

import (
	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/antispam"
)

// rejectSpamText applies the shared anti-spam text rules to user-written
// content other than posts (comments, guild names and descriptions). It runs
// before any database access so rejected content never costs a query.
func rejectSpamText(texts ...string) error {
	if reason := antispam.TextRejectReason(texts...); reason != "" {
		return apperrors.NewSpamRejection(apperrors.ErrContentSpamRejected, reason)
	}
	return nil
}
