package repository

import (
	"context"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

// SessionRepository defines operations for managing refresh-token sessions.
//
// Sessions form rotation chains ("families"): see model.Session.
type SessionRepository interface {
	CreateSession(ctx context.Context, s *model.Session) error
	GetSessionByRefreshToken(ctx context.Context, token string) (*model.Session, error)
	// RotateSession marks currentID as rotated and inserts next in one
	// transaction. It returns apperror.ErrSessionAlreadyRotated when a
	// concurrent refresh got there first, leaving the table untouched.
	RotateSession(ctx context.Context, currentID string, next *model.Session) error
	DeleteSession(ctx context.Context, token string) error
	// DeleteByUserID revokes every session of one user and reports how many
	// rows went, so callers can tell an administrator what they just did.
	DeleteByUserID(ctx context.Context, userID string) (int64, error)
	DeleteByFamilyID(ctx context.Context, familyID string) error
	// DeleteAll revokes every session of every user.
	DeleteAll(ctx context.Context) (int64, error)
	// DeleteExpired prunes one user's dead rows: tokens past their sliding
	// deadline and every row of a family past its absolute deadline.
	DeleteExpired(ctx context.Context, userID string) (int64, error)
	UpdateSession(ctx context.Context, s *model.Session) error
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) CreateSession(ctx context.Context, s *model.Session) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *sessionRepository) GetSessionByRefreshToken(ctx context.Context, token string) (*model.Session, error) {
	var sess model.Session
	if err := r.db.WithContext(ctx).Where("refresh_token = ?", token).First(&sess).Error; err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *sessionRepository) RotateSession(ctx context.Context, currentID string, next *model.Session) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The rotated_at IS NULL predicate is the concurrency control: two
		// parallel refreshes read the same row, but only one UPDATE matches.
		// The loser is told so and falls back to the grace path instead of
		// inserting a second successor for the same parent.
		res := tx.Model(&model.Session{}).
			Where("id = ? AND rotated_at IS NULL", currentID).
			Updates(map[string]any{
				"rotated_at":  next.CreatedAt,
				"replaced_by": next.RefreshToken,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperrors.ErrSessionAlreadyRotated
		}
		return tx.Create(next).Error
	})
}

func (r *sessionRepository) DeleteSession(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Where("refresh_token = ?", token).Delete(&model.Session{}).Error
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	res := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.Session{})
	return res.RowsAffected, res.Error
}

func (r *sessionRepository) DeleteByFamilyID(ctx context.Context, familyID string) error {
	return r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&model.Session{}).Error
}

func (r *sessionRepository) DeleteAll(ctx context.Context) (int64, error) {
	// GORM refuses a Delete with no conditions, so the always-true predicate is
	// required rather than decorative.
	res := r.db.WithContext(ctx).Where("1 = 1").Delete(&model.Session{})
	return res.RowsAffected, res.Error
}

func (r *sessionRepository) DeleteExpired(ctx context.Context, userID string) (int64, error) {
	now := time.Now()
	// Rotated rows are covered by expires_at too: past it the token could no
	// longer have been redeemed, so keeping it for replay detection buys
	// nothing and only grows the table.
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at < ? OR absolute_expires_at < ?)", userID, now, now).
		Delete(&model.Session{})
	return res.RowsAffected, res.Error
}

func (r *sessionRepository) UpdateSession(ctx context.Context, s *model.Session) error {
	return r.db.WithContext(ctx).Save(s).Error
}
