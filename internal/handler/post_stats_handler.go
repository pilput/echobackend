package handler

import (
	"strconv"

	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type PostStatsHandler struct {
	statsService service.PostStatsService
}

func NewPostStatsHandler(statsService service.PostStatsService) *PostStatsHandler {
	return &PostStatsHandler{statsService: statsService}
}

func dateRangeFromQuery(c *echo.Context) dto.DateRangeQuery {
	return dto.DateRangeQuery{
		StartDate: c.QueryParam("startDate"),
		EndDate:   c.QueryParam("endDate"),
	}
}

func (h *PostStatsHandler) GetStats(c *echo.Context) error {
	limit, _ := ParsePaginationParams(c, 10)
	var tagID *int
	if raw := c.QueryParam("tagId"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			tagID = &parsed
		}
	}
	stats, err := h.statsService.GetPostStats(c.Request().Context(), dateRangeFromQuery(c), limit, tagID)
	if err != nil {
		return response.InternalServerError(c, "Failed to fetch post stats", err)
	}
	return response.Success(c, "Post stats fetched successfully", stats)
}

func (h *PostStatsHandler) GetEngagement(c *echo.Context) error {
	metrics, err := h.statsService.GetEngagementMetrics(c.Request().Context(), dateRangeFromQuery(c))
	if err != nil {
		return response.InternalServerError(c, "Failed to fetch engagement metrics", err)
	}
	return response.Success(c, "Engagement metrics fetched successfully", metrics)
}
