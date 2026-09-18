package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupChatConversationRoutes(api *echo.Group) {
	conversations := api.Group("/chat/conversations")
	{
		conversations.POST("", r.chatConversationHandler.CreateConversation, r.authMiddleware.Auth())
		conversations.POST("/stream", r.chatConversationHandler.CreateConversationStream, r.authMiddleware.Auth())
		conversations.GET("", r.chatConversationHandler.GetConversations, r.authMiddleware.Auth())
		conversations.GET("/:id", r.chatConversationHandler.GetConversation, r.authMiddleware.Auth())
		conversations.PUT("/:id", r.chatConversationHandler.UpdateConversation, r.authMiddleware.Auth())
		conversations.DELETE("/:id", r.chatConversationHandler.DeleteConversation, r.authMiddleware.Auth())
		conversations.POST("/:conversationId/messages", r.chatConversationHandler.CreateMessage, r.authMiddleware.Auth())
		conversations.POST("/:conversationId/messages/stream", r.chatConversationHandler.CreateMessageStream, r.authMiddleware.Auth())
		conversations.GET("/:conversationId/messages", r.chatConversationHandler.GetMessages, r.authMiddleware.Auth())
	}

	messages := api.Group("/chat/messages")
	{
		messages.GET("/:messageId", r.chatConversationHandler.GetMessage, r.authMiddleware.Auth())
		messages.DELETE("/:messageId", r.chatConversationHandler.DeleteMessage, r.authMiddleware.Auth())
	}
}
