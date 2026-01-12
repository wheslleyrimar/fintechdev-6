#!/bin/bash

# Script de teste de carga para demonstrar escalabilidade e observabilidade
# Demonstra: rate limiting, backpressure, métricas de latência, percentis

echo "=== Teste de Carga - Escalabilidade e Observabilidade ==="
echo ""

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PAYMENT_URL="http://localhost:8080/payments"

# Função para criar um pagamento
create_payment() {
  local account_id="acc-$(shuf -i 1000-9999 -n 1)"
  local amount=$(shuf -i 10-1000 -n 1).$(shuf -i 10-99 -n 1)
  local correlation_id="test-$(date +%s)-$(shuf -i 1000-9999 -n 1)"
  
  response=$(curl -s -w "\n%{http_code}\n%{time_total}" \
    -X POST "$PAYMENT_URL" \
    -H "Content-Type: application/json" \
    -H "X-Correlation-ID: $correlation_id" \
    -d "{
      \"accountId\": \"$account_id\",
      \"amount\": $amount,
      \"currency\": \"BRL\"
    }")
  
  http_code=$(echo "$response" | tail -n 2 | head -n 1)
  time_total=$(echo "$response" | tail -n 1)
  
  echo "$http_code|$time_total|$correlation_id"
}

# Teste 1: Carga baixa (baseline)
echo -e "${GREEN}Teste 1: Carga Baixa (10 requisições)${NC}"
echo "Observar: latência média, p50, p90, p95, p99"
echo ""

results=()
for i in {1..10}; do
  result=$(create_payment)
  results+=("$result")
  http_code=$(echo "$result" | cut -d'|' -f1)
  time_total=$(echo "$result" | cut -d'|' -f2)
  
  if [ "$http_code" = "201" ]; then
    echo -e "  ✓ Requisição $i: ${GREEN}${http_code}${NC} - ${time_total}s"
  elif [ "$http_code" = "429" ]; then
    echo -e "  ⚠ Requisição $i: ${YELLOW}${http_code} (Rate Limited)${NC} - ${time_total}s"
  else
    echo -e "  ✗ Requisição $i: ${RED}${http_code}${NC} - ${time_total}s"
  fi
  
  sleep 0.1
done

echo ""
echo "Verifique métricas no Prometheus: http://localhost:9090"
echo "Verifique traces no Jaeger: http://localhost:16686"
echo ""
read -p "Pressione Enter para continuar..."

# Teste 2: Carga média (demonstrar escalabilidade)
echo ""
echo -e "${YELLOW}Teste 2: Carga Média (50 requisições rápidas)${NC}"
echo "Observar: aumento de latência, rate limiting ativando"
echo ""

success=0
rate_limited=0
errors=0

for i in {1..50}; do
  result=$(create_payment)
  http_code=$(echo "$result" | cut -d'|' -f1)
  
  if [ "$http_code" = "201" ]; then
    ((success++))
  elif [ "$http_code" = "429" ]; then
    ((rate_limited++))
  else
    ((errors++))
  fi
  
  if [ $((i % 10)) -eq 0 ]; then
    echo "  Processadas: $i/50 (Sucesso: $success, Rate Limited: $rate_limited, Erros: $errors)"
  fi
done

echo ""
echo "Resultados:"
echo "  ✓ Sucesso: $success"
echo "  ⚠ Rate Limited (Backpressure): $rate_limited"
echo "  ✗ Erros: $errors"
echo ""
read -p "Pressione Enter para continuar..."

# Teste 3: Carga alta (demonstrar backpressure e circuit breaker)
echo ""
echo -e "${RED}Teste 3: Carga Alta (100 requisições simultâneas)${NC}"
echo "Observar: backpressure, circuit breaker, latência de cauda"
echo ""

pids=()
for i in {1..100}; do
  (
    result=$(create_payment)
    http_code=$(echo "$result" | cut -d'|' -f1)
    time_total=$(echo "$result" | cut -d'|' -f2)
    correlation_id=$(echo "$result" | cut -d'|' -f3)
    
    if [ "$http_code" = "201" ]; then
      echo "SUCCESS|$time_total|$correlation_id" >> /tmp/load_test_results.txt
    elif [ "$http_code" = "429" ]; then
      echo "RATE_LIMITED|$time_total|$correlation_id" >> /tmp/load_test_results.txt
    elif [ "$http_code" = "503" ]; then
      echo "CIRCUIT_BREAKER|$time_total|$correlation_id" >> /tmp/load_test_results.txt
    else
      echo "ERROR|$time_total|$correlation_id" >> /tmp/load_test_results.txt
    fi
  ) &
  pids+=($!)
done

echo "Aguardando todas as requisições..."
for pid in "${pids[@]}"; do
  wait $pid
done

# Analisar resultados
total=$(wc -l < /tmp/load_test_results.txt)
success=$(grep -c "SUCCESS" /tmp/load_test_results.txt || echo "0")
rate_limited=$(grep -c "RATE_LIMITED" /tmp/load_test_results.txt || echo "0")
circuit_breaker=$(grep -c "CIRCUIT_BREAKER" /tmp/load_test_results.txt || echo "0")
errors=$(grep -c "ERROR" /tmp/load_test_results.txt || echo "0")

echo ""
echo "Resultados do Teste de Carga Alta:"
echo "  Total: $total"
echo "  ✓ Sucesso: $success"
echo "  ⚠ Rate Limited (Backpressure): $rate_limited"
echo "  ⚠ Circuit Breaker: $circuit_breaker"
echo "  ✗ Erros: $errors"
echo ""

# Calcular percentis de latência (aproximado)
echo "Análise de Latência (verifique Prometheus para valores precisos):"
echo "  - Acesse: http://localhost:9090"
echo "  - Query: histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))"
echo "  - Query: histogram_quantile(0.90, rate(http_request_duration_seconds_bucket[5m]))"
echo "  - Query: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))"
echo "  - Query: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))"
echo ""

rm -f /tmp/load_test_results.txt

echo -e "${GREEN}=== Teste de Carga Concluído ===${NC}"
echo ""
echo "Próximos passos:"
echo "1. Verifique métricas RED no Prometheus"
echo "2. Analise traces distribuídos no Jaeger"
echo "3. Observe logs estruturados nos serviços"
echo "4. Verifique dashboards no Grafana (se configurados)"
