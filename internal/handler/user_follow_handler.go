package handler

import (
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type UserFollowHandler struct {
	userFollowService service.UserFollowService
}

func NewUserFollowHandler(userFollowService service.UserFollowService) *UserFollowHandler {
	return &UserFollowHandler{userFollowService: userFollowService}
}

func (h *UserFollowHandler) FollowUser(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	var followReq dto.FollowRequest
	if err := c.Bind(&followReq); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}

	if err := c.Validate(followReq); err != nil {
		return response.FromValidateError(c, err)
	}

	followResponse, err := h.userFollowService.FollowUser(c.Request().Context(), userID, followReq.UserID)
	if err != nil {
		return respondError(c, "Failed to follow user", err)
	}

	return response.Success(c, followResponse.Message, followResponse)
}

func (h *UserFollowHandler) UnfollowUser(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	userIDToUnfollow := c.Param("id")
	if userIDToUnfollow == "" {
		return response.BadRequest(c, "User ID is required", nil)
	}

	followResponse, err := h.userFollowService.UnfollowUser(c.Request().Context(), userID, userIDToUnfollow)
	if err != nil {
		return respondError(c, "Failed to unfollow user", err)
	}

	return response.Success(c, followResponse.Message, followResponse)
}

func (h *UserFollowHandler) GetFollowers(c *echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return response.BadRequest(c, "User ID is required", nil)
	}

	limit, offset := ParsePaginationParams(c, 10)

	followers, total, err := h.userFollowService.GetFollowers(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return respondError(c, "Failed to get followers", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)

	return response.SuccessWithMeta(c, "Successfully retrieved followers", followers, meta)
}

func (h *UserFollowHandler) GetFollowing(c *echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return response.BadRequest(c, "User ID is required", nil)
	}

	limit, offset := ParsePaginationParams(c, 10)

	following, total, err := h.userFollowService.GetFollowing(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return respondError(c, "Failed to get following", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)

	return response.SuccessWithMeta(c, "Successfully retrieved following", following, meta)
}

func (h *UserFollowHandler) GetFollowStats(c *echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return response.BadRequest(c, "User ID is required", nil)
	}

	stats, err := h.userFollowService.GetFollowStats(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, "Failed to get follow statistics", err)
	}

	return response.Success(c, "Successfully retrieved follow statistics", stats)
}

func (h *UserFollowHandler) CheckFollowStatus(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		return response.BadRequest(c, "User ID is required", nil)
	}

	isFollowing, err := h.userFollowService.IsFollowing(c.Request().Context(), userID, targetUserID)
	if err != nil {
		return respondError(c, "Failed to check follow status", err)
	}

	return response.Success(c, "Successfully checked follow status", map[string]bool{
		"is_following": isFollowing,
	})
}

func (h *UserFollowHandler) GetMutualFollows(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "Authentication required")
	}

	otherUserID := c.Param("id")
	if otherUserID == "" {
		return response.BadRequest(c, "User ID is required", nil)
	}

	mutualFollows, err := h.userFollowService.GetMutualFollows(c.Request().Context(), userID, otherUserID)
	if err != nil {
		return respondError(c, "Failed to get mutual follows", err)
	}

	return response.Success(c, "Successfully retrieved mutual follows", mutualFollows)
}
