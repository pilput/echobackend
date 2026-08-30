package handler

import (
	"errors"
	"strconv"
	"strings"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

type PostHandler struct {
	postService     service.PostService
	postViewService service.PostViewService
}

// maxPostLimit caps the page size for post listing queries.
const maxPostLimit = 100

func (h *PostHandler) respondPostError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrPostNotFound):
		return response.NotFound(c, message, err)
	case errors.Is(err, apperrors.ErrNotAuthor), errors.Is(err, apperrors.ErrPostNotOwned):
		return response.Forbidden(c, message)
	case errors.Is(err, apperrors.ErrFileNil), errors.Is(err, apperrors.ErrFileTooLarge), errors.Is(err, apperrors.ErrInvalidFileType), errors.Is(err, apperrors.ErrStorageUnavailable):
		return response.BadRequest(c, message, err)
	default:
		return response.InternalServerError(c, message, err)
	}
}

func NewPostHandler(postService service.PostService, postViewService service.PostViewService) *PostHandler {
	return &PostHandler{
		postService:     postService,
		postViewService: postViewService,
	}
}

func (h *PostHandler) GetPosts(c *echo.Context) error {
	filter := &dto.PostQueryFilter{
		Limit:     10,
		Offset:    0,
		Search:    c.QueryParam("search"),
		SortBy:    c.QueryParam("sort_by"),
		SortOrder: c.QueryParam("sort_order"),
		StartDate: c.QueryParam("start_date"),
		EndDate:   c.QueryParam("end_date"),
		CreatedBy: c.QueryParam("created_by"),
	}

	if limit := c.QueryParam("limit"); limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 {
			filter.Limit = limitInt
		}
	}
	// Cap limit to prevent unbounded queries (DoS vector).
	if filter.Limit > maxPostLimit {
		filter.Limit = maxPostLimit
	}

	if offset := c.QueryParam("offset"); offset != "" {
		if offsetInt, err := strconv.Atoi(offset); err == nil && offsetInt > 0 {
			filter.Offset = offsetInt
		}
	}

	if published := c.QueryParam("published"); published != "" {
		if pubBool, err := strconv.ParseBool(published); err == nil {
			filter.Published = &pubBool
		}
	}

	if tags := c.QueryParam("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
		for i, tag := range filter.Tags {
			filter.Tags[i] = strings.TrimSpace(tag)
		}
	}

	posts, total, err := h.postService.GetPostsFiltered(c.Request().Context(), filter)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.SuccessWithMeta(c, "Successfully retrieved posts", posts,
		response.CalculatePaginationMeta(total, filter.Offset, filter.Limit))
}

func (h *PostHandler) CreatePost(c *echo.Context) error {
	var postReq dto.CreatePostRequest
	if err := c.Bind(&postReq); err != nil {
		return response.BadRequest(c, "Failed to create post", err)
	}

	if err := c.Validate(postReq); err != nil {
		return response.FromValidateError(c, err)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	newpost, err := h.postService.CreatePost(c.Request().Context(), &postReq, userID)
	if err != nil {
		return h.respondPostError(c, "Failed to create post", err)
	}
	return response.Created(c, "Successfully created post", map[string]any{
		"id": newpost.ID,
	})
}

func (h *PostHandler) UpdatePost(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	var updateDTO dto.UpdatePostRequest
	if err := c.Bind(&updateDTO); err != nil {
		return response.BadRequest(c, "Failed to update post", err)
	}

	if err := c.Validate(updateDTO); err != nil {
		return response.FromValidateError(c, err)
	}

	updatedPost, err := h.postService.UpdatePost(c.Request().Context(), id, &updateDTO)
	if err != nil {
		return h.respondPostError(c, "Failed to update post", err)
	}

	return response.Success(c, "Post updated successfully", updatedPost)
}

func (h *PostHandler) UpdateMyPost(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	var updateDTO dto.UpdatePostRequest
	if err := c.Bind(&updateDTO); err != nil {
		return response.BadRequest(c, "Failed to update post", err)
	}

	if err := c.Validate(updateDTO); err != nil {
		return response.FromValidateError(c, err)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	updatedPost, err := h.postService.UpdateMyPost(c.Request().Context(), id, userID, &updateDTO)
	if err != nil {
		return h.respondPostError(c, "Failed to update post", err)
	}

	return response.Success(c, "Post updated successfully", updatedPost)
}

func (h *PostHandler) GetPostBySlugAndUsername(c *echo.Context) error {
	slug := c.Param("slug")
	username := c.Param("username")
	post, err := h.postService.GetPostBySlugAndUsername(c.Request().Context(), slug, username)
	if err != nil {
		return h.respondPostError(c, "Failed to get post", err)
	}

	return response.Success(c, "Successfully retrieved post", post)
}

func (h *PostHandler) GetPost(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	post, err := h.postService.GetPostByID(c.Request().Context(), id)
	if err != nil {
		return h.respondPostError(c, "Failed to get post", err)
	}

	return response.Success(c, "Successfully retrieved post", post)
}

func (h *PostHandler) GetMyPost(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	if err := h.postService.IsAuthor(c.Request().Context(), id, userID); err != nil {
		return h.respondPostError(c, "Failed to check post ownership", err)
	}

	post, err := h.postService.GetPostByID(c.Request().Context(), id)
	if err != nil {
		return h.respondPostError(c, "Failed to get post", err)
	}

	return response.Success(c, "Successfully retrieved post", post)
}

func (h *PostHandler) DeletePost(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	err := h.postService.DeletePostByID(c.Request().Context(), id)
	if err != nil {
		return h.respondPostError(c, "Failed to delete post", err)
	}

	return response.Success(c, "Successfully deleted post", nil)
}

func (h *PostHandler) DeleteMyPost(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return response.BadRequest(c, "Invalid post ID", nil)
	}

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	if err := h.postService.DeleteMyPost(c.Request().Context(), id, userID); err != nil {
		return h.respondPostError(c, "Failed to delete post", err)
	}

	return response.Success(c, "Successfully deleted post", nil)
}

func (h *PostHandler) GetPostsRandom(c *echo.Context) error {
	limit, _ := ParsePaginationParams(c, 9)
	if limit > 20 {
		limit = 20
	}
	posts, err := h.postService.GetPostsRandom(c.Request().Context(), limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.Success(c, "Successfully retrieved posts", posts)
}

func (h *PostHandler) GetPostsTrending(c *echo.Context) error {
	limit, _ := ParsePaginationParams(c, 10)

	posts, err := h.postService.GetPostsTrending(c.Request().Context(), limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to get trending posts", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.Success(c, "Successfully retrieved trending posts", posts)
}

func (h *PostHandler) GetMyPosts(c *echo.Context) error {
	limit, offset := ParsePaginationParams(c, 10)

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	posts, total, err := h.postService.GetPostsByCreatedBy(c.Request().Context(), userID, offset, limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.SuccessWithMeta(c, "Successfully retrieved posts", posts,
		response.CalculatePaginationMeta(total, offset, limit))
}

func (h *PostHandler) GetMyPostsAnalytics(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	q := &dto.MyPostsAnalyticsQuery{
		StartDate: c.QueryParam("start_date"),
		EndDate:   c.QueryParam("end_date"),
	}

	analytics, err := h.postViewService.GetMyPostsAnalytics(c.Request().Context(), userID, q)
	if err != nil {
		return response.InternalServerError(c, "Failed to get post analytics", err)
	}

	return response.Success(c, "Successfully retrieved post analytics", analytics)
}

func (h *PostHandler) GetMyPostsLikesByMonth(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	months := 12
	if m := c.QueryParam("months"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 1 && v <= 24 {
			months = v
		}
	}

	data, err := h.postViewService.GetMyPostsLikesByMonth(c.Request().Context(), userID, &dto.MyPostsLikesByMonthQuery{
		Months: months,
	})
	if err != nil {
		return response.InternalServerError(c, "Failed to get likes by month", err)
	}

	return response.Success(c, "Successfully retrieved likes by month", data)
}

func (h *PostHandler) GetPostsForYou(c *echo.Context) error {
	limit, offset := ParsePaginationParams(c, 10)

	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	posts, total, err := h.postService.GetPostsForYou(c.Request().Context(), userID, offset, limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.SuccessWithMeta(c, "Successfully retrieved for-you posts", posts,
		response.CalculatePaginationMeta(total, offset, limit))
}

func (h *PostHandler) GetPostsByUsername(c *echo.Context) error {
	username := c.Param("username")
	limit, offset := ParsePaginationParams(c, 10)

	posts, total, err := h.postService.GetPostsByUsername(c.Request().Context(), username, offset, limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.SuccessWithMeta(c, "Successfully retrieved posts", posts,
		response.CalculatePaginationMeta(total, offset, limit))
}

func (h *PostHandler) GetPostsByTag(c *echo.Context) error {
	tag := c.Param("tag")
	limit, offset := ParsePaginationParams(c, 10)

	posts, total, err := h.postService.GetPostsByTag(c.Request().Context(), tag, limit, offset)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts by tag", err)
	}

	dto.TruncatePostBodies(posts, 250)

	return response.SuccessWithMeta(c, "Successfully retrieved posts by tag", posts,
		response.CalculatePaginationMeta(total, offset, limit))
}

func (h *PostHandler) UploadImagePosts(c *echo.Context) error {
	file, err := c.FormFile("image")
	if err != nil {
		return response.BadRequest(c, "Failed to upload image", err)
	}

	if file == nil {
		return response.BadRequest(c, "No file uploaded", nil)
	}

	imagePath, err := h.postService.UploadImagePosts(c.Request().Context(), file)
	if err != nil {
		return h.respondPostError(c, "Failed to upload image", err)
	}
	return response.Created(c, "Successfully uploaded image", map[string]string{
		"path": imagePath,
		"url":  imagePath,
	})
}

func (h *PostHandler) GetPostsForSitemap(c *echo.Context) error {
	posts, err := h.postService.GetPostsForSitemap(c.Request().Context(), 1000)
	if err != nil {
		return response.InternalServerError(c, "Failed to get posts for sitemap", err)
	}

	return response.Success(c, "Successfully retrieved posts for sitemap", posts)
}
