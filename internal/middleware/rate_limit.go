package middleware

import (
	"strconv"
	"sync"
	"time"

	"echobackend/internal/platform/cache"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type fixedWindowVisitor struct {
	count      int
	windowEnds time.Time
}

type fixedWindowStore struct {
	mu       sync.Mutex
	visitors map[string]fixedWindowVisitor
}

// maxTrackedVisitors bounds the in-memory fallback store so a flood of unique
// (or spoofed) client IPs cannot grow the map without limit. Once the cap is
// reached, expired entries are evicted first; if none are expired, the store
// stops admitting *new* identifiers until slots free up — existing visitors
// keep their counters. This trades strict per-IP accuracy under extreme flood
// for bounded memory, which is the right trade-off for an abuse-protection
// fallback path.
const maxTrackedVisitors = 10_000

// FixedWindowRateLimiter limits each client IP to maxRequests within window.
// It is intended for low-volume abuse protection on sensitive routes.
func FixedWindowRateLimiter(maxRequests int, window time.Duration) echo.MiddlewareFunc {
	return FixedWindowRateLimiterWithCache(nil, "", maxRequests, window)
}

// FixedWindowRateLimiterWithCache uses Redis when available so limits are
// shared across app instances. If cache is nil or errors, it falls back to memory.
func FixedWindowRateLimiterWithCache(redisCache *cache.RedisCache, name string, maxRequests int, window time.Duration) echo.MiddlewareFunc {
	store := &fixedWindowStore{visitors: make(map[string]fixedWindowVisitor)}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if maxRequests <= 0 || window <= 0 {
				return next(c)
			}

			now := time.Now()
			identifier := c.RealIP()
			if identifier == "" {
				identifier = c.Request().RemoteAddr
			}

			allowed, retryAfter := allowFixedWindow(c, redisCache, store, name, identifier, maxRequests, window, now)
			if !allowed {
				seconds := max(int(retryAfter.Seconds()), 1)
				c.Response().Header().Set("Retry-After", strconv.Itoa(seconds))
				return response.TooManyRequests(c, "Too many attempts. Please try again later.")
			}

			return next(c)
		}
	}
}

func allowFixedWindow(
	c *echo.Context,
	redisCache *cache.RedisCache,
	store *fixedWindowStore,
	name string,
	identifier string,
	maxRequests int,
	window time.Duration,
	now time.Time,
) (bool, time.Duration) {
	if redisCache == nil {
		return store.allow(identifier, maxRequests, window, now)
	}

	cacheKey := redisCache.BuildKey("rate_limit", name, identifier)
	count, retryAfter, err := redisCache.IncrementFixedWindow(c.Request().Context(), cacheKey, window)
	if err != nil {
		log.Warn("falling back to in-memory store", "name", name, "error", err)
		return store.allow(identifier, maxRequests, window, now)
	}
	if count == 0 {
		return store.allow(identifier, maxRequests, window, now)
	}

	if count > maxRequests {
		return false, retryAfter
	}
	return true, 0
}

func (s *fixedWindowStore) allow(identifier string, maxRequests int, window time.Duration, now time.Time) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	visitor, ok := s.visitors[identifier]
	if !ok || !now.Before(visitor.windowEnds) {
		if !ok {
			s.admitLocked(identifier, now, window)
		} else {
			s.visitors[identifier] = fixedWindowVisitor{
				count:      1,
				windowEnds: now.Add(window),
			}
		}
		return true, 0
	}

	if visitor.count >= maxRequests {
		return false, visitor.windowEnds.Sub(now)
	}

	visitor.count++
	s.visitors[identifier] = visitor
	return true, 0
}

// admitLocked inserts a new visitor, evicting expired entries first and, if
// the store is still at capacity, refusing to track brand-new identifiers.
func (s *fixedWindowStore) admitLocked(identifier string, now time.Time, window time.Duration) {
	if len(s.visitors) >= maxTrackedVisitors {
		s.cleanup(now)
	}
	if len(s.visitors) >= maxTrackedVisitors {
		// Store is full of active visitors; do not track this new identifier.
		// It is treated as allowed (first request) but not counted, keeping
		// memory bounded. Redis-backed mode is unaffected.
		return
	}
	s.visitors[identifier] = fixedWindowVisitor{
		count:      1,
		windowEnds: now.Add(window),
	}
}

func (s *fixedWindowStore) cleanup(now time.Time) {
	for identifier, visitor := range s.visitors {
		if !now.Before(visitor.windowEnds) {
			delete(s.visitors, identifier)
		}
	}
}
