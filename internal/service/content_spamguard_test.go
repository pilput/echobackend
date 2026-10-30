package service

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
)

// Guild repo mocks panic on any unstubbed call and comment mocks would report
// a missing post, so these tests also prove the guard runs before any
// database access.

func TestCommentService_RejectsSpam(t *testing.T) {
	svc := NewCommentService(&mockCommentRepo{}, &mockPostRepo{}, nil)

	req := &dto.CreateCommentRequest{Text: "Mampir ke situs g4c0r, link di bit.ly/xyz"}
	if _, err := svc.CreateComment(context.Background(), validPostID, req, validUserID); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("CreateComment err = %v, want ErrContentSpamRejected", err)
	}
	if _, err := svc.UpdateComment(context.Background(), validPostID, "comment-id", req.Text, validUserID); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("UpdateComment err = %v, want ErrContentSpamRejected", err)
	}
}

func TestGuildService_RejectsSpam(t *testing.T) {
	svc := NewGuildService(&mockGuildRepo{})

	description := "Komunitas slot gacor, daftar di https://slot88.example"
	createReq := &dto.CreateGuildRequest{Name: "Info Harian", Description: &description}
	if _, err := svc.CreateGuild(context.Background(), validUserID, createReq); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("CreateGuild err = %v, want ErrContentSpamRejected", err)
	}

	name := "T O G E L Club"
	updateReq := &dto.UpdateGuildRequest{Name: &name}
	if _, err := svc.UpdateGuild(context.Background(), "info-harian", validUserID, updateReq); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("UpdateGuild err = %v, want ErrContentSpamRejected", err)
	}
}
