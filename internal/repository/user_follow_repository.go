package repository

import (
	"context"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type UserFollowRepository interface {
	Follow(ctx context.Context, followerID, followingID string) error
	Unfollow(ctx context.Context, followerID, followingID string) error
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
	GetFollowers(ctx context.Context, userID string, limit, offset int) ([]*model.User, int64, error)
	GetFollowing(ctx context.Context, userID string, limit, offset int) ([]*model.User, int64, error)
	GetFollowStats(ctx context.Context, userID string) (*dto.UserFollowStats, error)
	UpdateFollowCounts(ctx context.Context, userID string) error
	GetMutualFollows(ctx context.Context, userID1, userID2 string) ([]*model.User, error)
}

type userFollowRepository struct {
	db *gorm.DB
}

func NewUserFollowRepository(db *gorm.DB) UserFollowRepository {
	return &userFollowRepository{db: db}
}

func (r *userFollowRepository) Follow(ctx context.Context, followerID, followingID string) error {
	isFollowing, err := r.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if isFollowing {
		return apperrors.ErrAlreadyFollowing
	}

	if followerID == followingID {
		return apperrors.ErrCannotFollowSelf
	}

	follow := &model.UserFollow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}

	// followers_count / following_count are maintained automatically by the
	// database triggers (trigger_update_follow_counts_insert/delete on
	// user_follows), so only the follow row is created here. The previous
	// app-level gorm.Expr increments ran *in addition* to the triggers and
	// double-counted every follow.
	return r.db.WithContext(ctx).Create(follow).Error
}

func (r *userFollowRepository) Unfollow(ctx context.Context, followerID, followingID string) error {
	isFollowing, err := r.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if !isFollowing {
		return apperrors.ErrNotFollowing
	}

	// Count columns are maintained by DB triggers; only the follow row is deleted.
	return r.db.WithContext(ctx).Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&model.UserFollow{}).Error
}

func (r *userFollowRepository) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

// followListQuery selects the non-deleted users linked to userID through live
// (non-soft-deleted) user_follows rows. joinColumn is the user_follows column
// that points at the listed user, filterColumn the one that must equal userID.
// Model(&model.User{}) keeps GORM's users.deleted_at scope; the join itself
// gets no soft-delete scope, so user_follows.deleted_at is filtered by hand.
func (r *userFollowRepository) followListQuery(ctx context.Context, joinColumn, filterColumn, userID string) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Joins("JOIN user_follows ON users.id = user_follows."+joinColumn+" AND user_follows.deleted_at IS NULL").
		Where("user_follows."+filterColumn+" = ?", userID)
}

// listFollowUsers counts and pages a followListQuery; total uses the same join
// so it always agrees with the rows.
func (r *userFollowRepository) listFollowUsers(ctx context.Context, joinColumn, filterColumn, userID string, limit, offset int) ([]*model.User, int64, error) {
	var total int64
	if err := r.followListQuery(ctx, joinColumn, filterColumn, userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	users := make([]*model.User, 0)
	err := r.followListQuery(ctx, joinColumn, filterColumn, userID).
		Select("users.*").
		Preload("Profile").
		Order("user_follows.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetFollowers lists the users following userID, newest follow first.
func (r *userFollowRepository) GetFollowers(ctx context.Context, userID string, limit, offset int) ([]*model.User, int64, error) {
	return r.listFollowUsers(ctx, "follower_id", "following_id", userID, limit, offset)
}

// GetFollowing lists the users userID follows, newest follow first.
func (r *userFollowRepository) GetFollowing(ctx context.Context, userID string, limit, offset int) ([]*model.User, int64, error) {
	return r.listFollowUsers(ctx, "following_id", "follower_id", userID, limit, offset)
}

func (r *userFollowRepository) GetFollowStats(ctx context.Context, userID string) (*dto.UserFollowStats, error) {
	stats := &dto.UserFollowStats{UserID: userID}

	if err := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Where("following_id = ?", userID).Count(&stats.FollowersCount).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Where("follower_id = ?", userID).Count(&stats.FollowingCount).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *userFollowRepository) UpdateFollowCounts(ctx context.Context, userID string) error {
	stats, err := r.GetFollowStats(ctx, userID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"followers_count": stats.FollowersCount,
			"following_count": stats.FollowingCount,
		}).Error
}

// GetMutualFollows lists the non-deleted users that both userID1 and userID2
// follow through live follow rows, ordered by username. The IN subqueries use
// the UserFollow model, so GORM applies user_follows.deleted_at IS NULL, and
// they cannot produce duplicate rows the way a double join could.
func (r *userFollowRepository) GetMutualFollows(ctx context.Context, userID1, userID2 string) ([]*model.User, error) {
	db := r.db.WithContext(ctx)
	followedBy := func(followerID string) *gorm.DB {
		return db.Model(&model.UserFollow{}).Select("following_id").Where("follower_id = ?", followerID)
	}

	users := make([]*model.User, 0)
	err := db.Model(&model.User{}).
		Preload("Profile").
		Where("users.id IN (?)", followedBy(userID1)).
		Where("users.id IN (?)", followedBy(userID2)).
		Order("users.username ASC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}
