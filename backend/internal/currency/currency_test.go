package currency

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		minor int64
		code  string
		loc   string
		want  string
	}{
		// v1.0 behavior (plus thousands grouping, ADR-0012).
		{7900, "BRL", "pt-BR", "R$ 79,00"},
		{7900, "BRL", "en-US", "R$ 79.00"},
		{123456, "BRL", "pt-BR", "R$ 1.234,56"},
		{500, "XXX", "en-US", "XXX 5.00"},
		// v2.0 markets.
		{150000, "IDR", "id-ID", "Rp 150.000"}, // zero-decimal, dot groups
		{150000, "IDR", "en-US", "Rp 150,000"}, // zero-decimal, comma groups
		{45900, "THB", "th-TH", "฿ 459.00"},    // satang keeps 2 decimals
		{120000, "VND", "vi-VN", "120.000 ₫"},  // suffix convention
		{3200, "TWD", "zh-TW", "NT$ 32.00"},    // fen keeps 2 decimals
		{-7900, "BRL", "pt-BR", "R$ -79,00"},   // negative safety
		{999999, "VND", "vi-VN", "999.999 ₫"},  // no fractional part ever
	}
	for _, tt := range tests {
		if got := Format(tt.minor, tt.code, tt.loc); got != tt.want {
			t.Fatalf("Format(%d,%q,%q) = %q, want %q", tt.minor, tt.code, tt.loc, got, tt.want)
		}
	}
}
