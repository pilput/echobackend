package apperror

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenExpired       = errors.New("token has expired")

	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidUserID    = errors.New("invalid user ID format")
	ErrAlreadyFollowing = errors.New("already following this user")
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	ErrNotFollowing     = errors.New("not following this user")

	ErrPostNotFound    = errors.New("post not found")
	ErrNotAuthor       = errors.New("not author")
	ErrAlreadyLiked    = errors.New("user has already liked this post")
	ErrNotLiked        = errors.New("user has not liked this post")
	ErrCommentNotOwned = errors.New("not authorized to modify this comment")
	ErrPostNotOwned    = errors.New("not authorized to modify this post")
	ErrInvalidPostID   = errors.New("invalid post ID format")
	ErrEmptyPostID     = errors.New("post ID cannot be empty")

	ErrTagNotFound     = errors.New("tag not found")
	ErrTagNameRequired = errors.New("tag name is required")
	ErrTagNameEmpty    = errors.New("tag name cannot be empty")
	ErrInvalidTagID    = errors.New("invalid tag ID")

	ErrChatConversationNotFound = errors.New("chat conversation not found")
	ErrChatMessageNotFound      = errors.New("chat message not found")
	ErrConversationNotOwned     = errors.New("access denied: conversation does not belong to user")

	ErrInvalidPaginationLimit   = errors.New("limit must be greater than 0")
	ErrPaginationLimitExceeded  = errors.New("limit must not exceed 100")
	ErrPaginationOffsetNegative = errors.New("offset must be non-negative")

	ErrFileNil            = errors.New("file cannot be nil")
	ErrFileTooLarge       = errors.New("file size must not exceed 1 MB")
	ErrInvalidFileType    = errors.New("file must be a JPEG, PNG, or WebP image")
	ErrStorageUnavailable = errors.New("storage is unavailable")

	ErrHoldingNotFound        = errors.New("holding not found")
	ErrHoldingTypeNotFound    = errors.New("holding type not found")
	ErrHoldingNotOwned        = errors.New("not authorized to modify this holding")
	ErrHoldingDuplicateSame   = errors.New("source and target month/year are the same")
	ErrHoldingInvalidRange    = errors.New("end month/year must be on or before start month/year")
	ErrHoldingRangeTooLarge   = errors.New("requested month/year range is too large")
	ErrBookmarkNotFound       = errors.New("bookmark not found")
	ErrBookmarkFolderNotFound = errors.New("bookmark folder not found")
	ErrNotificationNotFound   = errors.New("notification not found")

	ErrPasswordTooShort          = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong           = errors.New("password must be at most 128 characters")
	ErrPasswordNoUpper           = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower           = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit           = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecial         = errors.New("password must contain at least one special character")
	ErrPasswordResetTokenUsed    = errors.New("password reset token has already been used")
	ErrPasswordResetTokenExpired = errors.New("password reset token has expired")
	ErrOAuthNotConfigured        = errors.New("GitHub OAuth is not configured")
)
