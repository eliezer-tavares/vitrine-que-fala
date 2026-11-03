package locale

import "testing"

func TestNegotiate(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"empty header falls back to default", "", Default},
		{"exact match", "en-US,en;q=0.9", ENUS},
		{"language only", "en", ENUS},
		{"pt prefix", "pt", PTBR},
		{"unsupported only falls back to default", "fr-FR,fr;q=0.9", Default},
		{"fallback chain walks to supported en", "kok-KOK,kok;q=0.9,en;q=0.8", ENUS},
		{"q ordering wins", "pt;q=0.5, en;q=0.9", ENUS},
		{"case insensitive", "PT-br", PTBR},
		{"wildcard falls back to default", "*", Default},
		{"garbage falls back to default", ";;,", Default},
		// v2.0: the four Shopee-market locales are now first-class (ADR-0003).
		{"thai exact", "th-TH,th;q=0.9,en;q=0.8", THTH},
		{"thai prefix", "th", THTH},
		{"indonesian prefix", "id", IDID},
		{"indonesian exact", "id-ID,id;q=0.9", IDID},
		{"vietnamese", "vi-VN,vi;q=0.9,en;q=0.8", VIVN},
		{"traditional chinese exact", "zh-TW", ZHTW},
		{"chinese prefix falls to supported variant", "zh-CN", ZHTW},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Negotiate(tt.header); got != tt.want {
				t.Fatalf("Negotiate(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}
