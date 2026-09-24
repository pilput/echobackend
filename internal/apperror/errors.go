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

	ErrPostNotFound    = errors.New("post not found")
	ErrNotAuthor       = errors.New("not author")
	ErrAlreadyLiked    = errors.New("user has already liked this post")
	ErrNotLiked        = errors.New("user has not liked this post")
	ErrCommentNotFound = errors.New("comment not found")
	ErrCommentNotOwned = errors.New("not authorized to modify this comment")
	ErrPostNotOwned    = errors.New("not authorized to modify this post")
	ErrInvalidPostID   = errors.New("invalid post ID format")
	ErrEmptyPostID     = errors.New("post ID cannot be empty")

	ErrDateRangeTooLarge = errors.New("date range must not exceed 366 days")

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
	ErrAvatarFileTooLarge = errors.New("file size must not exceed 5 MB")
	ErrInvalidFileType    = errors.New("file must be a JPEG, PNG, GIF, or WebP image")
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

	ErrGuildNotFound         = errors.New("guild not found")
	ErrGuildSlugExists       = errors.New("guild slug already taken")
	ErrGuildSlugInvalid      = errors.New("guild name must contain at least one letter or digit")
	ErrGuildSlugReserved     = errors.New("guild slug is reserved")
	ErrGuildNotOwned         = errors.New("not authorized to modify this guild")
	ErrAlreadyGuildMember    = errors.New("already a member of this guild")
	ErrNotGuildMember        = errors.New("not a member of this guild")
	ErrGuildOwnerCannotLeave = errors.New("guild owner cannot leave; transfer ownership or delete the guild")

	ErrGuildChannelNotFound      = errors.New("guild channel not found")
	ErrGuildChannelNameExists    = errors.New("a channel with this name already exists in the guild")
	ErrGuildChannelNameInvalid   = errors.New("channel name must contain at least one letter or digit")
	ErrGuildMessageNotFound      = errors.New("message not found")
	ErrGuildMessageNotOwned      = errors.New("not authorized to modify this message")
	ErrGuildMessageEmpty         = errors.New("message content cannot be empty")
	ErrGuildMessageReplyNotFound = errors.New("replied-to message not found in this channel")
	ErrInvalidMessageCursor      = errors.New("invalid message cursor")

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
