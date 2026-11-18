package portfolio

// Hand-written mocks for the portfolio service tests; see
// internal/service/mocks_test.go for the conventions.

import "context"

var _ HoldingRepository = (*mockHoldingRepo)(nil)

// ---- HoldingRepository mock ---------------------------------------------------

type holdingFilter = struct {
	Month     *int
	Year      *int
	SortBy    string
	SortOrder string
}

type breakdownRow = struct {
	Name         string
	Invested     float64
	CurrentValue float64
}

type trendRow = struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
}

type monthlyRow = struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
	Count        int64
}

type mockHoldingRepo struct {
	findAllFn              func(ctx context.Context, userID string, filter *holdingFilter) ([]Holding, error)
	findByIDFn             func(ctx context.Context, id int64, userID string) (*Holding, error)
	createFn               func(ctx context.Context, h *Holding) error
	updateFn               func(ctx context.Context, h *Holding) error
	deleteFn               func(ctx context.Context, id int64, userID string) error
	findHoldingTypesFn     func(ctx context.Context) ([]HoldingType, error)
	findHoldingTypeByIDFn  func(ctx context.Context, id int) (*HoldingType, error)
	findForSyncFn          func(ctx context.Context, userID string, month, year int) ([]Holding, error)
	updateFieldsFn         func(ctx context.Context, id int64, userID string, fields map[string]any) error
	findForDuplicateFn     func(ctx context.Context, userID string, month, year int) ([]Holding, error)
	deleteByMonthYearFn    func(ctx context.Context, userID string, month, year int) error
	countByMonthYearFn     func(ctx context.Context, userID string, month, year int) (int64, error)
	getSummaryFn           func(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error)
	getTypeBreakdownFn     func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error)
	getPlatformBreakdownFn func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error)
	getTrendsFn            func(ctx context.Context, userID string, years []int) ([]trendRow, error)
	getMonthlyDataFn       func(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error)
	findStockSymbolsFn     func(ctx context.Context, userID string) ([]string, error)
}

func (m *mockHoldingRepo) FindAll(ctx context.Context, userID string, filter *holdingFilter) ([]Holding, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx, userID, filter)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindByID(ctx context.Context, id int64, userID string) (*Holding, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id, userID)
	}
	return nil, nil
}
func (m *mockHoldingRepo) Create(ctx context.Context, h *Holding) error {
	if m.createFn != nil {
		return m.createFn(ctx, h)
	}
	return nil
}
func (m *mockHoldingRepo) Update(ctx context.Context, h *Holding) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, h)
	}
	return nil
}
func (m *mockHoldingRepo) Delete(ctx context.Context, id int64, userID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, userID)
	}
	return nil
}
func (m *mockHoldingRepo) FindHoldingTypes(ctx context.Context) ([]HoldingType, error) {
	if m.findHoldingTypesFn != nil {
		return m.findHoldingTypesFn(ctx)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindHoldingTypeByID(ctx context.Context, id int) (*HoldingType, error) {
	if m.findHoldingTypeByIDFn != nil {
		return m.findHoldingTypeByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindForSync(ctx context.Context, userID string, month, year int) ([]Holding, error) {
	if m.findForSyncFn != nil {
		return m.findForSyncFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) UpdateFields(ctx context.Context, id int64, userID string, fields map[string]any) error {
	if m.updateFieldsFn != nil {
		return m.updateFieldsFn(ctx, id, userID, fields)
	}
	return nil
}
func (m *mockHoldingRepo) FindForDuplicate(ctx context.Context, userID string, month, year int) ([]Holding, error) {
	if m.findForDuplicateFn != nil {
		return m.findForDuplicateFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) DeleteByUserMonthYear(ctx context.Context, userID string, month, year int) error {
	if m.deleteByMonthYearFn != nil {
		return m.deleteByMonthYearFn(ctx, userID, month, year)
	}
	return nil
}
func (m *mockHoldingRepo) CountByUserMonthYear(ctx context.Context, userID string, month, year int) (int64, error) {
	if m.countByMonthYearFn != nil {
		return m.countByMonthYearFn(ctx, userID, month, year)
	}
	return 0, nil
}
func (m *mockHoldingRepo) GetSummary(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error) {
	if m.getSummaryFn != nil {
		return m.getSummaryFn(ctx, userID, month, year)
	}
	return 0, 0, 0, nil
}
func (m *mockHoldingRepo) GetTypeBreakdown(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
	if m.getTypeBreakdownFn != nil {
		return m.getTypeBreakdownFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) GetPlatformBreakdown(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
	if m.getPlatformBreakdownFn != nil {
		return m.getPlatformBreakdownFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) GetTrends(ctx context.Context, userID string, years []int) ([]trendRow, error) {
	if m.getTrendsFn != nil {
		return m.getTrendsFn(ctx, userID, years)
	}
	return nil, nil
}
func (m *mockHoldingRepo) GetMonthlyData(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error) {
	if m.getMonthlyDataFn != nil {
		return m.getMonthlyDataFn(ctx, userID, sm, sy, em, ey)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindStockSymbols(ctx context.Context, userID string) ([]string, error) {
	if m.findStockSymbolsFn != nil {
		return m.findStockSymbolsFn(ctx, userID)
	}
	return nil, nil
}
