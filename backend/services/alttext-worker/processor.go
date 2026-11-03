package main

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/broker"
)

// Message is one pending alt-text job (the broker's message type).
type Message = broker.Message

// AIClient asks the ai-vision-service for per-locale alt-text.
type AIClient interface {
	Describe(ctx context.Context, imageURL string) (map[string]string, error)
}

// Embedder turns texts into vectors for the semantic-search index
// (ADR-0009). Optional in the processor: nil keeps the pipeline
// alt-text-only.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// Store persists worker output into the catalog through its gRPC contract
// (ADR-0007).
type Store interface {
	SetAltText(ctx context.Context, productID string, alt map[string]string) error
	SetEmbedding(ctx context.Context, productID, locale string, vec []float64) error
}

// Queue abstracts the stream operations the processor needs. Redis Streams
// and Google Pub/Sub both satisfy it via internal/broker (ADR-0008).
type Queue interface {
	Ack(ctx context.Context, topic, group, id string) error
	Requeue(ctx context.Context, topic string, values map[string]any) error
	Dead(ctx context.Context, deadTopic string, values map[string]any) error
}

// Processor consumes one message at a time: describe, persist, index
// (embeddings), ack — or requeue with an attempts counter, dead-lettering
// exhausted jobs.
type Processor struct {
	AI          AIClient
	Embedder    Embedder // optional; nil = skip the embedding step
	Store       Store
	Queue       Queue
	Stream      string
	Group       string
	DeadStream  string
	MaxAttempts int
}

// Handle processes a message. It returns an error only for storage failures
// (the message stays pending for redelivery); AI failures are handled by
// requeue/dead-letter policy inside this function. Embedding failures are
// logged and skipped: search falls back to lexical for that row, and the
// next full re-index (batch) repairs it.
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

	p.embedAndStore(ctx, productID, alt)
	return p.Queue.Ack(ctx, p.Stream, p.Group, m.ID)
}

// embedAndStore indexes the freshly written alt-text. Best-effort by design
// (ADR-0009): a failed embedding never fails the job.
func (p *Processor) embedAndStore(ctx context.Context, productID string, alt map[string]string) {
	if p.Embedder == nil || len(alt) == 0 {
		return
	}
	locales := make([]string, 0, len(alt))
	texts := make([]string, 0, len(alt))
	for loc, txt := range alt {
		locales = append(locales, loc)
		texts = append(texts, txt)
	}
	embeddings, err := p.Embedder.Embed(ctx, texts)
	if err != nil {
		slog.Warn("embedding skipped for product", "product", productID, "error", err)
		return
	}
	if len(embeddings) != len(locales) {
		slog.Warn("embedding count mismatch", "product", productID,
			"want", len(locales), "got", len(embeddings))
		return
	}
	for i, loc := range locales {
		if err := p.Store.SetEmbedding(ctx, productID, loc, embeddings[i]); err != nil {
			slog.Warn("embedding write failed", "product", productID, "locale", loc, "error", err)
		}
	}
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
