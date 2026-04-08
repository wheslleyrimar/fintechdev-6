#!/bin/bash

set -e

curl -s -X POST http://localhost:8080/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{"chattyDb": true}' > /dev/null

echo "chatty_db habilitado"
