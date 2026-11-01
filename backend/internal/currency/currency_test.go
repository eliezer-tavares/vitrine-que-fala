package currency

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		minor int64
		code  string
		loc   string
		want  string
	}{
		{7900, "BRL", "pt-BR", "R$ 79,00"},
		{7900, "BRL", "en-US", "R$ 79.00"},
		{123456, "BRL", "pt-BR", "R$ 1234,56"},
		{500, "XXX", "en-US", "XXX 5.00"},
	}
	for _, tt := range tests {
		if got := Format(tt.minor, tt.code, tt.loc); got != tt.want {
			t.Fatalf("Format(%d,%q,%q) = %q, want %q", tt.minor, tt.code, tt.loc, got, tt.want)
		}
	}
}
