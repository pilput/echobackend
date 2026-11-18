package portfolio

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeExchangeRateCache struct {
	values map[string]ExchangeRateResponse
	setTTL time.Duration
}

func (f *fakeExchangeRateCache) BuildKey(parts ...string) string {
	var key strings.Builder
	for i, part := range parts {
		if i > 0 {
			key.WriteString(":")
		}
		key.WriteString(part)
	}
	return key.String()
}

func (f *fakeExchangeRateCache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	value, ok := f.values[key]
	if !ok {
		return false, nil
	}
	*(dest.(*ExchangeRateResponse)) = value
	return true, nil
}

func (f *fakeExchangeRateCache) SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	if f.values == nil {
		f.values = map[string]ExchangeRateResponse{}
	}
	f.values[key] = *(value.(*ExchangeRateResponse))
	f.setTTL = ttl
	return nil
}

func TestExchangeRateService_GetRate_DirectQuote(t *testing.T) {
	cache := &fakeExchangeRateCache{}
	svc := NewExchangeRateService(&stubQuoteClient{
		quotes: map[string]float64{"USDIDR=X": 16250},
	}, cache)

	got, err := svc.GetRate(context.Background(), "usd", "idr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.From != "USD" || got.To != "IDR" {
		t.Fatalf("unexpected pair: %+v", got)
	}
	if got.Symbol != "USDIDR=X" || got.Rate != 16250 {
		t.Fatalf("unexpected quote: %+v", got)
	}
	if got.Cached {
		t.Fatal("first fetch should not be marked cached")
	}
	if cache.setTTL != exchangeRateCacheTTL {
		t.Fatalf("cache ttl = %v, want %v", cache.setTTL, exchangeRateCacheTTL)
	}
}

func TestExchangeRateService_GetRate_InverseFallback(t *testing.T) {
	svc := NewExchangeRateService(&stubQuoteClient{
		quotes: map[string]float64{"USDIDR=X": 16000},
	}, &fakeExchangeRateCache{})

	got, err := svc.GetRate(context.Background(), "IDR", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Symbol != "USDIDR=X" {
		t.Fatalf("symbol = %q, want inverse symbol", got.Symbol)
	}
	if got.Rate != 0.0000625 {
		t.Fatalf("rate = %v, want 0.0000625", got.Rate)
	}
}

func TestExchangeRateService_GetRate_CacheHit(t *testing.T) {
	cache := &fakeExchangeRateCache{values: map[string]ExchangeRateResponse{
		"exchange-rate:USD:IDR": {From: "USD", To: "IDR", Symbol: "USDIDR=X", Rate: 16000},
	}}
	svc := NewExchangeRateService(&stubQuoteClient{
		err: context.Canceled,
	}, cache)

	got, err := svc.GetRate(context.Background(), "USD", "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Cached {
		t.Fatal("cached response should be marked cached")
	}
	if got.Rate != 16000 {
		t.Fatalf("rate = %v", got.Rate)
	}
}

func TestExchangeRateService_GetRate_InvalidCurrency(t *testing.T) {
	svc := NewExchangeRateService(&stubQuoteClient{}, nil)
	if _, err := svc.GetRate(context.Background(), "USDT", "IDR"); !errors.Is(err, ErrInvalidCurrencyPair) {
		t.Fatalf("expected ErrInvalidCurrencyPair, got %v", err)
	}
}
