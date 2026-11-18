package blog

import (
	"echobackend/internal/middleware"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

type PostViewHandler struct {
	postViewService PostViewService
	admins          AdminChecker
}

func NewPostViewHandler(postViewService PostViewService, admins AdminChecker) *PostViewHandler {
	return &PostViewHandler{postViewService: postViewService, admins: admins}
}

// RecordView records a view for logged-in users and anonymous visitors alike
// (the route uses OptionalAuth, so a missing or invalid token is anonymous).
func (h *PostViewHandler) RecordView(c *echo.Context) error {
	postID := c.Param("id")
	if !validator.IsValidUUID(postID) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	var userID string
	if uid, ok := middleware.UserIDFromContext(c); ok {
		userID = uid
	}

	// RealIP honours HTTP_TRUST_PROXY; never read X-Forwarded-For directly.
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
	if !validator.IsValidUUID(postID) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	limit, offset := response.ParsePaginationParams(c, 10)
	ctx := c.Request().Context()

	views, total, err := h.postViewService.GetViewsByPostID(ctx, postID, userID, isSuperAdmin(ctx, h.admins, userID), limit, offset)
	if err != nil {
		return respondError(c, "Failed to get post views", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)

	return response.SuccessWithMeta(c, "Successfully retrieved post views", views, meta)
}

func (h *PostViewHandler) GetPostViewStats(c *echo.Context) error {
	postID := c.Param("id")
	if !validator.IsValidUUID(postID) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	stats, err := h.postViewService.GetViewStats(c.Request().Context(), postID)
	if err != nil {
		return respondError(c, "Failed to get view statistics", err)
	}

	return response.Success(c, "Successfully retrieved view statistics", stats)
}

func (h *PostViewHandler) CheckUserViewed(c *echo.Context) error {
	postID := c.Param("id")
	if !validator.IsValidUUID(postID) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	userID, ok := middleware.UserIDFromContext(c)
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
