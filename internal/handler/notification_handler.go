package handler

import (
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type NotificationHandler struct {
	notificationService service.NotificationService
}

func NewNotificationHandler(notificationService service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) GetNotifications(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	limit, offset := ParsePaginationParams(c, 20)
	filter := &dto.NotificationListFilter{
		Unread: c.QueryParam("unread") == "true",
		Limit:  limit,
		Offset: offset,
	}

	notifications, total, err := h.notificationService.GetNotifications(c.Request().Context(), userID, filter)
	if err != nil {
		return respondError(c, "Failed to get notifications", err)
	}
	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Notifications fetched successfully", notifications, meta)
}

func (h *NotificationHandler) GetUnreadCount(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	count, err := h.notificationService.GetUnreadCount(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, "Failed to get unread notification count", err)
	}
	return response.Success(c, "Unread notification count fetched successfully", count)
}

func (h *NotificationHandler) MarkAsRead(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	id := c.Param("id")
	if id == "" {
		return response.BadRequest(c, "Notification ID is required", nil)
	}
	notification, err := h.notificationService.MarkAsRead(c.Request().Context(), id, userID)
	if err != nil {
		return respondError(c, "Failed to mark notification as read", err)
	}
	return response.Success(c, "Notification marked as read successfully", notification)
}

func (h *NotificationHandler) MarkAllAsRead(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	result, err := h.notificationService.MarkAllAsRead(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, "Failed to mark all notifications as read", err)
	}
	return response.Success(c, "All notifications marked as read successfully", result)
}
