package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
	"echobackend/pkg/market"
)

type holdingCache interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

type holdingService struct {
	holdingRepo repository.HoldingRepository
	quoteClient market.QuoteClient
	cache       holdingCache
}

type HoldingService interface {
	GetHoldings(ctx context.Context, userID string, filter *dto.HoldingQueryFilter) ([]model.Holding, error)
	GetHoldingByID(ctx context.Context, id int64, userID string) (*model.Holding, error)
	CreateHolding(ctx context.Context, userID string, req *dto.CreateHoldingRequest) (*model.Holding, error)
	UpdateHolding(ctx context.Context, id int64, userID string, req *dto.UpdateHoldingRequest) (*model.Holding, error)
	DeleteHolding(ctx context.Context, id int64, userID string) error
	GetHoldingTypes(ctx context.Context) ([]model.HoldingType, error)
	GetSummary(ctx context.Context, userID string, q *dto.HoldingSummaryQuery) (*dto.HoldingSummaryResponse, error)
	GetTrends(ctx context.Context, userID string, q *dto.HoldingTrendsQuery) ([]dto.HoldingTrendResponse, error)
	CompareMonths(ctx context.Context, userID string, q *dto.HoldingCompareQuery) (*dto.HoldingMonthComparisonResponse, error)
	GetMonthlyData(ctx context.Context, userID string, q *dto.HoldingMonthlyQuery) ([]dto.HoldingMonthlyDataResponse, error)
	SyncPrices(ctx context.Context, userID string) (*dto.HoldingSyncResponse, error)
	DuplicateHoldings(ctx context.Context, userID string, req *dto.DuplicateHoldingRequest) ([]dto.DuplicateResultItem, error)
}

func NewHoldingService(holdingRepo repository.HoldingRepository, quoteClient market.QuoteClient, cache ...holdingCache) HoldingService {
	var c holdingCache
	if len(cache) > 0 {
		c = cache[0]
	}
	return &holdingService{
		holdingRepo: holdingRepo,
		quoteClient: quoteClient,
		cache:       c,
	}
}

func (s *holdingService) GetHoldings(ctx context.Context, userID string, filter *dto.HoldingQueryFilter) ([]model.Holding, error) {
	repoFilter := &struct {
		Month     *int
		Year      *int
		SortBy    string
		SortOrder string
	}{
		Month:     filter.Month,
		Year:      filter.Year,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	}
	return s.holdingRepo.FindAll(ctx, userID, repoFilter)
}

func (s *holdingService) GetHoldingByID(ctx context.Context, id int64, userID string) (*model.Holding, error) {
	return s.holdingRepo.FindByID(ctx, id, userID)
}

func (s *holdingService) CreateHolding(ctx context.Context, userID string, req *dto.CreateHoldingRequest) (*model.Holding, error) {
	if _, err := s.holdingRepo.FindHoldingTypeByID(ctx, req.HoldingTypeID); err != nil {
		return nil, err
	}

	holding := &model.Holding{
		UserID:         userID,
		Name:           req.Name,
		Symbol:         req.Symbol,
		Platform:       req.Platform,
		HoldingTypeID:  req.HoldingTypeID,
		Currency:       req.Currency,
		InvestedAmount: req.InvestedAmount,
		CurrentValue:   req.CurrentValue,
		Units:          req.Units,
		AvgBuyPrice:    req.AvgBuyPrice,
		CurrentPrice:   req.CurrentPrice,
		Notes:          req.Notes,
		Month:          req.Month,
		Year:           req.Year,
	}

	if req.LastUpdated != nil {
		t, err := time.Parse(time.RFC3339, *req.LastUpdated)
		if err == nil {
			holding.LastUpdated = &t
		}
	}

	if err := s.holdingRepo.Create(ctx, holding); err != nil {
		return nil, err
	}

	return s.holdingRepo.FindByID(ctx, holding.ID, userID)
}

func (s *holdingService) UpdateHolding(ctx context.Context, id int64, userID string, req *dto.UpdateHoldingRequest) (*model.Holding, error) {
	existing, err := s.holdingRepo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Symbol != nil {
		existing.Symbol = req.Symbol
	}
	if req.Platform != nil {
		existing.Platform = *req.Platform
	}
	if req.HoldingTypeID != nil {
		if _, err := s.holdingRepo.FindHoldingTypeByID(ctx, *req.HoldingTypeID); err != nil {
			return nil, err
		}
		existing.HoldingTypeID = *req.HoldingTypeID
	}
	if req.Currency != nil {
		existing.Currency = *req.Currency
	}
	if req.InvestedAmount != nil {
		existing.InvestedAmount = *req.InvestedAmount
	}
	if req.CurrentValue != nil {
		existing.CurrentValue = *req.CurrentValue
	}
	if req.Units != nil {
		existing.Units = req.Units
	}
	if req.AvgBuyPrice != nil {
		existing.AvgBuyPrice = req.AvgBuyPrice
	}
	if req.CurrentPrice != nil {
		existing.CurrentPrice = req.CurrentPrice
	}
	if req.Notes != nil {
		existing.Notes = req.Notes
	}
	if req.Month != nil {
		existing.Month = *req.Month
	}
	if req.Year != nil {
		existing.Year = *req.Year
	}
	if req.LastUpdated != nil {
		t, err := time.Parse(time.RFC3339, *req.LastUpdated)
		if err == nil {
			existing.LastUpdated = &t
		}
	}

	now := time.Now()
	existing.UpdatedAt = now

	if err := s.holdingRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return s.holdingRepo.FindByID(ctx, id, userID)
}

func (s *holdingService) DeleteHolding(ctx context.Context, id int64, userID string) error {
	return s.holdingRepo.Delete(ctx, id, userID)
}

func (s *holdingService) GetHoldingTypes(ctx context.Context) ([]model.HoldingType, error) {
	if s.cache != nil {
		var cached []model.HoldingType
		key := s.cache.BuildKey("holding-types")
		if ok, err := s.cache.GetJSON(ctx, key, &cached); err == nil && ok {
			return cached, nil
		}
	}

	types, err := s.holdingRepo.FindHoldingTypes(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		key := s.cache.BuildKey("holding-types")
		_ = s.cache.SetJSONWithTTL(ctx, key, types, 24*time.Hour)
	}

	return types, nil
}

func (s *holdingService) GetSummary(ctx context.Context, userID string, q *dto.HoldingSummaryQuery) (*dto.HoldingSummaryResponse, error) {
	invested, current, count, err := s.holdingRepo.GetSummary(ctx, userID, q.Month, q.Year)
	if err != nil {
		return nil, err
	}

	profitLoss := current - invested
	profitLossPct := calcPercent(invested, current)

	typeBreakdown, err := s.buildTypeBreakdown(ctx, userID, q.Month, q.Year)
	if err != nil {
		return nil, err
	}
	platformBreakdown, err := s.buildPlatformBreakdown(ctx, userID, q.Month, q.Year)
	if err != nil {
		return nil, err
	}

	return &dto.HoldingSummaryResponse{
		TotalInvested:             formatFloat(invested),
		TotalCurrentValue:         formatFloat(current),
		TotalProfitLoss:           formatFloat(profitLoss),
		TotalProfitLossPercentage: profitLossPct,
		HoldingsCount:             count,
		TypeBreakdown:             typeBreakdown,
		PlatformBreakdown:         platformBreakdown,
	}, nil
}

func (s *holdingService) GetTrends(ctx context.Context, userID string, q *dto.HoldingTrendsQuery) ([]dto.HoldingTrendResponse, error) {
	var years []int
	if q != nil && len(q.Years) > 0 {
		years = q.Years
	}

	data, err := s.holdingRepo.GetTrends(ctx, userID, years)
	if err != nil {
		return nil, err
	}

	var result []dto.HoldingTrendResponse
	for _, d := range data {
		pl := d.CurrentValue - d.Invested
		result = append(result, dto.HoldingTrendResponse{
			Date:                 fmt.Sprintf("%04d-%02d", d.Year, d.Month),
			Invested:             formatFloat(d.Invested),
			Current:              formatFloat(d.CurrentValue),
			ProfitLoss:           formatFloat(pl),
			ProfitLossPercentage: calcPercent(d.Invested, d.CurrentValue),
		})
	}
	return result, nil
}

func (s *holdingService) CompareMonths(ctx context.Context, userID string, q *dto.HoldingCompareQuery) (*dto.HoldingMonthComparisonResponse, error) {
	fromSummary, err := s.GetSummary(ctx, userID, &dto.HoldingSummaryQuery{Month: q.FromMonth, Year: q.FromYear})
	if err != nil {
		return nil, err
	}
	toSummary, err := s.GetSummary(ctx, userID, &dto.HoldingSummaryQuery{Month: &q.ToMonth, Year: &q.ToYear})
	if err != nil {
		return nil, err
	}

	fromInvested, _ := strconv.ParseFloat(fromSummary.TotalInvested, 64)
	fromCurrent, _ := strconv.ParseFloat(fromSummary.TotalCurrentValue, 64)
	toInvested, _ := strconv.ParseFloat(toSummary.TotalInvested, 64)
	toCurrent, _ := strconv.ParseFloat(toSummary.TotalCurrentValue, 64)

	fromPL, _ := strconv.ParseFloat(fromSummary.TotalProfitLoss, 64)
	toPL, _ := strconv.ParseFloat(toSummary.TotalProfitLoss, 64)

	fm := *q.FromMonth
	fy := *q.FromYear

	typeComp, err := s.buildTypeCompareBreakdown(ctx, userID, q.FromMonth, q.FromYear, &q.ToMonth, &q.ToYear)
	if err != nil {
		return nil, err
	}
	platformComp, err := s.buildPlatformCompareBreakdown(ctx, userID, q.FromMonth, q.FromYear, &q.ToMonth, &q.ToYear)
	if err != nil {
		return nil, err
	}

	return &dto.HoldingMonthComparisonResponse{
		FromMonth: dto.HoldingMonthPoint{Month: fm, Year: fy},
		ToMonth:   dto.HoldingMonthPoint{Month: q.ToMonth, Year: q.ToYear},
		Summary: dto.HoldingCompareSummary{
			From:                        toSummaryValues(fromSummary),
			To:                          toSummaryValues(toSummary),
			InvestedDiff:                toInvested - fromInvested,
			CurrentValueDiff:            toCurrent - fromCurrent,
			ProfitLossDiff:              toPL - fromPL,
			HoldingsCountDiff:           toSummary.HoldingsCount - fromSummary.HoldingsCount,
			InvestedDiffPercentage:      calcPercentNumber(fromInvested, toInvested),
			CurrentValueDiffPercentage:  calcPercentNumber(fromCurrent, toCurrent),
			HoldingsCountDiffPercentage: calcPercentIntNumber(fromSummary.HoldingsCount, toSummary.HoldingsCount),
		},
		TypeComparison:     typeComp,
		PlatformComparison: platformComp,
	}, nil
}

func (s *holdingService) GetMonthlyData(ctx context.Context, userID string, q *dto.HoldingMonthlyQuery) ([]dto.HoldingMonthlyDataResponse, error) {
	// The iteration walks from (EndMonth, EndYear) FORWARD to (StartMonth, StartYear),
	// so End must be on or before Start chronologically. Reject inverted ranges up
	// front — without this guard the loop below never terminates and burns CPU/RAM.
	if q.EndYear > q.StartYear || (q.EndYear == q.StartYear && q.EndMonth > q.StartMonth) {
		return nil, apperrors.ErrHoldingInvalidRange
	}

	data, err := s.holdingRepo.GetMonthlyData(ctx, userID, q.StartMonth, q.StartYear, q.EndMonth, q.EndYear)
	if err != nil {
		return nil, err
	}

	dataMap := make(map[string]struct {
		Month        int
		Year         int
		Invested     float64
		CurrentValue float64
		Count        int64
	})
	for _, d := range data {
		key := fmt.Sprintf("%04d-%02d", d.Year, d.Month)
		dataMap[key] = d
	}

	var result []dto.HoldingMonthlyDataResponse
	sm, sy := q.EndMonth, q.EndYear
	em, ey := q.StartMonth, q.StartYear

	curM, curY := sm, sy
	for {
		key := fmt.Sprintf("%04d-%02d", curY, curM)
		if d, ok := dataMap[key]; ok {
			result = append(result, dto.HoldingMonthlyDataResponse{
				Month:             d.Month,
				Year:              d.Year,
				Date:              key,
				TotalCurrentValue: formatFloat(d.CurrentValue),
				TotalInvested:     formatFloat(d.Invested),
				HoldingsCount:     d.Count,
			})
		} else {
			result = append(result, dto.HoldingMonthlyDataResponse{
				Month:             curM,
				Year:              curY,
				Date:              key,
				TotalCurrentValue: "0",
				TotalInvested:     "0",
				HoldingsCount:     0,
			})
		}

		if curM == em && curY == ey {
			break
		}
		curM++
		if curM > 12 {
			curM = 1
			curY++
		}
	}

	return result, nil
}

func (s *holdingService) SyncPrices(ctx context.Context, userID string) (*dto.HoldingSyncResponse, error) {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	holdings, err := s.holdingRepo.FindForSync(ctx, userID, month, year)
	if err != nil {
		return nil, err
	}

	if len(holdings) == 0 {
		return &dto.HoldingSyncResponse{
			SyncedCount: 0,
			Month:       month,
			Year:        year,
		}, nil
	}

	symbols := make([]string, 0, len(holdings))
	for _, holding := range holdings {
		if holding.Symbol == nil {
			continue
		}
		symbols = append(symbols, *holding.Symbol)
	}

	quotes, err := s.quoteClient.GetQuotes(ctx, symbols)
	if err != nil {
		return nil, err
	}

	var syncedCount int64
	for _, holding := range holdings {
		if holding.Symbol == nil || holding.Units == nil {
			continue
		}

		symbol := strings.ToUpper(strings.TrimSpace(*holding.Symbol))
		price, ok := quotes[symbol]
		if !ok || price <= 0 {
			continue
		}

		units, err := strconv.ParseFloat(strings.TrimSpace(*holding.Units), 64)
		if err != nil || units <= 0 {
			continue
		}

		currentValue := math.Round(units*price*100) / 100
		if err := s.holdingRepo.UpdateFields(ctx, holding.ID, userID, map[string]any{
			"current_price": formatHoldingPrice(price),
			"current_value": formatHoldingValue(currentValue),
			"last_updated":  now,
			"updated_at":    now,
		}); err != nil {
			return nil, err
		}
		syncedCount++
	}

	return &dto.HoldingSyncResponse{
		SyncedCount: syncedCount,
		Month:       month,
		Year:        year,
	}, nil
}

func (s *holdingService) DuplicateHoldings(ctx context.Context, userID string, req *dto.DuplicateHoldingRequest) ([]dto.DuplicateResultItem, error) {
	if req.FromMonth == req.ToMonth && req.FromYear == req.ToYear {
		return nil, apperrors.ErrHoldingDuplicateSame
	}

	holdings, err := s.holdingRepo.FindForDuplicate(ctx, userID, req.FromMonth, req.FromYear)
	if err != nil {
		return nil, err
	}

	if len(holdings) == 0 {
		return nil, apperrors.ErrHoldingNotFound
	}

	if req.Overwrite {
		if err := s.holdingRepo.DeleteByUserMonthYear(ctx, userID, req.ToMonth, req.ToYear); err != nil {
			return nil, err
		}
	}

	var results []dto.DuplicateResultItem
	for _, h := range holdings {
		newHolding := &model.Holding{
			UserID:         userID,
			Name:           h.Name,
			Symbol:         h.Symbol,
			Platform:       h.Platform,
			HoldingTypeID:  h.HoldingTypeID,
			Currency:       h.Currency,
			InvestedAmount: h.InvestedAmount,
			CurrentValue:   h.CurrentValue,
			Units:          h.Units,
			AvgBuyPrice:    h.AvgBuyPrice,
			CurrentPrice:   h.CurrentPrice,
			Notes:          h.Notes,
			Month:          req.ToMonth,
			Year:           req.ToYear,
		}
		if err := s.holdingRepo.Create(ctx, newHolding); err != nil {
			return nil, err
		}
		results = append(results, dto.DuplicateResultItem{
			ID:    strconv.FormatInt(newHolding.ID, 10),
			Name:  newHolding.Name,
			Month: newHolding.Month,
			Year:  newHolding.Year,
		})
	}

	return results, nil
}

func (s *holdingService) buildTypeBreakdown(ctx context.Context, userID string, month, year *int) ([]dto.HoldingTypeBreakdown, error) {
	data, err := s.holdingRepo.GetTypeBreakdown(ctx, userID, month, year)
	if err != nil {
		return nil, err
	}
	var result []dto.HoldingTypeBreakdown
	for _, d := range data {
		pl := d.CurrentValue - d.Invested
		result = append(result, dto.HoldingTypeBreakdown{
			Name:                 d.Name,
			Invested:             formatFloat(d.Invested),
			Current:              formatFloat(d.CurrentValue),
			ProfitLoss:           formatFloat(pl),
			ProfitLossPercentage: calcPercent(d.Invested, d.CurrentValue),
		})
	}
	return result, nil
}

func (s *holdingService) buildPlatformBreakdown(ctx context.Context, userID string, month, year *int) ([]dto.HoldingPlatformBreakdown, error) {
	data, err := s.holdingRepo.GetPlatformBreakdown(ctx, userID, month, year)
	if err != nil {
		return nil, err
	}
	var result []dto.HoldingPlatformBreakdown
	for _, d := range data {
		pl := d.CurrentValue - d.Invested
		result = append(result, dto.HoldingPlatformBreakdown{
			Name:                 d.Name,
			Invested:             formatFloat(d.Invested),
			Current:              formatFloat(d.CurrentValue),
			ProfitLoss:           formatFloat(pl),
			ProfitLossPercentage: calcPercent(d.Invested, d.CurrentValue),
		})
	}
	return result, nil
}

func buildCompareMap(data []struct {
	Name         string
	Invested     float64
	CurrentValue float64
}) map[string]struct{ Invested, CurrentValue float64 } {
	m := make(map[string]struct{ Invested, CurrentValue float64 })
	for _, d := range data {
		m[d.Name] = struct{ Invested, CurrentValue float64 }{d.Invested, d.CurrentValue}
	}
	return m
}

func buildCompareBreakdownResult(allNames map[string]bool, fromMap, toMap map[string]struct{ Invested, CurrentValue float64 }) []dto.HoldingCompareBreakdown {
	names := make([]string, 0, len(allNames))
	for name := range allNames {
		names = append(names, name)
	}
	sort.Strings(names)

	var result []dto.HoldingCompareBreakdown
	for _, name := range names {
		from := fromMap[name]
		to := toMap[name]
		fromPL := from.CurrentValue - from.Invested
		toPL := to.CurrentValue - to.Invested
		result = append(result, dto.HoldingCompareBreakdown{
			Name: name,
			From: dto.HoldingBreakdownValues{
				Invested:             from.Invested,
				Current:              from.CurrentValue,
				ProfitLoss:           fromPL,
				ProfitLossPercentage: calcPercentNumber(from.Invested, from.CurrentValue),
			},
			To: dto.HoldingBreakdownValues{
				Invested:             to.Invested,
				Current:              to.CurrentValue,
				ProfitLoss:           toPL,
				ProfitLossPercentage: calcPercentNumber(to.Invested, to.CurrentValue),
			},
			InvestedDiff:               to.Invested - from.Invested,
			CurrentValueDiff:           to.CurrentValue - from.CurrentValue,
			ProfitLossDiff:             toPL - fromPL,
			InvestedDiffPercentage:     calcPercentNumber(from.Invested, to.Invested),
			CurrentValueDiffPercentage: calcPercentNumber(from.CurrentValue, to.CurrentValue),
		})
	}
	return result
}

func (s *holdingService) buildTypeCompareBreakdown(ctx context.Context, userID string, fromMonth, fromYear, toMonth, toYear *int) ([]dto.HoldingCompareBreakdown, error) {
	fromData, err := s.holdingRepo.GetTypeBreakdown(ctx, userID, fromMonth, fromYear)
	if err != nil {
		return nil, err
	}
	toData, err := s.holdingRepo.GetTypeBreakdown(ctx, userID, toMonth, toYear)
	if err != nil {
		return nil, err
	}
	fromMap := buildCompareMap(fromData)
	toMap := buildCompareMap(toData)
	allNames := mergeNameKeys(fromMap, toMap)
	return buildCompareBreakdownResult(allNames, fromMap, toMap), nil
}

func (s *holdingService) buildPlatformCompareBreakdown(ctx context.Context, userID string, fromMonth, fromYear, toMonth, toYear *int) ([]dto.HoldingCompareBreakdown, error) {
	fromData, err := s.holdingRepo.GetPlatformBreakdown(ctx, userID, fromMonth, fromYear)
	if err != nil {
		return nil, err
	}
	toData, err := s.holdingRepo.GetPlatformBreakdown(ctx, userID, toMonth, toYear)
	if err != nil {
		return nil, err
	}
	fromMap := buildPlatformCompareMap(fromData)
	toMap := buildPlatformCompareMap(toData)
	allNames := mergeNameKeys(fromMap, toMap)
	return buildCompareBreakdownResult(allNames, fromMap, toMap), nil
}

func buildPlatformCompareMap(data []struct {
	Name         string
	Invested     float64
	CurrentValue float64
}) map[string]struct{ Invested, CurrentValue float64 } {
	m := make(map[string]struct{ Invested, CurrentValue float64 })
	for _, d := range data {
		m[d.Name] = struct{ Invested, CurrentValue float64 }{d.Invested, d.CurrentValue}
	}
	return m
}

func mergeNameKeys(maps ...map[string]struct{ Invested, CurrentValue float64 }) map[string]bool {
	result := make(map[string]bool)
	for _, m := range maps {
		for k := range m {
			result[k] = true
		}
	}
	return result
}

func formatFloat(f float64) string {
	if f == 0 {
		return "0"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func formatHoldingPrice(price float64) string {
	return strconv.FormatFloat(price, 'f', 8, 64)
}

func formatHoldingValue(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func calcPercent(base, value float64) string {
	return formatFloat(calcPercentNumber(base, value))
}

func calcPercentNumber(base, value float64) float64 {
	if base == 0 {
		return 0
	}
	pct := ((value - base) / base) * 100
	return math.Round(pct*100) / 100
}

func calcPercentIntNumber(base, value int64) float64 {
	if base == 0 {
		return 0
	}
	pct := (float64(value-base) / float64(base)) * 100
	return math.Round(pct*100) / 100
}

func parseFloatValue(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func toNamedBreakdowns(types []dto.HoldingTypeBreakdown) []dto.HoldingNamedBreakdown {
	result := make([]dto.HoldingNamedBreakdown, 0, len(types))
	for _, item := range types {
		result = append(result, dto.HoldingNamedBreakdown{
			Name:                 item.Name,
			Invested:             parseFloatValue(item.Invested),
			Current:              parseFloatValue(item.Current),
			ProfitLoss:           parseFloatValue(item.ProfitLoss),
			ProfitLossPercentage: parseFloatValue(item.ProfitLossPercentage),
		})
	}
	return result
}

func toNamedPlatformBreakdowns(platforms []dto.HoldingPlatformBreakdown) []dto.HoldingNamedBreakdown {
	result := make([]dto.HoldingNamedBreakdown, 0, len(platforms))
	for _, item := range platforms {
		result = append(result, dto.HoldingNamedBreakdown{
			Name:                 item.Name,
			Invested:             parseFloatValue(item.Invested),
			Current:              parseFloatValue(item.Current),
			ProfitLoss:           parseFloatValue(item.ProfitLoss),
			ProfitLossPercentage: parseFloatValue(item.ProfitLossPercentage),
		})
	}
	return result
}

func toSummaryValues(s *dto.HoldingSummaryResponse) dto.HoldingSummaryValues {
	return dto.HoldingSummaryValues{
		TotalInvested:             parseFloatValue(s.TotalInvested),
		TotalCurrentValue:         parseFloatValue(s.TotalCurrentValue),
		TotalProfitLoss:           parseFloatValue(s.TotalProfitLoss),
		TotalProfitLossPercentage: parseFloatValue(s.TotalProfitLossPercentage),
		HoldingsCount:             s.HoldingsCount,
		TypeBreakdown:             toNamedBreakdowns(s.TypeBreakdown),
		PlatformBreakdown:         toNamedPlatformBreakdowns(s.PlatformBreakdown),
	}
}
