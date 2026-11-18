package blog

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
)

const authorUserID = "018f4d39-3a4f-7c4f-9b2a-2cf6f8c4f4d4"

func postByAuthor(authorID string) *mockPostRepo {
	return &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return &Post{ID: id, CreatedBy: &authorID}, nil
		},
	}
}

func TestPostReportService_ReportPost_Success(t *testing.T) {
	var got *PostReport
	repo := &mockPostReportRepo{
		createFn: func(ctx context.Context, r *PostReport) error {
			got = r
			return nil
		},
	}
	svc := NewPostReportService(repo, postByAuthor(authorUserID))

	empty := ""
	err := svc.ReportPost(context.Background(), validPostID, validUserID, CreatePostReportRequest{Reason: "spam", Details: &empty})
	if err != nil {
		t.Fatalf("ReportPost returned error: %v", err)
	}
	if got == nil || got.PostID != validPostID || got.ReporterID != validUserID || got.Reason != "spam" {
		t.Fatalf("unexpected report: %+v", got)
	}
	if got.Status != PostReportStatusPending {
		t.Errorf("status = %q, want pending", got.Status)
	}
	if got.Details != nil {
		t.Errorf("empty details should be stored as nil, got %q", *got.Details)
	}
}

func TestPostReportService_ReportPost_OwnPost(t *testing.T) {
	svc := NewPostReportService(&mockPostReportRepo{}, postByAuthor(validUserID))
	err := svc.ReportPost(context.Background(), validPostID, validUserID, CreatePostReportRequest{Reason: "spam"})
	if !errors.Is(err, ErrCannotReportOwnPost) {
		t.Fatalf("err = %v, want ErrCannotReportOwnPost", err)
	}
}

func TestPostReportService_ReportPost_PostNotFound(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return nil, ErrPostNotFound
		},
	}
	svc := NewPostReportService(&mockPostReportRepo{}, postRepo)
	err := svc.ReportPost(context.Background(), validPostID, validUserID, CreatePostReportRequest{Reason: "spam"})
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestPostReportService_ReportPost_InvalidIDs(t *testing.T) {
	svc := NewPostReportService(&mockPostReportRepo{}, &mockPostRepo{})
	req := CreatePostReportRequest{Reason: "spam"}
	if err := svc.ReportPost(context.Background(), "bad", validUserID, req); !errors.Is(err, ErrInvalidPostID) {
		t.Errorf("bad post id: err = %v", err)
	}
	if err := svc.ReportPost(context.Background(), validPostID, "bad", req); !errors.Is(err, apperrors.ErrInvalidUserID) {
		t.Errorf("bad user id: err = %v", err)
	}
}

func TestPostReportService_ListReportedPosts_DefaultsToPending(t *testing.T) {
	var gotStatus string
	repo := &mockPostReportRepo{
		listGroupedByPostFn: func(ctx context.Context, status string, limit, offset int) ([]PostReportGroup, int64, error) {
			gotStatus = status
			return nil, 0, nil
		},
	}
	svc := NewPostReportService(repo, &mockPostRepo{})
	if _, _, err := svc.ListReportedPosts(context.Background(), "", 20, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotStatus != PostReportStatusPending {
		t.Errorf("status = %q, want pending", gotStatus)
	}
}

func TestPostReportService_ListReportedPosts_InvalidStatus(t *testing.T) {
	svc := NewPostReportService(&mockPostReportRepo{}, &mockPostRepo{})
	_, _, err := svc.ListReportedPosts(context.Background(), "bogus", 20, 0)
	if !errors.Is(err, ErrInvalidReportStatus) {
		t.Fatalf("err = %v, want ErrInvalidReportStatus", err)
	}
}

func TestPostReportService_ReportPost_HiddenPost(t *testing.T) {
	now := time.Now()
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			author := authorUserID
			return &Post{ID: id, CreatedBy: &author, HiddenAt: &now}, nil
		},
	}
	svc := NewPostReportService(&mockPostReportRepo{}, postRepo)
	err := svc.ReportPost(context.Background(), validPostID, validUserID, CreatePostReportRequest{Reason: "spam"})
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound for a hidden post", err)
	}
}

func TestPostReportService_ModeratePost_PassesArguments(t *testing.T) {
	var gotAction, gotAdmin string
	var gotNote *string
	repo := &mockPostReportRepo{
		moderateFn: func(ctx context.Context, postID, adminID, action string, note *string) error {
			gotAction, gotAdmin, gotNote = action, adminID, note
			return nil
		},
	}
	svc := NewPostReportService(repo, &mockPostRepo{})
	note := "confirmed spam"
	err := svc.ModeratePost(context.Background(), validPostID, validUserID, ModeratePostRequest{Action: ModerationActionHide, Note: &note})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAction != ModerationActionHide || gotAdmin != validUserID || gotNote == nil || *gotNote != note {
		t.Errorf("action=%q admin=%q note=%v", gotAction, gotAdmin, gotNote)
	}
}

func TestPostReportService_ModeratePost_PropagatesStateErrors(t *testing.T) {
	for _, want := range []error{ErrNoPendingReports, ErrPostAlreadyHidden, ErrPostNotHidden, ErrPostNotFound} {
		repo := &mockPostReportRepo{
			moderateFn: func(ctx context.Context, postID, adminID, action string, note *string) error { return want },
		}
		svc := NewPostReportService(repo, &mockPostRepo{})
		err := svc.ModeratePost(context.Background(), validPostID, validUserID, ModeratePostRequest{Action: ModerationActionDismiss})
		if !errors.Is(err, want) {
			t.Errorf("err = %v, want %v", err, want)
		}
	}
}

func TestPostReportService_ModeratePost_InvalidIDs(t *testing.T) {
	svc := NewPostReportService(&mockPostReportRepo{}, &mockPostRepo{})
	req := ModeratePostRequest{Action: ModerationActionHide}
	if err := svc.ModeratePost(context.Background(), "bad", validUserID, req); !errors.Is(err, ErrInvalidPostID) {
		t.Errorf("bad post id: err = %v", err)
	}
	if err := svc.ModeratePost(context.Background(), validPostID, "bad", req); !errors.Is(err, apperrors.ErrInvalidUserID) {
		t.Errorf("bad admin id: err = %v", err)
	}
}

func TestPostReportService_ListModerationActions_InvalidID(t *testing.T) {
	svc := NewPostReportService(&mockPostReportRepo{}, &mockPostRepo{})
	if _, _, err := svc.ListModerationActions(context.Background(), "bad", 20, 0); !errors.Is(err, ErrInvalidPostID) {
		t.Fatalf("err = %v, want ErrInvalidPostID", err)
	}
}

func TestPostLikeService_LikePost_HiddenPost(t *testing.T) {
	now := time.Now()
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*Post, error) {
			return &Post{ID: id, HiddenAt: &now}, nil
		},
	}
	svc := NewPostLikeService(&mockPostLikeRepo{}, postRepo)
	err := svc.LikePost(context.Background(), validPostID, validUserID)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound for a hidden post", err)
	}
}

func TestEnsurePostInteractable(t *testing.T) {
	now := time.Now()
	if err := ensurePostInteractable(&Post{}); err != nil {
		t.Errorf("visible post: %v", err)
	}
	if err := ensurePostInteractable(&Post{HiddenAt: &now}); !errors.Is(err, ErrPostNotFound) {
		t.Errorf("hidden post: %v", err)
	}
	if err := ensurePostInteractable(nil); err != nil {
		t.Errorf("nil post: %v", err)
	}
}
