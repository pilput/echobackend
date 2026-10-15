package service

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

// The repo mock panics on any unstubbed call, so these tests also prove the
// spam guard runs before any tag resolution or database write.
func TestPostService_CreatePost_RejectsSpam(t *testing.T) {
	svc := NewPostService(&mockPostRepo{}, &mockTagService{}, nil, nil)

	spam := []dto.CreatePostRequest{
		{
			Title: "Situs gacor terpercaya",
			Slug:  "situs-gacor-terpercaya",
			Body:  "Daftar sekarang dan klaim bonus maxwin setiap hari.",
		},
		{
			Title: "Promo murah",
			Slug:  "promo-murah-meriah",
			Body:  "https://a.example https://b.example https://c.example https://d.example https://e.example https://f.example",
		},
	}
	for i, req := range spam {
		if _, err := svc.CreatePost(context.Background(), &req, validUserID); !errors.Is(err, apperrors.ErrPostSpamRejected) {
			t.Errorf("case %d: err = %v, want ErrPostSpamRejected", i, err)
		}
	}
}

func TestPostService_CreatePost_AllowsCleanPost(t *testing.T) {
	repo := &mockPostRepo{
		createPostWithTagsFn: func(ctx context.Context, post *model.Post, tags []model.Tag) (*model.Post, error) {
			return post, nil
		},
	}
	tags := &mockTagService{
		findOrCreateByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
			return &model.Tag{Name: name}, nil
		},
	}
	svc := NewPostService(repo, tags, nil, nil)

	req := &dto.CreatePostRequest{
		Title: "How I built my first API",
		Slug:  "how-i-built-my-first-api",
		Body:  "A genuine write-up with one link to https://example.com/docs for reference.",
		Tags:  []string{"golang"},
	}
	if _, err := svc.CreatePost(context.Background(), req, validUserID); err != nil {
		t.Fatalf("CreatePost returned error: %v", err)
	}
}

func TestPostService_UpdatePost_RejectsSpam(t *testing.T) {
	svc := NewPostService(&mockPostRepo{}, &mockTagService{}, nil, nil)

	req := &dto.UpdatePostRequest{Body: "Info rtp live 98% malam ini, join sekarang"}
	if _, err := svc.UpdatePost(context.Background(), validPostID, req); !errors.Is(err, apperrors.ErrPostSpamRejected) {
		t.Fatalf("err = %v, want ErrPostSpamRejected", err)
	}
	if _, err := svc.UpdateMyPost(context.Background(), validPostID, validUserID, req); !errors.Is(err, apperrors.ErrPostSpamRejected) {
		t.Fatalf("err = %v, want ErrPostSpamRejected", err)
	}
}
