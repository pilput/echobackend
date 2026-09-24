package service

import "echobackend/pkg/applog"

var (
	authLog       = applog.Component("auth")
	openRouterLog = applog.Component("openrouter")
	guildChatLog  = applog.Component("guild_chat")
)
