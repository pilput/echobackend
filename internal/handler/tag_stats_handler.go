package handler

import (
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type TagStatsHandler struct {
	statsService service.TagStatsService
}

func NewTagStatsHandler(statsService service.TagStatsService) *TagStatsHandler {
	return &TagStatsHandler{statsService: statsService}
}

func (h *TagStatsHandler) GetStats(c *echo.Context) error {
	limit, _ := ParsePaginationParams(c, 10)
	stats, err := h.statsService.GetTagStats(c.Request().Context(), limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to fetch tag stats", err)
	}
	return response.Success(c, "Tag stats fetched successfully", stats)
}
