package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

// ---- Inlined Service Mocks ----------------------------------------------------

type mockTagService struct {
	findOrCreateByNameFn func(ctx context.Context, name string) (*model.Tag, error)
}

func (m *mockTagService) CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*model.Tag, error) {
	return nil, nil
}
func (m *mockTagService) GetTags(ctx context.Context) ([]model.Tag, error) { return nil, nil }
func (m *mockTagService) GetTagByID(ctx context.Context, id uint) (*model.Tag, error) {
	return nil, nil
}
func (m *mockTagService) GetTagByName(ctx context.Context, name string) (*model.Tag, error) {
	return nil, nil
}
func (m *mockTagService) GetTrendingTags(ctx context.Context) ([]*dto.TrendingTagResponse, error) {
	return nil, nil
}
func (m *mockTagService) GetTagsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapTag, error) {
	return nil, nil
}
func (m *mockTagService) FindOrCreateByName(ctx context.Context, name string) (*model.Tag, error) {
	if m.findOrCreateByNameFn != nil {
		return m.findOrCreateByNameFn(ctx, name)
	}
	return nil, nil
}
func (m *mockTagService) UpdateTag(ctx context.Context, id uint, req *dto.UpdateTagRequest) (*model.Tag, error) {
	return nil, nil
}
func (m *mockTagService) DeleteTag(ctx context.Context, id uint) error { return nil }

type mockCacheStore struct {
	buildKeyFn func(parts ...string) string
	getJSONFn  func(ctx context.Context, key string, dest any) (bool, error)
	setJSONFn  func(ctx context.Context, key string, value any) error
}

func (m *mockCacheStore) BuildKey(parts ...string) string {
	if m.buildKeyFn != nil {
		return m.buildKeyFn(parts...)
	}
	return ""
}
func (m *mockCacheStore) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	if m.getJSONFn != nil {
		return m.getJSONFn(ctx, key, dest)
	}
	return false, nil
}
func (m *mockCacheStore) SetJSON(ctx context.Context, key string, value any) error {
	if m.setJSONFn != nil {
		return m.setJSONFn(ctx, key, value)
	}
	return nil
}

// ---- Test Cases ---------------------------------------------------------------

func TestUploadImagePostsRejectsFilesLargerThanOneMiB(t *testing.T) {
	svc := NewPostService(&mockPostRepo{}, nil, nil, nil)

	_, err := svc.UploadImagePosts(context.Background(), &multipart.FileHeader{
		Filename: "large.jpg",
		Size:     maxPostImageSize + 1,
	})

	if !errors.Is(err, apperrors.ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestIsAuthor(t *testing.T) {
	ctx := context.Background()
	authorID := "author-uuid"

	t.Run("authorized author", func(t *testing.T) {
		repo := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return &model.Post{ID: id, CreatedBy: &authorID}, nil
			},
		}
		svc := NewPostService(repo, nil, nil, nil)
		err := svc.IsAuthor(ctx, "post-id", authorID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("unauthorized mismatch", func(t *testing.T) {
		repo := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				wrongAuthor := "other-uuid"
				return &model.Post{ID: id, CreatedBy: &wrongAuthor}, nil
			},
		}
		svc := NewPostService(repo, nil, nil, nil)
		err := svc.IsAuthor(ctx, "post-id", authorID)
		if !errors.Is(err, apperrors.ErrNotAuthor) {
			t.Fatalf("expected ErrNotAuthor, got %v", err)
		}
	})

	t.Run("post not found error", func(t *testing.T) {
		repo := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return nil, apperrors.ErrPostNotFound
			},
		}
		svc := NewPostService(repo, nil, nil, nil)
		err := svc.IsAuthor(ctx, "post-id", authorID)
		if !errors.Is(err, apperrors.ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestGetPostByID(t *testing.T) {
	ctx := context.Background()
	postID := "post-uuid"
	title := "Post Title"

	t.Run("success", func(t *testing.T) {
		repo := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return &model.Post{ID: id, Title: &title}, nil
			},
		}
		svc := NewPostService(repo, nil, nil, nil)
		resp, err := svc.GetPostByID(ctx, postID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if resp.ID != postID || *resp.Title != title {
			t.Fatalf("mismatched post data: %v", resp)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockPostRepo{
			getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
				return nil, apperrors.ErrPostNotFound
			},
		}
		svc := NewPostService(repo, nil, nil, nil)
		_, err := svc.GetPostByID(ctx, postID)
		if !errors.Is(err, apperrors.ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestCreatePost(t *testing.T) {
	ctx := context.Background()
	creatorID := "creator-uuid"
	req := &dto.CreatePostRequest{
		Title: "New Post",
		Slug:  "new-post",
		Body:  "Body content",
		Tags:  []string{"Tech", "Go"},
	}

	t.Run("success with tags", func(t *testing.T) {
		tagCalls := 0
		mockTagSvc := &mockTagService{
			findOrCreateByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
				tagCalls++
				return &model.Tag{Name: name}, nil
			},
		}

		repo := &mockPostRepo{
			createPostWithTagsFn: func(ctx context.Context, post *model.Post, tags []model.Tag) (*model.Post, error) {
				post.ID = "new-post-uuid"
				post.Tags = tags
				return post, nil
			},
		}

		svc := NewPostService(repo, mockTagSvc, nil, nil)
		resp, err := svc.CreatePost(ctx, req, creatorID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if tagCalls != 2 {
			t.Fatalf("expected 2 calls to findOrCreateByName, got %d", tagCalls)
		}
		if resp.ID != "new-post-uuid" {
			t.Fatalf("expected ID new-post-uuid, got %s", resp.ID)
		}
		if len(resp.Tags) != 2 {
			t.Fatalf("expected 2 tags in response, got %d", len(resp.Tags))
		}
	})
}

func TestUpdatePost(t *testing.T) {
	ctx := context.Background()
	postID := "post-uuid"
	updatedTitle := "Updated Title"

	t.Run("update title without tags", func(t *testing.T) {
		repo := &mockPostRepo{
			updatePostFn: func(ctx context.Context, id string, updates map[string]any) (*model.Post, error) {
				if id != postID {
					t.Fatalf("expected post ID %s, got %s", postID, id)
				}
				if updates["title"] != updatedTitle {
					t.Fatalf("expected title %s, got %v", updatedTitle, updates["title"])
				}
				return &model.Post{ID: id, Title: &updatedTitle}, nil
			},
		}

		svc := NewPostService(repo, nil, nil, nil)
		resp, err := svc.UpdatePost(ctx, postID, &dto.UpdatePostRequest{
			Title: updatedTitle,
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if resp.ID != postID || *resp.Title != updatedTitle {
			t.Fatalf("unexpected response: %v", resp)
		}
	})

	t.Run("update with tags", func(t *testing.T) {
		tagServiceCalled := false
		tagSvc := &mockTagService{
			findOrCreateByNameFn: func(ctx context.Context, name string) (*model.Tag, error) {
				tagServiceCalled = true
				return &model.Tag{ID: 1, Name: name}, nil
			},
		}

		repo := &mockPostRepo{
			updatePostWithTagsFn: func(ctx context.Context, id string, updates map[string]any, tags []model.Tag) (*model.Post, error) {
				if len(tags) != 1 || tags[0].Name != "golang" {
					t.Fatalf("expected tag 'golang', got %v", tags)
				}
				return &model.Post{ID: id, Title: &updatedTitle, Tags: tags}, nil
			},
		}

		svc := NewPostService(repo, tagSvc, nil, nil)
		resp, err := svc.UpdatePost(ctx, postID, &dto.UpdatePostRequest{
			Title: updatedTitle,
			Tags:  []string{"golang"},
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !tagServiceCalled {
			t.Fatalf("expected findOrCreateByName to be called")
		}
		if len(resp.Tags) != 1 || resp.Tags[0].Name != "golang" {
			t.Fatalf("expected tag in response, got %v", resp.Tags)
		}
	})
}

func TestDeletePostByID(t *testing.T) {
	ctx := context.Background()
	postID := "post-uuid"
	t.Run("success", func(t *testing.T) {
		called := false
		repo := &mockPostRepo{
			deletePostByIDFn: func(ctx context.Context, id string) error {
				called = true
				return nil
			},
		}
		svc := NewPostService(repo, nil, nil, nil)
		err := svc.DeletePostByID(ctx, postID)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !called {
			t.Fatalf("expected deletePostByIDFn to be called")
		}
	})
}

func TestGetPostsTrending(t *testing.T) {
	ctx := context.Background()
	limit := 5

	t.Run("cache hit", func(t *testing.T) {
		title := "Trending Title"
		mockCache := &mockCacheStore{
			buildKeyFn: func(parts ...string) string {
				return "posts:trending:5"
			},
			getJSONFn: func(ctx context.Context, key string, dest any) (bool, error) {
				typedDest, ok := dest.(*trendingPostsCacheEntry)
				if ok {
					typedDest.Posts = []*dto.PostResponse{{ID: "post-1", Title: &title}}
					return true, nil
				}
				return false, nil
			},
		}

		// repo is not set up with getPostsTrendingFn, so if it's called, it will panic.
		// A successful test with no panic guarantees a cache hit was resolved.
		svc := NewPostService(&mockPostRepo{}, nil, nil, mockCache)
		resp, err := svc.GetPostsTrending(ctx, limit)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if len(resp) != 1 || *resp[0].Title != title {
			t.Fatalf("unexpected response: %v", resp)
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		title := "Db Title"
		mockCache := &mockCacheStore{
			buildKeyFn: func(parts ...string) string {
				return "posts:trending:5"
			},
			getJSONFn: func(ctx context.Context, key string, dest any) (bool, error) {
				return false, nil
			},
			setJSONFn: func(ctx context.Context, key string, value any) error {
				return nil
			},
		}

		repo := &mockPostRepo{
			getPostsTrendingFn: func(ctx context.Context, lim int) ([]*model.Post, error) {
				return []*model.Post{{ID: "post-1", Title: &title}}, nil
			},
		}

		svc := NewPostService(repo, nil, nil, mockCache)
		resp, err := svc.GetPostsTrending(ctx, limit)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if len(resp) != 1 || *resp[0].Title != title {
			t.Fatalf("unexpected response: %v", resp)
		}
	})
}

type mockFileUploader struct {
	saveFn func(ctx context.Context, path string, file io.Reader, contentType string) error
}

func (m *mockFileUploader) Save(ctx context.Context, path string, file io.Reader, contentType string) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, path, file, contentType)
	}
	return nil
}

func TestUploadImagePosts(t *testing.T) {
	ctx := context.Background()

	t.Run("nil file", func(t *testing.T) {
		svc := NewPostService(&mockPostRepo{}, nil, &mockFileUploader{}, nil)
		_, err := svc.UploadImagePosts(ctx, nil)
		if !errors.Is(err, apperrors.ErrFileNil) {
			t.Fatalf("expected ErrFileNil, got %v", err)
		}
	})

	t.Run("storage nil", func(t *testing.T) {
		svc := NewPostService(&mockPostRepo{}, nil, nil, nil)
		header := &multipart.FileHeader{
			Filename: "test.jpg",
			Size:     100,
		}
		_, err := svc.UploadImagePosts(ctx, header)
		if !errors.Is(err, apperrors.ErrStorageUnavailable) {
			t.Fatalf("expected ErrStorageUnavailable, got %v", err)
		}
	})
}
