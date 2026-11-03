package main

import (
	"context"
	"testing"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeWriter struct {
	alt     map[string]string
	embeds  map[string][]float64
	lastPID string
	err     error
}

func (f *fakeWriter) SetAltText(_ context.Context, productID string, alt map[string]string) error {
	if f.err != nil {
		return f.err
	}
	f.lastPID = productID
	f.alt = alt
	return nil
}

func (f *fakeWriter) SetEmbedding(_ context.Context, productID, locale string, vec []float64) error {
	if f.err != nil {
		return f.err
	}
	f.lastPID = productID
	if f.embeds == nil {
		f.embeds = map[string][]float64{}
	}
	f.embeds[locale] = vec
	return nil
}

func TestGRPCListProductsDefaultsAndPrices(t *testing.T) {
	s := &CatalogGRPC{
		Repo:  &fakeRepo{items: ptProducts()},
		Cache: &memCache{m: map[string]string{}},
		TTL:   time.Minute,
	}
	res, err := s.ListProducts(context.Background(), &vitrine.ListProductsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetLocale() != "pt-BR" {
		t.Fatalf("default locale = %q, want pt-BR", res.GetLocale())
	}
	if len(res.GetItems()) != 1 || res.GetItems()[0].GetPrice().GetFormatted() != "R$ 79,00" {
		t.Fatalf("items = %v", res.GetItems())
	}
}

func TestGRPCGetProductNotFound(t *testing.T) {
	s := &CatalogGRPC{Repo: &fakeRepo{byID: map[string]Product{}}}
	_, err := s.GetProduct(context.Background(), &vitrine.GetProductRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want NotFound", status.Code(err))
	}
}

func TestGRPCSearchRequiresQuery(t *testing.T) {
	s := &CatalogGRPC{Repo: &fakeRepo{}}
	_, err := s.SearchProducts(context.Background(), &vitrine.SearchProductsRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestGRPCSearchWithoutEmbedderIsLexical(t *testing.T) {
	s := &CatalogGRPC{Repo: &fakeRepo{items: ptProducts()}}
	res, err := s.SearchProducts(context.Background(), &vitrine.SearchProductsRequest{Query: "poncho"})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetStrategy() != "lexical" {
		t.Fatalf("strategy = %q, want lexical", res.GetStrategy())
	}
}

func TestGRPCUpdateAltTextWritesThroughWriter(t *testing.T) {
	w := &fakeWriter{}
	s := &CatalogGRPC{Repo: &fakeRepo{}, Writer: w}
	if _, err := s.UpdateAltText(context.Background(), &vitrine.UpdateAltTextRequest{
		ProductId: "p-1",
		AltTexts:  map[string]string{"pt-BR": "Poncho amarelo", "en-US": "Yellow poncho"},
	}); err != nil {
		t.Fatal(err)
	}
	if w.lastPID != "p-1" || w.alt["pt-BR"] != "Poncho amarelo" {
		t.Fatalf("writer got pid=%q alt=%v", w.lastPID, w.alt)
	}
}

func TestGRPCUpdateAltTextRequiresProductID(t *testing.T) {
	s := &CatalogGRPC{Repo: &fakeRepo{}, Writer: &fakeWriter{}}
	_, err := s.UpdateAltText(context.Background(), &vitrine.UpdateAltTextRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestGRPCUpdateEmbeddingValidatesAndConverts(t *testing.T) {
	s := &CatalogGRPC{Repo: &fakeRepo{}, Writer: &fakeWriter{}}
	_, err := s.UpdateEmbedding(context.Background(), &vitrine.UpdateEmbeddingRequest{ProductId: "p", Locale: "pt-BR"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty vector must be InvalidArgument, got %v", status.Code(err))
	}

	w := &fakeWriter{}
	s.Writer = w
	if _, err := s.UpdateEmbedding(context.Background(), &vitrine.UpdateEmbeddingRequest{
		ProductId: "p-1", Locale: "th-TH", Vector: []float32{1, 2, 3},
	}); err != nil {
		t.Fatal(err)
	}
	if got := w.embeds["th-TH"]; len(got) != 3 || got[2] != 3 {
		t.Fatalf("embedding = %v", got)
	}
}

func TestGRPCHandlerErrorMapsToInternal(t *testing.T) {
	w := &fakeWriter{err: context.DeadlineExceeded}
	s := &CatalogGRPC{Repo: &fakeRepo{}, Writer: w}
	_, err := s.UpdateAltText(context.Background(), &vitrine.UpdateAltTextRequest{
		ProductId: "p-1", AltTexts: map[string]string{"pt-BR": "x"},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}
