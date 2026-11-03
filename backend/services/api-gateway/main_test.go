package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/jwtutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const secret = "test-secret"

// fakeCatalog is the in-memory CatalogService used behind bufconn.
type fakeCatalog struct {
	vitrine.UnimplementedCatalogServiceServer
	listLocale   string
	getLocale    string
	getID        string
	searchLocale string
	searchQuery  string
	searchResult *vitrine.SearchProductsResponse
	getErr       error
	searchErr    error
}

func (f *fakeCatalog) ListProducts(_ context.Context, req *vitrine.ListProductsRequest) (*vitrine.ListProductsResponse, error) {
	f.listLocale = req.GetLocale()
	return &vitrine.ListProductsResponse{
		Locale: req.GetLocale(),
		Items: []*vitrine.Product{{
			Id:          "p-1",
			Name:        "เสื้อกันฝนกันน้ำมีฮู้ด",
			Description: "desc",
			AltText:     "alt",
			Price:       &vitrine.Price{MinorUnits: 45900, Currency: "THB", Formatted: "฿ 459.00"},
		}},
	}, nil
}

func (f *fakeCatalog) GetProduct(_ context.Context, req *vitrine.GetProductRequest) (*vitrine.GetProductResponse, error) {
	f.getLocale = req.GetLocale()
	f.getID = req.GetId()
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &vitrine.GetProductResponse{Product: &vitrine.Product{
		Id:    req.GetId(),
		Name:  "Produto",
		Price: &vitrine.Price{MinorUnits: 7900, Currency: "BRL", Formatted: "R$ 79,00"},
	}}, nil
}

func (f *fakeCatalog) SearchProducts(_ context.Context, req *vitrine.SearchProductsRequest) (*vitrine.SearchProductsResponse, error) {
	f.searchLocale = req.GetLocale()
	f.searchQuery = req.GetQuery()
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchResult != nil {
		return f.searchResult, nil
	}
	return &vitrine.SearchProductsResponse{Locale: req.GetLocale(), Strategy: "semantic", Items: []*vitrine.Product{
		{Id: "p-9", Name: "Nón lá", Price: &vitrine.Price{MinorUnits: 120000, Currency: "VND", Formatted: "120.000 ₫"}},
	}}, nil
}

// fakeOrder is the in-memory OrderService used behind bufconn.
type fakeOrder struct {
	vitrine.UnimplementedOrderServiceServer
	got *vitrine.CreateOrderRequest
}

func (f *fakeOrder) CreateOrder(_ context.Context, req *vitrine.CreateOrderRequest) (*vitrine.CreateOrderResponse, error) {
	f.got = req
	return &vitrine.CreateOrderResponse{OrderId: "ord-1", Locale: req.GetLocale(), Message: "ยืนยันคำสั่งซื้อแล้ว"}, nil
}

// startBufconn serves reg on an in-memory listener and returns a client conn.
func startBufconn(t *testing.T, reg func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	reg(s)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// newGateway builds the Gateway over bufconn fakes plus an httptest mux that
// mirrors the production routes under test.
func newGateway(t *testing.T, cat *fakeCatalog, ord *fakeOrder) *httptest.Server {
	t.Helper()
	catConn := startBufconn(t, func(s *grpc.Server) { vitrine.RegisterCatalogServiceServer(s, cat) })
	ordConn := startBufconn(t, func(s *grpc.Server) { vitrine.RegisterOrderServiceServer(s, ord) })

	g := &Gateway{
		Secret: secret,
		Clients: &Clients{
			Catalog: vitrine.NewCatalogServiceClient(catConn),
			Order:   vitrine.NewOrderServiceClient(ordConn),
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/token", g.tokenHandler)
	mux.HandleFunc("GET /api/v1/products", g.productsHandler)
	mux.HandleFunc("GET /api/v1/products/{id}", g.productHandler)
	mux.HandleFunc("GET /api/v1/search", g.searchHandler)
	mux.HandleFunc("POST /api/v1/orders", g.ordersHandler)
	return httptest.NewServer(mux)
}

func TestTokenEndpointIssuesParsableJWT(t *testing.T) {
	g := &Gateway{Secret: secret}
	rec := httptest.NewRecorder()
	g.tokenHandler(rec, httptest.NewRequest("POST", "/api/v1/auth/token", strings.NewReader(`{"sub":"demo-user"}`)))

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	sub, err := jwtutil.Parse(secret, out.AccessToken)
	if err != nil || sub != "demo-user" {
		t.Fatalf("token must parse to demo-user, got %q err %v", sub, err)
	}
}

func TestTokenEndpointRejectsEmptySubject(t *testing.T) {
	g := &Gateway{Secret: secret}
	rec := httptest.NewRecorder()
	g.tokenHandler(rec, httptest.NewRequest("POST", "/api/v1/auth/token", strings.NewReader(`{}`)))
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOrdersRequireBearer(t *testing.T) {
	ord := &fakeOrder{}
	gw := newGateway(t, &fakeCatalog{}, ord)

	resp, err := http.Post(gw.URL+"/api/v1/orders", "application/json",
		strings.NewReader(`{"items":[{"product_id":"x","quantity":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if ord.got != nil {
		t.Fatal("unauthenticated requests must not reach the order service")
	}
}

func TestOrdersTravelOverGRPCLocaleAndSubject(t *testing.T) {
	ord := &fakeOrder{}
	gw := newGateway(t, &fakeCatalog{}, ord)
	tok, _ := jwtutil.Generate(secret, "demo-user", time.Minute)

	req, _ := http.NewRequest("POST", gw.URL+"/api/v1/orders",
		strings.NewReader(`{"items":[{"product_id":"p-1","quantity":2}]}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept-Language", "th-TH,th;q=0.9,en;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if ord.got == nil {
		t.Fatal("order service was not called")
	}
	// v2.0: th-TH is first-class, so the negotiated locale must be th-TH.
	if ord.got.GetLocale() != "th-TH" {
		t.Fatalf("locale = %q, want th-TH", ord.got.GetLocale())
	}
	if ord.got.GetSubject() != "demo-user" {
		t.Fatalf("subject = %q, want demo-user", ord.got.GetSubject())
	}
	if len(ord.got.GetItems()) != 1 || ord.got.GetItems()[0].GetProductId() != "p-1" {
		t.Fatalf("items = %v", ord.got.GetItems())
	}
	var body struct {
		OrderID string `json:"order_id"`
		Message string `json:"message"`
		Locale  string `json:"locale"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.OrderID != "ord-1" || body.Locale != "th-TH" || body.Message != "ยืนยันคำสั่งซื้อแล้ว" {
		t.Fatalf("receipt = %+v", body)
	}
}

func TestProductsListOverGRPC(t *testing.T) {
	cat := &fakeCatalog{}
	gw := newGateway(t, cat, &fakeOrder{})

	resp, err := http.Get(gw.URL + "/api/v1/products")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if cat.listLocale != "pt-BR" {
		t.Fatalf("locale sent to catalog = %q, want pt-BR (default)", cat.listLocale)
	}
	var body struct {
		Locale string `json:"locale"`
		Items  []struct {
			ID    string `json:"id"`
			Price struct {
				Formatted string `json:"formatted"`
			} `json:"price"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Locale != "pt-BR" || len(body.Items) != 1 || body.Items[0].Price.Formatted != "฿ 459.00" {
		t.Fatalf("body = %+v", body)
	}
}

func TestProductsDetailNotFoundMapsTo404(t *testing.T) {
	cat := &fakeCatalog{getErr: status.Error(codes.NotFound, "product not found")}
	gw := newGateway(t, cat, &fakeOrder{})

	resp, err := http.Get(gw.URL + "/api/v1/products/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if cat.getID != "missing" {
		t.Fatalf("id sent = %q, want missing", cat.getID)
	}
}

func TestSearchEndpointOverGRPC(t *testing.T) {
	cat := &fakeCatalog{}
	gw := newGateway(t, cat, &fakeOrder{})

	resp, err := http.Get(gw.URL + "/api/v1/search?q=" + url.QueryEscape("nón lá") + "&limit=5")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if cat.searchQuery != "nón lá" || cat.searchLocale != "pt-BR" {
		t.Fatalf("search request = locale %q query %q", cat.searchLocale, cat.searchQuery)
	}
	var body struct {
		Strategy string `json:"strategy"`
		Items    []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Strategy != "semantic" || len(body.Items) != 1 || body.Items[0].ID != "p-9" {
		t.Fatalf("body = %+v", body)
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	gw := newGateway(t, &fakeCatalog{}, &fakeOrder{})
	resp, err := http.Get(gw.URL + "/api/v1/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAIProxyRewritesToDescribe(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
	}))
	defer backend.Close()

	target, _ := url.Parse(backend.URL)
	g := &Gateway{Secret: secret, AIURL: target}
	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/ai/alt-text", g.proxy(g.AIURL, false, "/describe"))
	gw := httptest.NewServer(mux)
	defer gw.Close()

	resp, err := http.Post(gw.URL+"/api/v1/ai/alt-text", "multipart/form-data; boundary=x", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotPath != "/describe" {
		t.Fatalf("backend path = %q, want /describe", gotPath)
	}
}

func TestImageUploadStampsLocaleAndSubject(t *testing.T) {
	var gotLocale, gotSubject string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLocale = r.Header.Get("X-Client-Locale")
		gotSubject = r.Header.Get("X-Auth-Subject")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	target, _ := url.Parse(backend.URL)
	g := &Gateway{Secret: secret, CatalogHTTP: target}
	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/products/{id}/image", g.proxy(g.CatalogHTTP, true, ""))
	gw := httptest.NewServer(mux)
	defer gw.Close()

	tok, _ := jwtutil.Generate(secret, "demo-user", time.Minute)
	req, _ := http.NewRequest("POST", gw.URL+"/api/v1/products/p-1/image",
		strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"image\"; filename=\"a.jpg\"\r\n\r\nbytes\r\n--x--\r\n"))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if gotLocale != "vi-VN" {
		t.Fatalf("stamped locale = %q, want vi-VN", gotLocale)
	}
	if gotSubject != "demo-user" {
		t.Fatalf("subject header = %q, want demo-user", gotSubject)
	}
}
