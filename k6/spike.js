import http from "k6/http";
import { check } from "k6";

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  stages: [
    { duration: "30s", target: 20 },
    { duration: "30s", target: 120 },
    { duration: "1m", target: 120 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.10"],
  },
};

export default function () {
  const res = http.post(`${baseUrl}/payments`, JSON.stringify({
    accountId: `acc-spike-${__VU}-${__ITER}`,
    amount: 999.99,
    currency: "BRL",
  }), {
    headers: {
      "Content-Type": "application/json",
      "X-Correlation-ID": `spike-${__VU}-${__ITER}`,
    },
    tags: { scenario: "spike" },
  });

  check(res, { "status is 201 or protected": (r) => r.status === 201 || r.status === 429 || r.status === 503 });
}
