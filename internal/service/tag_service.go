package service

import (
	"context"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
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
	tagRepo repository.TagRepository
	cache   tagCache
}

type TagService interface {
	CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*model.Tag, error)
	GetTags(ctx context.Context) ([]model.Tag, error)
	GetTagByID(ctx context.Context, id uint) (*model.Tag, error)
	GetTagByName(ctx context.Context, name string) (*model.Tag, error)
	GetTrendingTags(ctx context.Context) ([]*dto.TrendingTagResponse, error)
	GetTagsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapTag, error)
	FindOrCreateByName(ctx context.Context, name string) (*model.Tag, error)
	UpdateTag(ctx context.Context, id uint, req *dto.UpdateTagRequest) (*model.Tag, error)
	DeleteTag(ctx context.Context, id uint) error
}

func NewTagService(tagRepo repository.TagRepository, cache ...tagCache) TagService {
	var c tagCache
	if len(cache) > 0 {
		c = cache[0]
	}
	return &tagService{tagRepo: tagRepo, cache: c}
}

func (s *tagService) CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*model.Tag, error) {
	if req.Name == "" {
		return nil, apperrors.ErrTagNameRequired
	}
	tag := &model.Tag{Name: req.Name}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *tagService) GetTags(ctx context.Context) ([]model.Tag, error) {
	return s.tagRepo.FindAll(ctx)
}

func (s *tagService) GetTagByID(ctx context.Context, id uint) (*model.Tag, error) {
	return s.tagRepo.FindByID(ctx, id)
}

func (s *tagService) UpdateTag(ctx context.Context, id uint, req *dto.UpdateTagRequest) (*model.Tag, error) {
	if req.Name == "" {
		return nil, apperrors.ErrTagNameRequired
	}
	tag := &model.Tag{ID: id, Name: req.Name}
	if err := s.tagRepo.Update(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *tagService) GetTagByName(ctx context.Context, name string) (*model.Tag, error) {
	return s.tagRepo.FindByName(ctx, name)
}

func (s *tagService) GetTrendingTags(ctx context.Context) ([]*dto.TrendingTagResponse, error) {
	cacheKey := ""
	if s.cache != nil {
		cacheKey = s.cache.BuildKey("tags", "trending")
		var cachedTags []*dto.TrendingTagResponse
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

func (s *tagService) GetTagsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapTag, error) {
	return s.tagRepo.GetTagsForSitemap(ctx, limit)
}

func (s *tagService) FindOrCreateByName(ctx context.Context, name string) (*model.Tag, error) {
	if name == "" {
		return nil, apperrors.ErrTagNameEmpty
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
