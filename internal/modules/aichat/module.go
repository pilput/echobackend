// Package aichat is the AI module: per-user chat conversations with
// streamed OpenRouter replies, and admin-only article generation.
//
// It is self-contained: models, DTOs, errors, repository, services, handlers
// and routes all live here. It may depend on core packages (config,
// middleware, platform, pkg) but never on another module; only di and routes
// import it.
package aichat

import (
	"time"

	"echobackend/config"
	"echobackend/internal/middleware"
	"echobackend/internal/platform/cache"
	"echobackend/internal/platform/openrouter"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// Deps are the shared resources the module needs from the container.
type Deps struct {
	Config *config.Config
	DB     *gorm.DB
	// Cache may be nil (Redis disabled); the rate limiter then fails open.
	Cache *cache.RedisCache
}

// Guards are the auth middlewares routes are mounted behind.
type Guards struct {
	// Auth requires a logged-in user.
	Auth echo.MiddlewareFunc
	// Admin requires a super admin; it must run after Auth.
	Admin echo.MiddlewareFunc
}

// Module holds the wired AI handlers.
type Module struct {
	cache        *cache.RedisCache
	conversation *ChatConversationHandler
	ai           *AIHandler
}

// New wires the module's repository, services and handlers. With no
// OPENROUTER_API_KEY the services degrade (chat echoes, article generation
// reports unavailable) instead of failing.
func New(deps Deps) *Module {
	openRouterClient := openrouter.NewClient(deps.Config.OpenRouter)

	conversationService := NewChatConversationService(NewChatConversationRepository(deps.DB), openRouterClient, deps.Config)
	articleService := NewArticleGenerationService(openRouterClient, deps.Config)

	return &Module{
		cache:        deps.Cache,
		conversation: NewChatConversationHandler(conversationService),
		ai:           NewAIHandler(articleService),
	}
}

// RegisterRoutes mounts the module under api (the /api group).
// Auth is attached per route, not per group, to keep the original routing
// (a guarded group would answer unknown sub-paths with 401 instead of 404).
func (m *Module) RegisterRoutes(api *echo.Group, guards Guards) {
	conversations := api.Group("/chat/conversations")
	{
		conversations.POST("", m.conversation.CreateConversation, guards.Auth)
		conversations.POST("/stream", m.conversation.CreateConversationStream, guards.Auth)
		conversations.GET("", m.conversation.GetConversations, guards.Auth)
		conversations.GET("/:id", m.conversation.GetConversation, guards.Auth)
		conversations.PUT("/:id", m.conversation.UpdateConversation, guards.Auth)
		conversations.DELETE("/:id", m.conversation.DeleteConversation, guards.Auth)
		conversations.POST("/:conversationId/messages", m.conversation.CreateMessage, guards.Auth)
		conversations.POST("/:conversationId/messages/stream", m.conversation.CreateMessageStream, guards.Auth)
		conversations.GET("/:conversationId/messages", m.conversation.GetMessages, guards.Auth)
	}

	messages := api.Group("/chat/messages")
	{
		messages.GET("/:messageId", m.conversation.GetMessage, guards.Auth)
		messages.DELETE("/:messageId", m.conversation.DeleteMessage, guards.Auth)
	}

	generateArticleRateLimit := middleware.FixedWindowRateLimiterWithCache(m.cache, "ai:generate-article", 20, time.Hour)

	ai := api.Group("/ai")
	// Super admins only for now (rollout); privileges are read from the DB.
	ai.POST("/generate-article", m.ai.GenerateArticle, guards.Auth, guards.Admin, generateArticleRateLimit)
}
