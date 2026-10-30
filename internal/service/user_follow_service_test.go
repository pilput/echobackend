package service

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
)

var _ repository.UserFollowRepository = (*mockUserFollowRepo)(nil)

// mockUserFollowRepo stubs repository.UserFollowRepository for the methods
// these tests reach; anything else panics so an unexpected call is obvious.
type mockUserFollowRepo struct {
	getFollowStatsFn func(ctx context.Context, userID string) (*dto.UserFollowStats, error)
}

func (m *mockUserFollowRepo) Follow(ctx context.Context, followerID, followingID string) error {
	panic("Follow not stubbed")
}
func (m *mockUserFollowRepo) Unfollow(ctx context.Context, followerID, followingID string) error {
	panic("Unfollow not stubbed")
}
func (m *mockUserFollowRepo) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	panic("IsFollowing not stubbed")
}
func (m *mockUserFollowRepo) GetFollowers(ctx context.Context, userID string, limit, offset int) ([]*model.User, int64, error) {
	panic("GetFollowers not stubbed")
}
func (m *mockUserFollowRepo) GetFollowing(ctx context.Context, userID string, limit, offset int) ([]*model.User, int64, error) {
	panic("GetFollowing not stubbed")
}
func (m *mockUserFollowRepo) GetFollowStats(ctx context.Context, userID string) (*dto.UserFollowStats, error) {
	if m.getFollowStatsFn != nil {
		return m.getFollowStatsFn(ctx, userID)
	}
	return &dto.UserFollowStats{UserID: userID}, nil
}
func (m *mockUserFollowRepo) UpdateFollowCounts(ctx context.Context, userID string) error {
	panic("UpdateFollowCounts not stubbed")
}
func (m *mockUserFollowRepo) GetMutualFollows(ctx context.Context, userID1, userID2 string) ([]*model.User, error) {
	panic("GetMutualFollows not stubbed")
}

func TestUserFollowService_GetFollowStats_UnknownUserIsNotFound(t *testing.T) {
	follows := &mockUserFollowRepo{
		getFollowStatsFn: func(ctx context.Context, userID string) (*dto.UserFollowStats, error) {
			t.Fatal("counts must not be read for a missing user")
			return nil, nil
		},
	}
	users := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			if deletedOnly {
				t.Fatal("soft-deleted users must not be looked up")
			}
			return nil, apperrors.ErrUserNotFound
		},
	}
	_, err := NewUserFollowService(follows, users, nil).GetFollowStats(context.Background(), "u1")
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserFollowService_GetFollowStats_LiveCounts(t *testing.T) {
	follows := &mockUserFollowRepo{
		getFollowStatsFn: func(ctx context.Context, userID string) (*dto.UserFollowStats, error) {
			return &dto.UserFollowStats{UserID: userID, FollowersCount: 2, FollowingCount: 1}, nil
		},
	}
	users := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			// The stored count columns are ignored: stats come from the repo.
			return &model.User{ID: id, FollowersCount: 99, FollowingCount: 99}, nil
		},
	}
	stats, err := NewUserFollowService(follows, users, nil).GetFollowStats(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.UserID != "u1" || stats.FollowersCount != 2 || stats.FollowingCount != 1 {
		t.Fatalf("unexpected stats %+v", stats)
	}
}
