package main

import (
	"context"
	"strconv"
)

// Message is one pending alt-text job.
type Message struct {
	ID     string
	Values map[string]any
}

// AIClient asks the ai-vision-service for per-locale alt-text.
type AIClient interface {
	Describe(ctx context.Context, imageURL string) (map[string]string, error)
}

// Store persists per-locale alt-text into the catalog translations table.
type Store interface {
	SetAltText(ctx context.Context, productID string, alt map[string]string) error
}

// Queue abstracts the stream operations the processor needs. v1.0 is Redis
// Streams; v2.0 swaps Google Pub/Sub behind this interface (ADR-0004).
type Queue interface {
	Ack(ctx context.Context, stream, group, id string) error
	Requeue(ctx context.Context, stream string, values map[string]any) error
	Dead(ctx context.Context, stream string, values map[string]any) error
}

// Processor consumes one message at a time: describe, persist, ack — or
// requeue with an attempts counter, dead-lettering exhausted jobs.
type Processor struct {
	AI          AIClient
	Store       Store
	Queue       Queue
	Stream      string
	Group       string
	DeadStream  string
	MaxAttempts int
}

// Handle processes a message. It returns an error only for storage failures
// (the message stays pending for redelivery); AI failures are handled by
// requeue/dead-letter policy inside this function.
func (p *Processor) Handle(ctx context.Context, m Message) error {
	productID, _ := m.Values["product_id"].(string)
	imageURL, _ := m.Values["image_url"].(string)

	alt, err := p.AI.Describe(ctx, imageURL)
	if err != nil {
		attempts := toInt64(m.Values["attempts"])
		if attempts+1 >= int64(p.MaxAttempts) {
			if derr := p.Queue.Dead(ctx, p.DeadStream, m.Values); derr != nil {
				return derr
			}
		} else {
			nv := cloneValues(m.Values)
			nv["attempts"] = attempts + 1
			if rerr := p.Queue.Requeue(ctx, p.Stream, nv); rerr != nil {
				return rerr
			}
		}
		return p.Queue.Ack(ctx, p.Stream, p.Group, m.ID)
	}

	if err := p.Store.SetAltText(ctx, productID, alt); err != nil {
		return err
	}
	return p.Queue.Ack(ctx, p.Stream, p.Group, m.ID)
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	}
	return 0
}

func cloneValues(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}
