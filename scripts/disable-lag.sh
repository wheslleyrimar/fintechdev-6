#!/bin/bash

# Script helper para desativar lag intencional via docker-compose

set -e

COMPOSE_FILE="docker-compose.yml"

# Cores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== Desativando Lag Intencional ===${NC}"
echo ""

# Verificar se docker-compose.yml existe
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}Erro: $COMPOSE_FILE não encontrado${NC}"
    exit 1
fi

# Desativar lag no docker-compose.yml
# Comentar as variáveis de ambiente
sed -i.bak \
    -e 's/- INTENTIONAL_LAG_ENABLED=true/# - INTENTIONAL_LAG_ENABLED=true/' \
    -e 's/- INTENTIONAL_LAG_DATABASE_MS=2000/# - INTENTIONAL_LAG_DATABASE_MS=2000/' \
    -e 's/- INTENTIONAL_LAG_CACHE_MS=500/# - INTENTIONAL_LAG_CACHE_MS=500/' \
    -e 's/- INTENTIONAL_LAG_EXTERNAL_MS=1000/# - INTENTIONAL_LAG_EXTERNAL_MS=1000/' \
    "$COMPOSE_FILE"

# Remover arquivo .bak criado pelo sed
rm -f "${COMPOSE_FILE}.bak"

echo -e "${GREEN}✓ Variáveis de ambiente de lag desativadas no docker-compose.yml${NC}"
echo ""
echo "Reiniciando payment-service..."
docker compose up -d payment-service

echo ""
echo -e "${GREEN}✓ Lag intencional desativado!${NC}"
echo ""
echo "O serviço voltou ao comportamento normal."
echo ""
