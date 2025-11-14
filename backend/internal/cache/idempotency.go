package cache

import (
	context "context"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyCache stores serialized responses keyed by request id.
type IdempotencyCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewIdempotencyCache(client *redis.Client, ttl time.Duration) *IdempotencyCache {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &IdempotencyCache{client: client, ttl: ttl}
}

func (c *IdempotencyCache) Get(ctx context.Context, key string) ([]byte, bool) {
	if c == nil || c.client == nil || key == "" {
		return nil, false
	}
	data, err := c.client.Get(ctx, c.prefixed(key)).Bytes()
	if err != nil {
		return nil, false
	}
	return data, true
}

func (c *IdempotencyCache) Set(ctx context.Context, key string, value []byte) {
	if c == nil || c.client == nil || key == "" || len(value) == 0 {
		return
	}
	c.client.Set(ctx, c.prefixed(key), value, c.ttl)
}

func (c *IdempotencyCache) prefixed(key string) string {
	return "idem:" + key
}
