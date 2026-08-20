package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

// stubQuoteClient implements market.QuoteClient for SyncPrices tests.
type stubQuoteClient struct {
	quotes map[string]float64
	err    error
}

func (s *stubQuoteClient) GetQuotes(ctx context.Context, symbols []string) (map[string]float64, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.quotes, nil
}

// helpers --------------------------------------------------------------------

//go:fix inline
func intPtr(v int) *int { return new(v) }

// CreateHolding --------------------------------------------------------------

func TestHoldingService_CreateHolding_TypeNotFound(t *testing.T) {
	repo := &mockHoldingRepo{
		findHoldingTypeByIDFn: func(ctx context.Context, id int) (*model.HoldingType, error) {
			return nil, apperrors.ErrHoldingTypeNotFound
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	_, err := svc.CreateHolding(context.Background(), "user-1", &dto.CreateHoldingRequest{
		Name: "BBCA", Platform: "Mandiri", HoldingTypeID: 1, Currency: "IDR",
		InvestedAmount: "1000", CurrentValue: "1100", Month: 5, Year: 2026,
	})
	if !errors.Is(err, apperrors.ErrHoldingTypeNotFound) {
		t.Fatalf("expected ErrHoldingTypeNotFound, got %v", err)
	}
}

func TestHoldingService_CreateHolding_Success(t *testing.T) {
	created := false
	repo := &mockHoldingRepo{
		findHoldingTypeByIDFn: func(ctx context.Context, id int) (*model.HoldingType, error) {
			return &model.HoldingType{ID: id, Name: "Stock"}, nil
		},
		createFn: func(ctx context.Context, h *model.Holding) error {
			created = true
			h.ID = 99
			if h.UserID != "user-1" {
				t.Errorf("UserID = %q", h.UserID)
			}
			return nil
		},
		findByIDFn: func(ctx context.Context, id int64, userID string) (*model.Holding, error) {
			if id != 99 || userID != "user-1" {
				t.Errorf("unexpected (%d,%s)", id, userID)
			}
			return &model.Holding{ID: id, UserID: userID, Name: "BBCA"}, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	got, err := svc.CreateHolding(context.Background(), "user-1", &dto.CreateHoldingRequest{
		Name: "BBCA", Platform: "Mandiri", HoldingTypeID: 1, Currency: "IDR",
		InvestedAmount: "1000", CurrentValue: "1100", Month: 5, Year: 2026,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected Create to be called")
	}
	if got == nil || got.ID != 99 {
		t.Fatalf("unexpected holding: %+v", got)
	}
}

// DeleteHolding -------------------------------------------------------------

func TestHoldingService_DeleteHolding(t *testing.T) {
	called := false
	repo := &mockHoldingRepo{
		deleteFn: func(ctx context.Context, id int64, userID string) error {
			called = true
			if id != 5 || userID != "u1" {
				t.Errorf("unexpected (%d,%s)", id, userID)
			}
			return nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	if err := svc.DeleteHolding(context.Background(), 5, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected Delete to be called")
	}
}

// GetSummary ----------------------------------------------------------------

func TestHoldingService_GetSummary_NoData(t *testing.T) {
	repo := &mockHoldingRepo{
		getSummaryFn: func(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error) {
			return 0, 0, 0, nil
		},
		getTypeBreakdownFn: func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
			return nil, nil
		},
		getPlatformBreakdownFn: func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
			return nil, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	resp, err := svc.GetSummary(context.Background(), "u1", &dto.HoldingSummaryQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HoldingsCount != 0 {
		t.Errorf("HoldingsCount = %d", resp.HoldingsCount)
	}
	if resp.TotalProfitLossPercentage != "0" {
		t.Errorf("expected zero percentage, got %q", resp.TotalProfitLossPercentage)
	}
}

func TestHoldingService_GetSummary_WithData(t *testing.T) {
	repo := &mockHoldingRepo{
		getSummaryFn: func(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error) {
			return 1000, 1200, 3, nil
		},
		getTypeBreakdownFn: func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
			return []breakdownRow{{Name: "Stock", Invested: 1000, CurrentValue: 1200}}, nil
		},
		getPlatformBreakdownFn: func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
			return []breakdownRow{{Name: "Mandiri", Invested: 1000, CurrentValue: 1200}}, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	resp, err := svc.GetSummary(context.Background(), "u1", &dto.HoldingSummaryQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HoldingsCount != 3 {
		t.Errorf("HoldingsCount = %d, want 3", resp.HoldingsCount)
	}
	if resp.TotalInvested != "1000" {
		t.Errorf("TotalInvested = %q, want %q", resp.TotalInvested, "1000")
	}
	if resp.TotalCurrentValue != "1200" {
		t.Errorf("TotalCurrentValue = %q, want %q", resp.TotalCurrentValue, "1200")
	}
	if resp.TotalProfitLoss != "200" {
		t.Errorf("TotalProfitLoss = %q, want %q", resp.TotalProfitLoss, "200")
	}
	// (1200-1000)/1000 * 100 = 20%
	if resp.TotalProfitLossPercentage != "20" {
		t.Errorf("TotalProfitLossPercentage = %q, want %q", resp.TotalProfitLossPercentage, "20")
	}
	if len(resp.TypeBreakdown) != 1 || resp.TypeBreakdown[0].Name != "Stock" {
		t.Errorf("TypeBreakdown unexpected: %+v", resp.TypeBreakdown)
	}
}

// SyncPrices ----------------------------------------------------------------

func TestHoldingService_SyncPrices_NoHoldings(t *testing.T) {
	repo := &mockHoldingRepo{
		findForSyncFn: func(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
			return nil, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	resp, err := svc.SyncPrices(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SyncedCount != 0 {
		t.Errorf("SyncedCount = %d, want 0", resp.SyncedCount)
	}
}

func TestHoldingService_SyncPrices_UpdatesQuotes(t *testing.T) {
	holdings := []model.Holding{
		{ID: 1, UserID: "u1", Symbol: new("BBCA.JK"), Units: new("10")},
		{ID: 2, UserID: "u1", Symbol: new("AAPL"), Units: new("5")},
		// Skipped: nil symbol
		{ID: 3, UserID: "u1", Symbol: nil, Units: new("3")},
		// Skipped: nil units
		{ID: 4, UserID: "u1", Symbol: new("MSFT"), Units: nil},
		// Skipped: missing quote
		{ID: 5, UserID: "u1", Symbol: new("UNKNOWN"), Units: new("1")},
	}

	updated := map[int64]map[string]any{}
	repo := &mockHoldingRepo{
		findForSyncFn: func(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
			return holdings, nil
		},
		updateFieldsFn: func(ctx context.Context, id int64, userID string, fields map[string]any) error {
			updated[id] = fields
			return nil
		},
	}
	quote := &stubQuoteClient{quotes: map[string]float64{"BBCA.JK": 6150, "AAPL": 200}}
	svc := NewHoldingService(repo, quote)

	resp, err := svc.SyncPrices(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SyncedCount != 2 {
		t.Errorf("SyncedCount = %d, want 2", resp.SyncedCount)
	}
	if len(updated) != 2 {
		t.Errorf("UpdateFields called %d times, want 2", len(updated))
	}
	if _, ok := updated[1]; !ok {
		t.Errorf("expected holding 1 (BBCA.JK) to be updated")
	}
	if _, ok := updated[2]; !ok {
		t.Errorf("expected holding 2 (AAPL) to be updated")
	}
	if cv, ok := updated[1]["current_value"].(string); !ok || !strings.HasPrefix(cv, "61500") {
		t.Errorf("BBCA.JK current_value = %v, want ~61500.00", updated[1]["current_value"])
	}
}

func TestHoldingService_SyncPrices_QuoteClientError(t *testing.T) {
	wantErr := errors.New("quote client failure")
	repo := &mockHoldingRepo{
		findForSyncFn: func(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
			return []model.Holding{{ID: 1, UserID: "u1", Symbol: new("AAPL"), Units: new("1")}}, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{err: wantErr})
	_, err := svc.SyncPrices(context.Background(), "u1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped err, got %v", err)
	}
}

// DuplicateHoldings ---------------------------------------------------------

func TestHoldingService_DuplicateHoldings_SameMonthYear(t *testing.T) {
	svc := NewHoldingService(&mockHoldingRepo{}, &stubQuoteClient{})
	_, err := svc.DuplicateHoldings(context.Background(), "u1", &dto.DuplicateHoldingRequest{
		FromMonth: 5, FromYear: 2026, ToMonth: 5, ToYear: 2026,
	})
	if !errors.Is(err, apperrors.ErrHoldingDuplicateSame) {
		t.Fatalf("expected ErrHoldingDuplicateSame, got %v", err)
	}
}

func TestHoldingService_DuplicateHoldings_NoSourceData(t *testing.T) {
	repo := &mockHoldingRepo{
		findForDuplicateFn: func(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
			return nil, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	_, err := svc.DuplicateHoldings(context.Background(), "u1", &dto.DuplicateHoldingRequest{
		FromMonth: 4, FromYear: 2026, ToMonth: 5, ToYear: 2026,
	})
	if !errors.Is(err, apperrors.ErrHoldingNotFound) {
		t.Fatalf("expected ErrHoldingNotFound, got %v", err)
	}
}

func TestHoldingService_DuplicateHoldings_OverwriteAndCopy(t *testing.T) {
	deleted := false
	createCount := 0
	repo := &mockHoldingRepo{
		findForDuplicateFn: func(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
			return []model.Holding{
				{ID: 1, UserID: userID, Name: "BBCA", Platform: "Mandiri", HoldingTypeID: 1, Currency: "IDR", InvestedAmount: "1000", CurrentValue: "1100", Month: 4, Year: 2026},
				{ID: 2, UserID: userID, Name: "AAPL", Platform: "IBKR", HoldingTypeID: 1, Currency: "USD", InvestedAmount: "500", CurrentValue: "600", Month: 4, Year: 2026},
			}, nil
		},
		deleteByMonthYearFn: func(ctx context.Context, userID string, month, year int) error {
			deleted = true
			if month != 5 || year != 2026 {
				t.Errorf("unexpected delete (%d,%d)", month, year)
			}
			return nil
		},
		createFn: func(ctx context.Context, h *model.Holding) error {
			createCount++
			if h.Month != 5 || h.Year != 2026 {
				t.Errorf("expected target month/year, got (%d,%d)", h.Month, h.Year)
			}
			h.ID = int64(createCount * 100)
			return nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	results, err := svc.DuplicateHoldings(context.Background(), "u1", &dto.DuplicateHoldingRequest{
		FromMonth: 4, FromYear: 2026, ToMonth: 5, ToYear: 2026, Overwrite: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected DeleteByUserMonthYear to be called when Overwrite=true")
	}
	if createCount != 2 {
		t.Errorf("Create called %d times, want 2", createCount)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Month != 5 || results[0].Year != 2026 {
		t.Errorf("result[0] = %+v", results[0])
	}
}

func TestHoldingService_DuplicateHoldings_NoOverwrite(t *testing.T) {
	repo := &mockHoldingRepo{
		findForDuplicateFn: func(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
			return []model.Holding{{ID: 1, UserID: userID, Name: "BBCA", Month: 4, Year: 2026, Platform: "M", HoldingTypeID: 1, Currency: "IDR", InvestedAmount: "1", CurrentValue: "1"}}, nil
		},
		deleteByMonthYearFn: func(ctx context.Context, userID string, month, year int) error {
			t.Fatal("DeleteByUserMonthYear should not be called when Overwrite=false")
			return nil
		},
		createFn: func(ctx context.Context, h *model.Holding) error {
			h.ID = 1
			return nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	if _, err := svc.DuplicateHoldings(context.Background(), "u1", &dto.DuplicateHoldingRequest{
		FromMonth: 4, FromYear: 2026, ToMonth: 5, ToYear: 2026, Overwrite: false,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// GetTrends -----------------------------------------------------------------

func TestHoldingService_GetTrends(t *testing.T) {
	repo := &mockHoldingRepo{
		getTrendsFn: func(ctx context.Context, userID string, years []int) ([]trendRow, error) {
			return []trendRow{
				{Month: 1, Year: 2026, Invested: 100, CurrentValue: 110},
				{Month: 2, Year: 2026, Invested: 200, CurrentValue: 180},
			}, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	got, err := svc.GetTrends(context.Background(), "u1", &dto.HoldingTrendsQuery{Years: []int{2026}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Date != "2026-01" {
		t.Errorf("Date = %q, want 2026-01", got[0].Date)
	}
	if got[0].ProfitLoss != "10" {
		t.Errorf("ProfitLoss = %q", got[0].ProfitLoss)
	}
}

// Pure helper tests ---------------------------------------------------------

func TestCalcPercentNumber(t *testing.T) {
	tests := []struct {
		base, value float64
		want        float64
	}{
		{0, 100, 0},      // base=0 -> 0
		{1000, 1200, 20}, // 20% gain
		{1000, 800, -20}, // 20% loss
		{100, 100, 0},    // no change
	}
	for _, tt := range tests {
		got := calcPercentNumber(tt.base, tt.value)
		if got != tt.want {
			t.Errorf("calcPercentNumber(%v,%v) = %v, want %v", tt.base, tt.value, got, tt.want)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1.5, "1.5"},
		{100, "100"},
		{-3.14, "-3.14"},
	}
	for _, tt := range tests {
		if got := formatFloat(tt.in); got != tt.want {
			t.Errorf("formatFloat(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseFloatValue(t *testing.T) {
	if parseFloatValue("12.34") != 12.34 {
		t.Errorf("parseFloatValue valid number failed")
	}
	if parseFloatValue("not a number") != 0 {
		t.Errorf("parseFloatValue invalid input should return 0")
	}
	if parseFloatValue("") != 0 {
		t.Errorf("parseFloatValue empty should return 0")
	}
}

// Quick sanity check for monthly bounds — exercises the iteration loop in
// GetMonthlyData without actually hitting the DB.
//
// NOTE: the service's loop starts at EndMonth/EndYear and walks FORWARD to
// StartMonth/StartYear (i.e. End must be chronologically earlier than Start),
// so we deliberately set StartMonth=5 / EndMonth=3 here.
func TestHoldingService_GetMonthlyData_FillsGaps(t *testing.T) {
	repo := &mockHoldingRepo{
		getMonthlyDataFn: func(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error) {
			// Return only one of the three expected months; service should fill gaps with zeros.
			return []monthlyRow{{Month: 4, Year: 2026, Invested: 100, CurrentValue: 110, Count: 2}}, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	got, err := svc.GetMonthlyData(context.Background(), "u1", &dto.HoldingMonthlyQuery{
		StartMonth: 5, StartYear: 2026,
		EndMonth: 3, EndYear: 2026,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	wantDates := map[string]bool{"2026-05": true, "2026-04": true, "2026-03": true}
	for _, r := range got {
		if !wantDates[r.Date] {
			t.Errorf("unexpected date: %q", r.Date)
		}
		delete(wantDates, r.Date)
	}
	if len(wantDates) != 0 {
		t.Errorf("missing dates: %v", wantDates)
	}
}

// Regression test: an inverted range (End AFTER Start) used to drive the loop
// into infinite iteration, exhausting CPU/RAM. The service must reject it.
func TestHoldingService_GetMonthlyData_InvertedRangeRejected(t *testing.T) {
	repo := &mockHoldingRepo{
		getMonthlyDataFn: func(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error) {
			t.Fatal("repository should not be called for an invalid range")
			return nil, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})

	cases := []struct {
		name string
		q    dto.HoldingMonthlyQuery
	}{
		{"end month after start, same year", dto.HoldingMonthlyQuery{StartMonth: 3, StartYear: 2026, EndMonth: 5, EndYear: 2026}},
		{"end year after start year", dto.HoldingMonthlyQuery{StartMonth: 1, StartYear: 2025, EndMonth: 1, EndYear: 2026}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetMonthlyData(context.Background(), "u1", &tc.q)
			if !errors.Is(err, apperrors.ErrHoldingInvalidRange) {
				t.Fatalf("expected ErrHoldingInvalidRange, got %v", err)
			}
		})
	}
}

// Equal Start and End is a valid single-month query and must not be rejected.
func TestHoldingService_GetMonthlyData_EqualRange(t *testing.T) {
	repo := &mockHoldingRepo{
		getMonthlyDataFn: func(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error) {
			return []monthlyRow{{Month: 5, Year: 2026, Invested: 1, CurrentValue: 2, Count: 1}}, nil
		},
	}
	svc := NewHoldingService(repo, &stubQuoteClient{})
	got, err := svc.GetMonthlyData(context.Background(), "u1", &dto.HoldingMonthlyQuery{
		StartMonth: 5, StartYear: 2026, EndMonth: 5, EndYear: 2026,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Date != "2026-05" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

// Make sure unused intPtr helper is referenced so the file compiles cleanly.
var _ = intPtr

type fakeHoldingCache struct {
	buildKeyCalls []string
	getJSONCalls  []string
	setJSONCalls  []string

	keyToReturn string
	getOK       bool
	getData     []model.HoldingType
	getErr      error

	setErr error
}

func (f *fakeHoldingCache) BuildKey(parts ...string) string {
	f.buildKeyCalls = append(f.buildKeyCalls, strings.Join(parts, ":"))
	return f.keyToReturn
}

func (f *fakeHoldingCache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	f.getJSONCalls = append(f.getJSONCalls, key)
	if f.getOK && f.getErr == nil {
		if d, ok := dest.(*[]model.HoldingType); ok {
			*d = f.getData
		}
	}
	return f.getOK, f.getErr
}

func (f *fakeHoldingCache) SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	f.setJSONCalls = append(f.setJSONCalls, key)
	return f.setErr
}

func TestHoldingService_GetHoldingTypes_CacheHit(t *testing.T) {
	cache := &fakeHoldingCache{
		keyToReturn: "holding-types",
		getOK:       true,
		getData: []model.HoldingType{
			{ID: 1, Name: "Stock"},
			{ID: 2, Name: "Crypto"},
		},
	}
	// Repository should not be called at all
	repo := &mockHoldingRepo{
		findHoldingTypesFn: func(ctx context.Context) ([]model.HoldingType, error) {
			t.Fatal("repository findHoldingTypesFn should not have been called on cache hit")
			return nil, nil
		},
	}

	svc := NewHoldingService(repo, &stubQuoteClient{}, cache)
	types, err := svc.GetHoldingTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(types) != 2 || types[0].Name != "Stock" || types[1].Name != "Crypto" {
		t.Fatalf("unexpected types returned: %v", types)
	}

	if len(cache.getJSONCalls) != 1 || cache.getJSONCalls[0] != "holding-types" {
		t.Fatalf("unexpected getJSON calls: %v", cache.getJSONCalls)
	}
	if len(cache.setJSONCalls) != 0 {
		t.Fatalf("unexpected setJSON calls on cache hit: %v", cache.setJSONCalls)
	}
}

func TestHoldingService_GetHoldingTypes_CacheMiss(t *testing.T) {
	cache := &fakeHoldingCache{
		keyToReturn: "holding-types",
		getOK:       false,
	}
	repoCalled := false
	repo := &mockHoldingRepo{
		findHoldingTypesFn: func(ctx context.Context) ([]model.HoldingType, error) {
			repoCalled = true
			return []model.HoldingType{
				{ID: 3, Name: "Cash"},
			}, nil
		},
	}

	svc := NewHoldingService(repo, &stubQuoteClient{}, cache)
	types, err := svc.GetHoldingTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repoCalled {
		t.Fatal("expected repository findHoldingTypesFn to be called on cache miss")
	}

	if len(types) != 1 || types[0].Name != "Cash" {
		t.Fatalf("unexpected types returned: %v", types)
	}

	if len(cache.getJSONCalls) != 1 || cache.getJSONCalls[0] != "holding-types" {
		t.Fatalf("unexpected getJSON calls: %v", cache.getJSONCalls)
	}
	if len(cache.setJSONCalls) != 1 || cache.setJSONCalls[0] != "holding-types" {
		t.Fatalf("expected setJSON call on cache miss: %v", cache.setJSONCalls)
	}
}

func TestHoldingService_CompareMonths_DeterministicSorting(t *testing.T) {
	fromMonth, fromYear := 1, 2026
	toMonth, toYear := 2, 2026

	repo := &mockHoldingRepo{
		getSummaryFn: func(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error) {
			return 1000, 1500, 2, nil
		},
		getTypeBreakdownFn: func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
			// Deliberately unsorted in repo output
			return []breakdownRow{
				{Name: "Zebra Stock", Invested: 100, CurrentValue: 120},
				{Name: "Crypto", Invested: 400, CurrentValue: 600},
				{Name: "Alpha ETF", Invested: 500, CurrentValue: 780},
			}, nil
		},
		getPlatformBreakdownFn: func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
			return []breakdownRow{
				{Name: "Stockbit", Invested: 500, CurrentValue: 700},
				{Name: "Bibit", Invested: 500, CurrentValue: 800},
			}, nil
		},
	}

	svc := NewHoldingService(repo, &stubQuoteClient{})
	resp, err := svc.CompareMonths(context.Background(), "u1", &dto.HoldingCompareQuery{
		FromMonth: &fromMonth,
		FromYear:  &fromYear,
		ToMonth:   toMonth,
		ToYear:    toYear,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify TypeComparison is strictly sorted: Alpha ETF, Crypto, Zebra Stock
	if len(resp.TypeComparison) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.TypeComparison))
	}
	if resp.TypeComparison[0].Name != "Alpha ETF" ||
		resp.TypeComparison[1].Name != "Crypto" ||
		resp.TypeComparison[2].Name != "Zebra Stock" {
		t.Fatalf("unexpected TypeComparison sort order: %+v", resp.TypeComparison)
	}

	// Verify PlatformComparison is strictly sorted: Bibit, Stockbit
	if len(resp.PlatformComparison) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.PlatformComparison))
	}
	if resp.PlatformComparison[0].Name != "Bibit" ||
		resp.PlatformComparison[1].Name != "Stockbit" {
		t.Fatalf("unexpected PlatformComparison sort order: %+v", resp.PlatformComparison)
	}
}
