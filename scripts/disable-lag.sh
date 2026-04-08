#!/bin/bash

# Script helper para desativar lag intencional via endpoint administrativo local

set -e

ADMIN_URL="http://localhost:8080/admin/scenarios"

# Cores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== Desativando Lag Intencional ===${NC}"
echo ""

if ! curl -s -f "$ADMIN_URL" > /dev/null 2>&1; then
    echo -e "${RED}Erro: endpoint administrativo indisponível em $ADMIN_URL${NC}"
    exit 1
fi

curl -s -X POST "$ADMIN_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "intentionalLagEnabled": false,
    "databaseDelayMs": 2000,
    "cacheDelayMs": 500,
    "externalDelayMs": 1000,
    "lagPercentage": 1.0
  }' > /dev/null

echo ""
echo -e "${GREEN}✓ Lag intencional desativado!${NC}"
echo ""
echo "O serviço voltou ao comportamento normal."
echo ""
