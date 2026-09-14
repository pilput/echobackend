package repository

import (
	"context"
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, comment *model.PostComment) error
	GetCommentsByPostID(ctx context.Context, postID string) ([]*model.PostComment, error)
	GetCommentByID(ctx context.Context, id string) (*model.PostComment, error)
	UpdateComment(ctx context.Context, comment *model.PostComment) error
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

func (r *commentRepository) CreateComment(ctx context.Context, comment *model.PostComment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *commentRepository) GetCommentsByPostID(ctx context.Context, postID string) ([]*model.PostComment, error) {
	var comments []*model.PostComment

	// Get all comments with user information
	if err := r.db.WithContext(ctx).
		Preload("User", preloadUserBrief).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Find(&comments).Error; err != nil {
		return nil, err
	}

	return comments, nil
}

func (r *commentRepository) GetCommentByID(ctx context.Context, id string) (*model.PostComment, error) {
	var comment model.PostComment
	if err := r.db.WithContext(ctx).
		Preload("User", preloadUserBrief).
		Where("id = ?", id).
		First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

func (r *commentRepository) UpdateComment(ctx context.Context, comment *model.PostComment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

func (r *commentRepository) DeleteComment(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.PostComment{}).Error
}
