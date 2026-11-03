package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/broker"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/locale"
	"github.com/google/uuid"
)

// OrderItem is one line of a purchase.
type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// NewOrder is the request body for POST /api/v1/orders.
type NewOrder struct {
	Items []OrderItem `json:"items"`
}

// Receipt is the localized response returned to the client.
type Receipt struct {
	OrderID string      `json:"order_id"`
	Locale  string      `json:"locale"`
	Message string      `json:"message"`
	Items   []OrderItem `json:"items"`
}

// Localized strings. v2.0 serves all six locales (ADR-0003); the table grows
// with locale.Supported and stays lookup-based so the gRPC and HTTP paths
// share one source of truth.
var (
	confirmedMessages = map[string]string{
		locale.PTBR: "Pedido confirmado",
		locale.ENUS: "Order confirmed",
		locale.IDID: "Pesanan dikonfirmasi",
		locale.THTH: "ยืนยันคำสั่งซื้อแล้ว",
		locale.VIVN: "Đã xác nhận đơn hàng",
		locale.ZHTW: "訂單已確認",
	}
	invalidMessages = map[string]string{
		locale.PTBR: "Itens do pedido inválidos",
		locale.ENUS: "Invalid order items",
		locale.IDID: "Item pesanan tidak valid",
		locale.THTH: "รายการคำสั่งซื้อไม่ถูกต้อง",
		locale.VIVN: "Mặt hàng đơn hàng không hợp lệ",
		locale.ZHTW: "訂單項目無效",
	}
)

// localized picks the message for loc, falling back to English — a missing
// translation must never break checkout.
func localized(table map[string]string, loc string) string {
	if v, ok := table[loc]; ok {
		return v
	}
	return table[locale.ENUS]
}

// ErrInvalidOrder maps to HTTP 400 / gRPC InvalidArgument.
var ErrInvalidOrder = errors.New("invalid order items")

// Handler validates checkout payloads and publishes OrderPlaced events.
type Handler struct {
	Broker broker.Publisher
	Stream string
}

func (h *Handler) validate(in NewOrder) bool {
	if len(in.Items) == 0 {
		return false
	}
	for _, it := range in.Items {
		if it.ProductID == "" || it.Quantity < 1 {
			return false
		}
	}
	return true
}

// CreateOrder is the shared checkout core: validate, publish OrderPlaced,
// return the localized receipt. The HTTP handler and the gRPC server both
// call it (ADR-0007) — one business rule, two transports.
func (h *Handler) CreateOrder(ctx context.Context, items []OrderItem, subject, loc string) (*Receipt, error) {
	in := NewOrder{Items: items}
	if !h.validate(in) {
		return nil, ErrInvalidOrder
	}

	orderID := uuid.NewString()
	itemsJSON, err := json.Marshal(in.Items)
	if err != nil {
		return nil, err
	}

	values := map[string]any{
		"order_id":   orderID,
		"subject":    subject,
		"locale":     loc,
		"items":      string(itemsJSON),
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.Broker.Publish(ctx, h.Stream, values); err != nil {
		return nil, err
	}

	return &Receipt{
		OrderID: orderID,
		Locale:  loc,
		Message: localized(confirmedMessages, loc),
		Items:   in.Items,
	}, nil
}

// Create serves POST /api/v1/orders. Auth is enforced by the gateway, which
// forwards the validated subject in X-Auth-Subject (internal trust boundary,
// documented in infra/cloud-run.md).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	loc := httputil.Negotiated(r)

	var in NewOrder
	if err := httputil.Decode(r, &in); err != nil {
		httputil.Error(w, http.StatusBadRequest, localized(invalidMessages, loc))
		return
	}

	receipt, err := h.CreateOrder(r.Context(), in.Items, r.Header.Get("X-Auth-Subject"), loc)
	if errors.Is(err, ErrInvalidOrder) {
		httputil.Error(w, http.StatusBadRequest, localized(invalidMessages, loc))
		return
	}
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "could not register the order")
		return
	}

	httputil.JSON(w, http.StatusCreated, receipt)
}
