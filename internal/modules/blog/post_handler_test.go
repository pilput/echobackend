package blog

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

var _ PostService = (*mockPostService)(nil)

type mockPostService struct {
	getPostsForSitemapFn func(ctx context.Context, limit int) ([]*SitemapPost, error)
	getPostsFilteredFn   func(ctx context.Context, filter *PostQueryFilter) ([]*PostResponse, int64, error)
}

func (m *mockPostService) GetPosts(ctx context.Context, limit int, offset int) ([]*PostResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) GetPostsFiltered(ctx context.Context, filter *PostQueryFilter) ([]*PostResponse, int64, error) {
	if m.getPostsFilteredFn != nil {
		return m.getPostsFilteredFn(ctx, filter)
	}
	return nil, 0, nil
}
func (m *mockPostService) GetPostsByUsername(ctx context.Context, username string, offset int, limit int) ([]*PostResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) GetPostsRandom(ctx context.Context, limit int) ([]*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) GetPostsTrending(ctx context.Context, limit int) ([]*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) GetPostByID(ctx context.Context, id string) (*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int, search string, published, hidden *bool) ([]*PostResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) GetPostsByTag(ctx context.Context, tag string, limit int, offset int, sortBy string, sortOrder string) ([]*PostResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*PostResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) DeletePostByID(ctx context.Context, id string) error {
	return nil
}
func (m *mockPostService) DeleteMyPost(ctx context.Context, id string, userID string) error {
	return nil
}
func (m *mockPostService) UploadImagePosts(ctx context.Context, file *multipart.FileHeader) (string, error) {
	return "", nil
}
func (m *mockPostService) CreatePost(ctx context.Context, req *CreatePostRequest, creatorID string) (*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) UpdatePost(ctx context.Context, id string, req *UpdatePostRequest) (*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) UpdateMyPost(ctx context.Context, id string, userID string, req *UpdatePostRequest) (*PostResponse, error) {
	return nil, nil
}
func (m *mockPostService) IsAuthor(ctx context.Context, id string, userid string) error {
	return nil
}
func (m *mockPostService) GetPostsForSitemap(ctx context.Context, limit int) ([]*SitemapPost, error) {
	if m.getPostsForSitemapFn != nil {
		return m.getPostsForSitemapFn(ctx, limit)
	}
	return nil, nil
}

func TestGetPostsForSitemap_Limits(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedLimit int
	}{
		{
			name:          "Default limit is 50000",
			query:         "",
			expectedLimit: 50000,
		},
		{
			name:          "Custom limit within bounds",
			query:         "limit=250",
			expectedLimit: 250,
		},
		{
			name:          "Custom limit exceeding max is capped at 50000",
			query:         "limit=100000",
			expectedLimit: 50000,
		},
		{
			name:          "Invalid limit falls back to default 50000",
			query:         "limit=-5",
			expectedLimit: 50000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedLimit int
			mockSvc := &mockPostService{
				getPostsForSitemapFn: func(ctx context.Context, limit int) ([]*SitemapPost, error) {
					capturedLimit = limit
					return []*SitemapPost{}, nil
				},
			}

			h := &PostHandler{
				postService: mockSvc,
			}

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/posts/sitemap?"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.GetPostsForSitemap(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}

			if capturedLimit != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, capturedLimit)
			}
		})
	}
}
