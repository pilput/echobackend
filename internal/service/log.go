package service

import "echobackend/pkg/applog"

var (
	authLog      = applog.Component("auth")
	guildChatLog = applog.Component("guild_chat")
)
