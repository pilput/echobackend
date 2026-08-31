package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
)

type FileUploader interface {
	Save(ctx context.Context, path string, file io.Reader, contentType string) error
}

type CacheStore interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSON(ctx context.Context, key string, value any) error
}

type PostService interface {
	GetPosts(ctx context.Context, limit int, offset int) ([]*dto.PostResponse, int64, error)
	GetPostsFiltered(ctx context.Context, filter *dto.PostQueryFilter) ([]*dto.PostResponse, int64, error)
	GetPostsByUsername(ctx context.Context, username string, offset int, limit int) ([]*dto.PostResponse, int64, error)
	GetPostsRandom(ctx context.Context, limit int) ([]*dto.PostResponse, error)
	GetPostsTrending(ctx context.Context, limit int) ([]*dto.PostResponse, error)
	GetPostByID(ctx context.Context, id string) (*dto.PostResponse, error)
	GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*dto.PostResponse, error)
	GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int) ([]*dto.PostResponse, int64, error)
	GetPostsByTag(ctx context.Context, tag string, limit int, offset int) ([]*dto.PostResponse, int64, error)
	GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*dto.PostResponse, int64, error)
	DeletePostByID(ctx context.Context, id string) error
	DeleteMyPost(ctx context.Context, id string, userID string) error
	UploadImagePosts(ctx context.Context, file *multipart.FileHeader) (string, error)
	CreatePost(ctx context.Context, req *dto.CreatePostRequest, creatorID string) (*dto.PostResponse, error)
	UpdatePost(ctx context.Context, id string, req *dto.UpdatePostRequest) (*dto.PostResponse, error)
	UpdateMyPost(ctx context.Context, id string, userID string, req *dto.UpdatePostRequest) (*dto.PostResponse, error)
	IsAuthor(ctx context.Context, id string, userid string) error
	GetPostsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapPost, error)
}

type postService struct {
	postRepo   repository.PostRepository
	tagService TagService
	s3storage  FileUploader
	cache      CacheStore
}

type trendingPostsCacheEntry struct {
	Posts []*dto.PostResponse `json:"posts"`
}

const maxPostImageSize = 1 * 1024 * 1024
const imageUploadPrefix = "posts/images"

func NewPostService(postRepo repository.PostRepository, tagService TagService, storageclient FileUploader, redisCache CacheStore) PostService {
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

func (s *postService) GetPostsByUsername(ctx context.Context, username string, offset int, limit int) ([]*dto.PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if username == "" {
		return []*dto.PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostByUsername(ctx, username, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, dto.PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) CreatePost(ctx context.Context, req *dto.CreatePostRequest, creatorID string) (*dto.PostResponse, error) {
	var tags []model.Tag
	if len(req.Tags) > 0 {
		for _, tagName := range req.Tags {
			if tagName == "" {
				continue
			}

			tag, err := s.findOrCreateTagByName(ctx, tagName)
			if err != nil {
				return nil, err
			}
			tags = append(tags, *tag)
		}
	}

	post := &model.Post{
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

	return dto.PostToResponse(created), nil
}

func (s *postService) GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*dto.PostResponse, error) {
	post, err := s.postRepo.GetPostBySlugAndUsername(ctx, slug, username)
	if err != nil {
		return nil, err
	}

	return dto.PostToResponse(post), nil
}

func (s *postService) DeletePostByID(ctx context.Context, id string) error {
	return s.postRepo.DeletePostByID(ctx, id)
}

func (s *postService) DeleteMyPost(ctx context.Context, id string, userID string) error {
	return s.postRepo.DeletePostByOwner(ctx, id, userID)
}

func (s *postService) UpdatePost(ctx context.Context, id string, req *dto.UpdatePostRequest) (*dto.PostResponse, error) {
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

	var tags []model.Tag
	if req.Tags != nil {
		tags = make([]model.Tag, 0, len(req.Tags))
		for _, tagName := range req.Tags {
			if tagName == "" {
				continue
			}
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
		return dto.PostToResponse(post), nil
	}

	var updatedPost *model.Post
	var err error
	if req.Tags != nil {
		updatedPost, err = s.postRepo.UpdatePostWithTags(ctx, id, updates, tags)
	} else {
		updatedPost, err = s.postRepo.UpdatePost(ctx, id, updates)
	}
	if err != nil {
		return nil, err
	}
	return dto.PostToResponse(updatedPost), nil
}

// UpdateMyPost updates a post scoped to its owner; the mutation itself is
// owner-checked at the repository level to avoid TOCTOU ownership races.
func (s *postService) UpdateMyPost(ctx context.Context, id string, userID string, req *dto.UpdatePostRequest) (*dto.PostResponse, error) {
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

	var tags []model.Tag
	if req.Tags != nil {
		tags = make([]model.Tag, 0, len(req.Tags))
		for _, tagName := range req.Tags {
			if tagName == "" {
				continue
			}
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
		return dto.PostToResponse(post), nil
	}

	var updatedPost *model.Post
	var err error
	if req.Tags != nil {
		updatedPost, err = s.postRepo.UpdatePostWithTagsByOwner(ctx, id, userID, updates, tags)
	} else {
		updatedPost, err = s.postRepo.UpdatePostByOwner(ctx, id, userID, updates)
	}
	if err != nil {
		return nil, err
	}
	return dto.PostToResponse(updatedPost), nil
}

func (s *postService) GetPostByID(ctx context.Context, id string) (*dto.PostResponse, error) {
	post, err := s.postRepo.GetPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return dto.PostToResponse(post), nil
}

func (s *postService) GetPosts(ctx context.Context, limit int, offset int) ([]*dto.PostResponse, int64, error) {
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

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postResponse := dto.PostToResponse(post)
		postsResponse = append(postsResponse, postResponse)
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsRandom(ctx context.Context, limit int) ([]*dto.PostResponse, error) {
	if limit < 0 {
		limit = 0
	}

	cacheKey := ""
	if s.cache != nil {
		cacheKey = s.cache.BuildKey("posts", "random", strconv.Itoa(limit))
		var cachedPosts []*dto.PostResponse
		found, err := s.cache.GetJSON(ctx, cacheKey, &cachedPosts)
		if err == nil && found {
			return cachedPosts, nil
		}
	}

	posts, err := s.postRepo.GetPostsRandom(ctx, limit)
	if err != nil {
		return nil, err
	}

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postResponse := dto.PostToResponse(post)
		postsResponse = append(postsResponse, postResponse)
	}

	if cacheKey != "" {
		_ = s.cache.SetJSON(ctx, cacheKey, postsResponse)
	}

	return postsResponse, nil
}

func (s *postService) GetPostsTrending(ctx context.Context, limit int) ([]*dto.PostResponse, error) {
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

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, dto.PostToResponse(post))
	}

	if cacheKey != "" {
		_ = s.cache.SetJSON(ctx, cacheKey, trendingPostsCacheEntry{
			Posts: postsResponse,
		})
	}

	return postsResponse, nil
}

func (s *postService) GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int) ([]*dto.PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if createdBy == "" {
		return []*dto.PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostsByCreatedBy(ctx, createdBy, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, dto.PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsByTag(ctx context.Context, tag string, limit int, offset int) ([]*dto.PostResponse, int64, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if tag == "" {
		return []*dto.PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostsByTag(ctx, tag, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, dto.PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsFiltered(ctx context.Context, filter *dto.PostQueryFilter) ([]*dto.PostResponse, int64, error) {
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

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postResponse := dto.PostToResponse(post)
		postsResponse = append(postsResponse, postResponse)
	}

	return postsResponse, total, nil
}

func (s *postService) GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*dto.PostResponse, int64, error) {
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
		return []*dto.PostResponse{}, 0, nil
	}

	posts, total, err := s.postRepo.GetPostsForYou(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	postsResponse := make([]*dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		postsResponse = append(postsResponse, dto.PostToResponse(post))
	}

	return postsResponse, total, nil
}

func (s *postService) UploadImagePosts(ctx context.Context, file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", apperrors.ErrFileNil
	}
	if file.Size > maxPostImageSize {
		return "", apperrors.ErrFileTooLarge
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
		return "", apperrors.ErrFileTooLarge
	}

	contentType, ext, ok := detectAllowedImage(data)
	if !ok {
		return "", apperrors.ErrInvalidFileType
	}

	objectKey, err := randomImageObjectKey(imageUploadPrefix, ext)
	if err != nil {
		return "", err
	}

	if err := s.s3storage.Save(ctx, objectKey, bytes.NewReader(data), contentType); err != nil {
		return "", err
	}

	return objectKey, nil
}

func (s *postService) findOrCreateTagByName(ctx context.Context, tagName string) (*model.Tag, error) {
	return s.tagService.FindOrCreateByName(ctx, tagName)
}

func detectAllowedImage(data []byte) (contentType string, ext string, ok bool) {
	contentType = http.DetectContentType(data)
	switch contentType {
	case "image/jpeg":
		return contentType, ".jpg", true
	case "image/png":
		return contentType, ".png", true
	case "image/webp":
		return contentType, ".webp", true
	case "image/gif":
		return contentType, ".gif", true
	default:
		return "", "", false
	}
}

// randomImageObjectKey generates a random object key under the given storage
// prefix (e.g. "posts/images" or "users/avatars").
func randomImageObjectKey(prefix, ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s%s", prefix, hex.EncodeToString(b), ext), nil
}

func (s *postService) GetPostsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapPost, error) {
	if limit < 0 {
		limit = 0
	}
	return s.postRepo.GetPostsForSitemap(ctx, limit)
}
