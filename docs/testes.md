# Testes e Demonstrações

[← Voltar ao README](../README.md)

---

### Teste 1: Requisição Básica

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: test-123" \
  -d '{
    "accountId": "acc-1",
    "amount": 100.50,
    "currency": "BRL"
  }'
```

**Observar:**
- Response com `X-Correlation-ID` e `X-Trace-ID`
- Logs estruturados nos serviços
- Trace no Jaeger

### Teste 2: Teste de Carga

```bash
./scripts/load-test.sh
```

**Observar:**
- Rate limiting ativando (HTTP 429)
- Métricas no Prometheus
- Percentis de latência (p50, p90, p99)

### Teste 3: Simulação de Gargalos

```bash
./scripts/simulate-bottleneck.sh
```

**Observar:**
- Traces com latência alta
- Cache misses
- Serviços chatty

### Teste 4: Demonstração de Lag Intencional

**O lag é um problema que aparece naturalmente nas requisições normais.**

**Ativar lag:**
```bash
# Opção 1: Script helper (recomendado)
./scripts/enable-lag.sh

# Opção 2: Manual
# Edite docker-compose.yml e descomente as variáveis INTENTIONAL_LAG_*
# Depois: docker compose up -d payment-service
```

**Demonstração completa:**
```bash
./scripts/demonstrate-lag.sh
```

Este script demonstra:
1. Baseline (sem lag)
2. Ativação de lag intencional
3. Geração de requisições com lag
4. Investigação via observabilidade
5. Diagnóstico
6. Ações e escalonamento

**Desativar lag:**
```bash
./scripts/disable-lag.sh
```

**Métricas de lag no Prometheus:**
```promql
# Verificar se lag está ativo
intentional_lag_enabled

# Duração do lag no banco
intentional_lag_database_duration_seconds

# Duração do lag no cache
intentional_lag_cache_duration_seconds

# Duração do lag em chamadas externas
intentional_lag_external_duration_seconds
```

### Teste 5: Análise de Métricas

**No Prometheus (http://localhost:9090):**

```promql
# Rate de requisições
rate(http_requests_total[5m])

# Taxa de erro
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Latência p50
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))

# Latência p99
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# CPU utilization
cpu_utilization_percent

# Queue depth (saturação)
message_queue_depth
```

### Teste 6: Análise de Traces

**No Jaeger (http://localhost:16686):**
1. Selecione serviço: `payment-service`
2. Clique em "Find Traces"
3. Veja árvore de spans
4. Identifique spans lentos
5. Veja propagação de trace ID

---

[← Voltar ao README](../README.md)
