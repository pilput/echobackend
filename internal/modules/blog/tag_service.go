package blog

import (
	"context"
	"time"
)

const (
	trendingTagsLimit = 5
	trendingTagsTTL   = 30 * time.Minute
)

type tagCache interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

type tagService struct {
	tagRepo TagRepository
	cache   tagCache
}

type TagService interface {
	CreateTag(ctx context.Context, req *CreateTagRequest) (*Tag, error)
	GetTags(ctx context.Context) ([]Tag, error)
	GetTagByID(ctx context.Context, id uint) (*Tag, error)
	GetTagByName(ctx context.Context, name string) (*Tag, error)
	GetTrendingTags(ctx context.Context) ([]*TrendingTagResponse, error)
	GetTagsForSitemap(ctx context.Context, limit int) ([]*SitemapTag, error)
	FindOrCreateByName(ctx context.Context, name string) (*Tag, error)
	UpdateTag(ctx context.Context, id uint, req *UpdateTagRequest) (*Tag, error)
	DeleteTag(ctx context.Context, id uint) error
}

func NewTagService(tagRepo TagRepository, cache ...tagCache) TagService {
	var c tagCache
	if len(cache) > 0 {
		c = cache[0]
	}
	return &tagService{tagRepo: tagRepo, cache: c}
}

func (s *tagService) CreateTag(ctx context.Context, req *CreateTagRequest) (*Tag, error) {
	if req.Name == "" {
		return nil, ErrTagNameRequired
	}
	tag := &Tag{Name: req.Name}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *tagService) GetTags(ctx context.Context) ([]Tag, error) {
	return s.tagRepo.FindAll(ctx)
}

func (s *tagService) GetTagByID(ctx context.Context, id uint) (*Tag, error) {
	return s.tagRepo.FindByID(ctx, id)
}

func (s *tagService) UpdateTag(ctx context.Context, id uint, req *UpdateTagRequest) (*Tag, error) {
	if req.Name == "" {
		return nil, ErrTagNameRequired
	}
	if err := s.tagRepo.Update(ctx, &Tag{ID: id, Name: req.Name}); err != nil {
		return nil, err
	}
	// Return the persisted row rather than the update stub.
	return s.tagRepo.FindByID(ctx, id)
}

func (s *tagService) GetTagByName(ctx context.Context, name string) (*Tag, error) {
	return s.tagRepo.FindByName(ctx, name)
}

func (s *tagService) GetTrendingTags(ctx context.Context) ([]*TrendingTagResponse, error) {
	cacheKey := ""
	if s.cache != nil {
		cacheKey = s.cache.BuildKey("tags", "trending")
		var cachedTags []*TrendingTagResponse
		found, err := s.cache.GetJSON(ctx, cacheKey, &cachedTags)
		if err == nil && found {
			return cachedTags, nil
		}
	}

	tags, err := s.tagRepo.GetTrendingTags(ctx, trendingTagsLimit)
	if err != nil {
		return nil, err
	}

	if cacheKey != "" {
		_ = s.cache.SetJSONWithTTL(ctx, cacheKey, tags, trendingTagsTTL)
	}

	return tags, nil
}

func (s *tagService) GetTagsForSitemap(ctx context.Context, limit int) ([]*SitemapTag, error) {
	return s.tagRepo.GetTagsForSitemap(ctx, limit)
}

func (s *tagService) FindOrCreateByName(ctx context.Context, name string) (*Tag, error) {
	if name == "" {
		return nil, ErrTagNameEmpty
	}

	// The find-then-create sequence lives in the repository: only there can the
	// insert carry ON CONFLICT, which is what makes two concurrent requests for
	// the same new tag safe.
	return s.tagRepo.FindOrCreateByName(ctx, name)
}

func (s *tagService) DeleteTag(ctx context.Context, id uint) error {
	_, err := s.tagRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return s.tagRepo.Delete(ctx, id)
}
