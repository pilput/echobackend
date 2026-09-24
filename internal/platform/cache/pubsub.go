package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Publish sends payload to a Redis pub/sub channel. A nil cache is a no-op that
// reports no error, so callers must decide on their own fallback delivery.
func (c *RedisCache) Publish(ctx context.Context, channel string, payload []byte) error {
	if c == nil || c.client == nil || channel == "" {
		return nil
	}
	if err := c.client.Publish(ctx, channel, payload).Err(); err != nil {
		log.Warn("cache: Publish error", "channel", channel, "error", err)
		return err
	}
	return nil
}

// PSubscribe opens a pattern subscription. It returns nil when caching is
// disabled. go-redis reconnects the subscription on its own; the caller owns
// the returned PubSub and must Close it.
func (c *RedisCache) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.PSubscribe(ctx, patterns...)
}
