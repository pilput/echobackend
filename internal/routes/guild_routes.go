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
	}
}
