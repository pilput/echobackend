package apperror

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenExpired       = errors.New("token has expired")
	// ErrSessionAlreadyRotated is returned when a concurrent refresh rotated the
	// session first. It is an internal signal, never surfaced to the client.
	ErrSessionAlreadyRotated = errors.New("session already rotated")

	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidUserID    = errors.New("invalid user ID format")
	ErrAlreadyFollowing = errors.New("already following this user")
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	ErrNotFollowing     = errors.New("not following this user")

	// ErrContentSpamRejected covers comments and guilds; posts keep their own
	// error so existing clients matching on it are unaffected.
	ErrContentSpamRejected = errors.New("content rejected by anti-spam policy")
	ErrNotAuthor           = errors.New("not author")

	ErrDateRangeTooLarge = errors.New("date range must not exceed 366 days")

	ErrInvalidPaginationLimit   = errors.New("limit must be greater than 0")
	ErrPaginationLimitExceeded  = errors.New("limit must not exceed 100")
	ErrPaginationOffsetNegative = errors.New("offset must be non-negative")

	ErrFileNil            = errors.New("file cannot be nil")
	ErrAvatarFileTooLarge = errors.New("file size must not exceed 5 MB")
	ErrInvalidFileType    = errors.New("file must be a JPEG, PNG, GIF, or WebP image")
	ErrStorageUnavailable = errors.New("storage is unavailable")

	ErrNotificationNotFound = errors.New("notification not found")

	ErrPasswordTooShort            = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong             = errors.New("password must be at most 128 characters")
	ErrPasswordNoUpper             = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower             = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit             = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecial           = errors.New("password must contain at least one special character")
	ErrPasswordResetTokenUsed      = errors.New("password reset token has already been used")
	ErrPasswordResetTokenExpired   = errors.New("password reset token has expired")
	ErrAccountDeletionTokenUsed    = errors.New("account deletion token has already been used")
	ErrAccountDeletionTokenExpired = errors.New("account deletion token has expired")
	ErrEmailNotConfigured          = errors.New("email delivery is not configured")
	ErrOAuthNotConfigured          = errors.New("GitHub OAuth is not configured")
)
