package market

import (
	"context"
	"strings"
	"testing"
	"time"
)

type mockQuoteCache struct {
	store map[string]float64
}

func (m *mockQuoteCache) BuildKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func (m *mockQuoteCache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	val, ok := m.store[key]
	if !ok {
		return false, nil
	}
	*(dest.(*float64)) = val
	return true, nil
}

func (m *mockQuoteCache) SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	if m.store == nil {
		m.store = make(map[string]float64)
	}
	m.store[key] = value.(float64)
	return nil
}

type mockInnerQuoteClient struct {
	calls         int
	lastRequested []string
	quotes        map[string]float64
}

func (m *mockInnerQuoteClient) GetQuotes(ctx context.Context, symbols []string) (map[string]float64, error) {
	m.calls++
	m.lastRequested = symbols
	result := make(map[string]float64)
	for _, s := range symbols {
		if p, ok := m.quotes[s]; ok {
			result[s] = p
		}
	}
	return result, nil
}

func TestCachedQuoteClient_AllCacheHit(t *testing.T) {
	cache := &mockQuoteCache{
		store: map[string]float64{
			"quote:BBCA.JK": 9800,
			"quote:AAPL":    185,
		},
	}
	inner := &mockInnerQuoteClient{
		quotes: map[string]float64{"BBCA.JK": 9800, "AAPL": 185},
	}

	client := NewCachedQuoteClient(inner, cache)
	quotes, err := client.GetQuotes(context.Background(), []string{"BBCA.JK", "AAPL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.calls != 0 {
		t.Errorf("expected 0 inner calls on full cache hit, got %d", inner.calls)
	}
	if quotes["BBCA.JK"] != 9800 || quotes["AAPL"] != 185 {
		t.Errorf("unexpected quotes: %+v", quotes)
	}
}

func TestCachedQuoteClient_PartialCacheHit(t *testing.T) {
	cache := &mockQuoteCache{
		store: map[string]float64{
			"quote:BBCA.JK": 9800,
		},
	}
	inner := &mockInnerQuoteClient{
		quotes: map[string]float64{"AAPL": 185},
	}

	client := NewCachedQuoteClient(inner, cache)
	quotes, err := client.GetQuotes(context.Background(), []string{"BBCA.JK", "AAPL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.calls != 1 {
		t.Errorf("expected 1 inner call on partial cache hit, got %d", inner.calls)
	}
	if len(inner.lastRequested) != 1 || inner.lastRequested[0] != "AAPL" {
		t.Errorf("expected inner client to only be called for AAPL, got: %v", inner.lastRequested)
	}
	if quotes["BBCA.JK"] != 9800 || quotes["AAPL"] != 185 {
		t.Errorf("unexpected combined quotes: %+v", quotes)
	}
	if cache.store["quote:AAPL"] != 185 {
		t.Errorf("expected AAPL to be written to cache, got %v", cache.store["quote:AAPL"])
	}
}

func TestCachedQuoteClient_NilCache(t *testing.T) {
	inner := &mockInnerQuoteClient{
		quotes: map[string]float64{"BBCA.JK": 9800},
	}

	client := NewCachedQuoteClient(inner, nil)
	quotes, err := client.GetQuotes(context.Background(), []string{"BBCA.JK"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.calls != 1 {
		t.Errorf("expected 1 inner call with nil cache, got %d", inner.calls)
	}
	if quotes["BBCA.JK"] != 9800 {
		t.Errorf("unexpected quotes: %+v", quotes)
	}
}
