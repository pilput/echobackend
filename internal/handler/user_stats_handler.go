package handler

import (
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type UserStatsHandler struct {
	statsService service.UserStatsService
}

func NewUserStatsHandler(statsService service.UserStatsService) *UserStatsHandler {
	return &UserStatsHandler{statsService: statsService}
}

func (h *UserStatsHandler) GetStats(c *echo.Context) error {
	limit, _ := ParsePaginationParams(c, 10)
	stats, err := h.statsService.GetUserStats(c.Request().Context(), dto.DateRangeQuery{
		StartDate: c.QueryParam("startDate"),
		EndDate:   c.QueryParam("endDate"),
	}, limit)
	if err != nil {
		return response.InternalServerError(c, "Failed to fetch user stats", err)
	}
	return response.Success(c, "User stats fetched successfully", stats)
}
