package service

import (
	"context"

	"echobackend/internal/dto"
	"echobackend/internal/repository"
)

type TagStatsService interface {
	GetTagStats(ctx context.Context, limit int) ([]dto.TagPerformance, error)
}

type tagStatsService struct {
	repo repository.TagStatsRepository
}

func NewTagStatsService(repo repository.TagStatsRepository) TagStatsService {
	return &tagStatsService{repo: repo}
}

func (s *tagStatsService) GetTagStats(ctx context.Context, limit int) ([]dto.TagPerformance, error) {
	return s.repo.GetTagPerformance(ctx, limit)
}
