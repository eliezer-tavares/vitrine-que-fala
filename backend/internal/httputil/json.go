// Package httputil holds tiny HTTP helpers shared by the services: JSON
// encoding/decoding and the locale negotiation entry point used at the edge
// and behind it.
package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/locale"
)

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes a JSON error payload.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Decode strictly decodes the request body into dst.
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// Negotiated resolves the request locale. The gateway sets X-Client-Locale
// after negotiating Accept-Language; downstream services trust it. Direct
// callers (dev, tests) still get Accept-Language negotiation.
func Negotiated(r *http.Request) string {
	if x := r.Header.Get("X-Client-Locale"); x != "" {
		return x
	}
	return locale.Negotiate(r.Header.Get("Accept-Language"))
}
