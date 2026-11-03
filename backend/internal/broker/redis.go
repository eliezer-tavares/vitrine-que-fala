package broker

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher publishes to a stream (v1.0 behavior, still the local and
// CI default).
type RedisPublisher struct {
	R *redis.Client
}

// NewRedisPublisher connects to REDIS_ADDR (or the docker-compose default).
func NewRedisPublisher() *RedisPublisher {
	return &RedisPublisher{R: redis.NewClient(&redis.Options{Addr: envOrDefault("REDIS_ADDR", "redis:6379")})}
}

func (p *RedisPublisher) Publish(ctx context.Context, topic string, values map[string]any) error {
	return p.R.XAdd(ctx, &redis.XAddArgs{Stream: topic, Values: values}).Err()
}

// RedisSubscriber consumes a stream through a consumer group (ADR-0004).
type RedisSubscriber struct {
	R        *redis.Client
	Consumer string
}

// NewRedisSubscriber binds the subscriber to a consumer name.
func NewRedisSubscriber(consumer string) *RedisSubscriber {
	return &RedisSubscriber{
		R:        redis.NewClient(&redis.Options{Addr: envOrDefault("REDIS_ADDR", "redis:6379")}),
		Consumer: consumer,
	}
}

func (s *RedisSubscriber) EnsureGroup(ctx context.Context, topic, group string) error {
	err := s.R.XGroupCreateMkStream(ctx, topic, group, "0").Err()
	if err != nil && isBusyGroup(err) {
		return nil
	}
	return err
}

func (s *RedisSubscriber) Read(ctx context.Context, topic, group string, count int64) ([]Message, error) {
	res, err := s.R.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: s.Consumer,
		Streams:  []string{topic, ">"},
		Count:    count,
		Block:    readBlock,
	}).Result()
	if err != nil {
		if err == redis.Nil { // empty stream: block timeout, nothing to do
			return nil, nil
		}
		return nil, err
	}
	var out []Message
	for _, stream := range res {
		for _, m := range stream.Messages {
			out = append(out, Message{ID: m.ID, Values: m.Values})
		}
	}
	return out, nil
}

func (s *RedisSubscriber) Ack(ctx context.Context, topic, group, id string) error {
	return s.R.XAck(ctx, topic, group, id).Err()
}

func (s *RedisSubscriber) Requeue(ctx context.Context, topic string, values map[string]any) error {
	return s.R.XAdd(ctx, &redis.XAddArgs{Stream: topic, Values: values}).Err()
}

func (s *RedisSubscriber) Dead(ctx context.Context, deadTopic string, values map[string]any) error {
	return s.R.XAdd(ctx, &redis.XAddArgs{Stream: deadTopic, Values: values}).Err()
}
