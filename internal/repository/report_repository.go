package repository

import (
	"context"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type ReportRepository interface {
	GetOverviewCounts(ctx context.Context) (totalUsers, totalPosts, totalViews, totalLikes, totalComments, newUsersToday, newPostsToday, activeUsersThisWeek int64, err error)
	GetUserCounts(ctx context.Context, startDate, endDate string) (totalUsers, newUsers, activeUsers int64, err error)
	GetTopContributors(ctx context.Context, limit int) ([]dto.TopContributor, error)
	GetUserGrowthTrendData(ctx context.Context, start, end time.Time) (cumulativeBefore int64, dailyCounts map[string]int64, err error)
	GetPostCounts(ctx context.Context, startDate, endDate string) (totalPosts, newPosts, totalComments, totalViews, totalLikes int64, err error)
	GetTopPosts(ctx context.Context, limit int, tagID *int) ([]dto.PostPerformanceData, error)
	GetTagPerformance(ctx context.Context, limit int) ([]dto.TagPerformance, error)
	GetEngagementCounts(ctx context.Context, startDate, endDate string, prevPeriodStart, prevPeriodEnd time.Time) (currentLikes, currentComments, totalPosts, totalViews, prevLikes int64, err error)
}

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetOverviewCounts(ctx context.Context) (totalUsers, totalPosts, totalViews, totalLikes, totalComments, newUsersToday, newPostsToday, activeUsersThisWeek int64, err error) {
	today := time.Now().Format("2006-01-02")
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	type result struct{ Count int64 }
	var uResult, pResult, vResult, lResult, cResult, nuResult, npResult, auResult result

	if err = r.db.WithContext(ctx).Model(&model.User{}).Count(&uResult.Count).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.Post{}).Where("published = ?", true).Count(&pResult.Count).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.PostView{}).Count(&vResult.Count).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.PostLike{}).Count(&lResult.Count).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.PostComment{}).Count(&cResult.Count).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.User{}).Where("DATE(created_at) >= ?", today).Count(&nuResult.Count).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&model.Post{}).Where("published = ? AND DATE(created_at) >= ?", true, today).Count(&npResult.Count).Error; err != nil {
		return
	}
	// Table() opts out of the soft-delete scope GORM adds for Model(), so the
	// deleted_at predicate has to be written by hand here.
	if err = r.db.WithContext(ctx).Table("post_views").Select("COUNT(DISTINCT user_id) AS count").Where("user_id IS NOT NULL AND deleted_at IS NULL AND DATE(created_at) >= ?", weekAgo).Scan(&auResult).Error; err != nil {
		return
	}

	return uResult.Count, pResult.Count, vResult.Count, lResult.Count, cResult.Count, nuResult.Count, npResult.Count, auResult.Count, nil
}

func (r *reportRepository) GetUserCounts(ctx context.Context, startDate, endDate string) (totalUsers, newUsers, activeUsers int64, err error) {
	if err = r.db.WithContext(ctx).Model(&model.User{}).Count(&totalUsers).Error; err != nil {
		return
	}

	newUsersQuery := r.db.WithContext(ctx).Model(&model.User{})
	if startDate != "" {
		newUsersQuery = newUsersQuery.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		newUsersQuery = newUsersQuery.Where("DATE(created_at) <= ?", endDate)
	}
	if err = newUsersQuery.Count(&newUsers).Error; err != nil {
		return
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err = r.db.WithContext(ctx).Table("post_views").Select("COUNT(DISTINCT user_id)").Where("user_id IS NOT NULL AND deleted_at IS NULL AND created_at >= ?", thirtyDaysAgo).Scan(&activeUsers).Error; err != nil {
		return
	}

	return
}

func (r *reportRepository) GetTopContributors(ctx context.Context, limit int) ([]dto.TopContributor, error) {
	type contributorRow struct {
		ID         string
		Username   *string
		FirstName  *string
		LastName   *string
		PostCount  int64
		TotalViews int64
		TotalLikes int64
	}
	var topRows []contributorRow
	if err := r.db.WithContext(ctx).
		Table("users").
		Select("users.id, users.username, users.first_name, users.last_name, COUNT(posts.id) AS post_count, COALESCE(SUM(posts.view_count), 0) AS total_views, COALESCE(SUM(posts.like_count), 0) AS total_likes").
		Joins("LEFT JOIN posts ON users.id = posts.created_by").
		Where("users.deleted_at IS NULL").
		Group("users.id, users.username, users.first_name, users.last_name").
		Order("COUNT(posts.id) DESC").
		Limit(limit).
		Scan(&topRows).Error; err != nil {
		return nil, err
	}

	topContributors := make([]dto.TopContributor, 0, len(topRows))
	for _, row := range topRows {
		topContributors = append(topContributors, dto.TopContributor{
			ID:         row.ID,
			Username:   row.Username,
			FirstName:  row.FirstName,
			LastName:   row.LastName,
			PostCount:  row.PostCount,
			TotalViews: row.TotalViews,
			TotalLikes: row.TotalLikes,
		})
	}
	return topContributors, nil
}

func (r *reportRepository) GetUserGrowthTrendData(ctx context.Context, start, end time.Time) (int64, map[string]int64, error) {
	var cumulative int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("DATE(created_at) < ?", start.Format("2006-01-02")).Count(&cumulative).Error; err != nil {
		return 0, nil, err
	}

	type dayCount struct {
		Date  string
		Count int64
	}
	var rows []dayCount
	// The cumulative count above goes through Model(), which filters soft-deleted
	// users automatically; Table() below does not, so it must match explicitly or
	// the trend and its baseline would be counted on different populations.
	if err := r.db.WithContext(ctx).Table("users").
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Group("DATE(created_at)").
		Order("DATE(created_at) ASC").
		Scan(&rows).Error; err != nil {
		return 0, nil, err
	}

	dailyCounts := make(map[string]int64, len(rows))
	for _, row := range rows {
		dailyCounts[row.Date] = row.Count
	}

	return cumulative, dailyCounts, nil
}

func (r *reportRepository) GetPostCounts(ctx context.Context, startDate, endDate string) (totalPosts, newPosts, totalComments, totalViews, totalLikes int64, err error) {
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

func (r *reportRepository) GetTopPosts(ctx context.Context, limit int, tagID *int) ([]dto.PostPerformanceData, error) {
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

func (r *reportRepository) GetTagPerformance(ctx context.Context, limit int) ([]dto.TagPerformance, error) {
	type tagRow struct {
		ID         int
		Name       string
		PostCount  int64
		TotalViews int64
		TotalLikes int64
	}
	var tagRows []tagRow
	if err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.id, tags.name, COUNT(posts_to_tags.post_id) AS post_count, COALESCE(SUM(posts.view_count), 0) AS total_views, COALESCE(SUM(posts.like_count), 0) AS total_likes").
		Joins("INNER JOIN posts_to_tags ON tags.id = posts_to_tags.tag_id").
		Joins("INNER JOIN posts ON posts_to_tags.post_id = posts.id").
		Where("posts.published = ?", true).
		Group("tags.id, tags.name").
		Order("COUNT(posts_to_tags.post_id) DESC").
		Limit(limit).
		Scan(&tagRows).Error; err != nil {
		return nil, err
	}

	tagPerformance := make([]dto.TagPerformance, 0, len(tagRows))
	for _, row := range tagRows {
		tagPerformance = append(tagPerformance, dto.TagPerformance(row))
	}
	return tagPerformance, nil
}

func (r *reportRepository) GetEngagementCounts(ctx context.Context, startDate, endDate string, prevPeriodStart, prevPeriodEnd time.Time) (currentLikes, currentComments, totalPosts, totalViews, prevLikes int64, err error) {
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
