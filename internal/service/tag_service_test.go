package service

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

func TestTagService_CreateTag_Empty(t *testing.T) {
	svc := NewTagService(&mockTagRepo{})
	_, err := svc.CreateTag(context.Background(), &dto.CreateTagRequest{Name: ""})
	if !errors.Is(err, apperrors.ErrTagNameRequired) {
		t.Fatalf("expected ErrTagNameRequired, got %v", err)
	}
}

func TestTagService_CreateTag_Success(t *testing.T) {
	created := false
	repo := &mockTagRepo{
		createFn: func(ctx context.Context, tag *model.Tag) error {
			created = true
			return nil
		},
	}
	svc := NewTagService(repo)
	if _, err := svc.CreateTag(context.Background(), &dto.CreateTagRequest{Name: "go"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected Create to be called")
	}
}

func TestTagService_FindOrCreateByName_EmptyName(t *testing.T) {
	repo := &mockTagRepo{
		findOrCreateByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
			t.Fatal("repository should not be reached for an empty name")
			return nil, nil
		},
	}
	svc := NewTagService(repo)
	_, err := svc.FindOrCreateByName(context.Background(), "")
	if !errors.Is(err, apperrors.ErrTagNameEmpty) {
		t.Fatalf("expected ErrTagNameEmpty, got %v", err)
	}
}

// The find-then-insert logic itself now lives in tagRepository, where the INSERT
// can carry ON CONFLICT; the service is only responsible for rejecting an empty
// name and passing everything else straight through.
func TestTagService_FindOrCreateByName_Delegates(t *testing.T) {
	existing := &model.Tag{ID: 1, Name: "go"}
	var gotName string
	repo := &mockTagRepo{
		findOrCreateByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
			gotName = name
			return existing, nil
		},
		findByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
			t.Fatal("FindByName should not be called directly by the service")
			return nil, nil
		},
		createFn: func(ctx context.Context, tag *model.Tag) error {
			t.Fatal("Create should not be called directly by the service")
			return nil
		},
	}
	svc := NewTagService(repo)
	got, err := svc.FindOrCreateByName(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != existing {
		t.Fatalf("expected to return the repository's tag, got %+v", got)
	}
	if gotName != "go" {
		t.Fatalf("expected name %q to reach the repository, got %q", "go", gotName)
	}
}

func TestTagService_FindOrCreateByName_RepoError(t *testing.T) {
	wantErr := errors.New("db error")
	repo := &mockTagRepo{
		findOrCreateByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
			return nil, wantErr
		},
	}
	svc := NewTagService(repo)
	_, err := svc.FindOrCreateByName(context.Background(), "rust")
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}

func TestTagService_DeleteTag_NotFound(t *testing.T) {
	repo := &mockTagRepo{
		findByIDFn: func(ctx context.Context, id uint) (*model.Tag, error) {
			return nil, apperrors.ErrTagNotFound
		},
		deleteFn: func(ctx context.Context, id uint) error {
			t.Fatal("Delete should not be called if FindByID fails")
			return nil
		},
	}
	svc := NewTagService(repo)
	if err := svc.DeleteTag(context.Background(), 1); !errors.Is(err, apperrors.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagService_DeleteTag_Success(t *testing.T) {
	deleted := false
	repo := &mockTagRepo{
		findByIDFn: func(ctx context.Context, id uint) (*model.Tag, error) {
			return &model.Tag{ID: id, Name: "go"}, nil
		},
		deleteFn: func(ctx context.Context, id uint) error {
			if id != 1 {
				t.Errorf("unexpected id %d", id)
			}
			deleted = true
			return nil
		},
	}
	svc := NewTagService(repo)
	if err := svc.DeleteTag(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected Delete to be called")
	}
}

func TestTagService_GetTags(t *testing.T) {
	repo := &mockTagRepo{
		findAllFn: func(ctx context.Context) ([]model.Tag, error) {
			return []model.Tag{{ID: 1, Name: "go"}, {ID: 2, Name: "rust"}}, nil
		},
	}
	svc := NewTagService(repo)
	tags, err := svc.GetTags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("len = %d, want 2", len(tags))
	}
}
