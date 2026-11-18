package blog

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTagService_CreateTag_Empty(t *testing.T) {
	svc := NewTagService(&mockTagRepo{})
	_, err := svc.CreateTag(context.Background(), &CreateTagRequest{Name: ""})
	if !errors.Is(err, ErrTagNameRequired) {
		t.Fatalf("expected ErrTagNameRequired, got %v", err)
	}
}

func TestTagService_CreateTag_Success(t *testing.T) {
	created := false
	repo := &mockTagRepo{
		createFn: func(ctx context.Context, tag *Tag) error {
			created = true
			return nil
		},
	}
	svc := NewTagService(repo)
	if _, err := svc.CreateTag(context.Background(), &CreateTagRequest{Name: "go"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected Create to be called")
	}
}

func TestTagService_FindOrCreateByName_EmptyName(t *testing.T) {
	repo := &mockTagRepo{
		findOrCreateByNameFn: func(ctx context.Context, name string) (*Tag, error) {
			t.Fatal("repository should not be reached for an empty name")
			return nil, nil
		},
	}
	svc := NewTagService(repo)
	_, err := svc.FindOrCreateByName(context.Background(), "")
	if !errors.Is(err, ErrTagNameEmpty) {
		t.Fatalf("expected ErrTagNameEmpty, got %v", err)
	}
}

// The find-then-insert logic itself now lives in tagRepository, where the INSERT
// can carry ON CONFLICT; the service is only responsible for rejecting an empty
// name and passing everything else straight through.
func TestTagService_FindOrCreateByName_Delegates(t *testing.T) {
	existing := &Tag{ID: 1, Name: "go"}
	var gotName string
	repo := &mockTagRepo{
		findOrCreateByNameFn: func(ctx context.Context, name string) (*Tag, error) {
			gotName = name
			return existing, nil
		},
		findByNameFn: func(ctx context.Context, name string) (*Tag, error) {
			t.Fatal("FindByName should not be called directly by the service")
			return nil, nil
		},
		createFn: func(ctx context.Context, tag *Tag) error {
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
		findOrCreateByNameFn: func(ctx context.Context, name string) (*Tag, error) {
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
		findByIDFn: func(ctx context.Context, id uint) (*Tag, error) {
			return nil, ErrTagNotFound
		},
		deleteFn: func(ctx context.Context, id uint) error {
			t.Fatal("Delete should not be called if FindByID fails")
			return nil
		},
	}
	svc := NewTagService(repo)
	if err := svc.DeleteTag(context.Background(), 1); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagService_DeleteTag_Success(t *testing.T) {
	deleted := false
	repo := &mockTagRepo{
		findByIDFn: func(ctx context.Context, id uint) (*Tag, error) {
			return &Tag{ID: id, Name: "go"}, nil
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
		findAllFn: func(ctx context.Context) ([]Tag, error) {
			return []Tag{{ID: 1, Name: "go"}, {ID: 2, Name: "rust"}}, nil
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

func TestTagService_UpdateTag_NotFoundSkipsReload(t *testing.T) {
	repo := &mockTagRepo{
		updateFn: func(ctx context.Context, tag *Tag) error {
			return ErrTagNotFound
		},
		findByIDFn: func(ctx context.Context, id uint) (*Tag, error) {
			t.Fatal("FindByID should not be called when Update fails")
			return nil, nil
		},
	}
	svc := NewTagService(repo)
	if _, err := svc.UpdateTag(context.Background(), 99, &UpdateTagRequest{Name: "go"}); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagService_UpdateTag_ReturnsPersistedRow(t *testing.T) {
	created := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	var updated *Tag
	repo := &mockTagRepo{
		updateFn: func(ctx context.Context, tag *Tag) error {
			updated = tag
			return nil
		},
		findByIDFn: func(ctx context.Context, id uint) (*Tag, error) {
			return &Tag{ID: id, Name: "rust", CreatedAt: &created}, nil
		},
	}
	svc := NewTagService(repo)
	tag, err := svc.UpdateTag(context.Background(), 5, &UpdateTagRequest{Name: "rust"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil || updated.ID != 5 || updated.Name != "rust" {
		t.Fatalf("unexpected update stub %+v", updated)
	}
	if tag.CreatedAt == nil || !tag.CreatedAt.Equal(created) {
		t.Fatalf("expected persisted created_at, got %+v", tag.CreatedAt)
	}
}
