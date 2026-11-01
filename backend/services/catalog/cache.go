package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheStore abstracts the locale-keyed catalog cache so handlers can be
// tested against an in-memory fake.
type CacheStore interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, val string, ttl time.Duration)
}

// CacheKey embeds the locale: serving en-US content to a pt-BR client is a
// correctness bug, not a performance issue (ADR-0003).
func CacheKey(loc string) string {
	return "catalog:v1:" + loc + ":products"
}

// RedisCache is the production CacheStore.
type RedisCache struct {
	R   *redis.Client
	TTL time.Duration
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, bool) {
	v, err := c.R.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (c *RedisCache) Set(ctx context.Context, key, val string, ttl time.Duration) {
	_ = c.R.Set(ctx, key, val, ttl).Err()
}
