package handler

import (
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type PostViewHandler struct {
	postViewService service.PostViewService
	userService     service.UserService
}

func NewPostViewHandler(postViewService service.PostViewService, userService service.UserService) *PostViewHandler {
	return &PostViewHandler{postViewService: postViewService, userService: userService}
}

func (h *PostViewHandler) RecordView(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	var userID string
	if uid, ok := GetUserIDFromClaims(c); ok {
		userID = uid
	}

	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := h.postViewService.RecordView(c.Request().Context(), postID, userID, &ipAddress, &userAgent)
	if err != nil {
		return respondError(c, "Failed to record view", err)
	}

	return response.Success(c, "View recorded successfully", nil)
}

func (h *PostViewHandler) GetPostViews(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	limit, offset := ParsePaginationParams(c, 10)
	ctx := c.Request().Context()

	views, total, err := h.postViewService.GetViewsByPostID(ctx, postID, userID, isSuperAdmin(ctx, h.userService, userID), limit, offset)
	if err != nil {
		return respondError(c, "Failed to get post views", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)

	return response.SuccessWithMeta(c, "Successfully retrieved post views", views, meta)
}

func (h *PostViewHandler) GetPostViewStats(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	stats, err := h.postViewService.GetViewStats(c.Request().Context(), postID)
	if err != nil {
		return respondError(c, "Failed to get view statistics", err)
	}

	return response.Success(c, "Successfully retrieved view statistics", stats)
}

func (h *PostViewHandler) CheckUserViewed(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	hasViewed, err := h.postViewService.HasUserViewedPost(c.Request().Context(), postID, userID)
	if err != nil {
		return respondError(c, "Failed to check view status", err)
	}

	return response.Success(c, "Successfully checked view status", map[string]bool{
		"has_viewed": hasViewed,
	})
}
