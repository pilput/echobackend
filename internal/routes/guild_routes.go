package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupGuildRoutes(api *echo.Group) {
	guilds := api.Group("/guilds")
	{
		// Public directory.
		guilds.GET("", r.guildHandler.ListGuilds)

		// Static segments are registered before "/:slug" so a guild can never
		// shadow them (same pattern as the post routes).
		guilds.POST("", r.guildHandler.CreateGuild, r.authMiddleware.Auth())
		guilds.GET("/me", r.guildHandler.GetMyGuilds, r.authMiddleware.Auth())

		// Public-but-personalized: membership fields appear only when signed in.
		guilds.GET("/:slug", r.guildHandler.GetGuild, r.authMiddleware.OptionalAuth())
		guilds.GET("/:slug/members", r.guildHandler.GetMembers, r.authMiddleware.OptionalAuth())

		guilds.PATCH("/:slug", r.guildHandler.UpdateGuild, r.authMiddleware.Auth())
		guilds.DELETE("/:slug", r.guildHandler.DeleteGuild, r.authMiddleware.Auth())
		guilds.POST("/:slug/join", r.guildHandler.JoinGuild, r.authMiddleware.Auth())
		guilds.DELETE("/:slug/leave", r.guildHandler.LeaveGuild, r.authMiddleware.Auth())

		// Channels: visible to whoever can see the guild; managed by owner/admin.
		guilds.GET("/:slug/channels", r.guildChannelHandler.ListChannels, r.authMiddleware.OptionalAuth())
		guilds.POST("/:slug/channels", r.guildChannelHandler.CreateChannel, r.authMiddleware.Auth())
		guilds.PATCH("/:slug/channels/:channelId", r.guildChannelHandler.UpdateChannel, r.authMiddleware.Auth())
		guilds.DELETE("/:slug/channels/:channelId", r.guildChannelHandler.DeleteChannel, r.authMiddleware.Auth())

		// Channel messages: members only.
		guilds.GET("/:slug/channels/:channelId/messages", r.guildChannelHandler.ListMessages, r.authMiddleware.Auth())
		guilds.POST("/:slug/channels/:channelId/messages", r.guildChannelHandler.SendMessage, r.authMiddleware.Auth())
		guilds.PATCH("/:slug/channels/:channelId/messages/:messageId", r.guildChannelHandler.EditMessage, r.authMiddleware.Auth())
		guilds.DELETE("/:slug/channels/:channelId/messages/:messageId", r.guildChannelHandler.DeleteMessage, r.authMiddleware.Auth())

		// Realtime events for every channel of the guild (SSE), members only.
		guilds.GET("/:slug/events", r.guildChannelHandler.StreamEvents, r.authMiddleware.Auth())
	}
}
