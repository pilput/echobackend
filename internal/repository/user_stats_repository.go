package repository

import (
	"context"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type UserStatsRepository interface {
	GetUserCounts(ctx context.Context, startDate, endDate string) (totalUsers, newUsers, activeUsers int64, err error)
	GetSnapshotCounts(ctx context.Context) (newUsersToday, activeUsersThisWeek int64, err error)
	GetTopContributors(ctx context.Context, limit int) ([]dto.TopContributor, error)
	GetUserGrowthTrendData(ctx context.Context, start, end time.Time) (cumulativeBefore int64, dailyCounts map[string]int64, err error)
}

type userStatsRepository struct {
	db *gorm.DB
}

func NewUserStatsRepository(db *gorm.DB) UserStatsRepository {
	return &userStatsRepository{db: db}
}

func (r *userStatsRepository) GetSnapshotCounts(ctx context.Context) (newUsersToday, activeUsersThisWeek int64, err error) {
	today := time.Now().Format("2006-01-02")
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	if err = r.db.WithContext(ctx).Model(&model.User{}).Where("DATE(created_at) >= ?", today).Count(&newUsersToday).Error; err != nil {
		return
	}
	// Table() opts out of the soft-delete scope GORM adds for Model(), so the
	// deleted_at predicate has to be written by hand here.
	err = r.db.WithContext(ctx).Table("post_views").Select("COUNT(DISTINCT user_id)").Where("user_id IS NOT NULL AND deleted_at IS NULL AND DATE(created_at) >= ?", weekAgo).Scan(&activeUsersThisWeek).Error
	return
}

func (r *userStatsRepository) GetUserCounts(ctx context.Context, startDate, endDate string) (totalUsers, newUsers, activeUsers int64, err error) {
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

func (r *userStatsRepository) GetTopContributors(ctx context.Context, limit int) ([]dto.TopContributor, error) {
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

func (r *userStatsRepository) GetUserGrowthTrendData(ctx context.Context, start, end time.Time) (int64, map[string]int64, error) {
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
