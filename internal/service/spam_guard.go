package service

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

// optionalText treats an absent optional field as empty text, which never
// trips any anti-spam rule.
func optionalText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
