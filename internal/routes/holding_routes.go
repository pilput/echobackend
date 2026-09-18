package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupHoldingRoutes(api *echo.Group) {
	holdings := api.Group("/holdings", r.authMiddleware.Auth())
	{
		holdings.GET("", r.holdingHandler.GetHoldings)
		holdings.GET("/summary", r.holdingHandler.GetSummary)
		holdings.GET("/trends", r.holdingHandler.GetTrends)
		holdings.GET("/compare", r.holdingHandler.CompareMonths)
		holdings.GET("/monthly", r.holdingHandler.GetMonthlyData)

		// Corporate Actions calendar (IDX dividends & RUPS)
		holdings.GET("/calendar", r.corporateActionHandler.GetCalendar)

		holdings.POST("", r.holdingHandler.CreateHolding)
		holdings.POST("/duplicate", r.holdingHandler.DuplicateHoldings)
		holdings.POST("/sync", r.holdingHandler.SyncPrices)
		holdings.GET("/:id", r.holdingHandler.GetHoldingByID)
		holdings.PUT("/:id", r.holdingHandler.UpdateHolding)
		holdings.DELETE("/:id", r.holdingHandler.DeleteHolding)
	}

	holdingTypes := api.Group("/holding-types", r.authMiddleware.Auth())
	{
		holdingTypes.GET("", r.holdingHandler.GetHoldingTypes)
	}
}
