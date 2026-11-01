package main

import (
	"encoding/json"
	"net/http"
	"time"

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

// Handler validates checkout payloads and publishes OrderPlaced events.
type Handler struct {
	Broker Broker
	Stream string
}

// Localized strings. v2.0 moves this into the translations infrastructure;
// two locales are small enough to live in code for now.
func msg(loc, pt, en string) string {
	if loc == locale.PTBR {
		return pt
	}
	return en
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

// Create serves POST /api/v1/orders. Auth is enforced by the gateway, which
// forwards the validated subject in X-Auth-Subject (internal trust boundary,
// documented in infra/cloud-run.md).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	loc := httputil.Negotiated(r)

	var in NewOrder
	if err := httputil.Decode(r, &in); err != nil || !h.validate(in) {
		httputil.Error(w, http.StatusBadRequest,
			msg(loc, "Itens do pedido inválidos", "Invalid order items"))
		return
	}

	orderID := uuid.NewString()
	itemsJSON, err := json.Marshal(in.Items)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	values := map[string]any{
		"order_id":   orderID,
		"subject":    r.Header.Get("X-Auth-Subject"),
		"locale":     loc,
		"items":      string(itemsJSON),
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.Broker.Publish(r.Context(), h.Stream, values); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "could not register the order")
		return
	}

	httputil.JSON(w, http.StatusCreated, Receipt{
		OrderID: orderID,
		Locale:  loc,
		Message: msg(loc, "Pedido confirmado", "Order confirmed"),
		Items:   in.Items,
	})
}
