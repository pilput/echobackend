package blog

import (
	"echobackend/internal/middleware"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

type CommentHandler struct {
	commentService CommentService
	admins         AdminChecker
}

func NewCommentHandler(commentService CommentService, admins AdminChecker) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		admins:         admins,
	}
}

func (h *CommentHandler) CreateComment(c *echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	var commentDTO CreateCommentRequest
	if err := c.Bind(&commentDTO); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}

	if err := c.Validate(commentDTO); err != nil {
		return response.FromValidateError(c, err)
	}

	userID, ok := middleware.UserIDFromContext(c)
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
	postID := c.Param("id")
	if !validator.IsValidUUID(postID) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}
	commentID := c.Param("comment_id")
	if !validator.IsValidUUID(commentID) {
		return response.BadRequest(c, "Invalid comment ID", nil)
	}

	var commentDTO CreateCommentRequest
	if err := c.Bind(&commentDTO); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}

	if err := c.Validate(commentDTO); err != nil {
		return response.FromValidateError(c, err)
	}

	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	comment, err := h.commentService.UpdateComment(c.Request().Context(), postID, commentID, commentDTO.Text, userID)
	if err != nil {
		return respondError(c, "Failed to update comment", err)
	}

	return response.Success(c, "Comment updated successfully", comment)
}

func (h *CommentHandler) DeleteComment(c *echo.Context) error {
	postID := c.Param("id")
	if !validator.IsValidUUID(postID) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}
	commentID := c.Param("comment_id")
	if !validator.IsValidUUID(commentID) {
		return response.BadRequest(c, "Invalid comment ID", nil)
	}

	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	// Super admins may delete any comment for moderation purposes.
	isAdmin := isSuperAdmin(c.Request().Context(), h.admins, userID)

	if err := h.commentService.DeleteComment(c.Request().Context(), postID, commentID, userID, isAdmin); err != nil {
		return respondError(c, "Failed to delete comment", err)
	}

	return response.Success(c, "Comment deleted successfully", nil)
}
