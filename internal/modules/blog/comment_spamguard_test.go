package blog

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
)

// Comment mocks would report a missing post, so this test also proves the
// guard runs before any database access.

func TestCommentService_RejectsSpam(t *testing.T) {
	svc := NewCommentService(&mockCommentRepo{}, &mockPostRepo{}, nil)

	req := &CreateCommentRequest{Text: "Mampir ke situs g4c0r, link di bit.ly/xyz"}
	if _, err := svc.CreateComment(context.Background(), validPostID, req, validUserID); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("CreateComment err = %v, want ErrContentSpamRejected", err)
	}
	if _, err := svc.UpdateComment(context.Background(), validPostID, "comment-id", req.Text, validUserID); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("UpdateComment err = %v, want ErrContentSpamRejected", err)
	}
}
