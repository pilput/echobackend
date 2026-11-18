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

	ErrPostNotFound     = errors.New("post not found")
	ErrPostSpamRejected = errors.New("post rejected by anti-spam policy")
	// ErrContentSpamRejected covers comments and guilds; posts keep their own
	// error so existing clients matching on it are unaffected.
	ErrContentSpamRejected = errors.New("content rejected by anti-spam policy")
	ErrNotAuthor           = errors.New("not author")
	ErrAlreadyLiked        = errors.New("user has already liked this post")
	ErrNotLiked            = errors.New("user has not liked this post")
	ErrCommentNotFound     = errors.New("comment not found")
	ErrCommentNotOwned     = errors.New("not authorized to modify this comment")
	ErrPostNotOwned        = errors.New("not authorized to modify this post")
	ErrInvalidPostID       = errors.New("invalid post ID format")
	ErrEmptyPostID         = errors.New("post ID cannot be empty")

	ErrCannotReportOwnPost = errors.New("cannot report your own post")
	ErrNoPendingReports    = errors.New("post has no pending reports")
	ErrInvalidReportStatus = errors.New("invalid report status")
	ErrPostAlreadyHidden   = errors.New("post is already hidden")
	ErrPostNotHidden       = errors.New("post is not hidden")

	ErrDateRangeTooLarge = errors.New("date range must not exceed 366 days")

	ErrTagNotFound     = errors.New("tag not found")
	ErrTagNameRequired = errors.New("tag name is required")
	ErrTagNameEmpty    = errors.New("tag name cannot be empty")
	ErrInvalidTagID    = errors.New("invalid tag ID")

	ErrInvalidPaginationLimit   = errors.New("limit must be greater than 0")
	ErrPaginationLimitExceeded  = errors.New("limit must not exceed 100")
	ErrPaginationOffsetNegative = errors.New("offset must be non-negative")

	ErrFileNil            = errors.New("file cannot be nil")
	ErrFileTooLarge       = errors.New("file size must not exceed 1 MB")
	ErrAvatarFileTooLarge = errors.New("file size must not exceed 5 MB")
	ErrInvalidFileType    = errors.New("file must be a JPEG, PNG, GIF, or WebP image")
	ErrStorageUnavailable = errors.New("storage is unavailable")

	ErrBookmarkNotFound       = errors.New("bookmark not found")
	ErrBookmarkFolderNotFound = errors.New("bookmark folder not found")
	ErrNotificationNotFound   = errors.New("notification not found")

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
