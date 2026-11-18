package portfolio

import (
	"echobackend/pkg/response"
	"errors"

	"github.com/labstack/echo/v5"
)

type ExchangeRateHandler struct {
	exchangeRateService ExchangeRateService
}

func NewExchangeRateHandler(exchangeRateService ExchangeRateService) *ExchangeRateHandler {
	return &ExchangeRateHandler{exchangeRateService: exchangeRateService}
}

func (h *ExchangeRateHandler) GetRate(c *echo.Context) error {
	from := c.QueryParam("from")
	to := c.QueryParam("to")

	result, err := h.exchangeRateService.GetRate(c.Request().Context(), from, to)
	if err != nil {
		if errors.Is(err, ErrInvalidCurrencyPair) {
			return response.BadRequest(c, "Invalid currency pair", err)
		}
		return response.InternalServerError(c, "Failed to get exchange rate", err)
	}

	return response.Success(c, "Exchange rate fetched successfully", result)
}
