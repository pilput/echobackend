package portfolio

import "time"

type CreateHoldingRequest struct {
	Name           string  `json:"name" validate:"required"`
	Symbol         *string `json:"symbol"`
	Platform       string  `json:"platform" validate:"required"`
	HoldingTypeID  int     `json:"holding_type_id" validate:"required"`
	Currency       string  `json:"currency" validate:"required,len=3"`
	InvestedAmount string  `json:"invested_amount" validate:"required"`
	CurrentValue   string  `json:"current_value" validate:"required"`
	Units          *string `json:"units"`
	AvgBuyPrice    *string `json:"avg_buy_price"`
	CurrentPrice   *string `json:"current_price"`
	LastUpdated    *string `json:"last_updated"`
	Notes          *string `json:"notes"`
	Month          int     `json:"month" validate:"required,min=1,max=12"`
	Year           int     `json:"year" validate:"required,min=2000"`
}

type UpdateHoldingRequest struct {
	Name           *string `json:"name"`
	Symbol         *string `json:"symbol"`
	Platform       *string `json:"platform"`
	HoldingTypeID  *int    `json:"holding_type_id"`
	Currency       *string `json:"currency" validate:"omitempty,len=3"`
	InvestedAmount *string `json:"invested_amount"`
	CurrentValue   *string `json:"current_value"`
	Units          *string `json:"units"`
	AvgBuyPrice    *string `json:"avg_buy_price"`
	CurrentPrice   *string `json:"current_price"`
	LastUpdated    *string `json:"last_updated"`
	Notes          *string `json:"notes"`
	Month          *int    `json:"month" validate:"omitempty,min=1,max=12"`
	Year           *int    `json:"year" validate:"omitempty,min=2000"`
}

type DuplicateHoldingRequest struct {
	FromMonth int  `json:"fromMonth" validate:"required,min=1,max=12"`
	FromYear  int  `json:"fromYear" validate:"required,min=1900,max=2100"`
	ToMonth   int  `json:"toMonth" validate:"required,min=1,max=12"`
	ToYear    int  `json:"toYear" validate:"required,min=1900,max=2100"`
	Overwrite bool `json:"overwrite"`
}

type HoldingQueryFilter struct {
	Month     *int
	Year      *int
	SortBy    string
	SortOrder string
}

type HoldingSummaryQuery struct {
	Month *int
	Year  *int
}

type HoldingTrendsQuery struct {
	Years []int
}

type HoldingCompareQuery struct {
	FromMonth *int
	FromYear  *int
	ToMonth   int
	ToYear    int
}

type HoldingMonthlyQuery struct {
	StartMonth int
	StartYear  int
	EndMonth   int
	EndYear    int
}

type HoldingSummaryResponse struct {
	TotalInvested             string                     `json:"totalInvested"`
	TotalCurrentValue         string                     `json:"totalCurrentValue"`
	TotalProfitLoss           string                     `json:"totalProfitLoss"`
	TotalProfitLossPercentage string                     `json:"totalProfitLossPercentage"`
	HoldingsCount             int64                      `json:"holdingsCount"`
	TypeBreakdown             []HoldingTypeBreakdown     `json:"typeBreakdown"`
	PlatformBreakdown         []HoldingPlatformBreakdown `json:"platformBreakdown"`
}

type HoldingTypeBreakdown struct {
	Name                 string `json:"name"`
	Invested             string `json:"invested"`
	Current              string `json:"current"`
	ProfitLoss           string `json:"profitLoss"`
	ProfitLossPercentage string `json:"profitLossPercentage"`
}

type HoldingPlatformBreakdown struct {
	Name                 string `json:"name"`
	Invested             string `json:"invested"`
	Current              string `json:"current"`
	ProfitLoss           string `json:"profitLoss"`
	ProfitLossPercentage string `json:"profitLossPercentage"`
}

type HoldingTrendResponse struct {
	Date                 string `json:"date"`
	Invested             string `json:"invested"`
	Current              string `json:"current"`
	ProfitLoss           string `json:"profitLoss"`
	ProfitLossPercentage string `json:"profitLossPercentage"`
}

type HoldingMonthComparisonResponse struct {
	FromMonth          HoldingMonthPoint         `json:"fromMonth"`
	ToMonth            HoldingMonthPoint         `json:"toMonth"`
	Summary            HoldingCompareSummary     `json:"summary"`
	TypeComparison     []HoldingCompareBreakdown `json:"typeComparison"`
	PlatformComparison []HoldingCompareBreakdown `json:"platformComparison"`
}

type HoldingMonthPoint struct {
	Month int `json:"month"`
	Year  int `json:"year"`
}

type HoldingCompareSummary struct {
	From                        HoldingSummaryValues `json:"from"`
	To                          HoldingSummaryValues `json:"to"`
	InvestedDiff                float64              `json:"investedDiff"`
	CurrentValueDiff            float64              `json:"currentValueDiff"`
	ProfitLossDiff              float64              `json:"profitLossDiff"`
	HoldingsCountDiff           int64                `json:"holdingsCountDiff"`
	InvestedDiffPercentage      float64              `json:"investedDiffPercentage"`
	CurrentValueDiffPercentage  float64              `json:"currentValueDiffPercentage"`
	HoldingsCountDiffPercentage float64              `json:"holdingsCountDiffPercentage"`
}

type HoldingSummaryValues struct {
	TotalInvested             float64                 `json:"totalInvested"`
	TotalCurrentValue         float64                 `json:"totalCurrentValue"`
	TotalProfitLoss           float64                 `json:"totalProfitLoss"`
	TotalProfitLossPercentage float64                 `json:"totalProfitLossPercentage"`
	HoldingsCount             int64                   `json:"holdingsCount"`
	TypeBreakdown             []HoldingNamedBreakdown `json:"typeBreakdown"`
	PlatformBreakdown         []HoldingNamedBreakdown `json:"platformBreakdown"`
}

type HoldingNamedBreakdown struct {
	Name                 string  `json:"name"`
	Invested             float64 `json:"invested"`
	Current              float64 `json:"current"`
	ProfitLoss           float64 `json:"profitLoss"`
	ProfitLossPercentage float64 `json:"profitLossPercentage"`
}

type HoldingCompareBreakdown struct {
	Name                       string                 `json:"name"`
	From                       HoldingBreakdownValues `json:"from"`
	To                         HoldingBreakdownValues `json:"to"`
	InvestedDiff               float64                `json:"investedDiff"`
	CurrentValueDiff           float64                `json:"currentValueDiff"`
	ProfitLossDiff             float64                `json:"profitLossDiff"`
	InvestedDiffPercentage     float64                `json:"investedDiffPercentage"`
	CurrentValueDiffPercentage float64                `json:"currentValueDiffPercentage"`
}

type HoldingBreakdownValues struct {
	Invested             float64 `json:"invested"`
	Current              float64 `json:"current"`
	ProfitLoss           float64 `json:"profitLoss"`
	ProfitLossPercentage float64 `json:"profitLossPercentage"`
}

type HoldingMonthlyDataResponse struct {
	Month             int    `json:"month"`
	Year              int    `json:"year"`
	Date              string `json:"date"`
	TotalCurrentValue string `json:"totalCurrentValue"`
	TotalInvested     string `json:"totalInvested"`
	HoldingsCount     int64  `json:"holdingsCount"`
}

type HoldingSyncResponse struct {
	SyncedCount int64 `json:"syncedCount"`
	Month       int   `json:"month"`
	Year        int   `json:"year"`
}

type HoldingTypeResponse struct {
	ID    int     `json:"id"`
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Notes *string `json:"notes"`
}

type DuplicateResultItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Month int    `json:"month"`
	Year  int    `json:"year"`
}

type MonthYearBucket struct {
	Month int
	Year  int
}

type MonthlyAggregate struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
	Count        int64
}

type TypePlatformAggregate struct {
	Name         string
	Invested     float64
	CurrentValue float64
	Count        int64
}

func FormatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339Nano)
	return &s
}
