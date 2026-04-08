#!/bin/bash

set -e

curl -s -X POST http://localhost:8080/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{"cacheStampede": true}' > /dev/null

echo "cache_stampede habilitado"
