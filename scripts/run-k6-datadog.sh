#!/bin/bash

set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Uso: ./scripts/run-k6-datadog.sh <script.js>"
  echo "Exemplo: ./scripts/run-k6-datadog.sh k6/baseline.js"
  exit 1
fi

SCRIPT_PATH="$1"
WEB_PORT="${K6_WEB_DASHBOARD_PORT:-5665}"
REPORT_DIR="${K6_REPORT_DIR:-reports}"
REPORT_FILE="${K6_WEB_DASHBOARD_EXPORT:-/work/$REPORT_DIR/compose-k6-datadog-report.html}"

mkdir -p "$REPORT_DIR"

echo "Executando k6 com Datadog via DogStatsD e dashboard web."
echo "Dashboard web: http://127.0.0.1:${WEB_PORT}"
echo "Script: ${SCRIPT_PATH}"

docker compose --profile k6 run --rm --service-ports \
  -e K6_WEB_DASHBOARD_PORT="${WEB_PORT}" \
  -e K6_WEB_DASHBOARD_EXPORT="${REPORT_FILE}" \
  k6-datadog run \
  --tag test_type=load \
  --tag team=platform \
  --tag env=dev \
  --out output-statsd \
  "/work/${SCRIPT_PATH}"
