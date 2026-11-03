// Package broker abstracts the event bus behind one interface (ADR-0004,
// extended by ADR-0008): Redis Streams locally and in CI, Google Cloud
// Pub/Sub in production. The transport is chosen by the BROKER env var
// (redis|pubsub); service code never knows which one is live.
//
// Topic names are shared verbatim between implementations: "images.pending",
// "images.dead" and "orders.events" — the same names the v1.0 streams used.
package broker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// readBlock is how long a Read may wait for work before returning empty.
const readBlock = 5 * time.Second

// Message is one event delivered to a consumer.
type Message struct {
	ID     string
	Values map[string]any
}

// Publisher writes events to a topic/stream.
type Publisher interface {
	Publish(ctx context.Context, topic string, values map[string]any) error
}

// Subscriber consumes events with at-least-once semantics: explicit Ack,
// Requeue for retries (the attempts counter travels inside values) and Dead
// for the dead-letter topic. One Subscriber instance is bound to one
// consumer name (worker goroutine).
type Subscriber interface {
	EnsureGroup(ctx context.Context, topic, group string) error
	Read(ctx context.Context, topic, group string, count int64) ([]Message, error)
	Ack(ctx context.Context, topic, group, id string) error
	Requeue(ctx context.Context, topic string, values map[string]any) error
	Dead(ctx context.Context, deadTopic string, values map[string]any) error
}

// Env returns the broker selection from BROKER (default "redis").
func Env() string {
	if os.Getenv("BROKER") == "pubsub" {
		return "pubsub"
	}
	return "redis"
}

// NewPublisher builds the Publisher selected by BROKER.
func NewPublisher(ctx context.Context) (Publisher, error) {
	if Env() == "pubsub" {
		return newPubSubPublisher(ctx)
	}
	return NewRedisPublisher(), nil
}

// NewSubscriber builds the Subscriber selected by BROKER. consumer names the
// worker goroutine (it maps to a Redis consumer name or a Pub/Sub
// subscription suffix).
func NewSubscriber(ctx context.Context, consumer string) (Subscriber, error) {
	if Env() == "pubsub" {
		return newPubSubSubscriber(ctx, consumer)
	}
	return NewRedisSubscriber(consumer), nil
}

// toStrings flattens values into Pub/Sub attributes (string-only).
func toStrings(values map[string]any) map[string]string {
	out := make(map[string]string, len(values))
	for k, v := range values {
		switch t := v.(type) {
		case string:
			out[k] = t
		case nil:
			out[k] = ""
		default:
			out[k] = fmt.Sprint(t)
		}
	}
	return out
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func isBusyGroup(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}
