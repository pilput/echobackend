package blog

import (
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type TagStatsHandler struct {
	statsService TagStatsService
}

func NewTagStatsHandler(statsService TagStatsService) *TagStatsHandler {
	return &TagStatsHandler{statsService: statsService}
}

func (h *TagStatsHandler) GetStats(c *echo.Context) error {
	limit, _ := response.ParsePaginationParams(c, 10)
	stats, err := h.statsService.GetTagStats(c.Request().Context(), limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to fetch tag stats", err)
	}
	return response.Success(c, "Tag stats fetched successfully", stats)
}
