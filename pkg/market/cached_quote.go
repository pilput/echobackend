package market

import (
	"context"
	"time"
)

// QuoteCache defines the caching operations required for storing market quotes.
type QuoteCache interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

// CachedQuoteClient wraps a QuoteClient and caches quotes in Redis / memory.
type CachedQuoteClient struct {
	inner QuoteClient
	cache QuoteCache
	ttl   time.Duration
}

const defaultQuoteCacheTTL = 15 * time.Minute

// NewCachedQuoteClient creates a new caching decorator around a QuoteClient.
// If cache is nil, caching is skipped and calls go directly to inner.
func NewCachedQuoteClient(inner QuoteClient, cache QuoteCache, ttl ...time.Duration) *CachedQuoteClient {
	cacheTTL := defaultQuoteCacheTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		cacheTTL = ttl[0]
	}
	return &CachedQuoteClient{
		inner: inner,
		cache: cache,
		ttl:   cacheTTL,
	}
}

// GetQuotes checks the cache for existing quotes, batches any cache misses to the inner client,
// stores newly fetched quotes in the cache, and returns the combined quote map.
func (c *CachedQuoteClient) GetQuotes(ctx context.Context, symbols []string) (map[string]float64, error) {
	normalized := NormalizeSymbols(symbols)
	if len(normalized) == 0 {
		return map[string]float64{}, nil
	}

	if c.cache == nil || c.ttl <= 0 {
		return c.inner.GetQuotes(ctx, normalized)
	}

	results := make(map[string]float64, len(normalized))
	missingSymbols := make([]string, 0, len(normalized))

	for _, sym := range normalized {
		key := c.cache.BuildKey("quote", sym)
		var price float64
		if ok, err := c.cache.GetJSON(ctx, key, &price); err == nil && ok && price > 0 {
			results[sym] = price
		} else {
			missingSymbols = append(missingSymbols, sym)
		}
	}

	if len(missingSymbols) == 0 {
		return results, nil
	}

	fetched, err := c.inner.GetQuotes(ctx, missingSymbols)
	if err != nil {
		// If we had some cache hits, return partial results or return error if all failed
		if len(results) > 0 {
			return results, nil
		}
		return nil, err
	}

	for sym, price := range fetched {
		if price <= 0 {
			continue
		}
		results[sym] = price
		key := c.cache.BuildKey("quote", sym)
		_ = c.cache.SetJSONWithTTL(ctx, key, price, c.ttl)
	}

	return results, nil
}
