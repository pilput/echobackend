package service

import (
	"context"
	"math"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/repository"
)

type PostStatsService interface {
	GetPostStats(ctx context.Context, q dto.DateRangeQuery, limit int, tagID *int) (*dto.PostStatsResponse, error)
	GetEngagementMetrics(ctx context.Context, q dto.DateRangeQuery) (*dto.EngagementMetricsResponse, error)
}

type postStatsService struct {
	repo repository.PostStatsRepository
}

func NewPostStatsService(repo repository.PostStatsRepository) PostStatsService {
	return &postStatsService{repo: repo}
}

func (s *postStatsService) GetPostStats(ctx context.Context, q dto.DateRangeQuery, limit int, tagID *int) (*dto.PostStatsResponse, error) {
	totalPosts, newPosts, totalComments, totalViews, totalLikes, err := s.repo.GetPostCounts(ctx, q.StartDate, q.EndDate)
	if err != nil {
		return nil, err
	}

	newPostsToday, err := s.repo.GetNewPostsToday(ctx)
	if err != nil {
		return nil, err
	}

	topPosts, err := s.repo.GetTopPosts(ctx, limit, tagID)
	if err != nil {
		return nil, err
	}

	// Calculate engagement rates in Service Layer
	for i := range topPosts {
		if topPosts[i].Views > 0 {
			topPosts[i].EngagementRate = math.Round((float64(topPosts[i].Likes+topPosts[i].Comments)/float64(topPosts[i].Views))*10000) / 100
		}
	}

	avgEngagementRate := 0.0
	if totalViews > 0 {
		avgEngagementRate = math.Round((float64(totalLikes+totalComments)/float64(totalViews))*10000) / 100
	}

	return &dto.PostStatsResponse{
		TotalPosts:         totalPosts,
		NewPostsThisPeriod: newPosts,
		TotalViews:         totalViews,
		TotalLikes:         totalLikes,
		TotalComments:      totalComments,
		NewPostsToday:      newPostsToday,
		AvgEngagementRate:  avgEngagementRate,
		TopPosts:           topPosts,
	}, nil
}

func (s *postStatsService) GetEngagementMetrics(ctx context.Context, q dto.DateRangeQuery) (*dto.EngagementMetricsResponse, error) {
	prevPeriodStart := time.Now().AddDate(0, 0, -60)
	prevPeriodEnd := time.Now().AddDate(0, 0, -30)
	if q.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", q.StartDate)
		if err == nil {
			endDate := time.Now()
			if q.EndDate != "" {
				if parsedEnd, parseErr := time.Parse("2006-01-02", q.EndDate); parseErr == nil {
					endDate = parsedEnd
				}
			}
			duration := endDate.Sub(startDate)
			prevPeriodEnd = startDate
			prevPeriodStart = startDate.Add(-duration)
		}
	}

	currentLikes, currentComments, totalPosts, totalViews, prevLikes, err := s.repo.GetEngagementCounts(ctx, q.StartDate, q.EndDate, prevPeriodStart, prevPeriodEnd)
	if err != nil {
		return nil, err
	}

	changePercent := 0.0
	if prevLikes > 0 {
		changePercent = math.Round(((float64(currentLikes-prevLikes) / float64(prevLikes)) * 100 * 100)) / 100
	}

	avgLikes := 0.0
	avgComments := 0.0
	avgViews := 0.0
	if totalPosts > 0 {
		avgLikes = math.Round((float64(currentLikes)/float64(totalPosts))*100) / 100
		avgComments = math.Round((float64(currentComments)/float64(totalPosts))*100) / 100
		avgViews = math.Round((float64(totalViews)/float64(totalPosts))*100) / 100
	}

	return &dto.EngagementMetricsResponse{
		TotalEngagements:   currentLikes + currentComments,
		AvgLikesPerPost:    avgLikes,
		AvgCommentsPerPost: avgComments,
		AvgViewsPerPost:    avgViews,
		PeriodComparison: dto.PeriodComparison{
			Current:       currentLikes,
			Previous:      prevLikes,
			ChangePercent: changePercent,
		},
	}, nil
}
