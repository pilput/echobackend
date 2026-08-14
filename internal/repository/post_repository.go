package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type PostRepository interface {
	CreatePost(ctx context.Context, post *model.Post) error
	CreatePostWithTags(ctx context.Context, post *model.Post, tags []model.Tag) (*model.Post, error)
	GetPosts(ctx context.Context, limit int, offset int) ([]*model.Post, int64, error)
	GetPostsFiltered(ctx context.Context, filter *dto.PostQueryFilter) ([]*model.Post, int64, error)
	GetPostByUsername(ctx context.Context, username string, offset int, limit int) ([]*model.Post, int64, error)
	GetPostsRandom(ctx context.Context, limit int) ([]*model.Post, error)
	GetPostsTrending(ctx context.Context, limit int) ([]*model.Post, error)
	GetPostByID(ctx context.Context, id string) (*model.Post, error)
	GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*model.Post, error)
	GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int) ([]*model.Post, int64, error)
	DeletePostByID(ctx context.Context, id string) error
	UpdatePost(ctx context.Context, id string, updates map[string]any) (*model.Post, error)
	UpdatePostWithTags(ctx context.Context, id string, updates map[string]any, tags []model.Tag) (*model.Post, error)
	GetPostsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapPost, error)
	SearchPosts(ctx context.Context, keyword string, limit int, offset int) ([]*model.Post, int64, error)
	GetPostsByTag(ctx context.Context, tag string, limit int, offset int) ([]*model.Post, int64, error)
	GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*model.Post, int64, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
	GetAuthorPostStats(ctx context.Context, userID string) (*dto.MyPostsAnalyticsSummary, error)
	GetTopPostsByAuthor(ctx context.Context, userID string, limit int) ([]dto.MyPostPerformance, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) CreatePost(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *postRepository) CreatePostWithTags(ctx context.Context, post *model.Post, tags []model.Tag) (*model.Post, error) {
	post.Tags = tags

	err := r.db.WithContext(ctx).Create(post).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create post with tags: %w", err)
	}

	err = r.db.WithContext(ctx).Preload("User", preloadUserBrief).Preload("Tags").First(post, "id = ?", post.ID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load created post with associations: %w", err)
	}

	return post, nil
}

func (r *postRepository) UpdatePost(ctx context.Context, id string, updates map[string]any) (*model.Post, error) {
	return r.UpdatePostWithTags(ctx, id, updates, nil)
}

func (r *postRepository) UpdatePostWithTags(ctx context.Context, id string, updates map[string]any, tags []model.Tag) (*model.Post, error) {
	if len(updates) > 0 {
		result := r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Updates(updates)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to update post: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil, apperrors.ErrPostNotFound
		}
	}

	var updatedPost model.Post
	err := r.db.WithContext(ctx).Preload("User", preloadUserBrief).Preload("Tags").First(&updatedPost, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to retrieve post for update: %w", err)
	}

	if tags != nil {
		if err := r.db.WithContext(ctx).Model(&updatedPost).Association("Tags").Replace(tags); err != nil {
			return nil, fmt.Errorf("failed to update post tags association: %w", err)
		}
		updatedPost.Tags = tags
	}

	return &updatedPost, nil
}

func (r *postRepository) GetPostByUsername(ctx context.Context, username string, offset int, limit int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Post{}).
		Joins("JOIN users ON users.id = posts.created_by").
		Where("users.username = ? AND users.deleted_at IS NULL", username)

	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts for username %s: %w", username, err)
	}

	err = r.db.WithContext(ctx).Model(&model.Post{}).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Joins("JOIN users ON users.id = posts.created_by").
		Where("users.username = ? AND users.deleted_at IS NULL", username).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get posts for username %s: %w", username, err)
	}

	return posts, count, nil
}

func (r *postRepository) DeletePostByID(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Post{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrPostNotFound
	}
	return nil
}

func (r *postRepository) GetPosts(ctx context.Context, limit int, offset int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64

	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Where("posts.published = ?", true).
		Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts: %w", err)
	}

	err = activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("posts.published = ?", true).
		Order("posts.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get posts: %w", err)
	}

	return posts, count, nil
}

func (r *postRepository) GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*model.Post, error) {
	var post model.Post
	err := r.db.WithContext(ctx).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Joins("JOIN users ON users.id = posts.created_by").
		Where("posts.slug = ? AND users.username = ? AND users.deleted_at IS NULL", slug, username).
		First(&post).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post by slug '%s' and username '%s': %w", slug, username, err)
	}
	return &post, nil
}

func (r *postRepository) GetPostByID(ctx context.Context, id string) (*model.Post, error) {
	var post model.Post
	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("posts.id = ?", id).
		First(&post).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post by ID: %w", err)
	}
	return &post, nil
}

func (r *postRepository) GetPostsRandom(ctx context.Context, limit int) ([]*model.Post, error) {
	var randomPosts []*model.Post
	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("posts.published = ?", true).
		Order("RANDOM()").
		Limit(limit).
		Find(&randomPosts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get random posts: %w", err)
	}
	return randomPosts, nil
}

func (r *postRepository) GetPostsTrending(ctx context.Context, limit int) ([]*model.Post, error) {
	var posts []*model.Post

	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("posts.published = ?", true).
		Order("posts.like_count * 2 + posts.bookmark_count * 2 + posts.view_count DESC").
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get trending posts: %w", err)
	}

	return posts, nil
}

func (r *postRepository) GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64

	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Where("posts.created_by = ?", createdBy).
		Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts by creator ID %s: %w", createdBy, err)
	}

	err = activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("posts.created_by = ?", createdBy).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get posts by creator ID %s: %w", createdBy, err)
	}
	return posts, count, nil
}

func (r *postRepository) GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64

	if userID == "" {
		return []*model.Post{}, 0, nil
	}

	followingIDs := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Select("following_id").
		Where("follower_id = ?", userID)

	base := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Where("posts.published = ?", true).
		Where("posts.created_by = ? OR posts.created_by IN (?)", userID, followingIDs)

	if err := base.Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count for-you posts: %w", err)
	}

	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("posts.published = ?", true).
		Where("posts.created_by = ? OR posts.created_by IN (?)", userID, followingIDs).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get for-you posts: %w", err)
	}

	return posts, count, nil
}

func (r *postRepository) SearchPosts(ctx context.Context, keyword string, limit int, offset int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64
	likePattern := "%" + keyword + "%"

	err := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Where("(posts.title ILIKE ? OR posts.body ILIKE ?) AND posts.published = ?", likePattern, likePattern, true).
		Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts for search: %w", err)
	}

	err = activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Where("(posts.title ILIKE ? OR posts.body ILIKE ?) AND posts.published = ?", likePattern, likePattern, true).
		Order("posts.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search posts: %w", err)
	}
	return posts, count, nil
}

func (r *postRepository) GetPostsByTag(ctx context.Context, tag string, limit int, offset int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Post{}).
		Joins("JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Joins("JOIN posts_to_tags ON posts_to_tags.post_id = posts.id").
		Joins("JOIN tags ON tags.id = posts_to_tags.tag_id").
		Where("tags.name = ? AND posts.published = ?", tag, true)

	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts by tag: %w", err)
	}

	err = r.db.WithContext(ctx).Model(&model.Post{}).
		Preload("User", preloadUserBrief).
		Preload("Tags").
		Joins("JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Joins("JOIN posts_to_tags ON posts_to_tags.post_id = posts.id").
		Joins("JOIN tags ON tags.id = posts_to_tags.tag_id").
		Where("tags.name = ? AND posts.published = ?", tag, true).
		Order("posts.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get posts by tag: %w", err)
	}
	return posts, count, nil
}

func (r *postRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check post existence: %w", err)
	}
	return count > 0, nil
}

func (r *postRepository) GetPostsFiltered(ctx context.Context, filter *dto.PostQueryFilter) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var count int64

	query := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{})).
		Preload("User", preloadUserBrief).
		Preload("Tags")

	if filter.Search != "" {
		likePattern := "%" + filter.Search + "%"
		query = query.Where("posts.title ILIKE ? AND posts.published = ?", likePattern, true)
	} else {
		query = query.Where("posts.published = ?", true)
	}

	if filter.StartDate != "" {
		query = query.Where("posts.created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("posts.created_at <= ?", filter.EndDate)
	}

	if filter.Search == "" && filter.Published != nil {
		query = query.Where("posts.published = ?", *filter.Published)
	}

	if filter.CreatedBy != "" {
		query = query.Where("posts.created_by = ?", filter.CreatedBy)
	}

	if len(filter.Tags) > 0 {
		query = query.Joins("JOIN posts_to_tags ON posts_to_tags.post_id = posts.id").
			Joins("JOIN tags ON tags.id = posts_to_tags.tag_id").
			Where("tags.name IN ?", filter.Tags)
	}

	countQuery := activePostUserJoin(r.db.WithContext(ctx).Model(&model.Post{}))

	if filter.Search != "" {
		likePattern := "%" + filter.Search + "%"
		countQuery = countQuery.Where("posts.title ILIKE ? AND posts.published = ?", likePattern, true)
	} else {
		countQuery = countQuery.Where("posts.published = ?", true)
	}

	if filter.StartDate != "" {
		countQuery = countQuery.Where("posts.created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		countQuery = countQuery.Where("posts.created_at <= ?", filter.EndDate)
	}
	if filter.Search == "" && filter.Published != nil {
		countQuery = countQuery.Where("posts.published = ?", *filter.Published)
	}
	if filter.CreatedBy != "" {
		countQuery = countQuery.Where("posts.created_by = ?", filter.CreatedBy)
	}
	if len(filter.Tags) > 0 {
		countQuery = countQuery.Joins("JOIN posts_to_tags ON posts_to_tags.post_id = posts.id").
			Joins("JOIN tags ON tags.id = posts_to_tags.tag_id").
			Where("tags.name IN ?", filter.Tags)
	}

	err := countQuery.Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts: %w", err)
	}

	sortField := filter.GetSortField()
	sortOrder := filter.GetSortOrder()
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	query = query.Limit(filter.Limit).Offset(filter.Offset)

	err = query.Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get filtered posts: %w", err)
	}

	return posts, count, nil
}

func (r *postRepository) GetPostsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapPost, error) {
	var sitemapPosts []*dto.SitemapPost

	err := r.db.WithContext(ctx).
		Table("posts").
		Select("users.username, posts.slug, posts.created_at, posts.updated_at").
		Joins("JOIN users ON users.id = posts.created_by").
		Where("posts.published = ? AND users.deleted_at IS NULL", true).
		Order("posts.created_at DESC").
		Limit(limit).
		Find(&sitemapPosts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get posts for sitemap: %w", err)
	}

	return sitemapPosts, nil
}

func (r *postRepository) GetAuthorPostStats(ctx context.Context, userID string) (*dto.MyPostsAnalyticsSummary, error) {
	stats := &dto.MyPostsAnalyticsSummary{}
	err := r.db.WithContext(ctx).
		Table("posts").
		Select(`
			COUNT(*) AS total_posts,
			COUNT(*) FILTER (WHERE published = true) AS published_posts,
			COALESCE(SUM(view_count), 0) AS total_views,
			COALESCE(SUM(like_count), 0) AS total_likes
		`).
		Where("created_by = ?", userID).
		Scan(stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get author post stats: %w", err)
	}
	return stats, nil
}

func (r *postRepository) GetTopPostsByAuthor(ctx context.Context, userID string, limit int) ([]dto.MyPostPerformance, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	var posts []dto.MyPostPerformance
	err := r.db.WithContext(ctx).
		Table("posts").
		Select("id, title, slug, view_count, like_count").
		Where("created_by = ?", userID).
		Order("view_count DESC, like_count DESC, created_at DESC").
		Limit(limit).
		Scan(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get top posts by author: %w", err)
	}
	return posts, nil
}
