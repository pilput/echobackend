package blog

import (
	"context"

	apperrors "echobackend/internal/apperror"
)

// Notification is a user notification raised by the blog (e.g. a new comment
// on someone's post).
type Notification struct {
	UserID  string
	Type    string
	Title   string
	Message *string
	Data    map[string]any
}

// Notifier delivers notifications; implemented in core (internal/di) on top
// of the notification service. Delivery is best-effort.
type Notifier interface {
	Notify(ctx context.Context, n *Notification) error
}

type CommentService interface {
	CreateComment(ctx context.Context, postID string, req *CreateCommentRequest, createdBy string) (*CommentResponse, error)
	GetCommentsByPostID(ctx context.Context, postID string) ([]*CommentResponse, error)
	GetCommentByID(ctx context.Context, id string) (*CommentResponse, error)
	UpdateComment(ctx context.Context, postID string, id string, content string, userID string) (*CommentResponse, error)
	DeleteComment(ctx context.Context, postID string, id string, userID string, isAdmin bool) error
	IsCommentAuthor(ctx context.Context, commentID string, userID string) error
}

type commentService struct {
	commentRepo CommentRepository
	postRepo    PostRepository
	notifier    Notifier
}

func NewCommentService(commentRepo CommentRepository, postRepo PostRepository, notifier Notifier) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		notifier:    notifier,
	}
}

func (s *commentService) CreateComment(ctx context.Context, postID string, req *CreateCommentRequest, createdBy string) (*CommentResponse, error) {
	if err := rejectSpamText(req.Text); err != nil {
		return nil, err
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if err := ensurePostInteractable(post); err != nil {
		return nil, err
	}

	comment := &PostComment{
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

	if s.notifier != nil && post.CreatedBy != nil && *post.CreatedBy != createdBy {
		message := "Someone commented on your post"
		title := "New comment"
		_ = s.notifier.Notify(ctx, &Notification{
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

	return CommentToResponse(created), nil
}

func (s *commentService) GetCommentsByPostID(ctx context.Context, postID string) ([]*CommentResponse, error) {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if err := ensurePostInteractable(post); err != nil {
		return nil, err
	}

	comments, err := s.commentRepo.GetCommentsByPostID(ctx, postID)
	if err != nil {
		return nil, err
	}

	responses := make([]*CommentResponse, len(comments))
	for i, comment := range comments {
		responses[i] = CommentToResponse(comment)
	}

	return responses, nil
}

func (s *commentService) GetCommentByID(ctx context.Context, id string) (*CommentResponse, error) {
	comment, err := s.commentRepo.GetCommentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return CommentToResponse(comment), nil
}

// UpdateComment edits a comment of postID. The lookup is scoped to the post in
// the path: a comment of another post is reported as ErrCommentNotFound (404)
// before ownership is checked.
func (s *commentService) UpdateComment(ctx context.Context, postID string, id string, text string, userID string) (*CommentResponse, error) {
	if err := rejectSpamText(text); err != nil {
		return nil, err
	}

	comment, err := s.commentRepo.GetCommentByPostAndID(ctx, postID, id)
	if err != nil {
		return nil, err
	}

	if comment.CreatedBy != userID {
		return nil, ErrCommentNotOwned
	}

	comment.Text = text
	if err := s.commentRepo.UpdateComment(ctx, comment); err != nil {
		return nil, err
	}

	updated, err := s.commentRepo.GetCommentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return CommentToResponse(updated), nil
}

// DeleteComment deletes a comment of postID (a comment of another post is
// ErrCommentNotFound). Ownership is enforced unless isAdmin is true, in which
// case a super admin may delete any comment (moderation).
func (s *commentService) DeleteComment(ctx context.Context, postID string, id string, userID string, isAdmin bool) error {
	comment, err := s.commentRepo.GetCommentByPostAndID(ctx, postID, id)
	if err != nil {
		return err
	}

	if !isAdmin && comment.CreatedBy != userID {
		return ErrCommentNotOwned
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
