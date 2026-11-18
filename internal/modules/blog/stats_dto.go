package blog

import "time"

// DateRangeQuery bounds the admin stats queries (YYYY-MM-DD, "" = open).
type DateRangeQuery struct {
	StartDate string
	EndDate   string
}

type PostPerformanceAuthor struct {
	ID        string  `json:"id"`
	Username  *string `json:"username"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
}

type PostPerformanceData struct {
	ID             string                `json:"id"`
	Title          *string               `json:"title"`
	Slug           *string               `json:"slug"`
	Views          int64                 `json:"views"`
	Likes          int64                 `json:"likes"`
	Comments       int64                 `json:"comments"`
	EngagementRate float64               `json:"engagementRate"`
	Author         PostPerformanceAuthor `json:"author"`
	CreatedAt      *string               `json:"createdAt"`
}

type TagPerformance struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	PostCount  int64  `json:"postCount"`
	TotalViews int64  `json:"totalViews"`
	TotalLikes int64  `json:"totalLikes"`
	// LastUsedAt is the publish time of the newest published post with this tag.
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

type PostStatsResponse struct {
	TotalPosts         int64                 `json:"totalPosts"`
	NewPostsThisPeriod int64                 `json:"newPostsThisPeriod"`
	TotalViews         int64                 `json:"totalViews"`
	TotalLikes         int64                 `json:"totalLikes"`
	TotalComments      int64                 `json:"totalComments"`
	NewPostsToday      int64                 `json:"newPostsToday"`
	AvgEngagementRate  float64               `json:"avgEngagementRate"`
	TopPosts           []PostPerformanceData `json:"topPosts"`
}

type PeriodComparison struct {
	Current       int64   `json:"current"`
	Previous      int64   `json:"previous"`
	ChangePercent float64 `json:"changePercent"`
}

type EngagementMetricsResponse struct {
	TotalEngagements   int64            `json:"totalEngagements"`
	AvgLikesPerPost    float64          `json:"avgLikesPerPost"`
	AvgCommentsPerPost float64          `json:"avgCommentsPerPost"`
	AvgViewsPerPost    float64          `json:"avgViewsPerPost"`
	PeriodComparison   PeriodComparison `json:"periodComparison"`
}
