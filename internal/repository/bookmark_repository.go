package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type BookmarkRepository interface {
	FindBookmarkByUserAndPost(ctx context.Context, userID, postID string) (*model.PostBookmark, error)
	FindBookmarkByID(ctx context.Context, id, userID string) (*model.PostBookmark, error)
	CreateBookmark(ctx context.Context, bookmark *model.PostBookmark) error
	DeleteBookmark(ctx context.Context, id string) error
	UpdateBookmark(ctx context.Context, bookmark *model.PostBookmark) error
	GetBookmarksByUser(ctx context.Context, userID string, folderID *string, limit, offset int) ([]*model.PostBookmark, int64, error)
	FindFolderByID(ctx context.Context, id, userID string) (*model.BookmarkFolder, error)
	CreateFolder(ctx context.Context, folder *model.BookmarkFolder) error
	UpdateFolder(ctx context.Context, folder *model.BookmarkFolder) error
	DeleteFolder(ctx context.Context, id, userID string) error
	GetFoldersByUser(ctx context.Context, userID string) ([]*model.BookmarkFolder, error)
}

type bookmarkRepository struct {
	db *gorm.DB
}

func NewBookmarkRepository(db *gorm.DB) BookmarkRepository {
	return &bookmarkRepository{db: db}
}

func (r *bookmarkRepository) FindBookmarkByUserAndPost(ctx context.Context, userID, postID string) (*model.PostBookmark, error) {
	var bookmark model.PostBookmark
	err := r.db.WithContext(ctx).
		Preload("Post").
		Preload("Post.User", preloadUserBrief).
		Preload("Folder").
		Where("user_id = ? AND post_id = ?", userID, postID).
		First(&bookmark).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrBookmarkNotFound
		}
		return nil, fmt.Errorf("failed to find bookmark by user and post: %w", err)
	}
	return &bookmark, nil
}

func (r *bookmarkRepository) FindBookmarkByID(ctx context.Context, id, userID string) (*model.PostBookmark, error) {
	var bookmark model.PostBookmark
	err := r.db.WithContext(ctx).
		Preload("Post").
		Preload("Post.User", preloadUserBrief).
		Preload("Folder").
		Where("id = ? AND user_id = ?", id, userID).
		First(&bookmark).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrBookmarkNotFound
		}
		return nil, fmt.Errorf("failed to find bookmark: %w", err)
	}
	return &bookmark, nil
}

func (r *bookmarkRepository) CreateBookmark(ctx context.Context, bookmark *model.PostBookmark) error {
	if err := r.db.WithContext(ctx).Create(bookmark).Error; err != nil {
		return fmt.Errorf("failed to create bookmark: %w", err)
	}
	return nil
}

func (r *bookmarkRepository) DeleteBookmark(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.PostBookmark{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete bookmark: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrBookmarkNotFound
	}
	return nil
}

func (r *bookmarkRepository) UpdateBookmark(ctx context.Context, bookmark *model.PostBookmark) error {
	result := r.db.WithContext(ctx).Save(bookmark)
	if result.Error != nil {
		return fmt.Errorf("failed to update bookmark: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrBookmarkNotFound
	}
	return nil
}

func (r *bookmarkRepository) GetBookmarksByUser(ctx context.Context, userID string, folderID *string, limit, offset int) ([]*model.PostBookmark, int64, error) {
	var bookmarks []*model.PostBookmark
	var total int64

	// Bookmarks of posts a moderator has hidden stay in the table (they reappear
	// if the post is unhidden) but are left out of the list.
	query := r.db.WithContext(ctx).Model(&model.PostBookmark{}).
		Where("user_id = ?", userID).
		Where("post_id NOT IN (SELECT id FROM posts WHERE hidden_at IS NOT NULL)")
	if folderID != nil {
		if *folderID == "" {
			query = query.Where("folder_id IS NULL")
		} else {
			query = query.Where("folder_id = ?", *folderID)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count bookmarks: %w", err)
	}

	err := query.
		Preload("Post").
		Preload("Post.User", preloadUserBrief).
		Preload("Folder").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&bookmarks).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch bookmarks: %w", err)
	}

	return bookmarks, total, nil
}

func (r *bookmarkRepository) FindFolderByID(ctx context.Context, id, userID string) (*model.BookmarkFolder, error) {
	var folder model.BookmarkFolder
	err := r.db.WithContext(ctx).
		Preload("PostBookmarks").
		Where("id = ? AND user_id = ?", id, userID).
		First(&folder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrBookmarkFolderNotFound
		}
		return nil, fmt.Errorf("failed to find bookmark folder: %w", err)
	}
	return &folder, nil
}

func (r *bookmarkRepository) CreateFolder(ctx context.Context, folder *model.BookmarkFolder) error {
	if err := r.db.WithContext(ctx).Create(folder).Error; err != nil {
		return fmt.Errorf("failed to create bookmark folder: %w", err)
	}
	return nil
}

func (r *bookmarkRepository) UpdateFolder(ctx context.Context, folder *model.BookmarkFolder) error {
	result := r.db.WithContext(ctx).Save(folder)
	if result.Error != nil {
		return fmt.Errorf("failed to update bookmark folder: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrBookmarkFolderNotFound
	}
	return nil
}

func (r *bookmarkRepository) DeleteFolder(ctx context.Context, id, userID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.BookmarkFolder{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete bookmark folder: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrBookmarkFolderNotFound
	}
	return nil
}

func (r *bookmarkRepository) GetFoldersByUser(ctx context.Context, userID string) ([]*model.BookmarkFolder, error) {
	var folders []*model.BookmarkFolder
	err := r.db.WithContext(ctx).
		Preload("PostBookmarks").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&folders).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bookmark folders: %w", err)
	}
	return folders, nil
}
