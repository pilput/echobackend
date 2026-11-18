package blog

import (
	"bytes"
	"context"
	"echobackend/internal/platform/storage"
	"io"
	"mime/multipart"
	"strconv"
	"strings"

	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/antispam"
)

type CacheStore interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSON(ctx context.Context, key string, value any) error
}

type PostService interface {
	GetPosts(ctx context.Context, limit int, offset int) ([]*PostResponse, int64, error)
	GetPostsFiltered(ctx context.Context, filter *PostQueryFilter) ([]*PostResponse, int64, error)
	GetPostsByUsername(ctx context.Context, username string, offset int, limit int) ([]*PostResponse, int64, error)
	GetPostsRandom(ctx context.Context, limit int) ([]*PostResponse, error)
	GetPostsTrending(ctx context.Context, limit int) ([]*PostResponse, error)
	GetPostByID(ctx context.Context, id string) (*PostResponse, error)
	GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*PostResponse, error)
	GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int, search string, published, hidden *bool) ([]*PostResponse, int64, error)
	GetPostsByTag(ctx context.Context, tag string, limit int, offset int, sortBy string, sortOrder string) ([]*PostResponse, int64, error)
	GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*PostResponse, int64, error)
	DeletePostByID(ctx context.Context, id string) error
	DeleteMyPost(ctx context.Context, id string, userID string) error
	UploadImagePosts(ctx context.Context, file *multipart.FileHeader) (string, error)
	CreatePost(ctx context.Context, req *CreatePostRequest, creatorID string) (*PostResponse, error)
	UpdatePost(ctx context.Context, id string, req *UpdatePostRequest) (*PostResponse, error)
	UpdateMyPost(ctx context.Context, id string, userID string, req *UpdatePostRequest) (*PostResponse, error)
	IsAuthor(ctx context.Context, id string, userid string) error
	GetPostsForSitemap(ctx context.Context, limit int) ([]*SitemapPost, error)
}

type postService struct {
	postRepo   PostRepository
	tagService TagService
	s3storage  storage.FileUploader
	cache      CacheStore
}

type trendingPostsCacheEntry struct {
	Posts []*PostResponse `json:"posts"`
}

const maxPostImageSize = 1 * 1024 * 1024
const imageUploadPrefix = "posts/images"

func NewPostService(postRepo PostRepository, tagService TagService, storageclient storage.FileUploader, redisCache CacheStore) PostService {
	return &postService{postRepo: postRepo, tagService: tagService, s3storage: storageclient, cache: redisCache}
}

func (s *postService) IsAuthor(ctx context.Context, id string, userid string) error {
	post, err := s.postRepo.GetPostByID(ctx, id)
	if err != nil {
		return err
	}
	if post.CreatedBy == nil || *post.CreatedBy != userid {
		return apperrors.ErrNotAuthor
	}
	return nil
}

func (s *postService) GetPostsByUsername(ctx context.Context, username string, offset int, limit int) ([]*PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if username == "" {
		return []*PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostByUsername(ctx, username, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) CreatePost(ctx context.Context, req *CreatePostRequest, creatorID string) (*PostResponse, error) {
	// Proactive spam guard: marketing spam is rejected with a 422 that tells
	// the author the rule, instead of polluting feeds until someone reports it.
	if reason := antispam.PostRejectReason(req.Title, req.Body, req.Tags); reason != "" {
		return nil, apperrors.NewSpamRejection(ErrPostSpamRejected, reason)
	}

	var tags []Tag
	if len(req.Tags) > 0 {
		for _, tagName := range normalizeTagNames(req.Tags) {
			tag, err := s.findOrCreateTagByName(ctx, tagName)
			if err != nil {
				return nil, err
			}
			tags = append(tags, *tag)
		}
	}

	post := &Post{
		Title:     &req.Title,
		Slug:      &req.Slug,
		Body:      &req.Body,
		CreatedBy: &creatorID,
		PhotoURL:  &req.PhotoURL,
		Published: &req.Published,
	}

	created, err := s.postRepo.CreatePostWithTags(ctx, post, tags)
	if err != nil {
		return nil, err
	}

	return PostToResponse(created), nil
}

func (s *postService) GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*PostResponse, error) {
	post, err := s.postRepo.GetPostBySlugAndUsername(ctx, slug, username)
	if err != nil {
		return nil, err
	}

	return PostToResponse(post), nil
}

func (s *postService) DeletePostByID(ctx context.Context, id string) error {
	return s.postRepo.DeletePostByID(ctx, id)
}

func (s *postService) DeleteMyPost(ctx context.Context, id string, userID string) error {
	return s.postRepo.DeletePostByOwner(ctx, id, userID)
}

func (s *postService) UpdatePost(ctx context.Context, id string, req *UpdatePostRequest) (*PostResponse, error) {
	// Same guard as creation; only the submitted fields are checked (empty
	// strings pass through without triggering any rule).
	if reason := antispam.PostRejectReason(req.Title, req.Body, req.Tags); reason != "" {
		return nil, apperrors.NewSpamRejection(ErrPostSpamRejected, reason)
	}

	updates := make(map[string]any)
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Body != "" {
		updates["body"] = req.Body
	}
	if req.Slug != "" {
		updates["slug"] = req.Slug
	}
	if req.PhotoURL != "" {
		updates["photo_url"] = req.PhotoURL
	}
	if req.Published != nil {
		updates["published"] = *req.Published
	}

	var tags []Tag
	if req.Tags != nil {
		tags = make([]Tag, 0, len(req.Tags))
		for _, tagName := range normalizeTagNames(req.Tags) {
			tag, err := s.findOrCreateTagByName(ctx, tagName)
			if err != nil {
				return nil, err
			}
			tags = append(tags, *tag)
		}
	}

	if len(updates) == 0 && req.Tags == nil {
		post, err := s.postRepo.GetPostByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return PostToResponse(post), nil
	}

	var updatedPost *Post
	var err error
	if req.Tags != nil {
		updatedPost, err = s.postRepo.UpdatePostWithTags(ctx, id, updates, tags)
	} else {
		updatedPost, err = s.postRepo.UpdatePost(ctx, id, updates)
	}
	if err != nil {
		return nil, err
	}
	return PostToResponse(updatedPost), nil
}

// UpdateMyPost updates a post scoped to its owner; the mutation itself is
// owner-checked at the repository level to avoid TOCTOU ownership races.
func (s *postService) UpdateMyPost(ctx context.Context, id string, userID string, req *UpdatePostRequest) (*PostResponse, error) {
	if reason := antispam.PostRejectReason(req.Title, req.Body, req.Tags); reason != "" {
		return nil, apperrors.NewSpamRejection(ErrPostSpamRejected, reason)
	}

	updates := make(map[string]any)
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Body != "" {
		updates["body"] = req.Body
	}
	if req.Slug != "" {
		updates["slug"] = req.Slug
	}
	if req.PhotoURL != "" {
		updates["photo_url"] = req.PhotoURL
	}
	if req.Published != nil {
		updates["published"] = *req.Published
	}

	var tags []Tag
	if req.Tags != nil {
		tags = make([]Tag, 0, len(req.Tags))
		for _, tagName := range normalizeTagNames(req.Tags) {
			tag, err := s.findOrCreateTagByName(ctx, tagName)
			if err != nil {
				return nil, err
			}
			tags = append(tags, *tag)
		}
	}

	if len(updates) == 0 && req.Tags == nil {
		post, err := s.postRepo.GetPostByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if post.CreatedBy == nil || *post.CreatedBy != userID {
			return nil, apperrors.ErrNotAuthor
		}
		return PostToResponse(post), nil
	}

	var updatedPost *Post
	var err error
	if req.Tags != nil {
		updatedPost, err = s.postRepo.UpdatePostWithTagsByOwner(ctx, id, userID, updates, tags)
	} else {
		updatedPost, err = s.postRepo.UpdatePostByOwner(ctx, id, userID, updates)
	}
	if err != nil {
		return nil, err
	}
	return PostToResponse(updatedPost), nil
}

func (s *postService) GetPostByID(ctx context.Context, id string) (*PostResponse, error) {
	post, err := s.postRepo.GetPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return PostToResponse(post), nil
}

func (s *postService) GetPosts(ctx context.Context, limit int, offset int) ([]*PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}

	posts, total, err := s.postRepo.GetPosts(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postResponse := PostToResponse(post)
		postsResponse = append(postsResponse, postResponse)
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsRandom(ctx context.Context, limit int) ([]*PostResponse, error) {
	if limit < 0 {
		limit = 0
	}

	cacheKey := ""
	if s.cache != nil {
		cacheKey = s.cache.BuildKey("posts", "random", strconv.Itoa(limit))
		var cachedPosts []*PostResponse
		found, err := s.cache.GetJSON(ctx, cacheKey, &cachedPosts)
		if err == nil && found {
			return cachedPosts, nil
		}
	}

	posts, err := s.postRepo.GetPostsRandom(ctx, limit)
	if err != nil {
		return nil, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postResponse := PostToResponse(post)
		postsResponse = append(postsResponse, postResponse)
	}

	if cacheKey != "" {
		_ = s.cache.SetJSON(ctx, cacheKey, postsResponse)
	}

	return postsResponse, nil
}

func (s *postService) GetPostsTrending(ctx context.Context, limit int) ([]*PostResponse, error) {
	if limit < 0 {
		limit = 10
	}
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	cacheKey := ""
	if s.cache != nil {
		cacheKey = s.cache.BuildKey(
			"posts",
			"trending",
			strconv.Itoa(limit),
		)
		var cachedTrending trendingPostsCacheEntry
		found, err := s.cache.GetJSON(ctx, cacheKey, &cachedTrending)
		if err == nil && found {
			return cachedTrending.Posts, nil
		}
	}

	posts, err := s.postRepo.GetPostsTrending(ctx, limit)
	if err != nil {
		return nil, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, PostToResponse(post))
	}

	if cacheKey != "" {
		_ = s.cache.SetJSON(ctx, cacheKey, trendingPostsCacheEntry{
			Posts: postsResponse,
		})
	}

	return postsResponse, nil
}

func (s *postService) GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int, search string, published, hidden *bool) ([]*PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if createdBy == "" {
		return []*PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostsByCreatedBy(ctx, createdBy, offset, limit, search, published, hidden)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsByTag(ctx context.Context, tag string, limit int, offset int, sortBy string, sortOrder string) ([]*PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if tag == "" {
		return []*PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostsByTag(ctx, tag, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsFiltered(ctx context.Context, filter *PostQueryFilter) ([]*PostResponse, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	posts, total, err := s.postRepo.GetPostsFiltered(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postResponse := PostToResponse(post)
		postsResponse = append(postsResponse, postResponse)
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*PostResponse, int64, error) {
	if limit < 0 {
		limit = 10
	}
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if userID == "" {
		return []*PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostsForYou(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) UploadImagePosts(ctx context.Context, file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", apperrors.ErrFileNil
	}
	if file.Size > maxPostImageSize {
		return "", ErrFileTooLarge
	}
	if s.s3storage == nil {
		return "", apperrors.ErrStorageUnavailable
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()

	data, err := io.ReadAll(io.LimitReader(src, maxPostImageSize+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxPostImageSize {
		return "", ErrFileTooLarge
	}

	contentType, ext, ok := storage.DetectAllowedImage(data)
	if !ok {
		return "", apperrors.ErrInvalidFileType
	}

	objectKey, err := storage.RandomImageObjectKey(imageUploadPrefix, ext)
	if err != nil {
		return "", err
	}

	if err := s.s3storage.Save(ctx, objectKey, bytes.NewReader(data), contentType); err != nil {
		return "", err
	}

	return objectKey, nil
}

func (s *postService) findOrCreateTagByName(ctx context.Context, tagName string) (*Tag, error) {
	return s.tagService.FindOrCreateByName(ctx, tagName)
}

func (s *postService) GetPostsForSitemap(ctx context.Context, limit int) ([]*SitemapPost, error) {
	if limit < 0 {
		limit = 0
	}
	return s.postRepo.GetPostsForSitemap(ctx, limit)
}

// normalizeTagNames trims each tag name, drops names that are empty after
// trimming and removes exact (case-sensitive) duplicates, keeping the order in
// which names first appear. Case is preserved. A non-nil input always yields a
// non-nil slice so an explicit empty tag list still clears a post's tags.
func normalizeTagNames(names []string) []string {
	if names == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized
}
