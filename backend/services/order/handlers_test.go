package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeBroker struct {
	streams   []string
	published []map[string]any
	err       error
}

func (f *fakeBroker) Publish(_ context.Context, stream string, values map[string]any) error {
	if f.err != nil {
		return f.err
	}
	f.streams = append(f.streams, stream)
	f.published = append(f.published, values)
	return nil
}

func post(h *Handler, body, acceptLanguage string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(body))
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	req.Header.Set("X-Auth-Subject", "demo-user")
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	return rec
}

func TestCreatePublishesOrderPlaced(t *testing.T) {
	b := &fakeBroker{}
	h := &Handler{Broker: b, Stream: "orders.events"}

	body := `{"items":[{"product_id":"11111111-1111-1111-1111-111111111111","quantity":2}]}`
	rec := post(h, body, "pt-BR")

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if len(b.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(b.published))
	}
	if b.streams[0] != "orders.events" {
		t.Fatalf("stream = %q", b.streams[0])
	}
	v := b.published[0]
	if v["locale"] != "pt-BR" {
		t.Fatalf("event locale = %v", v["locale"])
	}
	if v["subject"] != "demo-user" {
		t.Fatalf("event subject = %v", v["subject"])
	}
	if _, ok := v["order_id"].(string); !ok {
		t.Fatal("event must carry an order_id")
	}
	if !strings.Contains(rec.Body.String(), "Pedido confirmado") {
		t.Fatalf("pt-BR receipt expected, got: %s", rec.Body.String())
	}
}

func TestCreateLocalizedReceiptEN(t *testing.T) {
	b := &fakeBroker{}
	h := &Handler{Broker: b, Stream: "orders.events"}

	rec := post(h,
		`{"items":[{"product_id":"11111111-1111-1111-1111-111111111111","quantity":1}]}`,
		"en-US,en;q=0.9")

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Order confirmed") {
		t.Fatalf("en-US receipt expected, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"locale":"en-US"`) {
		t.Fatalf("receipt locale must be en-US: %s", rec.Body.String())
	}
}

func TestCreateRejectsEmptyItemsLocalized(t *testing.T) {
	b := &fakeBroker{}
	h := &Handler{Broker: b, Stream: "orders.events"}

	rec := post(h, `{"items":[]}`, "pt-BR")
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Itens do pedido inválidos") {
		t.Fatalf("pt-BR error expected, got: %s", rec.Body.String())
	}

	rec = post(h, `{"items":[]}`, "en-US")
	if !strings.Contains(rec.Body.String(), "Invalid order items") {
		t.Fatalf("en-US error expected, got: %s", rec.Body.String())
	}
	if len(b.published) != 0 {
		t.Fatal("invalid orders must not be published")
	}
}

func TestCreateRejectsBadQuantity(t *testing.T) {
	b := &fakeBroker{}
	h := &Handler{Broker: b, Stream: "orders.events"}

	rec := post(h, `{"items":[{"product_id":"x","quantity":0}]}`, "pt-BR")
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBrokerFailureMapsTo500(t *testing.T) {
	b := &fakeBroker{err: errors.New("redis down")}
	h := &Handler{Broker: b, Stream: "orders.events"}

	rec := post(h,
		`{"items":[{"product_id":"11111111-1111-1111-1111-111111111111","quantity":1}]}`,
		"pt-BR")
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
