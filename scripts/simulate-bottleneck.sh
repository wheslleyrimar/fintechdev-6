#!/bin/bash

# Script para simular gargalos e demonstrar observabilidade
# Demonstra: banco lento, cache miss, serviços chatty, sincronismo excessivo

echo "=== Simulação de Gargalos ==="
echo ""

PAYMENT_URL="http://localhost:8080/payments"

echo "Este script demonstra como a observabilidade ajuda a identificar gargalos:"
echo ""
echo "1. Gargalo de Banco de Dados:"
echo "   - 1% das requisições tem latência muito alta (cauda longa)"
echo "   - Observe p99 vs p50 no Prometheus"
echo ""
echo "2. Gargalo de Cache:"
echo "   - 20% cache miss (simulado no código)"
echo "   - Observe métricas de cache no tracing"
echo ""
echo "3. Serviços Chatty:"
echo "   - Notification service faz 4 chamadas (email, sms, push, webhook)"
echo "   - Observe spans no Jaeger"
echo ""
echo "4. Sincronismo Excessivo:"
echo "   - Múltiplas chamadas síncronas em sequência"
echo "   - Observe duração total vs duração individual"
echo ""

read -p "Pressione Enter para iniciar simulação..."

echo ""
echo "Enviando requisições para gerar gargalos observáveis..."
echo ""

for i in {1..20}; do
  account_id="acc-$(shuf -i 1000-9999 -n 1)"
  amount=$(shuf -i 10-1000 -n 1).$(shuf -i 10-99 -n 1)
  correlation_id="bottleneck-$(date +%s)-$i"
  
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
  
  if [ "$http_code" = "201" ]; then
    echo "✓ Requisição $i: $http_code - ${time_total}s (correlation_id: $correlation_id)"
  else
    echo "✗ Requisição $i: $http_code - ${time_total}s"
  fi
  
  sleep 0.2
done

echo ""
echo "=== Análise de Gargalos ==="
echo ""
echo "1. Verifique traces no Jaeger (http://localhost:16686):"
echo "   - Procure por spans com latência alta"
echo "   - Identifique 'database.query' com latência > 1s"
echo "   - Observe 'cache.lookup' com cache.hit=false"
echo ""
echo "2. Verifique métricas no Prometheus (http://localhost:9090):"
echo "   - Compare p50 vs p99: histogram_quantile(0.50, ...) vs histogram_quantile(0.99, ...)"
echo "   - Observe diferença entre média e percentis altos"
echo ""
echo "3. Verifique logs estruturados:"
echo "   docker compose logs payment-service | grep 'slow query'"
echo "   docker compose logs payment-service | grep 'cache.hit'"
echo ""
echo "4. Identifique serviços chatty:"
echo "   - No Jaeger, veja quantos spans 'external.call' existem por requisição"
echo "   - Observe que notification-service faz 4 chamadas paralelas"
echo ""
echo "=== Lições Aprendidas ==="
echo ""
echo "✓ Média esconde cauda. Cauda derruba SLA."
echo "✓ Percentis (p90, p95, p99) são essenciais para entender latência real"
echo "✓ Tracing distribuído revela gargalos invisíveis em métricas agregadas"
echo "✓ Logs estruturados com correlation_id permitem rastrear requisições"
echo "✓ Observabilidade transforma falhas em aprendizado"
