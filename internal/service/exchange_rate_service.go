package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"echobackend/internal/dto"
	"echobackend/pkg/market"
)

const exchangeRateCacheTTL = 15 * time.Minute

var ErrInvalidCurrencyPair = errors.New("invalid currency pair")

type ExchangeRateService interface {
	GetRate(ctx context.Context, from, to string) (*dto.ExchangeRateResponse, error)
}

type exchangeRateCache interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

type exchangeRateService struct {
	quoteClient market.QuoteClient
	cache       exchangeRateCache
	now         func() time.Time
}

func NewExchangeRateService(quoteClient market.QuoteClient, cache exchangeRateCache) ExchangeRateService {
	return &exchangeRateService{
		quoteClient: quoteClient,
		cache:       cache,
		now:         time.Now,
	}
}

func (s *exchangeRateService) GetRate(ctx context.Context, from, to string) (*dto.ExchangeRateResponse, error) {
	from = normalizeCurrencyCode(from)
	to = normalizeCurrencyCode(to)
	if !validCurrencyCode(from) || !validCurrencyCode(to) {
		return nil, ErrInvalidCurrencyPair
	}

	cacheKey := s.cacheKey(from, to)
	var cached dto.ExchangeRateResponse
	if s.cache != nil {
		if ok, _ := s.cache.GetJSON(ctx, cacheKey, &cached); ok {
			cached.Cached = true
			return &cached, nil
		}
	}

	if from == to {
		result := s.response(from, to, from+to+"=X", 1, false)
		s.setCache(ctx, cacheKey, result)
		return result, nil
	}

	directSymbol := yahooCurrencySymbol(from, to)
	inverseSymbol := yahooCurrencySymbol(to, from)
	quotes, err := s.quoteClient.GetQuotes(ctx, []string{directSymbol, inverseSymbol})
	if err != nil {
		return nil, err
	}

	if rate, ok := quotes[directSymbol]; ok && rate > 0 {
		result := s.response(from, to, directSymbol, rate, false)
		s.setCache(ctx, cacheKey, result)
		return result, nil
	}

	if inverseRate, ok := quotes[inverseSymbol]; ok && inverseRate > 0 {
		rate := math.Round((1/inverseRate)*1e8) / 1e8
		result := s.response(from, to, inverseSymbol, rate, false)
		s.setCache(ctx, cacheKey, result)
		return result, nil
	}

	return nil, fmt.Errorf("exchange rate not found for %s/%s", from, to)
}

func (s *exchangeRateService) setCache(ctx context.Context, key string, result *dto.ExchangeRateResponse) {
	if s.cache != nil {
		_ = s.cache.SetJSONWithTTL(ctx, key, result, exchangeRateCacheTTL)
	}
}

func (s *exchangeRateService) cacheKey(from, to string) string {
	if s.cache == nil {
		return ""
	}
	return s.cache.BuildKey("exchange-rate", from, to)
}

func (s *exchangeRateService) response(from, to, symbol string, rate float64, cached bool) *dto.ExchangeRateResponse {
	return &dto.ExchangeRateResponse{
		From:      from,
		To:        to,
		Symbol:    symbol,
		Rate:      rate,
		Source:    "RapidAPI",
		Cached:    cached,
		FetchedAt: s.now().UTC().Format(time.RFC3339),
	}
}

func normalizeCurrencyCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func validCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func yahooCurrencySymbol(from, to string) string {
	return from + to + "=X"
}
