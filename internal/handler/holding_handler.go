package handler

import (
	"errors"
	"strconv"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type HoldingHandler struct {
	holdingService service.HoldingService
}

func NewHoldingHandler(holdingService service.HoldingService) *HoldingHandler {
	return &HoldingHandler{holdingService: holdingService}
}

func (h *HoldingHandler) respondHoldingError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrHoldingNotFound):
		return response.NotFound(c, message, err)
	case errors.Is(err, apperrors.ErrHoldingNotOwned):
		return response.Forbidden(c, message)
	case errors.Is(err, apperrors.ErrHoldingTypeNotFound):
		return response.BadRequest(c, message, err)
	case errors.Is(err, apperrors.ErrHoldingDuplicateSame):
		return response.BadRequest(c, message, err)
	case errors.Is(err, apperrors.ErrHoldingInvalidRange):
		return response.BadRequest(c, message, err)
	case errors.Is(err, apperrors.ErrHoldingRangeTooLarge):
		return response.BadRequest(c, message, err)
	default:
		return response.InternalServerError(c, message, err)
	}
}

func (h *HoldingHandler) GetHoldings(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	filter := &dto.HoldingQueryFilter{
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	now := time.Now()
	curMonth := int(now.Month())
	curYear := now.Year()
	filter.Month = &curMonth
	filter.Year = &curYear

	if m := c.QueryParam("month"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 1 && v <= 12 {
			filter.Month = &v
		}
	}
	if y := c.QueryParam("year"); y != "" {
		if v, err := strconv.Atoi(y); err == nil {
			filter.Year = &v
		}
	}
	if s := c.QueryParam("sortBy"); s != "" {
		filter.SortBy = s
	}
	if o := c.QueryParam("order"); o != "" {
		filter.SortOrder = o
	}

	holdings, err := h.holdingService.GetHoldings(c.Request().Context(), userID, filter)
	if err != nil {
		return h.respondHoldingError(c, "Failed to get holdings", err)
	}

	return response.Success(c, "Holdings fetched successfully", holdings)
}

func (h *HoldingHandler) GetHoldingByID(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid holding ID", err)
	}

	holding, err := h.holdingService.GetHoldingByID(c.Request().Context(), id, userID)
	if err != nil {
		return h.respondHoldingError(c, "Failed to get holding", err)
	}

	return response.Success(c, "Holding fetched successfully", holding)
}

func (h *HoldingHandler) CreateHolding(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	var req dto.CreateHoldingRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request format", err)
	}

	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	holding, err := h.holdingService.CreateHolding(c.Request().Context(), userID, &req)
	if err != nil {
		return h.respondHoldingError(c, "Failed to create holding", err)
	}

	return response.Created(c, "Holding created successfully", []any{holding})
}

func (h *HoldingHandler) UpdateHolding(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid holding ID", err)
	}

	var req dto.UpdateHoldingRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request format", err)
	}

	holding, err := h.holdingService.UpdateHolding(c.Request().Context(), id, userID, &req)
	if err != nil {
		return h.respondHoldingError(c, "Failed to update holding", err)
	}

	return response.Success(c, "Holding updated successfully", []any{holding})
}

func (h *HoldingHandler) DeleteHolding(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid holding ID", err)
	}

	if err := h.holdingService.DeleteHolding(c.Request().Context(), id, userID); err != nil {
		return h.respondHoldingError(c, "Failed to delete holding", err)
	}

	return response.Success(c, "Holding deleted successfully", nil)
}

func (h *HoldingHandler) GetHoldingTypes(c *echo.Context) error {
	types, err := h.holdingService.GetHoldingTypes(c.Request().Context())
	if err != nil {
		return response.InternalServerError(c, "Failed to get holding types", err)
	}

	return response.Success(c, "Holding types fetched successfully", types)
}

func (h *HoldingHandler) GetSummary(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	q := &dto.HoldingSummaryQuery{}
	if m := c.QueryParam("month"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 1 && v <= 12 {
			q.Month = &v
		}
	}
	if y := c.QueryParam("year"); y != "" {
		if v, err := strconv.Atoi(y); err == nil {
			q.Year = &v
		}
	}

	summary, err := h.holdingService.GetSummary(c.Request().Context(), userID, q)
	if err != nil {
		return h.respondHoldingError(c, "Failed to get holdings summary", err)
	}

	return response.Success(c, "Holdings summary fetched successfully", summary)
}

func (h *HoldingHandler) GetTrends(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	q := &dto.HoldingTrendsQuery{}
	if y := c.QueryParam("years"); y != "" {
		for _, ys := range splitComma(y) {
			if v, err := strconv.Atoi(ys); err == nil {
				q.Years = append(q.Years, v)
			}
		}
	}

	trends, err := h.holdingService.GetTrends(c.Request().Context(), userID, q)
	if err != nil {
		return h.respondHoldingError(c, "Failed to get holdings trends", err)
	}

	return response.Success(c, "Holdings trends fetched successfully", trends)
}

func (h *HoldingHandler) CompareMonths(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	now := time.Now()
	q := &dto.HoldingCompareQuery{
		ToMonth: int(now.Month()),
		ToYear:  now.Year(),
	}

	if fm := c.QueryParam("fromMonth"); fm != "" {
		if v, err := strconv.Atoi(fm); err == nil && v >= 1 && v <= 12 {
			q.FromMonth = &v
		}
	}
	if fy := c.QueryParam("fromYear"); fy != "" {
		if v, err := strconv.Atoi(fy); err == nil {
			q.FromYear = &v
		}
	}
	if tm := c.QueryParam("toMonth"); tm != "" {
		if v, err := strconv.Atoi(tm); err == nil && v >= 1 && v <= 12 {
			q.ToMonth = v
		}
	}
	if ty := c.QueryParam("toYear"); ty != "" {
		if v, err := strconv.Atoi(ty); err == nil {
			q.ToYear = v
		}
	}

	switch {
	case q.FromMonth == nil && q.FromYear == nil:
		fromM, fromY := prevMonth(q.ToMonth, q.ToYear)
		q.FromMonth = &fromM
		q.FromYear = &fromY
	case q.FromMonth == nil:
		q.FromMonth = &q.ToMonth
	case q.FromYear == nil:
		q.FromYear = &q.ToYear
	}

	result, err := h.holdingService.CompareMonths(c.Request().Context(), userID, q)
	if err != nil {
		return h.respondHoldingError(c, "Failed to compare months", err)
	}

	return response.Success(c, "Month comparison fetched successfully", result)
}

func (h *HoldingHandler) GetMonthlyData(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	q, err := parseMonthlyQuery(c)
	if err != nil {
		return response.BadRequest(c, "Invalid monthly range", err)
	}

	result, err := h.holdingService.GetMonthlyData(c.Request().Context(), userID, q)
	if err != nil {
		return h.respondHoldingError(c, "Failed to get monthly data", err)
	}

	return response.Success(c, "Holdings monthly data fetched successfully", result)
}

// minHoldingYear/maxHoldingYearAhead bound the year query params so a caller
// can't force the service's month-by-month loop (holding_service.go) to walk
// millions of iterations with a value like startYear=2147483647.
const minHoldingYear = 1990

func parseMonthlyQuery(c *echo.Context) (*dto.HoldingMonthlyQuery, error) {
	now := time.Now()
	startMonth, startYear := int(now.Month()), now.Year()
	endMonth, endYear := int(now.Month()), now.Year()
	maxHoldingYear := now.Year() + 1

	var hasStartMonth, hasStartYear, hasEndMonth, hasEndYear bool

	if sm := c.QueryParam("startMonth"); sm != "" {
		if v, err := strconv.Atoi(sm); err == nil && v >= 1 && v <= 12 {
			startMonth = v
			hasStartMonth = true
		}
	}
	if sy := c.QueryParam("startYear"); sy != "" {
		v, err := strconv.Atoi(sy)
		if err != nil || v < minHoldingYear || v > maxHoldingYear {
			return nil, apperrors.ErrHoldingInvalidRange
		}
		startYear = v
		hasStartYear = true
	}
	if em := c.QueryParam("endMonth"); em != "" {
		if v, err := strconv.Atoi(em); err == nil && v >= 1 && v <= 12 {
			endMonth = v
			hasEndMonth = true
		}
	}
	if ey := c.QueryParam("endYear"); ey != "" {
		v, err := strconv.Atoi(ey)
		if err != nil || v < minHoldingYear || v > maxHoldingYear {
			return nil, apperrors.ErrHoldingInvalidRange
		}
		endYear = v
		hasEndYear = true
	}

	// Fill missing start components with current date.
	if !hasStartMonth {
		startMonth = int(now.Month())
	}
	if !hasStartYear {
		startYear = now.Year()
	}

	// If the end is not fully specified, derive it from the start so the range
	// stays predictable. A completely omitted end means "12 months up to start".
	if !hasEndMonth && !hasEndYear {
		endMonth, endYear = prevNMonths(startMonth, startYear, 11)
	} else {
		if !hasEndMonth {
			endMonth = startMonth
		}
		if !hasEndYear {
			endYear = startYear
		}
	}

	q := &dto.HoldingMonthlyQuery{
		StartMonth: startMonth,
		StartYear:  startYear,
		EndMonth:   endMonth,
		EndYear:    endYear,
	}

	// The service expects Start to be the chronologically latest month and End
	// the oldest. Normalize the endpoints so callers can pass them in any order.
	if q.EndYear > q.StartYear || (q.EndYear == q.StartYear && q.EndMonth > q.StartMonth) {
		q.StartMonth, q.EndMonth = q.EndMonth, q.StartMonth
		q.StartYear, q.EndYear = q.EndYear, q.StartYear
	}

	return q, nil
}

func (h *HoldingHandler) SyncPrices(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	result, err := h.holdingService.SyncPrices(c.Request().Context(), userID)
	if err != nil {
		return h.respondHoldingError(c, "Failed to sync prices", err)
	}

	return response.Success(c, "Prices synced successfully for current month", result)
}

func (h *HoldingHandler) DuplicateHoldings(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	var req dto.DuplicateHoldingRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request format", err)
	}

	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	results, err := h.holdingService.DuplicateHoldings(c.Request().Context(), userID, &req)
	if err != nil {
		return h.respondHoldingError(c, "Failed to duplicate holdings", err)
	}

	return response.Created(c, "Holdings duplicated successfully", results)
}

func splitComma(s string) []string {
	var result []string
	for _, v := range splitStr(s, ",") {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func splitStr(s, sep string) []string {
	var result []string
	for {
		idx := indexOfStr(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func prevMonth(month, year int) (int, int) {
	if month == 1 {
		return 12, year - 1
	}
	return month - 1, year
}

func prevNMonths(month, year, n int) (int, int) {
	for range n {
		if month == 1 {
			month = 12
			year--
		} else {
			month--
		}
	}
	return month, year
}
