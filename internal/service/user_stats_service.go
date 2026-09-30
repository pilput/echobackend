package service

import (
	"context"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/repository"
)

type UserStatsService interface {
	GetUserStats(ctx context.Context, q dto.DateRangeQuery, limit int) (*dto.UserStatsResponse, error)
}

type userStatsService struct {
	repo repository.UserStatsRepository
}

func NewUserStatsService(repo repository.UserStatsRepository) UserStatsService {
	return &userStatsService{repo: repo}
}

func (s *userStatsService) GetUserStats(ctx context.Context, q dto.DateRangeQuery, limit int) (*dto.UserStatsResponse, error) {
	totalUsers, newUsers, activeUsers, err := s.repo.GetUserCounts(ctx, q.StartDate, q.EndDate)
	if err != nil {
		return nil, err
	}

	newUsersToday, activeUsersThisWeek, err := s.repo.GetSnapshotCounts(ctx)
	if err != nil {
		return nil, err
	}

	topContributors, err := s.repo.GetTopContributors(ctx, limit)
	if err != nil {
		return nil, err
	}

	growthTrend, err := s.getUserGrowthTrend(ctx, q)
	if err != nil {
		return nil, err
	}

	return &dto.UserStatsResponse{
		TotalUsers:          totalUsers,
		NewUsersThisPeriod:  newUsers,
		ActiveUsers:         activeUsers,
		NewUsersToday:       newUsersToday,
		ActiveUsersThisWeek: activeUsersThisWeek,
		TopContributors:     topContributors,
		GrowthTrend:         growthTrend,
	}, nil
}

func (s *userStatsService) getUserGrowthTrend(ctx context.Context, q dto.DateRangeQuery) ([]dto.UserGrowthData, error) {
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()
	if q.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", q.StartDate); err == nil {
			start = parsed
		}
	}
	if q.EndDate != "" {
		if parsed, err := time.Parse("2006-01-02", q.EndDate); err == nil {
			end = parsed
		}
	}

	cumulative, dailyCounts, err := s.repo.GetUserGrowthTrendData(ctx, start, end)
	if err != nil {
		return nil, err
	}

	var result []dto.UserGrowthData
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		newUsers := dailyCounts[dateKey]
		cumulative += newUsers
		result = append(result, dto.UserGrowthData{
			Date:            dateKey,
			NewUsers:        newUsers,
			CumulativeUsers: cumulative,
		})
	}
	return result, nil
}
