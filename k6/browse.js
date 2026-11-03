// k6/browse.js — browsing flow against a DEPLOYED stack (ADR-0011).
// Never point this at an 8 GB notebook: run it against Cloud Run.
// Usage: BASE_URL=https://... k6 run k6/browse.js
import http from "k6/http";
import { check, group, sleep } from "k6";
import { Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const listLatency = new Trend("browse_list_latency");

// Ramp kept modest on purpose: this is a sanity profile for a small Cloud
// Run deployment, not a DDoS simulator.
export const options = {
  stages: [
    { duration: "30s", target: 5 },
    { duration: "1m", target: 5 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"],
    browse_list_latency: ["p(95)<800"],
  },
};

const LOCALES = ["pt-BR", "en-US", "id-ID", "th-TH", "vi-VN", "zh-TW"];

export default function () {
  group("browse catalog in every locale", () => {
    for (const locale of LOCALES) {
      const res = http.get(`${BASE_URL}/api/v1/products`, {
        headers: { "Accept-Language": locale },
      });
      listLatency.add(res.timings.duration);
      check(res, {
        [`${locale}: status 200`]: (r) => r.status === 200,
        [`${locale}: locale echoed`]: (r) => r.json("locale") !== undefined,
      });
    }
  });
  sleep(1);
}
