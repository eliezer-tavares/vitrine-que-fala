package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeRepo struct {
	items      []Product
	byID       map[string]Product
	lastLocale string
	searched   bool
	calls      int
}

func (f *fakeRepo) List(_ context.Context, loc string) ([]Product, error) {
	f.lastLocale = loc
	f.calls++
	return f.items, nil
}

func (f *fakeRepo) Get(_ context.Context, id, loc string) (*Product, error) {
	f.lastLocale = loc
	f.calls++
	if p, ok := f.byID[id]; ok {
		return &p, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) Search(_ context.Context, loc, _ string, _ int, _ []float64) ([]Product, error) {
	f.lastLocale = loc
	f.searched = true
	return f.items, nil
}

type memCache struct {
	m    map[string]string
	gets int
}

func (c *memCache) Get(_ context.Context, key string) (string, bool) {
	c.gets++
	v, ok := c.m[key]
	return v, ok
}

func (c *memCache) Set(_ context.Context, key, val string, _ time.Duration) {
	c.m[key] = val
}

func ptProducts() []Product {
	return []Product{{
		ID: "11111111-1111-1111-1111-111111111111", Name: "Poncho impermeável",
		Description: "Poncho de poliéster com capuz", AltText: "Poncho amarelo dobrado sobre uma mesa clara",
		PriceMinor: 7900, Currency: "BRL",
	}}
}

func TestListUsesNegotiatedLocale(t *testing.T) {
	repo := &fakeRepo{items: ptProducts()}
	h := &Handler{Repo: repo, TTL: time.Minute}

	req := httptest.NewRequest("GET", "/api/v1/products", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if repo.lastLocale != "en-US" {
		t.Fatalf("repo received locale %q, want en-US", repo.lastLocale)
	}
	if !strings.Contains(rec.Body.String(), `"locale":"en-US"`) {
		t.Fatalf("response should embed the locale, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"formatted":"R$ 79.00"`) {
		t.Fatalf("price should be formatted for en-US, got: %s", rec.Body.String())
	}
}

func TestListClientLocaleHeaderWins(t *testing.T) {
	repo := &fakeRepo{items: ptProducts()}
	h := &Handler{Repo: repo, TTL: time.Minute}

	req := httptest.NewRequest("GET", "/api/v1/products", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("X-Client-Locale", "pt-BR")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if repo.lastLocale != "pt-BR" {
		t.Fatalf("X-Client-Locale must win, got %q", repo.lastLocale)
	}
	if !strings.Contains(rec.Body.String(), `"formatted":"R$ 79,00"`) {
		t.Fatalf("price should be formatted for pt-BR, got: %s", rec.Body.String())
	}
}

func TestListCacheHitSkipsRepo(t *testing.T) {
	repo := &fakeRepo{items: ptProducts()}
	cache := &memCache{m: map[string]string{}}
	h := &Handler{Repo: repo, Cache: cache, TTL: time.Minute}

	req := httptest.NewRequest("GET", "/api/v1/products", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	h.List(httptest.NewRecorder(), req)
	if repo.calls != 1 {
		t.Fatalf("first request should hit repo, calls=%d", repo.calls)
	}
	if _, ok := cache.m[CacheKey("pt-BR")]; !ok {
		t.Fatal("cache should be filled after first request")
	}
	h.List(httptest.NewRecorder(), req)
	if repo.calls != 1 {
		t.Fatalf("second request should hit cache, calls=%d", repo.calls)
	}
}

func TestGetNotFound(t *testing.T) {
	repo := &fakeRepo{byID: map[string]Product{}}
	h := &Handler{Repo: repo, TTL: time.Minute}
	mux := routeMux(h)

	req := httptest.NewRequest("GET", "/api/v1/products/does-not-exist", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetReturnsLocalizedProduct(t *testing.T) {
	id := "22222222-2222-2222-2222-222222222222"
	repo := &fakeRepo{byID: map[string]Product{
		id: {ID: id, Name: "Automatic umbrella", AltText: "A black umbrella on a white background",
			PriceMinor: 4500, Currency: "BRL"},
	}}
	h := &Handler{Repo: repo, TTL: time.Minute}
	mux := routeMux(h)

	req := httptest.NewRequest("GET", "/api/v1/products/"+id, nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Automatic umbrella") {
		t.Fatalf("body should contain the localized name: %s", rec.Body.String())
	}
}

// routeMux mirrors main.go's routing so r.PathValue("id") works in tests.
func routeMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/products", h.List)
	mux.HandleFunc("GET /api/v1/products/{id}", h.Get)
	return mux
}

func TestCacheKeyEmbedsLocale(t *testing.T) {
	if CacheKey("pt-BR") == CacheKey("en-US") {
		t.Fatal("cache keys must differ per locale")
	}
	if !strings.Contains(CacheKey("pt-BR"), "pt-BR") {
		t.Fatal("cache key must embed the locale")
	}
}

func TestImageEnqueue(t *testing.T) {
	fake := &fakeQueuePublisher{}
	h := &ImageHandler{Queue: fake, Stream: "images.pending"}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/products/{id}/image", h.Enqueue)

	req := httptest.NewRequest("POST", "/api/v1/products/22222222-2222-2222-2222-222222222222/image",
		strings.NewReader(`{"image_url":"https://example.test/p.jpg"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 202 {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if fake.stream != "images.pending" {
		t.Fatalf("stream = %q", fake.stream)
	}
	if fake.values["attempts"] != int64(0) {
		t.Fatal("new jobs must start with attempts=0")
	}
	if fake.values["product_id"] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("product_id = %v", fake.values["product_id"])
	}

	req2 := httptest.NewRequest("POST", "/api/v1/products/x/image", strings.NewReader(`{}`))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 400 {
		t.Fatalf("missing image_url must be 400, got %d", rec2.Code)
	}
}

type fakeQueuePublisher struct {
	stream string
	values map[string]any
}

func (f *fakeQueuePublisher) Publish(_ context.Context, stream string, values map[string]any) error {
	f.stream = stream
	f.values = values
	return nil
}

func TestSearchUsesEmbedderAndReportsStrategy(t *testing.T) {
	repo := &fakeRepo{items: ptProducts()}
	h := &Handler{
		Repo: repo,
		Embedder: fakeEmbedderFunc(func(_ context.Context, texts []string) ([][]float64, error) {
			return [][]float64{{1, 2, 3}}, nil
		}),
	}

	req := httptest.NewRequest("GET", "/api/v1/search?q=poncho", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"strategy":"semantic"`) {
		t.Fatalf("semantic strategy expected, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"query":"poncho"`) {
		t.Fatalf("query echo expected, got: %s", rec.Body.String())
	}
}

func TestSearchEmbedderFailureFallsBackToLexical(t *testing.T) {
	repo := &fakeRepo{items: ptProducts()}
	h := &Handler{
		Repo: repo,
		Embedder: fakeEmbedderFunc(func(_ context.Context, _ []string) ([][]float64, error) {
			return nil, errors.New("model loading")
		}),
	}

	req := httptest.NewRequest("GET", "/api/v1/search?q=poncho", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"strategy":"lexical"`) {
		t.Fatalf("lexical fallback expected, got: %s", rec.Body.String())
	}
}

func TestSearchWithoutEmbedderIsLexical(t *testing.T) {
	repo := &fakeRepo{items: ptProducts()}
	h := &Handler{Repo: repo}

	req := httptest.NewRequest("GET", "/api/v1/search?q=mochila", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	if !strings.Contains(rec.Body.String(), `"strategy":"lexical"`) {
		t.Fatalf("lexical expected, got: %s", rec.Body.String())
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	h := &Handler{Repo: &fakeRepo{}}
	rec := httptest.NewRecorder()
	h.Search(rec, httptest.NewRequest("GET", "/api/v1/search", nil))
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// fakeEmbedderFunc adapts a function to the Embedder interface.
type fakeEmbedderFunc func(ctx context.Context, texts []string) ([][]float64, error)

func (f fakeEmbedderFunc) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return f(ctx, texts)
}
