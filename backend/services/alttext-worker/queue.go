package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueue implements Queue + reading over Redis Streams consumer groups
// (ADR-0004). Failed jobs are re-enqueued with attempts+1; exhausted jobs go
// to the dead stream.
type RedisQueue struct {
	R        *redis.Client
	Consumer string
}

// EnsureGroup creates the consumer group, tolerating BUSYGROUP (already exists).
func (q *RedisQueue) EnsureGroup(ctx context.Context, stream, group string) error {
	err := q.R.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

// Read pulls up to count new messages, blocking briefly when the stream is empty.
func (q *RedisQueue) Read(ctx context.Context, stream, group string, count int64) ([]Message, error) {
	res, err := q.R.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: q.Consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    5 * time.Second,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Message
	for _, s := range res {
		for _, m := range s.Messages {
			out = append(out, Message{ID: m.ID, Values: m.Values})
		}
	}
	return out, nil
}

func (q *RedisQueue) Ack(ctx context.Context, stream, group, id string) error {
	return q.R.XAck(ctx, stream, group, id).Err()
}

func (q *RedisQueue) Requeue(ctx context.Context, stream string, values map[string]any) error {
	return q.R.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: values}).Err()
}

func (q *RedisQueue) Dead(ctx context.Context, stream string, values map[string]any) error {
	return q.R.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: values}).Err()
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

// runLoop is one worker goroutine: block-read, process, repeat, until ctx is
// done. Storage failures are logged and left pending for redelivery.
func runLoop(ctx context.Context, q *RedisQueue, p *Processor, stream, group string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgs, err := q.Read(ctx, stream, group, 1)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(time.Second)
			continue
		}
		for _, m := range msgs {
			if err := p.Handle(ctx, m); err != nil {
				time.Sleep(time.Second)
			}
		}
	}
}
