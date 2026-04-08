#!/bin/bash

set -e

curl -s -X POST http://localhost:8080/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{"riskDependency": true}' > /dev/null

echo "risk_dependency habilitado"
