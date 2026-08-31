package routes

import (
	appmiddleware "echobackend/internal/middleware"
	"time"

	"github.com/labstack/echo/v5"
)

func (r *Routes) setupAuthRoutes(api *echo.Group) {
	auth := api.Group("/auth")
	loginRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:login", 5, 5*time.Minute)
	registerRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:register", 5, 5*time.Minute)
	forgotPasswordRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:forgot-password", 3, 5*time.Minute)
	resetPasswordRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:reset-password", 5, 5*time.Minute)
	refreshRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:refresh", 30, time.Minute)
	oauthExchangeRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:oauth-exchange", 10, time.Minute)
	checkUsernameRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "auth:check-username", 20, 5*time.Minute)
	{
		auth.POST("/register", r.authHandler.Register, registerRateLimit)
		auth.POST("/login", r.authHandler.Login, loginRateLimit)
		auth.POST("/forgot-password", r.authHandler.ForgotPassword, forgotPasswordRateLimit)
		auth.POST("/reset-password", r.authHandler.ResetPassword, resetPasswordRateLimit)
		auth.POST("/refresh", r.authHandler.RefreshToken, refreshRateLimit)
		auth.POST("/check-username", r.authHandler.CheckUsername, checkUsernameRateLimit)
		auth.POST("/logout", r.authHandler.Logout, r.authMiddleware.Auth())
		auth.GET("/profile", r.authHandler.GetProfile, r.authMiddleware.Auth())
		auth.PUT("/profile", r.authHandler.UpdateProfile, r.authMiddleware.Auth())
		auth.DELETE("/account", r.authHandler.DeleteAccount, r.authMiddleware.Auth())
		auth.PATCH("/password", r.authHandler.ChangePassword, r.authMiddleware.Auth())
		auth.GET("/activity-logs", r.authHandler.GetActivityLogs, r.authMiddleware.Auth())
		auth.GET("/activity-logs/recent", r.authHandler.GetRecentActivity, r.authMiddleware.Auth())
		auth.GET("/activity-logs/failed-logins", r.authHandler.GetFailedLogins, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())
		auth.GET("/oauth/github", r.authHandler.GithubOAuthRedirect)
		auth.GET("/oauth/github/callback", r.authHandler.GithubOAuthCallback)
		auth.POST("/oauth/exchange", r.authHandler.ExchangeOAuthCode, oauthExchangeRateLimit)
	}
}
