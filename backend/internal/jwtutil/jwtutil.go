// Package jwtutil issues and validates the HS256 tokens used by the gateway.
package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalid is returned for expired, tampered or malformed tokens.
var ErrInvalid = errors.New("invalid token")

// Generate returns a signed bearer token for the subject, valid for ttl.
func Generate(secret, subject string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    "vitrine-que-fala",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse validates the token signature and expiry, returning the subject.
func Parse(secret, tokenStr string) (string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !t.Valid {
		return "", ErrInvalid
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalid
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", ErrInvalid
	}
	return sub, nil
}
