package dto

type DateRangeQuery struct {
	StartDate string
	EndDate   string
}

type UserGrowthData struct {
	Date            string `json:"date"`
	NewUsers        int64  `json:"newUsers"`
	CumulativeUsers int64  `json:"cumulativeUsers"`
}

type TopContributor struct {
	ID         string  `json:"id"`
	Username   *string `json:"username"`
	FirstName  *string `json:"firstName"`
	LastName   *string `json:"lastName"`
	PostCount  int64   `json:"postCount"`
	TotalViews int64   `json:"totalViews"`
	TotalLikes int64   `json:"totalLikes"`
}

type UserStatsResponse struct {
	TotalUsers          int64 `json:"totalUsers"`
	NewUsersThisPeriod  int64 `json:"newUsersThisPeriod"`
	ActiveUsers         int64 `json:"activeUsers"`
	NewUsersToday       int64 `json:"newUsersToday"`
	ActiveUsersThisWeek int64 `json:"activeUsersThisWeek"`
	// AdminUsers counts non-deleted super admins.
	AdminUsers      int64            `json:"adminUsers"`
	TopContributors []TopContributor `json:"topContributors"`
	GrowthTrend     []UserGrowthData `json:"growthTrend"`
}
