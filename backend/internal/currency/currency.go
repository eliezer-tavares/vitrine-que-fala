// Package currency formats minor-unit prices for the supported markets.
// v2.0 serves BRL, USD, IDR, THB, VND and TWD (ADR-0012: per-market
// currencies are data — the maps below grow with the locale list, the
// frontend repeats the job with Intl.NumberFormat).
package currency

import (
	"fmt"
	"strings"
)

var symbols = map[string]string{
	"BRL": "R$",
	"USD": "US$",
	"IDR": "Rp",
	"THB": "฿",
	"VND": "₫",
	"TWD": "NT$",
}

// zeroDecimal lists currencies whose day-to-day prices have no fractional
// part: prices arrive already in whole units (minor == major).
var zeroDecimal = map[string]bool{"IDR": true, "VND": true}

// suffix lists currencies written after the amount in their market
// (Vietnamese: "120.000 ₫").
var suffix = map[string]bool{"VND": true}

// separators returns (thousands, decimal) for a locale. pt-BR, id-ID and
// vi-VN write 1.234,56; most other locales write 1,234.56.
func separators(loc string) (string, string) {
	l := strings.ToLower(loc)
	if strings.HasPrefix(l, "pt") || strings.HasPrefix(l, "id") || strings.HasPrefix(l, "vi") {
		return ".", ","
	}
	return ",", "."
}

// group inserts the thousands separator every three digits from the right.
func group(digits, sep string) string {
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	pre := len(digits) % 3
	if pre > 0 {
		b.WriteString(digits[:pre])
		b.WriteString(sep)
	}
	for i := pre; i < len(digits); i += 3 {
		b.WriteString(digits[i : i+3])
		if i+3 < len(digits) {
			b.WriteString(sep)
		}
	}
	return b.String()
}

// Format renders minor units as a display string for receipts and API
// convenience. It never fails: an unknown currency code is echoed verbatim.
func Format(minor int64, code, loc string) string {
	sym := symbols[code]
	if sym == "" {
		sym = code
	}
	thou, dec := separators(loc)

	neg := minor < 0
	if neg {
		minor = -minor
	}
	var body string
	if zeroDecimal[code] {
		body = group(fmt.Sprintf("%d", minor), thou)
	} else {
		body = group(fmt.Sprintf("%d", minor/100), thou) +
			dec + fmt.Sprintf("%02d", minor%100)
	}
	if neg {
		body = "-" + body
	}
	if suffix[code] {
		return fmt.Sprintf("%s %s", body, sym)
	}
	return fmt.Sprintf("%s %s", sym, body)
}
