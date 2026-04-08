#!/bin/bash

set -e

curl -s -X POST http://localhost:8081/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{"asyncLagMs": 1500}' > /dev/null

echo "async_lag habilitado no antifraud-service"
