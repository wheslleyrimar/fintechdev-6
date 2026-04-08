#!/bin/bash

set -e

curl -s -X POST http://localhost:8080/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{
    "intentionalLagEnabled": false,
    "chattyDb": false,
    "cacheStampede": false,
    "riskDependency": false
  }' > /dev/null

curl -s -X POST http://localhost:8081/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{"asyncLagMs": 0}' > /dev/null

echo "cenarios resetados"
