package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupExchangeRateRoutes(api *echo.Group) {
	exchangeRates := api.Group("/exchange-rates", r.authMiddleware.Auth())
	{
		exchangeRates.GET("", r.exchangeRateHandler.GetRate)
	}
}
