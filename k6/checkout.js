// k6/checkout.js — authenticated checkout flow against a DEPLOYED stack
// (ADR-0011). Exercises the token endpoint and the gRPC-backed order
// creation through the edge.
// Usage: BASE_URL=https://... k6 run k6/checkout.js
import http from "k6/http";
import { check, group, sleep } from "k6";
import { Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const checkoutLatency = new Trend("checkout_latency");

export const options = {
  stages: [
    { duration: "20s", target: 3 },
    { duration: "1m", target: 3 },
    { duration: "20s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.02"],
    checkout_latency: ["p(95)<1500"],
  },
};

const PRODUCT_ID = "11111111-1111-1111-1111-111111111111";

export default function () {
  group("token + checkout", () => {
    const tokenRes = http.post(
      `${BASE_URL}/api/v1/auth/token`,
      JSON.stringify({ sub: `k6-${__VU}` }),
      { headers: { "Content-Type": "application/json" } },
    );
    check(tokenRes, { "token issued": (r) => r.status === 200 });
    const token = tokenRes.json("access_token");

    const res = http.post(
      `${BASE_URL}/api/v1/orders`,
      JSON.stringify({ items: [{ product_id: PRODUCT_ID, quantity: 1 }] }),
      {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
          "Accept-Language": "th-TH",
        },
      },
    );
    checkoutLatency.add(res.timings.duration);
    check(res, {
      "checkout 201": (r) => r.status === 201,
      "receipt in Thai": (r) => r.json("locale") === "th-TH",
    });
  });
  sleep(2);
}
