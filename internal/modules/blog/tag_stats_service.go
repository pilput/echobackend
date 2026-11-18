package blog

import (
	"context"
)

type TagStatsService interface {
	GetTagStats(ctx context.Context, limit int) ([]TagPerformance, error)
}

type tagStatsService struct {
	repo TagStatsRepository
}

func NewTagStatsService(repo TagStatsRepository) TagStatsService {
	return &tagStatsService{repo: repo}
}

func (s *tagStatsService) GetTagStats(ctx context.Context, limit int) ([]TagPerformance, error) {
	return s.repo.GetTagPerformance(ctx, limit)
}
