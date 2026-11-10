package repository

import (
	"context"
	"errors"
	"time"

	"echobackend/internal/model"

	"gorm.io/gorm"
)

type AccountDeletionTokenRepository interface {
	Create(ctx context.Context, token *model.AccountDeletionToken) error
	FindByToken(ctx context.Context, token string) (*model.AccountDeletionToken, error)
	// MarkUsed sets used_at only if the token is still unused. It returns false
	// when another request already used it, so a token cannot be spent twice.
	MarkUsed(ctx context.Context, id string) (bool, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

type accountDeletionTokenRepository struct {
	db *gorm.DB
}

func NewAccountDeletionTokenRepository(db *gorm.DB) AccountDeletionTokenRepository {
	return &accountDeletionTokenRepository{db: db}
}

func (r *accountDeletionTokenRepository) Create(ctx context.Context, token *model.AccountDeletionToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *accountDeletionTokenRepository) FindByToken(ctx context.Context, token string) (*model.AccountDeletionToken, error) {
	var adt model.AccountDeletionToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&adt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &adt, nil
}

func (r *accountDeletionTokenRepository) MarkUsed(ctx context.Context, id string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.AccountDeletionToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", time.Now())
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *accountDeletionTokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.AccountDeletionToken{}).Error
}
