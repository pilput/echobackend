package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"echobackend/internal/dto"

	"github.com/labstack/echo/v5"
)

func TestGetPosts_InvalidFilterParams(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		message string
	}{
		{"bad start_date", "start_date=yesterday", "Invalid start_date format, expected YYYY-MM-DD or an RFC 3339 timestamp"},
		{"bad end_date", "end_date=2024-99-01", "Invalid end_date format, expected YYYY-MM-DD or an RFC 3339 timestamp"},
		{"bad created_by", "created_by=not-a-uuid", "Invalid created_by value, expected UUID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := &PostHandler{postService: &mockPostService{
				getPostsFilteredFn: func(ctx context.Context, filter *dto.PostQueryFilter) ([]*dto.PostResponse, int64, error) {
					called = true
					return nil, 0, nil
				},
			}}
			e := echo.New()
			e.GET("/api/posts", h.GetPosts)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/posts?"+tt.query, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			if body["message"] != tt.message {
				t.Errorf("message = %v, want %q", body["message"], tt.message)
			}
			if called {
				t.Error("service must not be called for invalid params")
			}
		})
	}
}

func TestGetPosts_NormalizesFilterParams(t *testing.T) {
	var captured *dto.PostQueryFilter
	h := &PostHandler{postService: &mockPostService{
		getPostsFilteredFn: func(ctx context.Context, filter *dto.PostQueryFilter) ([]*dto.PostResponse, int64, error) {
			captured = filter
			return []*dto.PostResponse{}, 0, nil
		},
	}}
	e := echo.New()
	e.GET("/api/posts", h.GetPosts)
	q := "start_date=2024-01-02&end_date=2024-01-03T10:00:00%2B07:00&created_by=%2001890a5d-ac96-774b-bcce-b302099a8057%20"
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/posts?"+q, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if captured == nil {
		t.Fatal("service not called")
	}
	if captured.StartDate != "2024-01-02T00:00:00Z" || captured.EndDate != "2024-01-03T03:00:00Z" {
		t.Errorf("dates = %q / %q", captured.StartDate, captured.EndDate)
	}
	if captured.CreatedBy != "01890a5d-ac96-774b-bcce-b302099a8057" {
		t.Errorf("created_by = %q", captured.CreatedBy)
	}
}

func TestPostViewHandler_InvalidPostID(t *testing.T) {
	h := &PostViewHandler{}
	e := echo.New()
	e.POST("/api/posts/:id/view", h.RecordView)
	e.GET("/api/posts/:id/views", h.GetPostViews)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/posts/not-a-uuid/view"},
		{http.MethodGet, "/api/posts/not-a-uuid/views"},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s %s: expected 400, got %d", tc.method, tc.path, rec.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["message"] != "Invalid post ID" {
			t.Errorf("%s %s: message = %v", tc.method, tc.path, body["message"])
		}
		if _, has := body["error"]; has {
			t.Errorf("%s %s: unexpected error field", tc.method, tc.path)
		}
	}
}
