package repository

import (
	"context"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type PostStatsRepository interface {
	GetPostCounts(ctx context.Context, startDate, endDate string) (totalPosts, newPosts, totalComments, totalViews, totalLikes int64, err error)
	GetNewPostsToday(ctx context.Context) (int64, error)
	GetTopPosts(ctx context.Context, limit int, tagID *int) ([]dto.PostPerformanceData, error)
	GetEngagementCounts(ctx context.Context, startDate, endDate string, prevPeriodStart, prevPeriodEnd time.Time) (currentLikes, currentComments, totalPosts, totalViews, prevLikes int64, err error)
}

type postStatsRepository struct {
	db *gorm.DB
}

func NewPostStatsRepository(db *gorm.DB) PostStatsRepository {
	return &postStatsRepository{db: db}
}

func (r *postStatsRepository) GetNewPostsToday(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("published = ? AND DATE(created_at) >= ?", true, time.Now().Format("2006-01-02")).
		Count(&count).Error
	return count, err
}

func (r *postStatsRepository) GetPostCounts(ctx context.Context, startDate, endDate string) (totalPosts, newPosts, totalComments, totalViews, totalLikes int64, err error) {
	if err = r.db.WithContext(ctx).Model(&model.Post{}).Where("published = ?", true).Count(&totalPosts).Error; err != nil {
		return
	}

	newPostsQuery := r.db.WithContext(ctx).Model(&model.Post{}).Where("published = ?", true)
	if startDate != "" {
		newPostsQuery = newPostsQuery.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		newPostsQuery = newPostsQuery.Where("DATE(created_at) <= ?", endDate)
	}
	if err = newPostsQuery.Count(&newPosts).Error; err != nil {
		return
	}

	type sumResult struct{ Total int64 }
	var vResult, lResult sumResult

	if err = r.db.WithContext(ctx).Model(&model.Post{}).Select("COALESCE(SUM(view_count), 0) AS total").Where("published = ?", true).Scan(&vResult).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.Post{}).Select("COALESCE(SUM(like_count), 0) AS total").Where("published = ?", true).Scan(&lResult).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.PostComment{}).Count(&totalComments).Error; err != nil {
		return
	}

	return totalPosts, newPosts, totalComments, vResult.Total, lResult.Total, nil
}

func (r *postStatsRepository) GetTopPosts(ctx context.Context, limit int, tagID *int) ([]dto.PostPerformanceData, error) {
	type postRow struct {
		ID              string
		Title           *string
		Slug            *string
		Views           int64
		Likes           int64
		CreatedAt       *string
		AuthorID        string
		AuthorUsername  *string
		AuthorFirstName *string
		AuthorLastName  *string
	}
	query := r.db.WithContext(ctx).
		Table("posts").
		Select("posts.id, posts.title, posts.slug, posts.view_count AS views, posts.like_count AS likes, posts.created_at, users.id AS author_id, users.username AS author_username, users.first_name AS author_first_name, users.last_name AS author_last_name").
		Joins("INNER JOIN users ON posts.created_by = users.id AND users.deleted_at IS NULL").
		Where("posts.published = ?", true).
		Order("posts.view_count DESC").
		Limit(limit)
	if tagID != nil {
		query = query.Joins("INNER JOIN posts_to_tags ON posts.id = posts_to_tags.post_id").Where("posts_to_tags.tag_id = ?", *tagID)
	}
	var topRows []postRow
	if err := query.Scan(&topRows).Error; err != nil {
		return nil, err
	}

	postIDs := make([]string, 0, len(topRows))
	for _, row := range topRows {
		postIDs = append(postIDs, row.ID)
	}

	type commentCountRow struct {
		PostID string
		Count  int64
	}
	commentCountMap := map[string]int64{}
	if len(postIDs) > 0 {
		var commentCounts []commentCountRow
		if err := r.db.WithContext(ctx).
			Table("post_comments").
			Select("post_id, COUNT(*) AS count").
			Where("post_id IN ?", postIDs).
			Group("post_id").
			Scan(&commentCounts).Error; err != nil {
			return nil, err
		}
		for _, row := range commentCounts {
			commentCountMap[row.PostID] = row.Count
		}
	}

	topPosts := make([]dto.PostPerformanceData, 0, len(topRows))
	for _, row := range topRows {
		topPosts = append(topPosts, dto.PostPerformanceData{
			ID:       row.ID,
			Title:    row.Title,
			Slug:     row.Slug,
			Views:    row.Views,
			Likes:    row.Likes,
			Comments: commentCountMap[row.ID],
			Author: dto.PostPerformanceAuthor{
				ID:        row.AuthorID,
				Username:  row.AuthorUsername,
				FirstName: row.AuthorFirstName,
				LastName:  row.AuthorLastName,
			},
			CreatedAt: row.CreatedAt,
		})
	}
	return topPosts, nil
}

func (r *postStatsRepository) GetEngagementCounts(ctx context.Context, startDate, endDate string, prevPeriodStart, prevPeriodEnd time.Time) (currentLikes, currentComments, totalPosts, totalViews, prevLikes int64, err error) {
	likesQuery := r.db.WithContext(ctx).Model(&model.PostLike{})
	commentsQuery := r.db.WithContext(ctx).Model(&model.PostComment{})
	if startDate != "" {
		likesQuery = likesQuery.Where("created_at >= ?", startDate)
		commentsQuery = commentsQuery.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		endDateTime, _ := time.Parse("2006-01-02", endDate)
		if !endDateTime.IsZero() {
			likesQuery = likesQuery.Where("created_at <= ?", endDateTime.Add(24*time.Hour))
			commentsQuery = commentsQuery.Where("created_at <= ?", endDateTime.Add(24*time.Hour))
		}
	}
	if err = likesQuery.Count(&currentLikes).Error; err != nil {
		return
	}
	if err = commentsQuery.Count(&currentComments).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.Post{}).Where("published = ?", true).Count(&totalPosts).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.Post{}).Select("COALESCE(SUM(view_count), 0)").Where("published = ?", true).Scan(&totalViews).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.PostLike{}).Where("created_at >= ? AND created_at <= ?", prevPeriodStart, prevPeriodEnd).Count(&prevLikes).Error; err != nil {
		return
	}

	return
}
