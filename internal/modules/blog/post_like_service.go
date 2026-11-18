package blog

import (
	"context"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/validator"
	"time"
)

type PostLikeService interface {
	LikePost(ctx context.Context, postID, userID string) error
	UnlikePost(ctx context.Context, postID, userID string) error
	GetLikesByPostID(ctx context.Context, postID string, limit, offset int) ([]*PostLikeResponse, int64, error)
	GetLikeStats(ctx context.Context, postID string) (*PostLikeStats, error)
	HasUserLikedPost(ctx context.Context, postID, userID string) (bool, error)
}

type postLikeService struct {
	postLikeRepo PostLikeRepository
	postRepo     PostRepository
}

func NewPostLikeService(
	postLikeRepo PostLikeRepository,
	postRepo PostRepository,
) PostLikeService {
	return &postLikeService{
		postLikeRepo: postLikeRepo,
		postRepo:     postRepo,
	}
}

func (s *postLikeService) LikePost(ctx context.Context, postID, userID string) error {
	if err := validatePostLikeIDs(postID, userID); err != nil {
		return err
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to check post existence: %w", err)
	}
	if err := ensurePostInteractable(post); err != nil {
		return err
	}

	hasLiked, err := s.postLikeRepo.HasUserLikedPost(ctx, postID, userID)
	if err != nil {
		return fmt.Errorf("failed to check like status: %w", err)
	}

	if hasLiked {
		return ErrAlreadyLiked
	}

	now := time.Now()
	like := &PostLike{
		PostID:    postID,
		UserID:    userID,
		CreatedAt: &now,
	}

	return s.postLikeRepo.CreateLike(ctx, like)
}

func (s *postLikeService) UnlikePost(ctx context.Context, postID, userID string) error {
	if err := validatePostLikeIDs(postID, userID); err != nil {
		return err
	}

	_, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to check post existence: %w", err)
	}

	hasLiked, err := s.postLikeRepo.HasUserLikedPost(ctx, postID, userID)
	if err != nil {
		return fmt.Errorf("failed to check like status: %w", err)
	}

	if !hasLiked {
		return ErrNotLiked
	}

	return s.postLikeRepo.DeleteLike(ctx, postID, userID)
}

func (s *postLikeService) GetLikesByPostID(ctx context.Context, postID string, limit, offset int) ([]*PostLikeResponse, int64, error) {
	if !validator.IsValidUUID(postID) {
		return nil, 0, ErrInvalidPostID
	}

	if err := validator.ValidatePagination(limit, offset); err != nil {
		return nil, 0, fmt.Errorf("validation error: %w", err)
	}

	likes, total, err := s.postLikeRepo.GetLikesByPostID(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*PostLikeResponse, len(likes))
	for i, like := range likes {
		responses[i] = PostLikeToResponse(like)
	}

	return responses, total, nil
}

func (s *postLikeService) GetLikeStats(ctx context.Context, postID string) (*PostLikeStats, error) {
	if !validator.IsValidUUID(postID) {
		return nil, ErrInvalidPostID
	}

	return s.postLikeRepo.GetLikeStats(ctx, postID)
}

func (s *postLikeService) HasUserLikedPost(ctx context.Context, postID, userID string) (bool, error) {
	if err := validatePostLikeIDs(postID, userID); err != nil {
		return false, err
	}

	return s.postLikeRepo.HasUserLikedPost(ctx, postID, userID)
}

func validatePostLikeIDs(postID, userID string) error {
	if !validator.IsValidUUID(postID) {
		return ErrInvalidPostID
	}
	if !validator.IsValidUUID(userID) {
		return apperrors.ErrInvalidUserID
	}
	return nil
}
