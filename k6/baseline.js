import http from "k6/http";
import { check, sleep } from "k6";

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  vus: 10,
  duration: "2m",
  thresholds: {
    http_req_failed: ["rate<0.02"],
    http_req_duration: ["p(95)<500"],
  },
};

export default function () {
  const id = `acc-${__VU}-${__ITER}`;
  const res = http.post(`${baseUrl}/payments`, JSON.stringify({
    accountId: id,
    amount: 100.5,
    currency: "BRL",
  }), {
    headers: {
      "Content-Type": "application/json",
      "X-Correlation-ID": `baseline-${__VU}-${__ITER}`,
    },
    tags: { scenario: "baseline" },
  });

  check(res, { "status is 201": (r) => r.status === 201 });
  sleep(0.2);
}
