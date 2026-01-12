#!/bin/bash

# Script para demonstrar lag intencional e investigação via observabilidade
# Demonstra: lag intencional → observabilidade → diagnóstico → escalonamento
# 
# IMPORTANTE: O lag agora é controlado via variáveis de ambiente no docker-compose.yml
# Para ativar o lag, edite o docker-compose.yml e reinicie o serviço

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PAYMENT_URL="http://localhost:8080/payments"
COMPOSE_FILE="docker-compose.yml"

echo -e "${BLUE}=== Demonstração: Lag Intencional e Observabilidade ===${NC}"
echo ""
echo "Este script demonstra o ciclo completo:"
echo "1. Baseline sem lag"
echo "2. Ativar lag intencional (via variáveis de ambiente)"
echo "3. Observar sinais de latência (métricas, logs, traces)"
echo "4. Diagnosticar a causa do problema"
echo "5. Documentar processo de escalonamento"
echo ""
echo -e "${YELLOW}NOTA:${NC} O lag é um problema que aparece naturalmente nas requisições."
echo "Ele é controlado via variáveis de ambiente no docker-compose.yml"
echo ""

# Verificar se o serviço está rodando
if ! curl -s -f "$PAYMENT_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}Erro: Payment service não está rodando em $PAYMENT_URL${NC}"
    echo "Execute: docker compose up -d"
    exit 1
fi

echo -e "${GREEN}✓ Serviço está rodando${NC}"
echo ""

# Verificar se lag está ativo
LAG_ENABLED=$(docker compose exec -T payment-service printenv INTENTIONAL_LAG_ENABLED 2>/dev/null || echo "")
if [ "$LAG_ENABLED" = "true" ]; then
    echo -e "${YELLOW}⚠ Lag intencional já está ativo!${NC}"
    echo "Para desativar, edite docker-compose.yml e reinicie: docker compose up -d payment-service"
    echo ""
    read -p "Deseja continuar mesmo com lag ativo? (s/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Ss]$ ]]; then
        echo "Edite docker-compose.yml para desativar o lag e execute: docker compose up -d payment-service"
        exit 0
    fi
fi

# Fase 1: Baseline - Sem lag
echo -e "${YELLOW}=== FASE 1: Baseline (Sem Lag) ===${NC}"
echo "Enviando 5 requisições para estabelecer baseline..."
echo ""

for i in {1..5}; do
    correlation_id="baseline-$(date +%s)-$i"
    response=$(curl -s -w "\n%{http_code}\n%{time_total}" \
        -X POST "$PAYMENT_URL" \
        -H "Content-Type: application/json" \
        -H "X-Correlation-ID: $correlation_id" \
        -d "{
            \"accountId\": \"acc-$i\",
            \"amount\": 100.50,
            \"currency\": \"BRL\"
        }" 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n 2 | head -n 1)
    time_total=$(echo "$response" | tail -n 1)
    
    if [ "$http_code" = "201" ]; then
        echo -e "  ${GREEN}✓${NC} Requisição $i: ${time_total}s (correlation_id: $correlation_id)"
    else
        echo -e "  ${RED}✗${NC} Requisição $i: HTTP $http_code"
    fi
done

echo ""
echo "Baseline estabelecido. Verifique métricas no Prometheus:"
echo "  - http://localhost:9090"
echo "  - Query: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))"
echo ""
read -p "Pressione Enter para continuar..."

# Fase 2: Instruções para ativar lag
echo ""
echo -e "${RED}=== FASE 2: Ativando Lag Intencional ===${NC}"
echo ""
echo "Para ativar o lag intencional (simular problema de latência):"
echo ""
echo "1. Edite o arquivo: $COMPOSE_FILE"
echo "2. Descomente as variáveis de ambiente no serviço payment-service:"
echo ""
echo -e "${BLUE}   environment:${NC}"
echo -e "${BLUE}     - INTENTIONAL_LAG_ENABLED=true${NC}"
echo -e "${BLUE}     - INTENTIONAL_LAG_DATABASE_MS=2000${NC}"
echo -e "${BLUE}     - INTENTIONAL_LAG_CACHE_MS=500${NC}"
echo -e "${BLUE}     - INTENTIONAL_LAG_EXTERNAL_MS=1000${NC}"
echo ""
echo "3. Reinicie o serviço:"
echo "   docker compose up -d payment-service"
echo ""
echo "OU use o script helper:"
echo "   ./scripts/enable-lag.sh"
echo ""
read -p "Pressione Enter após ativar o lag e reiniciar o serviço..."

# Verificar se lag foi ativado
LAG_ENABLED=$(docker compose exec -T payment-service printenv INTENTIONAL_LAG_ENABLED 2>/dev/null || echo "")
if [ "$LAG_ENABLED" != "true" ]; then
    echo -e "${RED}⚠ Lag não está ativo!${NC}"
    echo "Por favor, ative o lag conforme instruções acima e reinicie o serviço."
    read -p "Pressione Enter para continuar mesmo assim..."
fi

# Fase 3: Gerar requisições com lag
echo ""
echo -e "${RED}=== FASE 3: Gerando Requisições com Lag ===${NC}"
echo "Enviando 10 requisições (com lag ativo se configurado)..."
echo ""

for i in {1..10}; do
    correlation_id="lag-demo-$(date +%s)-$i"
    
    response=$(curl -s -w "\n%{http_code}\n%{time_total}" \
        -X POST "$PAYMENT_URL" \
        -H "Content-Type: application/json" \
        -H "X-Correlation-ID: $correlation_id" \
        -d "{
            \"accountId\": \"acc-$i\",
            \"amount\": 100.50,
            \"currency\": \"BRL\"
        }" 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n 2 | head -n 1)
    time_total=$(echo "$response" | tail -n 1)
    
    if [ "$http_code" = "201" ]; then
        # Comparar com baseline (se > 1s, provavelmente tem lag)
        if (( $(echo "$time_total > 1.0" | bc -l) )); then
            echo -e "  ${RED}⚠${NC} Requisição $i: ${time_total}s (correlation_id: $correlation_id) - ${RED}LENTA!${NC}"
        else
            echo -e "  ${GREEN}✓${NC} Requisição $i: ${time_total}s (correlation_id: $correlation_id)"
        fi
    else
        echo -e "  ${RED}✗${NC} Requisição $i: HTTP $http_code"
    fi
    
    sleep 0.5
done

echo ""
echo -e "${YELLOW}=== FASE 4: Investigação via Observabilidade ===${NC}"
echo ""
echo "Agora vamos investigar o problema usando observabilidade:"
echo ""

# Métricas
echo -e "${BLUE}1. MÉTRICAS (Prometheus: http://localhost:9090)${NC}"
echo ""
echo "Queries para executar:"
echo ""
echo "  # Verificar se lag está habilitado:"
echo "  intentional_lag_enabled"
echo ""
echo "  # Latência p99 (deve estar alta se lag ativo):"
echo "  histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[2m]))"
echo ""
echo "  # Comparar p50 vs p99 (cauda longa indica problema):"
echo "  histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[2m]))"
echo "  histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[2m]))"
echo ""
echo "  # Duração do lag no banco:"
echo "  intentional_lag_database_duration_seconds"
echo ""
echo "  # Duração do lag no cache:"
echo "  intentional_lag_cache_duration_seconds"
echo ""
read -p "Pressione Enter para continuar..."

# Logs
echo ""
echo -e "${BLUE}2. LOGS ESTRUTURADOS${NC}"
echo ""
echo "Comandos para executar:"
echo ""
echo "  # Ver logs com lag intencional:"
echo "  docker compose logs payment-service | grep 'intentional_lag'"
echo ""
echo "  # Ver logs de uma requisição específica:"
echo "  docker compose logs payment-service | grep 'lag-demo-'"
echo ""
echo "  # Ver logs estruturados em JSON:"
echo "  docker compose logs payment-service | jq 'select(.msg == \"intentional_lag_database\")'"
echo ""
read -p "Pressione Enter para continuar..."

# Traces
echo ""
echo -e "${BLUE}3. TRACES DISTRIBUÍDOS (Jaeger: http://localhost:16686)${NC}"
echo ""
echo "Passos para investigar:"
echo ""
echo "  1. Acesse: http://localhost:16686"
echo "  2. Selecione serviço: payment-service"
echo "  3. Procure por traces recentes"
echo "  4. Identifique spans com atributos:"
echo "     - lag.intentional=true"
echo "     - lag.type=database|cache|external"
echo "     - lag.duration_ms"
echo "  5. Observe a árvore de spans e identifique o gargalo"
echo ""
read -p "Pressione Enter para continuar..."

# Diagnóstico
echo ""
echo -e "${GREEN}=== FASE 5: Diagnóstico ===${NC}"
echo ""
echo "Com base na observabilidade, identifique:"
echo ""
echo "  ${RED}GARGALO IDENTIFICADO:${NC}"
echo "  - Database delay: 2000ms (maior impacto)"
echo "  - Cache delay: 500ms"
echo "  - External calls delay: 1000ms (3 chamadas = 3000ms total)"
echo ""
echo "  ${YELLOW}IMPACTO:${NC}"
echo "  - Latência total: ~5.5 segundos por requisição"
echo "  - p99 latência: > 5s"
echo "  - Experiência do usuário: degradada"
echo ""
echo "  ${BLUE}CAUSA RAIZ:${NC}"
echo "  - Lag intencional ativado via variáveis de ambiente"
echo "  - Database é o maior gargalo (2000ms)"
echo "  - Múltiplas chamadas externas acumulam latência"
echo ""
read -p "Pressione Enter para continuar..."

# Ações
echo ""
echo -e "${GREEN}=== FASE 6: Ações e Escalonamento ===${NC}"
echo ""
echo "Ações imediatas:"
echo "  1. ${YELLOW}Verificar métricas de lag${NC}"
echo "     - Query: intentional_lag_enabled"
echo "     - Se = 1, lag está ativo"
echo ""
echo "  2. ${YELLOW}Verificar traces no Jaeger${NC}"
echo "     - Identificar spans com lag.intentional=true"
echo "     - Verificar lag.duration_ms"
echo ""
echo "  3. ${YELLOW}Verificar logs${NC}"
echo "     - Buscar por 'intentional_lag'"
echo "     - Verificar correlation_id das requisições afetadas"
echo ""
echo "  4. ${YELLOW}Desativar lag (se necessário)${NC}"
echo "     - Edite docker-compose.yml e comente as variáveis INTENTIONAL_LAG_*"
echo "     - Execute: docker compose up -d payment-service"
echo "     - OU use: ./scripts/disable-lag.sh"
echo ""
echo "Quando escalar:"
echo "  - Se lag não é intencional: escalar para time de infra"
echo "  - Se é problema de banco: escalar para time de banco de dados"
echo "  - Se é problema de cache: escalar para time de plataforma"
echo "  - Se é problema de serviço externo: escalar para time do serviço"
echo ""
read -p "Pressione Enter para finalizar..."

echo ""
echo -e "${GREEN}=== Demonstração Concluída ===${NC}"
echo ""
echo "Resumo do aprendizado:"
echo "  ✓ Lag é um problema que aparece naturalmente nas requisições"
echo "  ✓ Observabilidade (métricas, logs, traces) identifica causa"
echo "  ✓ Diagnóstico claro permite ação rápida"
echo "  ✓ Runbook documenta processo de escalonamento"
echo ""
echo "Para desativar o lag:"
echo "  1. Edite docker-compose.yml"
echo "  2. Comente as variáveis INTENTIONAL_LAG_*"
echo "  3. Execute: docker compose up -d payment-service"
echo ""
echo "Consulte o README.md para mais detalhes sobre investigação de problemas"
echo ""
