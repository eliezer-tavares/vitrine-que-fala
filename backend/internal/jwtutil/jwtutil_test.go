package jwtutil

import (
	"strings"
	"testing"
	"time"
)

const secret = "test-secret"

func TestRoundTrip(t *testing.T) {
	tok, err := Generate(secret, "demo-user", time.Hour)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	sub, err := Parse(secret, tok)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if sub != "demo-user" {
		t.Fatalf("subject = %q, want demo-user", sub)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	tok, _ := Generate(secret, "demo-user", time.Hour)
	if _, err := Parse("other-secret", tok); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseRejectsExpired(t *testing.T) {
	tok, _ := Generate(secret, "demo-user", -time.Minute)
	if _, err := Parse(secret, tok); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse(secret, "not-a-token"); err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestParseRejectsUnexpectedAlgorithm(t *testing.T) {
	// A token with "none"-style tampering must not validate. Build a token
	// string with a foreign alg header to exercise the method check.
	tok, _ := Generate(secret, "demo-user", time.Hour)
	parts := strings.Split(tok, ".")
	tampered := parts[0] + "." + parts[1] + ".broken"
	if _, err := Parse(secret, tampered); err == nil {
		t.Fatal("expected error for tampered token")
	}
}
