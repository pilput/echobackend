package blog

import (
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/middleware"
	"echobackend/pkg/response"
	"errors"
	"io"

	"github.com/labstack/echo/v5"
)

type BookmarkHandler struct {
	bookmarkService BookmarkService
}

func NewBookmarkHandler(bookmarkService BookmarkService) *BookmarkHandler {
	return &BookmarkHandler{bookmarkService: bookmarkService}
}

func (h *BookmarkHandler) ToggleBookmark(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	postID := c.Param("post_id")
	if postID == "" {
		return response.BadRequest(c, "Post ID is required", nil)
	}

	var req ToggleBookmarkRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	result, err := h.bookmarkService.ToggleBookmark(c.Request().Context(), postID, userID, &req)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to toggle bookmark", err)
	}
	return response.Success(c, "Bookmark toggled successfully", result)
}

func (h *BookmarkHandler) GetBookmarks(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	limit, offset := response.ParsePaginationParams(c, 50)

	var folderID *string
	if raw := c.QueryParam("folder_id"); raw == "null" {
		empty := ""
		folderID = &empty
	} else if raw != "" {
		folderID = &raw
	}

	bookmarks, total, err := h.bookmarkService.GetBookmarksByUser(c.Request().Context(), userID, folderID, limit, offset)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to get bookmarks", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Bookmarks fetched successfully", bookmarks, meta)
}

func (h *BookmarkHandler) UpdateBookmark(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	bookmarkID := c.Param("bookmark_id")
	if bookmarkID == "" {
		return response.BadRequest(c, "Bookmark ID is required", nil)
	}

	var req UpdateBookmarkRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	bookmark, err := h.bookmarkService.UpdateBookmark(c.Request().Context(), bookmarkID, userID, &req)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to update bookmark", err)
	}
	return response.Success(c, "Bookmark updated successfully", bookmark)
}

func (h *BookmarkHandler) MoveBookmark(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	bookmarkID := c.Param("bookmark_id")
	if bookmarkID == "" {
		return response.BadRequest(c, "Bookmark ID is required", nil)
	}

	var req MoveBookmarkRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	bookmark, err := h.bookmarkService.MoveBookmark(c.Request().Context(), bookmarkID, userID, req.FolderID)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to move bookmark", err)
	}
	return response.Success(c, "Bookmark moved successfully", bookmark)
}

func (h *BookmarkHandler) CreateFolder(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req CreateBookmarkFolderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	folder, err := h.bookmarkService.CreateFolder(c.Request().Context(), userID, &req)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to create folder", err)
	}
	return response.Created(c, "Folder created successfully", folder)
}

func (h *BookmarkHandler) GetFolders(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	folders, err := h.bookmarkService.GetFoldersByUser(c.Request().Context(), userID)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to get folders", err)
	}
	return response.Success(c, "Folders fetched successfully", folders)
}

func (h *BookmarkHandler) UpdateFolder(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	folderID := c.Param("folder_id")
	if folderID == "" {
		return response.BadRequest(c, "Folder ID is required", nil)
	}

	var req UpdateBookmarkFolderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	folder, err := h.bookmarkService.UpdateFolder(c.Request().Context(), folderID, userID, &req)
	if err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to update folder", err)
	}
	return response.Success(c, "Folder updated successfully", folder)
}

func (h *BookmarkHandler) DeleteFolder(c *echo.Context) error {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	folderID := c.Param("folder_id")
	if folderID == "" {
		return response.BadRequest(c, "Folder ID is required", nil)
	}

	if err := h.bookmarkService.DeleteFolder(c.Request().Context(), folderID, userID); err != nil {
		if handled := handleBookmarkError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to delete folder", err)
	}
	return response.Success(c, "Folder deleted successfully", nil)
}

func handleBookmarkError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, ErrInvalidPostID), errors.Is(err, apperrors.ErrInvalidUserID):
		return response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, ErrPostNotFound):
		return response.NotFound(c, "Post not found", err)
	case errors.Is(err, ErrBookmarkNotFound):
		return response.NotFound(c, "Bookmark not found", err)
	case errors.Is(err, ErrBookmarkFolderNotFound):
		return response.NotFound(c, "Bookmark folder not found", err)
	default:
		return nil
	}
}
