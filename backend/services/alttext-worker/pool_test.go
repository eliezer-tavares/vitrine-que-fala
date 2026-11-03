package main

import (
	"context"
	"errors"
	"testing"
)

type fakeAI struct {
	alt map[string]string
	err error
}

func (f *fakeAI) Describe(_ context.Context, _ string) (map[string]string, error) {
	return f.alt, f.err
}

// fakeEmbedder is the ADR-0009 embedder fake used by the indexer tests.
type fakeEmbedder struct {
	vecs [][]float64
	err  error
}

func (f *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([][]float64, len(texts))
	for i := range texts {
		if i < len(f.vecs) {
			out[i] = f.vecs[i]
			continue
		}
		out[i] = []float64{0.1, 0.2, 0.3}
	}
	return out, nil
}

type fakeStore struct {
	alt    map[string]string
	embeds map[string][]float64 // locale -> vector
	err    error
}

func (f *fakeStore) SetAltText(_ context.Context, _ string, alt map[string]string) error {
	if f.err != nil {
		return f.err
	}
	f.alt = alt
	return nil
}

func (f *fakeStore) SetEmbedding(_ context.Context, _, locale string, vec []float64) error {
	if f.embeds == nil {
		f.embeds = map[string][]float64{}
	}
	f.embeds[locale] = vec
	return nil
}

type fakeQueue struct {
	acks     []string
	requeues []map[string]any
	deads    []map[string]any
}

func (f *fakeQueue) Ack(_ context.Context, stream, group, id string) error {
	f.acks = append(f.acks, id)
	return nil
}

func (f *fakeQueue) Requeue(_ context.Context, stream string, values map[string]any) error {
	f.requeues = append(f.requeues, values)
	return nil
}

func (f *fakeQueue) Dead(_ context.Context, stream string, values map[string]any) error {
	f.deads = append(f.deads, values)
	return nil
}

func newProcessor(ai AIClient, store *fakeStore, q *fakeQueue) *Processor {
	return &Processor{
		AI: ai, Store: store, Queue: q,
		Stream: "images.pending", Group: "alttext-workers",
		DeadStream: "images.dead", MaxAttempts: 3,
	}
}

func msg(values map[string]any) Message {
	return Message{ID: "msg-1", Values: values}
}

func baseValues(attempts any) map[string]any {
	return map[string]any{
		"product_id": "p-1",
		"image_url":  "https://example.test/p.jpg",
		"attempts":   attempts,
	}
}

func TestHandleSuccessPersistsAndAcks(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{alt: map[string]string{"pt-BR": "Poncho amarelo", "en-US": "Yellow poncho"}}, store, q)

	if err := p.Handle(context.Background(), msg(baseValues(int64(0)))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.alt["pt-BR"] != "Poncho amarelo" || store.alt["en-US"] != "Yellow poncho" {
		t.Fatalf("store got %v", store.alt)
	}
	if len(q.acks) != 1 || q.acks[0] != "msg-1" {
		t.Fatalf("acks = %v", q.acks)
	}
	if len(q.requeues) != 0 || len(q.deads) != 0 {
		t.Fatal("success must not requeue or dead-letter")
	}
}

func TestHandleSuccessIndexesEmbeddings(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{alt: map[string]string{"pt-BR": "Poncho amarelo", "en-US": "Yellow poncho"}}, store, q)
	p.Embedder = &fakeEmbedder{vecs: [][]float64{{1, 0}, {0, 1}}}

	if err := p.Handle(context.Background(), msg(baseValues(int64(0)))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(store.embeds) != 2 {
		t.Fatalf("expected embeddings for 2 locales, got %v", store.embeds)
	}
	if store.embeds["pt-BR"][0] != 1 || store.embeds["en-US"][1] != 1 {
		t.Fatalf("embeddings = %v", store.embeds)
	}
	if len(q.acks) != 1 {
		t.Fatal("job must ack after indexing")
	}
}

func TestEmbeddingFailureStillAcks(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{alt: map[string]string{"pt-BR": "x"}}, store, q)
	p.Embedder = &fakeEmbedder{err: errors.New("model loading")}

	// Embeddings are an enhancement (ADR-0009): their failure must never
	// fail the job or lose the alt-text.
	if err := p.Handle(context.Background(), msg(baseValues(int64(0)))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.alt["pt-BR"] != "x" {
		t.Fatal("alt-text must persist even when embedding fails")
	}
	if len(store.embeds) != 0 {
		t.Fatalf("no embedding should be written, got %v", store.embeds)
	}
	if len(q.acks) != 1 {
		t.Fatal("job must ack when only the embedding failed")
	}
}

func TestNoEmbedderSkipsIndexing(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{alt: map[string]string{"pt-BR": "x"}}, store, q)

	if err := p.Handle(context.Background(), msg(baseValues(int64(0)))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(store.embeds) != 0 {
		t.Fatalf("embeds = %v, want none", store.embeds)
	}
	if len(q.acks) != 1 {
		t.Fatal("job must ack")
	}
}

func TestHandleAIErrorRequeuesWithAttemptsPlusOne(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{err: errors.New("model busy")}, store, q)

	if err := p.Handle(context.Background(), msg(baseValues(int64(0)))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(q.requeues) != 1 {
		t.Fatalf("expected 1 requeue, got %d", len(q.requeues))
	}
	if q.requeues[0]["attempts"] != int64(1) {
		t.Fatalf("attempts = %v, want 1", q.requeues[0]["attempts"])
	}
	if len(q.acks) != 1 {
		t.Fatal("original message must be acked after requeue")
	}
	if len(q.deads) != 0 {
		t.Fatal("nothing should be dead at attempts=0")
	}
}

func TestHandleAIErrorDeadLettersAtMaxAttempts(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{err: errors.New("model busy")}, store, q)

	if err := p.Handle(context.Background(), msg(baseValues(int64(2)))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(q.deads) != 1 {
		t.Fatalf("expected 1 dead-letter, got %d", len(q.deads))
	}
	if len(q.requeues) != 0 {
		t.Fatal("exhausted jobs must not be requeued")
	}
	if len(q.acks) != 1 {
		t.Fatal("dead-lettered message must be acked")
	}
}

func TestHandleStringAttemptsStillCounts(t *testing.T) {
	store := &fakeStore{}
	q := &fakeQueue{}
	p := newProcessor(&fakeAI{err: errors.New("boom")}, store, q)

	// go-redis may deliver numbers as strings depending on the pipeline; the
	// counter must survive: string "2" + 1 = 3 >= MaxAttempts(3) -> dead.
	if err := p.Handle(context.Background(), msg(baseValues("2"))); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(q.deads) != 1 {
		t.Fatalf("attempts=3 (string '2' + 1) should dead-letter, got %d deads", len(q.deads))
	}
}

func TestHandleStoreErrorLeavesPending(t *testing.T) {
	q := &fakeQueue{}
	store := &fakeStore{err: errors.New("db down")}
	p := newProcessor(&fakeAI{alt: map[string]string{"pt-BR": "x"}}, store, q)

	if err := p.Handle(context.Background(), msg(baseValues(int64(0)))); err == nil {
		t.Fatal("storage failure must surface as error (message stays pending)")
	}
	if len(q.acks) != 0 {
		t.Fatal("failed store must not ack")
	}
}
