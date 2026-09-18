package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupNotificationRoutes(api *echo.Group) {
	notifications := api.Group("/notifications", r.authMiddleware.Auth())
	{
		notifications.GET("", r.notificationHandler.GetNotifications)
		notifications.GET("/unread-count", r.notificationHandler.GetUnreadCount)
		notifications.PATCH("/:id/read", r.notificationHandler.MarkAsRead)
		notifications.PATCH("/read-all", r.notificationHandler.MarkAllAsRead)
	}
}
