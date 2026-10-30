package dto

import (
	"echobackend/internal/model"
	"time"
)

type ToggleBookmarkRequest struct {
	FolderID *string `json:"folder_id" validate:"omitempty,uuid"`
	Name     *string `json:"name" validate:"omitempty,max=255"`
	Notes    *string `json:"notes" validate:"omitempty,max=2000"`
}

type UpdateBookmarkRequest struct {
	Name  *string `json:"name" validate:"omitempty,max=255"`
	Notes *string `json:"notes" validate:"omitempty,max=2000"`
}

type MoveBookmarkRequest struct {
	FolderID *string `json:"folder_id" validate:"omitempty,uuid"`
}

type CreateBookmarkFolderRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
}

type UpdateBookmarkFolderRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
}

type BookmarkFolderResponse struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description"`
	BookmarkCount int64      `json:"bookmark_count"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

type BookmarkResponse struct {
	ID        string                  `json:"id"`
	PostID    string                  `json:"post_id"`
	UserID    string                  `json:"user_id"`
	FolderID  *string                 `json:"folder_id"`
	Name      *string                 `json:"name"`
	Notes     *string                 `json:"notes"`
	Post      *PostResponse           `json:"post,omitempty"`
	Folder    *BookmarkFolderResponse `json:"folder,omitempty"`
	CreatedAt *time.Time              `json:"created_at"`
	UpdatedAt *time.Time              `json:"updated_at"`
}

type ToggleBookmarkResponse struct {
	Action   string            `json:"action"`
	Bookmark *BookmarkResponse `json:"bookmark,omitempty"`
}

func BookmarkFolderToResponse(folder *model.BookmarkFolder) *BookmarkFolderResponse {
	if folder == nil {
		return nil
	}

	count := int64(len(folder.PostBookmarks))
	if folder.BookmarkCount != nil {
		count = *folder.BookmarkCount
	}

	return &BookmarkFolderResponse{
		ID:            folder.ID,
		UserID:        folder.UserID,
		Name:          folder.Name,
		Description:   folder.Description,
		BookmarkCount: count,
		CreatedAt:     folder.CreatedAt,
		UpdatedAt:     folder.UpdatedAt,
	}
}

func BookmarkToResponse(bookmark *model.PostBookmark) *BookmarkResponse {
	if bookmark == nil {
		return nil
	}

	var postResp *PostResponse
	if bookmark.Post.ID != "" {
		postResp = PostToResponse(&bookmark.Post)
	}

	return &BookmarkResponse{
		ID:        bookmark.ID,
		PostID:    bookmark.PostID,
		UserID:    bookmark.UserID,
		FolderID:  bookmark.FolderID,
		Name:      bookmark.Name,
		Notes:     bookmark.Notes,
		Post:      postResp,
		Folder:    BookmarkFolderToResponse(bookmark.Folder),
		CreatedAt: bookmark.CreatedAt,
		UpdatedAt: bookmark.UpdatedAt,
	}
}
