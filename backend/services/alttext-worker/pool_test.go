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

type fakeStore struct {
	got map[string]string
	err error
}

func (f *fakeStore) SetAltText(_ context.Context, _ string, alt map[string]string) error {
	if f.err != nil {
		return f.err
	}
	f.got = alt
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
	if store.got["pt-BR"] != "Poncho amarelo" || store.got["en-US"] != "Yellow poncho" {
		t.Fatalf("store got %v", store.got)
	}
	if len(q.acks) != 1 || q.acks[0] != "msg-1" {
		t.Fatalf("acks = %v", q.acks)
	}
	if len(q.requeues) != 0 || len(q.deads) != 0 {
		t.Fatal("success must not requeue or dead-letter")
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
