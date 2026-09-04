package repository

import (
	"context"
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type AuthRepository interface {
	FindUserByEmail(ctx context.Context, email string) (*model.User, error)
	FindUserByIdentifier(ctx context.Context, identifier string) (*model.User, error)
	FindUserByGithubID(ctx context.Context, githubID int64) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) FindUserByIdentifier(ctx context.Context, identifier string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ? OR username = ?", identifier, identifier).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) CreateUser(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		// gorm.Config.TranslateError turns SQLSTATE 23505 into ErrDuplicatedKey,
		// so the unique-violation check no longer depends on the pgx driver.
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return apperrors.ErrUserExists
		}
		return result.Error
	}
	return nil
}

func (r *authRepository) FindUserByGithubID(ctx context.Context, githubID int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("github_id = ?", githubID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
