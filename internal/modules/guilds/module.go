// Package guilds is the Discord-style community module: guilds, membership,
// channels and channel messages, with realtime events over one SSE stream per
// guild.
//
// It is self-contained: models, DTOs, errors, repositories, services,
// handlers and routes all live here. It may depend on core packages (config,
// core, middleware, platform, pkg, apperror) but never on another module;
// only di and routes import it.
package guilds

import (
	"echobackend/internal/platform/realtime"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// Deps are the shared resources the module needs from the container.
type Deps struct {
	DB *gorm.DB
	// Hub fans events out to SSE streams. The container owns it (it is
	// closed on shutdown so open streams end), the module only uses it.
	Hub *realtime.Hub
}

// Guards are the auth middlewares routes are mounted behind.
type Guards struct {
	// Auth requires a logged-in user.
	Auth echo.MiddlewareFunc
	// OptionalAuth reads the user when a valid token is present.
	OptionalAuth echo.MiddlewareFunc
}

// Module holds the wired guild handlers.
type Module struct {
	guild   *GuildHandler
	channel *GuildChannelHandler
}

// New wires the module's repositories, services and handlers.
func New(deps Deps) *Module {
	return &Module{
		guild:   NewGuildHandler(NewGuildService(NewGuildRepository(deps.DB))),
		channel: NewGuildChannelHandler(NewGuildChannelService(NewGuildChannelRepository(deps.DB), deps.Hub)),
	}
}

// RegisterRoutes mounts the module under api (the /api group).
func (m *Module) RegisterRoutes(api *echo.Group, guards Guards) {
	guilds := api.Group("/guilds")
	{
		// Public directory.
		guilds.GET("", m.guild.ListGuilds)

		// Static segments are registered before "/:slug" so a guild can never
		// shadow them (same pattern as the post routes).
		guilds.POST("", m.guild.CreateGuild, guards.Auth)
		guilds.GET("/me", m.guild.GetMyGuilds, guards.Auth)

		// Public-but-personalized: membership fields appear only when signed in.
		guilds.GET("/:slug", m.guild.GetGuild, guards.OptionalAuth)
		guilds.GET("/:slug/members", m.guild.GetMembers, guards.OptionalAuth)

		guilds.PATCH("/:slug", m.guild.UpdateGuild, guards.Auth)
		guilds.DELETE("/:slug", m.guild.DeleteGuild, guards.Auth)
		guilds.POST("/:slug/join", m.guild.JoinGuild, guards.Auth)
		guilds.DELETE("/:slug/leave", m.guild.LeaveGuild, guards.Auth)

		// Channels: visible to whoever can see the guild; managed by owner/admin.
		guilds.GET("/:slug/channels", m.channel.ListChannels, guards.OptionalAuth)
		guilds.POST("/:slug/channels", m.channel.CreateChannel, guards.Auth)
		guilds.PATCH("/:slug/channels/:channelId", m.channel.UpdateChannel, guards.Auth)
		guilds.DELETE("/:slug/channels/:channelId", m.channel.DeleteChannel, guards.Auth)

		// Channel messages: members only.
		guilds.GET("/:slug/channels/:channelId/messages", m.channel.ListMessages, guards.Auth)
		guilds.POST("/:slug/channels/:channelId/messages", m.channel.SendMessage, guards.Auth)
		guilds.PATCH("/:slug/channels/:channelId/messages/:messageId", m.channel.EditMessage, guards.Auth)
		guilds.DELETE("/:slug/channels/:channelId/messages/:messageId", m.channel.DeleteMessage, guards.Auth)

		// Realtime events for every channel of the guild (SSE), members only.
		guilds.GET("/:slug/events", m.channel.StreamEvents, guards.Auth)
	}
}
