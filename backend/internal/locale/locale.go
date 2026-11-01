// Package locale implements Accept-Language negotiation for the supported
// locales, with the fallback chain documented in docs/adr/0003 and the
// OpenAPI spec: walk candidates in q-order, return the first supported match
// (exact tag first, then language prefix), otherwise the default locale.
package locale

import (
	"sort"
	"strconv"
	"strings"
)

const (
	PTBR    = "pt-BR"
	ENUS    = "en-US"
	Default = PTBR
)

// Supported lists the locales served in v1.0.
var Supported = []string{PTBR, ENUS}

type candidate struct {
	tag string
	q   float64
}

// Negotiate resolves an Accept-Language header value. It never fails: an
// empty or unparseable header yields the Default locale. Example from the
// plan: "th-TH,th;q=0.9,en;q=0.8" -> en-US (th unsupported, en is next).
func Negotiate(header string) string {
	cands := parse(header)
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].q > cands[j].q })
	for _, c := range cands {
		for _, s := range Supported {
			if strings.EqualFold(c.tag, s) {
				return s
			}
		}
		for _, s := range Supported {
			if langPrefixMatches(c.tag, s) {
				return s
			}
		}
	}
	return Default
}

// langPrefixMatches implements the language-prefix rule: "en-GB" matches
// "en-US" because the language subtag is the same.
func langPrefixMatches(tag, supported string) bool {
	tl, sl := tag, supported
	if i := strings.Index(tag, "-"); i > 0 {
		tl = tag[:i]
	}
	if i := strings.Index(supported, "-"); i > 0 {
		sl = supported[:i]
	}
	return strings.EqualFold(tl, sl)
}

func parse(header string) []candidate {
	var out []candidate
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := strings.Split(part, ";")
		tag := strings.TrimSpace(pieces[0])
		if tag == "" || tag == "*" {
			continue
		}
		q := 1.0
		for _, p := range pieces[1:] {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "q=") {
				if v, err := strconv.ParseFloat(strings.TrimPrefix(p, "q="), 64); err == nil {
					q = v
				}
			}
		}
		out = append(out, candidate{tag: tag, q: q})
	}
	return out
}
