package service

import (
	"context"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
)

type CommentService interface {
	CreateComment(ctx context.Context, postID string, req *dto.CreateCommentRequest, createdBy string) (*dto.CommentResponse, error)
	GetCommentsByPostID(ctx context.Context, postID string) ([]*dto.CommentResponse, error)
	GetCommentByID(ctx context.Context, id string) (*dto.CommentResponse, error)
	UpdateComment(ctx context.Context, id string, content string, userID string) (*dto.CommentResponse, error)
	DeleteComment(ctx context.Context, id string, userID string, isAdmin bool) error
	IsCommentAuthor(ctx context.Context, commentID string, userID string) error
}

type commentService struct {
	commentRepo         repository.CommentRepository
	postRepo            repository.PostRepository
	notificationService NotificationService
}

func NewCommentService(commentRepo repository.CommentRepository, postRepo repository.PostRepository, notificationService NotificationService) CommentService {
	return &commentService{
		commentRepo:         commentRepo,
		postRepo:            postRepo,
		notificationService: notificationService,
	}
}

func (s *commentService) CreateComment(ctx context.Context, postID string, req *dto.CreateCommentRequest, createdBy string) (*dto.CommentResponse, error) {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, apperrors.ErrPostNotFound
	}

	comment := &model.PostComment{
		PostID:    postID,
		Text:      req.Text,
		CreatedBy: createdBy,
	}

	if err := s.commentRepo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	created, err := s.commentRepo.GetCommentByID(ctx, comment.ID)
	if err != nil {
		return nil, err
	}

	if s.notificationService != nil && post.CreatedBy != nil && *post.CreatedBy != createdBy {
		message := "Someone commented on your post"
		title := "New comment"
		_, _ = s.notificationService.CreateNotification(ctx, &dto.CreateNotificationRequest{
			UserID:  *post.CreatedBy,
			Type:    "comment",
			Title:   title,
			Message: &message,
			Data: map[string]any{
				"post_id":    postID,
				"comment_id": created.ID,
				"actor_id":   createdBy,
			},
		})
	}

	return dto.CommentToResponse(created), nil
}

func (s *commentService) GetCommentsByPostID(ctx context.Context, postID string) ([]*dto.CommentResponse, error) {
	_, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, apperrors.ErrPostNotFound
	}

	comments, err := s.commentRepo.GetCommentsByPostID(ctx, postID)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.CommentResponse, len(comments))
	for i, comment := range comments {
		responses[i] = dto.CommentToResponse(comment)
	}

	return responses, nil
}

func (s *commentService) GetCommentByID(ctx context.Context, id string) (*dto.CommentResponse, error) {
	comment, err := s.commentRepo.GetCommentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.CommentToResponse(comment), nil
}

func (s *commentService) UpdateComment(ctx context.Context, id string, text string, userID string) (*dto.CommentResponse, error) {
	comment, err := s.commentRepo.GetCommentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if comment.CreatedBy != userID {
		return nil, apperrors.ErrCommentNotOwned
	}

	comment.Text = text
	if err := s.commentRepo.UpdateComment(ctx, comment); err != nil {
		return nil, err
	}

	updated, err := s.commentRepo.GetCommentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return dto.CommentToResponse(updated), nil
}

// DeleteComment deletes a comment. Ownership is enforced unless isAdmin is
// true, in which case a super admin may delete any comment (moderation).
func (s *commentService) DeleteComment(ctx context.Context, id string, userID string, isAdmin bool) error {
	comment, err := s.commentRepo.GetCommentByID(ctx, id)
	if err != nil {
		return err
	}

	if !isAdmin && comment.CreatedBy != userID {
		return apperrors.ErrCommentNotOwned
	}

	return s.commentRepo.DeleteComment(ctx, id)
}

func (s *commentService) IsCommentAuthor(ctx context.Context, commentID string, userID string) error {
	comment, err := s.commentRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.CreatedBy != userID {
		return apperrors.ErrNotAuthor
	}
	return nil
}
