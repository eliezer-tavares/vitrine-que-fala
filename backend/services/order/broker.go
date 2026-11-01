package main

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

// Broker abstracts the event bus so handlers can be tested with a fake.
// v1.0 uses Redis Streams; v2.0 swaps in Google Pub/Sub behind this exact
// interface (ADR-0004).
type Broker interface {
	Publish(ctx context.Context, stream string, values map[string]any) error
}

// RedisBroker publishes to a Redis Stream.
type RedisBroker struct {
	R *redis.Client
}

func (b *RedisBroker) Publish(ctx context.Context, stream string, values map[string]any) error {
	return b.R.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: values}).Err()
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
