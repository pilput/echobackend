package service

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

func TestUserService_GetByID_Success(t *testing.T) {
	first := "Alice"
	last := "Smith"
	repo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return &model.User{ID: id, Email: "a@b.com", FirstName: &first, LastName: &last}, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, err := svc.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "user-1" || resp.Email != "" {
		t.Errorf("unexpected response %+v", resp)
	}
	if resp.Name != "Alice Smith" {
		t.Errorf("Name = %q, want %q", resp.Name, "Alice Smith")
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return nil, apperrors.ErrUserNotFound
		},
	}
	svc := NewUserService(repo, nil)
	_, err := svc.GetByID(context.Background(), "missing")
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_GetMe_Success(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return &model.User{ID: id, Email: "me@example.com", IsSuperAdmin: new(true)}, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, err := svc.GetMe(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsSuperAdmin == nil || !*resp.IsSuperAdmin {
		t.Fatalf("expected IsSuperAdmin true, got %v", resp.IsSuperAdmin)
	}
}

func TestUserService_GetByUsername_Success(t *testing.T) {
	uname := "alice"
	repo := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return &model.User{ID: "1", Email: "alice@x.com", Username: &uname}, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, err := svc.GetByUsername(context.Background(), uname)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Username == nil || *resp.Username != uname {
		t.Fatalf("got %v", resp.Username)
	}
}

func TestUserService_GetUsers_Success(t *testing.T) {
	repo := &mockUserRepo{
		getUsersFn: func(ctx context.Context, offset, limit int, deletedFilter dto.UserDeletedFilter) ([]*model.User, int64, error) {
			return []*model.User{
				{ID: "u1", Email: "a@x.com", IsSuperAdmin: new(false)},
				nil, // nil entries should be skipped per service contract
				{ID: "u2", Email: "b@x.com", IsSuperAdmin: new(true)},
			}, 2, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, total, err := svc.GetUsers(context.Background(), 0, 10, dto.UserDeletedFilterActive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(resp) != 2 {
		t.Fatalf("len = %d, want 2", len(resp))
	}
	if resp[0].IsSuperAdmin == nil || *resp[0].IsSuperAdmin {
		t.Fatalf("expected u1 IsSuperAdmin false, got %v", resp[0].IsSuperAdmin)
	}
	if resp[1].IsSuperAdmin == nil || !*resp[1].IsSuperAdmin {
		t.Fatalf("expected u2 IsSuperAdmin true, got %v", resp[1].IsSuperAdmin)
	}
}

func TestUserService_GetUsers_RepoError(t *testing.T) {
	wantErr := errors.New("db down")
	repo := &mockUserRepo{
		getUsersFn: func(ctx context.Context, offset, limit int, deletedFilter dto.UserDeletedFilter) ([]*model.User, int64, error) {
			return nil, 0, wantErr
		},
	}
	svc := NewUserService(repo, nil)
	_, _, err := svc.GetUsers(context.Background(), 0, 10, dto.UserDeletedFilterActive)
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
}

func TestUserService_GetUsers_DeletedFilter(t *testing.T) {
	deletedAt := time.Now()
	var gotFilter dto.UserDeletedFilter
	repo := &mockUserRepo{
		getUsersFn: func(ctx context.Context, offset, limit int, deletedFilter dto.UserDeletedFilter) ([]*model.User, int64, error) {
			gotFilter = deletedFilter
			return []*model.User{
				{ID: "u1", Email: "a@x.com", DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true}},
			}, 1, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, total, err := svc.GetUsers(context.Background(), 0, 10, dto.UserDeletedFilterOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFilter != dto.UserDeletedFilterOnly {
		t.Fatalf("filter = %q, want %q", gotFilter, dto.UserDeletedFilterOnly)
	}
	if total != 1 || len(resp) != 1 {
		t.Fatalf("total=%d len=%d, want 1/1", total, len(resp))
	}
	if resp[0].DeletedAt == nil {
		t.Fatal("expected deleted_at to be set")
	}
}

func TestUserService_GetAdminByID_DeletedOnly(t *testing.T) {
	deletedAt := time.Now()
	repo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			if !deletedOnly {
				t.Fatal("expected deletedOnly=true")
			}
			return &model.User{ID: id, Email: "a@x.com", DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true}}, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, err := svc.GetAdminByID(context.Background(), "u1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "u1" || resp.DeletedAt == nil {
		t.Fatalf("unexpected response %+v", resp)
	}
}

func TestUserService_GetAdminByID_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return nil, apperrors.ErrUserNotFound
		},
	}
	svc := NewUserService(repo, nil)
	_, err := svc.GetAdminByID(context.Background(), "missing", true)
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_Delete(t *testing.T) {
	deleted := false
	repo := &mockUserRepo{
		softDeleteFn: func(ctx context.Context, id string) error {
			if id != "u1" {
				t.Errorf("unexpected id %q", id)
			}
			deleted = true
			return nil
		},
	}
	svc := NewUserService(repo, nil)
	if err := svc.Delete(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected SoftDeleteByID to be called")
	}
}

func TestUserService_Restore_Success(t *testing.T) {
	repo := &mockUserRepo{
		restoreByIDFn: func(ctx context.Context, id string) error {
			if id != "u1" {
				t.Errorf("unexpected id %q", id)
			}
			return nil
		},
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return &model.User{ID: id, Email: "a@x.com", IsSuperAdmin: new(false)}, nil
		},
	}
	svc := NewUserService(repo, nil)
	resp, err := svc.Restore(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "u1" || resp.DeletedAt != nil {
		t.Fatalf("unexpected response %+v", resp)
	}
}

func TestUserService_Restore_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		restoreByIDFn: func(ctx context.Context, id string) error {
			return apperrors.ErrUserNotFound
		},
	}
	svc := NewUserService(repo, nil)
	_, err := svc.Restore(context.Background(), "missing")
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_Restore_Conflict(t *testing.T) {
	repo := &mockUserRepo{
		restoreByIDFn: func(ctx context.Context, id string) error {
			return apperrors.ErrUserExists
		},
	}
	svc := NewUserService(repo, nil)
	_, err := svc.Restore(context.Background(), "u1")
	if !errors.Is(err, apperrors.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}
