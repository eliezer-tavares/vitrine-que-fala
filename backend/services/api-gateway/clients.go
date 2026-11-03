package main

import (
	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Clients bundles the internal gRPC stubs (ADR-0007). The edge speaks HTTP
// to browsers; between services it speaks gRPC.
type Clients struct {
	Catalog vitrine.CatalogServiceClient
	Order   vitrine.OrderServiceClient
}

func dial(addr string) (*grpc.ClientConn, error) {
	// Insecure on purpose: the mesh runs inside the trusted network boundary
	// (docker network / Cloud Run internal ingress with mTLS at the LB —
	// documented in infra/cloud-run.md).
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// NewClients dials catalog and order. The returned cleanup closes both
// connections; call it on shutdown.
func NewClients(catalogAddr, orderAddr string) (*Clients, func(), error) {
	cc, err := dial(catalogAddr)
	if err != nil {
		return nil, nil, err
	}
	oc, err := dial(orderAddr)
	if err != nil {
		_ = cc.Close()
		return nil, nil, err
	}
	cleanup := func() {
		_ = cc.Close()
		_ = oc.Close()
	}
	return &Clients{
		Catalog: vitrine.NewCatalogServiceClient(cc),
		Order:   vitrine.NewOrderServiceClient(oc),
	}, cleanup, nil
}
