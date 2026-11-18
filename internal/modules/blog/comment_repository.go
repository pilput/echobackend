package blog

import (
	"context"
	"echobackend/internal/core/userref"
	"errors"

	"gorm.io/gorm"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, comment *PostComment) error
	GetCommentsByPostID(ctx context.Context, postID string) ([]*PostComment, error)
	GetCommentByID(ctx context.Context, id string) (*PostComment, error)
	GetCommentByPostAndID(ctx context.Context, postID string, id string) (*PostComment, error)
	UpdateComment(ctx context.Context, comment *PostComment) error
	DeleteComment(ctx context.Context, id string) error
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{
		db: db,
	}
}

func (r *commentRepository) CreateComment(ctx context.Context, comment *PostComment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *commentRepository) GetCommentsByPostID(ctx context.Context, postID string) ([]*PostComment, error) {
	var comments []*PostComment

	// Get all comments with user information
	if err := r.db.WithContext(ctx).
		Preload("User", userref.PreloadBrief).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Find(&comments).Error; err != nil {
		return nil, err
	}

	return comments, nil
}

func (r *commentRepository) GetCommentByID(ctx context.Context, id string) (*PostComment, error) {
	var comment PostComment
	if err := r.db.WithContext(ctx).
		Preload("User", userref.PreloadBrief).
		Where("id = ?", id).
		First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

// GetCommentByPostAndID loads a comment only when it belongs to postID, so a
// comment cannot be reached through another post's URL.
func (r *commentRepository) GetCommentByPostAndID(ctx context.Context, postID string, id string) (*PostComment, error) {
	var comment PostComment
	if err := r.db.WithContext(ctx).
		Preload("User", userref.PreloadBrief).
		Where("id = ? AND post_id = ?", id, postID).
		First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

func (r *commentRepository) UpdateComment(ctx context.Context, comment *PostComment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

func (r *commentRepository) DeleteComment(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&PostComment{}).Error
}
