package blog

import (
	"context"
	"errors"
	"testing"
)

const (
	validPostID = "550e8400-e29b-41d4-a716-446655440000"
	validUserID = "018f4d39-3a4f-7c4f-9b2a-2cf6f8c4f4d3"
)

func TestPostLikeService_LikePost_Success(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return &Post{ID: id}, nil
		},
	}
	called := false
	likeRepo := &mockPostLikeRepo{
		hasUserLikedFn: func(ctx context.Context, postID, userID string) (bool, error) {
			return false, nil
		},
		createLikeFn: func(ctx context.Context, like *PostLike) error {
			called = true
			if like.PostID != validPostID || like.UserID != validUserID {
				t.Errorf("unexpected like: %+v", like)
			}
			if like.CreatedAt == nil {
				t.Error("expected CreatedAt to be set")
			}
			return nil
		},
	}
	svc := NewPostLikeService(likeRepo, postRepo)
	if err := svc.LikePost(context.Background(), validPostID, validUserID); err != nil {
		t.Fatalf("LikePost returned error: %v", err)
	}
	if !called {
		t.Fatal("expected CreateLike to be called")
	}
}

func TestPostLikeService_LikePost_InvalidUUID(t *testing.T) {
	svc := NewPostLikeService(&mockPostLikeRepo{}, &mockPostRepo{})
	err := svc.LikePost(context.Background(), "bad", validUserID)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPostLikeService_LikePost_PostNotFound(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return nil, ErrPostNotFound
		},
	}
	svc := NewPostLikeService(&mockPostLikeRepo{}, postRepo)
	err := svc.LikePost(context.Background(), validPostID, validUserID)
	if err == nil || !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected wrapped ErrPostNotFound, got %v", err)
	}
}

func TestPostLikeService_LikePost_AlreadyLiked(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return &Post{ID: id}, nil
		},
	}
	likeRepo := &mockPostLikeRepo{
		hasUserLikedFn: func(ctx context.Context, postID, userID string) (bool, error) {
			return true, nil
		},
		createLikeFn: func(ctx context.Context, like *PostLike) error {
			t.Fatal("CreateLike should not be called when already liked")
			return nil
		},
	}
	svc := NewPostLikeService(likeRepo, postRepo)
	err := svc.LikePost(context.Background(), validPostID, validUserID)
	if !errors.Is(err, ErrAlreadyLiked) {
		t.Fatalf("expected ErrAlreadyLiked, got %v", err)
	}
}

func TestPostLikeService_UnlikePost_Success(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return &Post{ID: id}, nil
		},
	}
	deleted := false
	likeRepo := &mockPostLikeRepo{
		hasUserLikedFn: func(ctx context.Context, postID, userID string) (bool, error) {
			return true, nil
		},
		deleteLikeFn: func(ctx context.Context, postID, userID string) error {
			deleted = true
			return nil
		},
	}
	svc := NewPostLikeService(likeRepo, postRepo)
	if err := svc.UnlikePost(context.Background(), validPostID, validUserID); err != nil {
		t.Fatalf("UnlikePost error: %v", err)
	}
	if !deleted {
		t.Fatal("expected DeleteLike to be called")
	}
}

func TestPostLikeService_UnlikePost_NotLiked(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return &Post{ID: id}, nil
		},
	}
	likeRepo := &mockPostLikeRepo{
		hasUserLikedFn: func(ctx context.Context, postID, userID string) (bool, error) {
			return false, nil
		},
		deleteLikeFn: func(ctx context.Context, postID, userID string) error {
			t.Fatal("DeleteLike should not be called when not liked")
			return nil
		},
	}
	svc := NewPostLikeService(likeRepo, postRepo)
	err := svc.UnlikePost(context.Background(), validPostID, validUserID)
	if !errors.Is(err, ErrNotLiked) {
		t.Fatalf("expected ErrNotLiked, got %v", err)
	}
}

func TestPostLikeService_GetLikesByPostID_InvalidUUID(t *testing.T) {
	svc := NewPostLikeService(&mockPostLikeRepo{}, &mockPostRepo{})
	_, _, err := svc.GetLikesByPostID(context.Background(), "bad", 10, 0)
	if !errors.Is(err, ErrInvalidPostID) {
		t.Fatalf("expected ErrInvalidPostID, got %v", err)
	}
}

func TestPostLikeService_GetLikesByPostID_InvalidPagination(t *testing.T) {
	svc := NewPostLikeService(&mockPostLikeRepo{}, &mockPostRepo{})
	_, _, err := svc.GetLikesByPostID(context.Background(), validPostID, 0, 0)
	if err == nil {
		t.Fatal("expected pagination validation error")
	}
}

func TestPostLikeService_GetLikesByPostID_Success(t *testing.T) {
	likeRepo := &mockPostLikeRepo{
		getLikesByPostIDFn: func(ctx context.Context, postID string, limit, offset int) ([]*PostLike, int64, error) {
			return []*PostLike{
				{ID: "like-1", PostID: postID, UserID: validUserID},
				{ID: "like-2", PostID: postID, UserID: validUserID},
			}, 2, nil
		},
	}
	svc := NewPostLikeService(likeRepo, &mockPostRepo{})
	resp, total, err := svc.GetLikesByPostID(context.Background(), validPostID, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

func TestPostLikeService_HasUserLikedPost_InvalidInput(t *testing.T) {
	svc := NewPostLikeService(&mockPostLikeRepo{}, &mockPostRepo{})
	_, err := svc.HasUserLikedPost(context.Background(), "bad", "bad")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPostLikeService_HasUserLikedPost_Success(t *testing.T) {
	likeRepo := &mockPostLikeRepo{
		hasUserLikedFn: func(ctx context.Context, postID, userID string) (bool, error) {
			return true, nil
		},
	}
	svc := NewPostLikeService(likeRepo, &mockPostRepo{})
	got, err := svc.HasUserLikedPost(context.Background(), validPostID, validUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected true")
	}
}

func TestPostLikeService_GetLikeStats_InvalidUUID(t *testing.T) {
	svc := NewPostLikeService(&mockPostLikeRepo{}, &mockPostRepo{})
	_, err := svc.GetLikeStats(context.Background(), "bad")
	if !errors.Is(err, ErrInvalidPostID) {
		t.Fatalf("expected ErrInvalidPostID, got %v", err)
	}
}
