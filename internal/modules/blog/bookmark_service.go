package blog

import (
	"context"
	"errors"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/validator"
)

type BookmarkService interface {
	ToggleBookmark(ctx context.Context, postID, userID string, req *ToggleBookmarkRequest) (*ToggleBookmarkResponse, error)
	GetBookmarksByUser(ctx context.Context, userID string, folderID *string, limit, offset int) ([]*BookmarkResponse, int64, error)
	UpdateBookmark(ctx context.Context, bookmarkID, userID string, req *UpdateBookmarkRequest) (*BookmarkResponse, error)
	MoveBookmark(ctx context.Context, bookmarkID, userID string, folderID *string) (*BookmarkResponse, error)
	CreateFolder(ctx context.Context, userID string, req *CreateBookmarkFolderRequest) (*BookmarkFolderResponse, error)
	GetFoldersByUser(ctx context.Context, userID string) ([]*BookmarkFolderResponse, error)
	UpdateFolder(ctx context.Context, folderID, userID string, req *UpdateBookmarkFolderRequest) (*BookmarkFolderResponse, error)
	DeleteFolder(ctx context.Context, folderID, userID string) error
}

type bookmarkService struct {
	bookmarkRepo BookmarkRepository
	postRepo     PostRepository
}

func NewBookmarkService(bookmarkRepo BookmarkRepository, postRepo PostRepository) BookmarkService {
	return &bookmarkService{bookmarkRepo: bookmarkRepo, postRepo: postRepo}
}

func (s *bookmarkService) ToggleBookmark(ctx context.Context, postID, userID string, req *ToggleBookmarkRequest) (*ToggleBookmarkResponse, error) {
	if err := validateBookmarkPostAndUser(postID, userID); err != nil {
		return nil, err
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	existing, err := s.bookmarkRepo.FindBookmarkByUserAndPost(ctx, userID, postID)
	if err == nil {
		if err := s.bookmarkRepo.DeleteBookmark(ctx, existing.ID); err != nil {
			return nil, err
		}
		return &ToggleBookmarkResponse{Action: "removed"}, nil
	}
	if !errors.Is(err, ErrBookmarkNotFound) {
		return nil, err
	}

	// Removing an existing bookmark above is always allowed; only adding a new
	// one to a hidden post is refused.
	if err := ensurePostInteractable(post); err != nil {
		return nil, err
	}

	if req.FolderID != nil && *req.FolderID != "" {
		if _, err := s.bookmarkRepo.FindFolderByID(ctx, *req.FolderID, userID); err != nil {
			return nil, err
		}
	}

	bookmark := &PostBookmark{
		PostID:   postID,
		UserID:   userID,
		FolderID: req.FolderID,
		Name:     req.Name,
		Notes:    req.Notes,
	}
	if err := s.bookmarkRepo.CreateBookmark(ctx, bookmark); err != nil {
		return nil, err
	}

	created, err := s.bookmarkRepo.FindBookmarkByID(ctx, bookmark.ID, userID)
	if err != nil {
		return nil, err
	}

	return &ToggleBookmarkResponse{
		Action:   "added",
		Bookmark: BookmarkToResponse(created),
	}, nil
}

func (s *bookmarkService) GetBookmarksByUser(ctx context.Context, userID string, folderID *string, limit, offset int) ([]*BookmarkResponse, int64, error) {
	if !validator.IsValidUUID(userID) {
		return nil, 0, apperrors.ErrInvalidUserID
	}
	if folderID != nil && *folderID != "" && !validator.IsValidUUID(*folderID) {
		return nil, 0, ErrBookmarkFolderNotFound
	}

	bookmarks, total, err := s.bookmarkRepo.GetBookmarksByUser(ctx, userID, folderID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*BookmarkResponse, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		result = append(result, BookmarkToResponse(bookmark))
	}
	return result, total, nil
}

func (s *bookmarkService) UpdateBookmark(ctx context.Context, bookmarkID, userID string, req *UpdateBookmarkRequest) (*BookmarkResponse, error) {
	if err := validateBookmarkAndUser(bookmarkID, userID); err != nil {
		return nil, err
	}

	bookmark, err := s.bookmarkRepo.FindBookmarkByID(ctx, bookmarkID, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		bookmark.Name = req.Name
	}
	if req.Notes != nil {
		bookmark.Notes = req.Notes
	}
	now := time.Now()
	bookmark.UpdatedAt = &now

	if err := s.bookmarkRepo.UpdateBookmark(ctx, bookmark); err != nil {
		return nil, err
	}
	updated, err := s.bookmarkRepo.FindBookmarkByID(ctx, bookmarkID, userID)
	if err != nil {
		return nil, err
	}
	return BookmarkToResponse(updated), nil
}

func (s *bookmarkService) MoveBookmark(ctx context.Context, bookmarkID, userID string, folderID *string) (*BookmarkResponse, error) {
	if err := validateBookmarkAndUser(bookmarkID, userID); err != nil {
		return nil, err
	}
	if folderID != nil && *folderID != "" && !validator.IsValidUUID(*folderID) {
		return nil, ErrBookmarkFolderNotFound
	}

	bookmark, err := s.bookmarkRepo.FindBookmarkByID(ctx, bookmarkID, userID)
	if err != nil {
		return nil, err
	}

	if folderID != nil && *folderID != "" {
		if _, err := s.bookmarkRepo.FindFolderByID(ctx, *folderID, userID); err != nil {
			return nil, err
		}
	}

	bookmark.FolderID = folderID
	now := time.Now()
	bookmark.UpdatedAt = &now
	if err := s.bookmarkRepo.UpdateBookmark(ctx, bookmark); err != nil {
		return nil, err
	}

	updated, err := s.bookmarkRepo.FindBookmarkByID(ctx, bookmarkID, userID)
	if err != nil {
		return nil, err
	}
	return BookmarkToResponse(updated), nil
}

func (s *bookmarkService) CreateFolder(ctx context.Context, userID string, req *CreateBookmarkFolderRequest) (*BookmarkFolderResponse, error) {
	if !validator.IsValidUUID(userID) {
		return nil, apperrors.ErrInvalidUserID
	}

	folder := &BookmarkFolder{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.bookmarkRepo.CreateFolder(ctx, folder); err != nil {
		return nil, err
	}
	created, err := s.bookmarkRepo.FindFolderByID(ctx, folder.ID, userID)
	if err != nil {
		return nil, err
	}
	return BookmarkFolderToResponse(created), nil
}

func (s *bookmarkService) GetFoldersByUser(ctx context.Context, userID string) ([]*BookmarkFolderResponse, error) {
	if !validator.IsValidUUID(userID) {
		return nil, apperrors.ErrInvalidUserID
	}

	folders, err := s.bookmarkRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*BookmarkFolderResponse, 0, len(folders))
	for _, folder := range folders {
		result = append(result, BookmarkFolderToResponse(folder))
	}
	return result, nil
}

func (s *bookmarkService) UpdateFolder(ctx context.Context, folderID, userID string, req *UpdateBookmarkFolderRequest) (*BookmarkFolderResponse, error) {
	if err := validateFolderAndUser(folderID, userID); err != nil {
		return nil, err
	}

	folder, err := s.bookmarkRepo.FindFolderByID(ctx, folderID, userID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		folder.Name = *req.Name
	}
	if req.Description != nil {
		folder.Description = req.Description
	}
	now := time.Now()
	folder.UpdatedAt = &now

	if err := s.bookmarkRepo.UpdateFolder(ctx, folder); err != nil {
		return nil, err
	}
	updated, err := s.bookmarkRepo.FindFolderByID(ctx, folderID, userID)
	if err != nil {
		return nil, err
	}
	return BookmarkFolderToResponse(updated), nil
}

func (s *bookmarkService) DeleteFolder(ctx context.Context, folderID, userID string) error {
	if err := validateFolderAndUser(folderID, userID); err != nil {
		return err
	}
	return s.bookmarkRepo.DeleteFolder(ctx, folderID, userID)
}

func validateBookmarkPostAndUser(postID, userID string) error {
	if !validator.IsValidUUID(postID) {
		return ErrInvalidPostID
	}
	if !validator.IsValidUUID(userID) {
		return apperrors.ErrInvalidUserID
	}
	return nil
}

func validateBookmarkAndUser(bookmarkID, userID string) error {
	if !validator.IsValidUUID(bookmarkID) {
		return ErrBookmarkNotFound
	}
	if !validator.IsValidUUID(userID) {
		return apperrors.ErrInvalidUserID
	}
	return nil
}

func validateFolderAndUser(folderID, userID string) error {
	if !validator.IsValidUUID(folderID) {
		return ErrBookmarkFolderNotFound
	}
	if !validator.IsValidUUID(userID) {
		return apperrors.ErrInvalidUserID
	}
	return nil
}
