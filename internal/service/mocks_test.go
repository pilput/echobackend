package service

// This file contains shared mock implementations of repository interfaces
// used by the service-layer unit tests. The mocks are deliberately simple:
// each method delegates to a function field on the struct, letting individual
// tests configure only the behavior they care about. Methods that are not
// stubbed return their zero value or panic with a clear message so it is
// obvious when a test reaches an unexpected code path.

import (
	"context"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
)

var (
	_ repository.UserRepository = (*mockUserRepo)(nil)
)

// ---- UserRepository mock ------------------------------------------------------

type mockUserRepo struct {
	getByIDFn             func(ctx context.Context, id string, deletedOnly bool) (*model.User, error)
	getByUsernameFn       func(ctx context.Context, username string) (*model.User, error)
	getUsersFn            func(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter, orderBy dto.UserListOrder, orderDesc bool) ([]*model.User, int64, error)
	softDeleteFn          func(ctx context.Context, id string) error
	restoreByIDFn         func(ctx context.Context, id string) error
	createFn              func(ctx context.Context, user *model.User) error
	updateFn              func(ctx context.Context, user *model.User) error
	existsFn              func(ctx context.Context, email string) (bool, error)
	getByEmailFn          func(ctx context.Context, email string) (*model.User, error)
	checkUserByUsernameFn func(ctx context.Context, username string) error
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) GetByID(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, deletedOnly)
	}
	return nil, nil
}
func (m *mockUserRepo) GetUsers(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter, orderBy dto.UserListOrder, orderDesc bool) ([]*model.User, int64, error) {
	if m.getUsersFn != nil {
		return m.getUsersFn(ctx, offset, limit, deletedFilter, orderBy, orderDesc)
	}
	return nil, 0, nil
}
func (m *mockUserRepo) GetUsersByEmail(ctx context.Context, email string) ([]*model.User, error) {
	panic("GetUsersByEmail not stubbed")
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, nil
}
func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) SoftDeleteByID(ctx context.Context, id string) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) RestoreByID(ctx context.Context, id string) error {
	if m.restoreByIDFn != nil {
		return m.restoreByIDFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) Exists(ctx context.Context, email string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepo) CheckUserByUsername(ctx context.Context, username string) error {
	if m.checkUserByUsernameFn != nil {
		return m.checkUserByUsernameFn(ctx, username)
	}
	return nil
}
