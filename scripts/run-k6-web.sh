#!/bin/bash

set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Uso: ./scripts/run-k6-web.sh <script.js>"
  echo "Exemplo: ./scripts/run-k6-web.sh k6/baseline.js"
  exit 1
fi

SCRIPT_PATH="$1"
WEB_PORT="${K6_WEB_DASHBOARD_PORT:-5665}"
REPORT_DIR="${K6_REPORT_DIR:-reports}"
REPORT_FILE="${K6_WEB_DASHBOARD_EXPORT:-$REPORT_DIR/$(basename "$SCRIPT_PATH" .js)-report.html}"

mkdir -p "$REPORT_DIR"

if command -v k6 >/dev/null 2>&1; then
  BASE_URL="${BASE_URL:-http://localhost:8080}" \
  K6_WEB_DASHBOARD=true \
  K6_WEB_DASHBOARD_PORT="$WEB_PORT" \
  K6_WEB_DASHBOARD_EXPORT="$REPORT_FILE" \
  k6 run "$SCRIPT_PATH"
  exit 0
fi

echo "k6 não encontrado localmente. Usando container grafana/k6."
echo "Dashboard: http://127.0.0.1:${WEB_PORT}"

docker run --rm -it \
  -p "${WEB_PORT}:${WEB_PORT}" \
  --add-host=host.docker.internal:host-gateway \
  -v "$PWD:/work" \
  -w /work \
  -e BASE_URL="${BASE_URL:-http://host.docker.internal:8080}" \
  -e K6_WEB_DASHBOARD=true \
  -e K6_WEB_DASHBOARD_PORT="${WEB_PORT}" \
  -e K6_WEB_DASHBOARD_EXPORT="${REPORT_FILE}" \
  grafana/k6:latest run "$SCRIPT_PATH"
