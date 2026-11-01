package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/jwtutil"
)

const secret = "test-secret"

func newGateway(t *testing.T, backendURL string) (*Gateway, *httptest.Server) {
	t.Helper()
	target, err := url.Parse(backendURL)
	if err != nil {
		t.Fatal(err)
	}
	g := &Gateway{Secret: secret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/token", g.tokenHandler)
	mux.Handle("GET /api/v1/products", g.proxy(target, false, ""))
	mux.Handle("POST /api/v1/orders", g.proxy(target, true, ""))
	mux.Handle("POST /api/v1/ai/alt-text", g.proxy(target, false, "/describe"))
	return g, httptest.NewServer(mux)
}

func TestTokenEndpointIssuesParsableJWT(t *testing.T) {
	g, _ := newGateway(t, "http://backend.invalid")
	body := strings.NewReader(`{"sub":"demo-user"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/token", body)
	rec := httptest.NewRecorder()
	g.tokenHandler(rec, req)

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
	g, _ := newGateway(t, "http://backend.invalid")
	rec := httptest.NewRecorder()
	g.tokenHandler(rec, httptest.NewRequest("POST", "/api/v1/auth/token", strings.NewReader(`{}`)))
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOrdersRequireBearer(t *testing.T) {
	hits := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(200)
	}))
	defer backend.Close()

	_, gw := newGateway(t, backend.URL)
	resp, err := http.Post(gw.URL+"/api/v1/orders", "application/json",
		strings.NewReader(`{"items":[{"product_id":"x","quantity":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if hits != 0 {
		t.Fatal("unauthenticated requests must not reach the backend")
	}
}

func TestProxyStampsLocaleAndSubject(t *testing.T) {
	var gotLocale, gotSubject string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLocale = r.Header.Get("X-Client-Locale")
		gotSubject = r.Header.Get("X-Auth-Subject")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	_, gw := newGateway(t, backend.URL)
	tok, _ := jwtutil.Generate(secret, "demo-user", time.Minute)

	req, _ := http.NewRequest("POST", gw.URL+"/api/v1/orders",
		strings.NewReader(`{"items":[{"product_id":"x","quantity":1}]}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept-Language", "th-TH,th;q=0.9,en;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if gotLocale != "en-US" {
		t.Fatalf("negotiated locale = %q, want en-US (th unsupported)", gotLocale)
	}
	if gotSubject != "demo-user" {
		t.Fatalf("subject header = %q, want demo-user", gotSubject)
	}
}

func TestAIProxyRewritesToDescribe(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
	}))
	defer backend.Close()

	_, gw := newGateway(t, backend.URL)
	resp, err := http.Post(gw.URL+"/api/v1/ai/alt-text", "multipart/form-data; boundary=x", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotPath != "/describe" {
		t.Fatalf("backend path = %q, want /describe", gotPath)
	}
}
