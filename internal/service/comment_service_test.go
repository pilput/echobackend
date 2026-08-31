package service

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

// ---- Inlined Comment Service Mocks --------------------------------------------

type mockCommentRepo struct {
	createCommentFn       func(ctx context.Context, comment *model.PostComment) error
	getCommentsByPostIDFn func(ctx context.Context, postID string) ([]*model.PostComment, error)
	getCommentByIDFn      func(ctx context.Context, id string) (*model.PostComment, error)
	updateCommentFn       func(ctx context.Context, comment *model.PostComment) error
	deleteCommentFn       func(ctx context.Context, id string) error
}

func (m *mockCommentRepo) CreateComment(ctx context.Context, comment *model.PostComment) error {
	if m.createCommentFn != nil {
		return m.createCommentFn(ctx, comment)
	}
	return nil
}
func (m *mockCommentRepo) GetCommentsByPostID(ctx context.Context, postID string) ([]*model.PostComment, error) {
	if m.getCommentsByPostIDFn != nil {
		return m.getCommentsByPostIDFn(ctx, postID)
	}
	return nil, nil
}
func (m *mockCommentRepo) GetCommentByID(ctx context.Context, id string) (*model.PostComment, error) {
	if m.getCommentByIDFn != nil {
		return m.getCommentByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockCommentRepo) UpdateComment(ctx context.Context, comment *model.PostComment) error {
	if m.updateCommentFn != nil {
		return m.updateCommentFn(ctx, comment)
	}
	return nil
}
func (m *mockCommentRepo) DeleteComment(ctx context.Context, id string) error {
	if m.deleteCommentFn != nil {
		return m.deleteCommentFn(ctx, id)
	}
	return nil
}

type mockNotificationService struct {
	createNotificationFn func(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error)
}

func (m *mockNotificationService) CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
	if m.createNotificationFn != nil {
		return m.createNotificationFn(ctx, req)
	}
	return nil, nil
}
func (m *mockNotificationService) GetNotifications(ctx context.Context, userID string, filter *dto.NotificationListFilter) ([]*dto.NotificationResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockNotificationService) GetUnreadCount(ctx context.Context, userID string) (*dto.NotificationUnreadCountResponse, error) {
	return nil, nil
}
func (m *mockNotificationService) MarkAsRead(ctx context.Context, id, userID string) (*dto.NotificationResponse, error) {
	return nil, nil
}
func (m *mockNotificationService) MarkAllAsRead(ctx context.Context, userID string) (*dto.NotificationMarkAllReadResponse, error) {
	return nil, nil
}

// ---- Test Cases ---------------------------------------------------------------

func TestCreateComment(t *testing.T) {
	ctx := context.Background()
	postID := "post-uuid"
	postAuthor := "author-uuid"
	commenterID := "commenter-uuid"

	t.Run("post not found", func(t *testing.T) {
		mockPost := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return nil, apperrors.ErrPostNotFound
			},
		}
		svc := NewCommentService(&mockCommentRepo{}, mockPost, nil)
		_, err := svc.CreateComment(ctx, postID, &dto.CreateCommentRequest{Text: "Test comment"}, commenterID)
		if !errors.Is(err, apperrors.ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})

	t.Run("success with notification", func(t *testing.T) {
		mockPost := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return &model.Post{ID: id, CreatedBy: &postAuthor}, nil
			},
		}

		commentText := "Nice post!"
		mockComment := &mockCommentRepo{
			createCommentFn: func(ctx context.Context, comment *model.PostComment) error {
				comment.ID = "comment-uuid"
				return nil
			},
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{
					ID:        id,
					PostID:    postID,
					Text:      commentText,
					CreatedBy: commenterID,
				}, nil
			},
		}

		notificationCreated := false
		mockNotif := &mockNotificationService{
			createNotificationFn: func(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
				notificationCreated = true
				if req.UserID != postAuthor {
					t.Errorf("expected notification recipient %s, got %s", postAuthor, req.UserID)
				}
				if req.Type != "comment" {
					t.Errorf("expected notification type comment, got %s", req.Type)
				}
				return &dto.NotificationResponse{}, nil
			},
		}

		svc := NewCommentService(mockComment, mockPost, mockNotif)
		resp, err := svc.CreateComment(ctx, postID, &dto.CreateCommentRequest{Text: commentText}, commenterID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if resp.ID != "comment-uuid" || resp.Text != commentText {
			t.Fatalf("unexpected comment response: %v", resp)
		}
		if !notificationCreated {
			t.Fatalf("expected notification to be created")
		}
	})

	t.Run("success without notification (comment by post author)", func(t *testing.T) {
		mockPost := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return &model.Post{ID: id, CreatedBy: &postAuthor}, nil
			},
		}

		mockComment := &mockCommentRepo{
			createCommentFn: func(ctx context.Context, comment *model.PostComment) error {
				comment.ID = "comment-uuid"
				return nil
			},
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{
					ID:        id,
					PostID:    postID,
					Text:      "Author's own comment",
					CreatedBy: postAuthor,
				}, nil
			},
		}

		notificationCreated := false
		mockNotif := &mockNotificationService{
			createNotificationFn: func(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
				notificationCreated = true
				return nil, nil
			},
		}

		svc := NewCommentService(mockComment, mockPost, mockNotif)
		_, err := svc.CreateComment(ctx, postID, &dto.CreateCommentRequest{Text: "Author's own comment"}, postAuthor)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if notificationCreated {
			t.Fatalf("expected no notification to be sent when author comments on their own post")
		}
	})
}

func TestGetCommentsByPostID(t *testing.T) {
	ctx := context.Background()
	postID := "post-uuid"

	t.Run("post not found", func(t *testing.T) {
		mockPost := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return nil, apperrors.ErrPostNotFound
			},
		}
		svc := NewCommentService(&mockCommentRepo{}, mockPost, nil)
		_, err := svc.GetCommentsByPostID(ctx, postID)
		if !errors.Is(err, apperrors.ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		mockPost := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return &model.Post{ID: id}, nil
			},
		}
		mockComment := &mockCommentRepo{
			getCommentsByPostIDFn: func(ctx context.Context, postID string) ([]*model.PostComment, error) {
				return []*model.PostComment{
					{ID: "c1", Text: "Comment 1"},
					{ID: "c2", Text: "Comment 2"},
				}, nil
			},
		}
		svc := NewCommentService(mockComment, mockPost, nil)
		resp, err := svc.GetCommentsByPostID(ctx, postID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if len(resp) != 2 || resp[0].ID != "c1" || resp[1].ID != "c2" {
			t.Fatalf("unexpected comment list: %v", resp)
		}
	})
}

func TestUpdateComment(t *testing.T) {
	ctx := context.Background()
	commentID := "comment-uuid"
	ownerID := "owner-uuid"

	t.Run("not owned by user", func(t *testing.T) {
		mockComment := &mockCommentRepo{
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{ID: id, CreatedBy: "other-user"}, nil
			},
		}
		svc := NewCommentService(mockComment, nil, nil)
		_, err := svc.UpdateComment(ctx, commentID, "New content", ownerID)
		if !errors.Is(err, apperrors.ErrCommentNotOwned) {
			t.Fatalf("expected ErrCommentNotOwned, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		currentText := "Old text"
		mockComment := &mockCommentRepo{
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{
					ID:        id,
					Text:      currentText,
					CreatedBy: ownerID,
				}, nil
			},
			updateCommentFn: func(ctx context.Context, comment *model.PostComment) error {
				currentText = comment.Text
				return nil
			},
		}
		svc := NewCommentService(mockComment, nil, nil)
		resp, err := svc.UpdateComment(ctx, commentID, "New text", ownerID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if resp.Text != "New text" {
			t.Fatalf("expected response text to be New text, got %s", resp.Text)
		}
	})
}

func TestDeleteComment(t *testing.T) {
	ctx := context.Background()
	commentID := "comment-uuid"
	ownerID := "owner-uuid"

	t.Run("not owned by user", func(t *testing.T) {
		mockComment := &mockCommentRepo{
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{ID: id, CreatedBy: "other-user"}, nil
			},
		}
		svc := NewCommentService(mockComment, nil, nil)
		err := svc.DeleteComment(ctx, commentID, ownerID, false)
		if !errors.Is(err, apperrors.ErrCommentNotOwned) {
			t.Fatalf("expected ErrCommentNotOwned, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		deleteCalled := false
		mockComment := &mockCommentRepo{
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{ID: id, CreatedBy: ownerID}, nil
			},
			deleteCommentFn: func(ctx context.Context, id string) error {
				deleteCalled = true
				return nil
			},
		}
		svc := NewCommentService(mockComment, nil, nil)
		err := svc.DeleteComment(ctx, commentID, ownerID, false)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !deleteCalled {
			t.Fatalf("expected deleteCommentFn to be called")
		}
	})

	t.Run("admin can delete a comment they do not own", func(t *testing.T) {
		deleteCalled := false
		mockComment := &mockCommentRepo{
			getCommentByIDFn: func(ctx context.Context, id string) (*model.PostComment, error) {
				return &model.PostComment{ID: id, CreatedBy: "other-user"}, nil
			},
			deleteCommentFn: func(ctx context.Context, id string) error {
				deleteCalled = true
				return nil
			},
		}
		svc := NewCommentService(mockComment, nil, nil)
		adminID := "admin-uuid"
		err := svc.DeleteComment(ctx, commentID, adminID, true)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !deleteCalled {
			t.Fatalf("expected deleteCommentFn to be called for admin moderation delete")
		}
	})
}
