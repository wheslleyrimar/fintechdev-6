#!/bin/bash

# Script helper para ativar lag intencional via endpoint administrativo local

set -e

ADMIN_URL="http://localhost:8080/admin/scenarios"

# Cores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== Ativando Lag Intencional ===${NC}"
echo ""

# Verificar se o serviço está respondendo
if ! curl -s -f "$ADMIN_URL" > /dev/null 2>&1; then
    echo -e "${RED}Erro: endpoint administrativo indisponível em $ADMIN_URL${NC}"
    exit 1
fi

curl -s -X POST "$ADMIN_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "intentionalLagEnabled": true,
    "databaseDelayMs": 2000,
    "cacheDelayMs": 500,
    "externalDelayMs": 1000,
    "lagPercentage": 1.0
  }' > /dev/null

echo ""
echo -e "${GREEN}✓ Lag intencional ativado!${NC}"
echo ""
echo "O lag aparecerá naturalmente nas requisições para /payments"
echo "Configuração:"
echo "  - Database delay: 2000ms"
echo "  - Cache delay: 500ms"
echo "  - External delay: 1000ms"
echo ""
echo "Para desativar: ./scripts/disable-lag.sh"
echo ""
