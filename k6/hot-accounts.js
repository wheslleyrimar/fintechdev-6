import http from "k6/http";
import { check, sleep } from "k6";

const hotAccounts = ["hot-1", "hot-2", "hot-3", "hot-4"];
const baseUrl = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  vus: 30,
  duration: "3m",
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<1500"],
  },
};

export default function () {
  const accountId = hotAccounts[__ITER % hotAccounts.length];
  const res = http.post(`${baseUrl}/payments`, JSON.stringify({
    accountId,
    amount: 250.75,
    currency: "BRL",
  }), {
    headers: {
      "Content-Type": "application/json",
      "X-Correlation-ID": `hot-${__VU}-${__ITER}`,
    },
    tags: { scenario: "hot_accounts" },
  });

  check(res, { "status is 201": (r) => r.status === 201 });
  sleep(0.1);
}
