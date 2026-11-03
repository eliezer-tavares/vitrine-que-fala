// k6/search.js — search flow (semantic pgvector + lexical fallback) against a
// DEPLOYED stack (ADR-0011). The response `strategy` field tells which path
// answered, so the report separates the two.
// Usage: BASE_URL=https://... k6 run k6/search.js
import http from "k6/http";
import { check, group, sleep } from "k6";
import { Trend, Counter } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const searchLatency = new Trend("search_latency");
const semanticHits = new Counter("search_semantic");
const lexicalHits = new Counter("search_lexical");

export const options = {
  stages: [
    { duration: "30s", target: 5 },
    { duration: "1m", target: 5 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"],
    search_latency: ["p(95)<1200"],
  },
};

// One Latin query and one Thai query: the Thai one is the multilingual
// smoke — lexical ILIKE rarely matches Thai input, so it exercises the
// semantic path when embeddings are indexed.
const QUERIES = [
  { locale: "pt-BR", q: "poncho" },
  { locale: "th-TH", q: "เสื้อกันฝน" },
  { locale: "vi-VN", q: "balo" },
];

export default function () {
  group("search across markets", () => {
    for (const { locale, q } of QUERIES) {
      const res = http.get(
        `${BASE_URL}/api/v1/search?q=${encodeURIComponent(q)}`,
        { headers: { "Accept-Language": locale } },
      );
      searchLatency.add(res.timings.duration);
      const strategy = res.json("strategy");
      if (strategy === "semantic") {
        semanticHits.add(1);
      } else if (strategy === "lexical") {
        lexicalHits.add(1);
      }
      check(res, {
        [`${locale}: status 200`]: (r) => r.status === 200,
        [`${locale}: strategy reported`]: (r) =>
          ["semantic", "lexical"].includes(String(r.json("strategy"))),
      });
    }
  });
  sleep(1);
}
