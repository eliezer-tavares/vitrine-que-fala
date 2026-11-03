package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/locale"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderGRPC serves the internal checkout contract (ADR-0007). The gateway is
// the only intended client: it is also the component that already validated
// the JWT, so subject and locale arrive trusted.
type OrderGRPC struct {
	vitrine.UnimplementedOrderServiceServer
	Handler *Handler
}

// CreateOrder validates and publishes the order, returning the receipt in
// the requested locale.
func (s *OrderGRPC) CreateOrder(ctx context.Context, req *vitrine.CreateOrderRequest) (*vitrine.CreateOrderResponse, error) {
	loc := req.GetLocale()
	if loc == "" {
		loc = locale.Default
	}
	items := make([]OrderItem, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		items = append(items, OrderItem{ProductID: it.GetProductId(), Quantity: int(it.GetQuantity())})
	}

	receipt, err := s.Handler.CreateOrder(ctx, items, req.GetSubject(), loc)
	if errors.Is(err, ErrInvalidOrder) {
		return nil, status.Error(codes.InvalidArgument, localized(invalidMessages, loc))
	}
	if err != nil {
		slog.Error("grpc CreateOrder failed", "error", err)
		return nil, status.Error(codes.Internal, "could not register the order")
	}

	return &vitrine.CreateOrderResponse{
		OrderId: receipt.OrderID,
		Locale:  receipt.Locale,
		Message: receipt.Message,
	}, nil
}
