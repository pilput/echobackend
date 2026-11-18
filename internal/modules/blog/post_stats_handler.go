package blog

import (
	"echobackend/pkg/response"
	"strconv"

	"github.com/labstack/echo/v5"
)

type PostStatsHandler struct {
	statsService PostStatsService
}

func NewPostStatsHandler(statsService PostStatsService) *PostStatsHandler {
	return &PostStatsHandler{statsService: statsService}
}

// dateRangeFromQuery reads the stats date range; unparsable dates are ignored.
func dateRangeFromQuery(c *echo.Context) DateRangeQuery {
	return DateRangeQuery{
		StartDate: parseStatsDate(c.QueryParam("startDate")),
		EndDate:   parseStatsDate(c.QueryParam("endDate")),
	}
}

func (h *PostStatsHandler) GetStats(c *echo.Context) error {
	limit, _ := response.ParsePaginationParams(c, 10)
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
