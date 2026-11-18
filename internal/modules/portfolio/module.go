// Package portfolio is the investment-tracking module: monthly holdings
// snapshots, IDX corporate-action calendar and exchange rates, backed by
// RapidAPI market data.
//
// It is self-contained: models, DTOs, repositories, services, handlers and
// routes all live here. It may depend on core packages (config, middleware,
// platform, pkg) but never on another module, and nothing outside di/routes
// imports it.
package portfolio

import (
	"echobackend/internal/platform/cache"
	"echobackend/internal/platform/market"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// Deps are the shared resources the module needs from the container.
type Deps struct {
	DB *gorm.DB
	// Cache may be nil (Redis disabled); every consumer is nil-safe.
	Cache *cache.RedisCache
	// RapidAPIKey may be empty; market lookups then degrade instead of failing.
	RapidAPIKey string
}

// Module holds the wired portfolio handlers.
type Module struct {
	holding         *HoldingHandler
	exchangeRate    *ExchangeRateHandler
	corporateAction *CorporateActionHandler
}

// New wires the module's repositories, services and handlers.
func New(deps Deps) *Module {
	// Market quotes: RapidAPI behind the shared Redis caching decorator.
	quoteClient := market.NewCachedQuoteClient(market.NewRapidAPIQuoteClient(deps.RapidAPIKey, nil), deps.Cache)
	idxClient := market.NewRapidAPIIDXClient(deps.RapidAPIKey, nil)

	holdingService := NewHoldingService(NewHoldingRepository(deps.DB), quoteClient, deps.Cache)
	exchangeRateService := NewExchangeRateService(quoteClient, deps.Cache)
	corporateActionService := NewCorporateActionService(idxClient, NewCorporateActionRepository(deps.DB))

	return &Module{
		holding:         NewHoldingHandler(holdingService),
		exchangeRate:    NewExchangeRateHandler(exchangeRateService),
		corporateAction: NewCorporateActionHandler(corporateActionService),
	}
}

// RegisterRoutes mounts the module under api (the /api group). requireAuth
// is the login-required middleware; every portfolio route is per-user.
func (m *Module) RegisterRoutes(api *echo.Group, requireAuth echo.MiddlewareFunc) {
	holdings := api.Group("/holdings", requireAuth)
	{
		holdings.GET("", m.holding.GetHoldings)
		holdings.GET("/summary", m.holding.GetSummary)
		holdings.GET("/trends", m.holding.GetTrends)
		holdings.GET("/compare", m.holding.CompareMonths)
		holdings.GET("/monthly", m.holding.GetMonthlyData)

		// Corporate Actions calendar (IDX dividends & RUPS)
		holdings.GET("/calendar", m.corporateAction.GetCalendar)

		holdings.POST("", m.holding.CreateHolding)
		holdings.POST("/duplicate", m.holding.DuplicateHoldings)
		holdings.POST("/sync", m.holding.SyncPrices)
		holdings.GET("/:id", m.holding.GetHoldingByID)
		holdings.PUT("/:id", m.holding.UpdateHolding)
		holdings.DELETE("/:id", m.holding.DeleteHolding)
	}

	holdingTypes := api.Group("/holding-types", requireAuth)
	{
		holdingTypes.GET("", m.holding.GetHoldingTypes)
	}

	exchangeRates := api.Group("/exchange-rates", requireAuth)
	{
		exchangeRates.GET("", m.exchangeRate.GetRate)
	}
}
