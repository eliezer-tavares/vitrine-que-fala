package main

import (
	"context"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CatalogClient persists worker output through the catalog's gRPC contract
// (ADR-0007): the worker no longer holds a database connection — the service
// that owns product_translations owns its writes.
type CatalogClient struct {
	Stub vitrine.CatalogServiceClient
}

// NewCatalogClient dials the catalog gRPC server (insecure is fine: the mesh
// runs inside the trusted network boundary; Cloud Run adds its own TLS —
// documented in infra/cloud-run.md).
func NewCatalogClient(addr string) (*CatalogClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &CatalogClient{Stub: vitrine.NewCatalogServiceClient(conn)}, nil
}

// SetAltText writes the per-locale alt-text produced by the AI pipeline.
func (c *CatalogClient) SetAltText(ctx context.Context, productID string, alt map[string]string) error {
	_, err := c.Stub.UpdateAltText(ctx, &vitrine.UpdateAltTextRequest{
		ProductId: productID,
		AltTexts:  alt,
	})
	return err
}

// SetEmbedding writes one 384-dim multilingual-e5 vector (ADR-0009).
func (c *CatalogClient) SetEmbedding(ctx context.Context, productID, locale string, vec []float64) error {
	f := make([]float32, len(vec))
	for i, v := range vec {
		f[i] = float32(v)
	}
	_, err := c.Stub.UpdateEmbedding(ctx, &vitrine.UpdateEmbeddingRequest{
		ProductId: productID,
		Locale:    locale,
		Vector:    f,
	})
	return err
}
