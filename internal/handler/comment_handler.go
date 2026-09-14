package handler

import (
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type CommentHandler struct {
	commentService service.CommentService
	userService    service.UserService
}

func NewCommentHandler(commentService service.CommentService, userService service.UserService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		userService:    userService,
	}
}

func (h *CommentHandler) CreateComment(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	var commentDTO dto.CreateCommentRequest
	if err := c.Bind(&commentDTO); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}

	if err := c.Validate(commentDTO); err != nil {
		return response.FromValidateError(c, err)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	comment, err := h.commentService.CreateComment(c.Request().Context(), postID, &commentDTO, userID)
	if err != nil {
		return respondError(c, "Failed to create comment", err)
	}

	return response.Created(c, "Comment created successfully", comment)
}

func (h *CommentHandler) GetCommentsByPostID(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	comments, err := h.commentService.GetCommentsByPostID(c.Request().Context(), postID)
	if err != nil {
		return respondError(c, "Comments fetched failed", err)
	}

	return response.Success(c, "Comments fetched successfully", comments)
}

func (h *CommentHandler) UpdateComment(c *echo.Context) error {
	commentID := c.Param("comment_id")
	if commentID == "" {
		return response.BadRequest(c, "Comment ID is required", nil)
	}

	var commentDTO dto.CreateCommentRequest
	if err := c.Bind(&commentDTO); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}

	if err := c.Validate(commentDTO); err != nil {
		return response.FromValidateError(c, err)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	comment, err := h.commentService.UpdateComment(c.Request().Context(), commentID, commentDTO.Text, userID)
	if err != nil {
		return respondError(c, "Failed to update comment", err)
	}

	return response.Success(c, "Comment updated successfully", comment)
}

func (h *CommentHandler) DeleteComment(c *echo.Context) error {
	commentID := c.Param("comment_id")
	if commentID == "" {
		return response.BadRequest(c, "Comment ID is required", nil)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	// Super admins may delete any comment for moderation purposes.
	isAdmin := isSuperAdmin(c.Request().Context(), h.userService, userID)

	if err := h.commentService.DeleteComment(c.Request().Context(), commentID, userID, isAdmin); err != nil {
		return respondError(c, "Failed to delete comment", err)
	}

	return response.Success(c, "Comment deleted successfully", nil)
}
