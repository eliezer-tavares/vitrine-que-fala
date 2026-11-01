// Package currency formats minor-unit prices for the supported markets.
// v1.0 serves BRL (Brazil); the map grows in v2.0 with IDR, THB, VND and TWD.
package currency

import (
	"fmt"
	"strings"
)

var symbols = map[string]string{
	"BRL": "R$",
	"USD": "US$",
}

// Format renders minor units (cents) as a display string. pt-BR locales use
// a comma decimal separator; other locales a dot. The frontend repeats the
// job with Intl.NumberFormat — this exists for receipts and API convenience.
func Format(minor int64, code, loc string) string {
	sym := symbols[code]
	if sym == "" {
		sym = code
	}
	sep := "."
	if strings.HasPrefix(strings.ToLower(loc), "pt") {
		sep = ","
	}
	return fmt.Sprintf("%s %d%s%02d", sym, minor/100, sep, minor%100)
}
