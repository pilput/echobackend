package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	pkgpassword "echobackend/pkg/password"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string, deletedOnly bool) (*model.User, error)
	GetUsers(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter) ([]*model.User, int64, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	SoftDeleteByID(ctx context.Context, id string) error
	RestoreByID(ctx context.Context, id string) error
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	CheckUserByUsername(ctx context.Context, username string) error
}

type UserService interface {
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	GetAdminByID(ctx context.Context, id string, deletedOnly bool) (*dto.UserResponse, error)
	GetMe(ctx context.Context, id string) (*dto.CurrentUserResponse, error)
	GetByUsername(ctx context.Context, username string) (*dto.UserResponse, error)
	GetUsers(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter) ([]*dto.UserResponse, int64, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (*dto.UserResponse, error)
	UploadAvatar(ctx context.Context, userID string, file *multipart.FileHeader) (string, error)
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	UpdateUser(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
}

type userService struct {
	userRepo  UserRepository
	s3storage FileUploader
}

// maxAvatarImageSize caps profile picture uploads to 5 MB, matching the
// frontend's own client-side size check.
const maxAvatarImageSize = 5 * 1024 * 1024
const avatarUploadPrefix = "users/avatars"

func NewUserService(userRepo UserRepository, s3storage FileUploader) UserService {
	return &userService{userRepo: userRepo, s3storage: s3storage}
}

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	return dto.UserToResponse(user), nil
}

func (s *userService) GetAdminByID(ctx context.Context, id string, deletedOnly bool) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id, deletedOnly)
	if err != nil {
		return nil, err
	}
	return dto.UserToAdminResponse(user), nil
}

func (s *userService) GetMe(ctx context.Context, id string) (*dto.CurrentUserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	return dto.UserToCurrentUserResponse(user), nil
}

func (s *userService) GetByUsername(ctx context.Context, username string) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return dto.UserToResponse(user), nil
}

func (s *userService) GetUsers(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter) ([]*dto.UserResponse, int64, error) {
	users, total, err := s.userRepo.GetUsers(ctx, offset, limit, deletedFilter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve users from repository: %w", err)
	}

	var userResponses []*dto.UserResponse
	for _, user := range users {
		if user == nil {
			continue
		}
		userResponses = append(userResponses, dto.UserToAdminResponse(user))
	}

	return userResponses, total, nil
}

func (s *userService) Delete(ctx context.Context, id string) error {
	return s.userRepo.SoftDeleteByID(ctx, id)
}

func (s *userService) Restore(ctx context.Context, id string) (*dto.UserResponse, error) {
	if err := s.userRepo.RestoreByID(ctx, id); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve restored user: %w", err)
	}

	return dto.UserToAdminResponse(user), nil
}

// UploadAvatar validates and stores a new profile picture for the given user,
// then persists the resulting object key on the user's Image field. It mirrors
// postService.UploadImagePosts's validation/upload pattern.
func (s *userService) UploadAvatar(ctx context.Context, userID string, file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", apperrors.ErrFileNil
	}
	if file.Size > maxAvatarImageSize {
		return "", apperrors.ErrAvatarFileTooLarge
	}
	if s.s3storage == nil {
		return "", apperrors.ErrStorageUnavailable
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()

	data, err := io.ReadAll(io.LimitReader(src, maxAvatarImageSize+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxAvatarImageSize {
		return "", apperrors.ErrAvatarFileTooLarge
	}

	contentType, ext, ok := detectAllowedImage(data)
	if !ok {
		return "", apperrors.ErrInvalidFileType
	}

	objectKey, err := randomImageObjectKey(avatarUploadPrefix, ext)
	if err != nil {
		return "", err
	}

	if err := s.s3storage.Save(ctx, objectKey, bytes.NewReader(data), contentType); err != nil {
		return "", err
	}

	user, err := s.userRepo.GetByID(ctx, userID, false)
	if err != nil {
		return "", err
	}

	user.Image = &objectKey
	if err := s.userRepo.Update(ctx, user); err != nil {
		return "", err
	}

	return objectKey, nil
}

// CreateUser is an admin action that creates a new user account, mirroring
// AuthService.Register's uniqueness checks and password hashing.
func (s *userService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	if _, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
		return nil, apperrors.ErrUserExists
	} else if !errors.Is(err, apperrors.ErrUserNotFound) {
		return nil, err
	}

	if err := s.userRepo.CheckUserByUsername(ctx, req.Username); err != nil {
		return nil, err
	}

	hashedPassword, err := pkgpassword.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	username := req.Username
	newUser := &model.User{
		Email:    req.Email,
		Username: &username,
		Password: &hashedPassword,
	}
	if req.FirstName != "" {
		newUser.FirstName = &req.FirstName
	}
	if req.LastName != "" {
		newUser.LastName = &req.LastName
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return dto.UserToAdminResponse(newUser), nil
}

// UpdateUser is an admin action that updates any user's profile fields,
// checking username/email for collisions with other users when changed.
func (s *userService) UpdateUser(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if user.Email != req.Email {
		existing, err := s.userRepo.GetByEmail(ctx, req.Email)
		if err == nil && existing.ID != id {
			return nil, apperrors.ErrUserExists
		}
		if err != nil && !errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, err
		}
	}

	if user.Username == nil || *user.Username != req.Username {
		if err := s.userRepo.CheckUserByUsername(ctx, req.Username); err != nil {
			return nil, err
		}
	}

	user.Email = req.Email
	username := req.Username
	user.Username = &username
	firstName := req.FirstName
	user.FirstName = &firstName
	lastName := req.LastName
	user.LastName = &lastName
	isSuperAdmin := req.IsSuperAdmin
	user.IsSuperAdmin = &isSuperAdmin

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return dto.UserToAdminResponse(user), nil
}
