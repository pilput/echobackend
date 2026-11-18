package guilds

import "errors"

// Guild domain errors.
var (
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
)
