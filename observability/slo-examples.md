# Exemplos de SLO, SLA e Error Budget

## Definições

### SLA (Service Level Agreement)
**Promessa externa** - Contrato com cliente/usuário

**Exemplo:**
- "99.9% de disponibilidade"
- "p99 latência < 500ms"
- **Consequência**: Penalidade se violado

### SLO (Service Level Objective)
**Objetivo interno** - Meta da equipe (mais rigoroso que SLA)

**Exemplo:**
- SLA: 99.9% → SLO: 99.95% (margem de segurança)
- SLA: p99 < 500ms → SLO: p99 < 400ms

### Error Budget
**Quanto pode falhar** - 100% - SLO

**Exemplo:**
- SLO: 99.9% → Error Budget: 0.1%
- Em um mês (2,592,000 segundos): 0.1% = 2,592 segundos = 43.2 minutos

## Exemplos Práticos

### Exemplo 1: Disponibilidade

**SLA:** 99.9% uptime
**SLO:** 99.95% uptime (margem de segurança)
**Error Budget:** 0.05% = 21.6 minutos/mês

**Cálculo no Prometheus:**
```promql
# Disponibilidade (últimas 24h)
avg_over_time(up[24h])

# Error Budget consumido
1 - avg_over_time(up[24h])
```

**Se error budget esgota:**
- Parar deploy de features
- Focar em estabilidade
- Não fazer mudanças arriscadas

### Exemplo 2: Latência

**SLA:** p99 < 500ms
**SLO:** p99 < 400ms (margem de segurança)
**Error Budget:** 1% das requisições podem ter latência > 400ms

**Cálculo no Prometheus:**
```promql
# p99 latência
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Requisições acima do SLO
rate(http_request_duration_seconds_bucket{le="0.4"}[5m]) / rate(http_request_duration_seconds_count[5m])
```

**Se error budget esgota:**
- Otimizar queries lentas
- Adicionar cache
- Escalar horizontalmente

### Exemplo 3: Taxa de Erro

**SLA:** Taxa de erro < 0.1%
**SLO:** Taxa de erro < 0.05%
**Error Budget:** 0.05% das requisições podem falhar

**Cálculo no Prometheus:**
```promql
# Taxa de erro
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Error budget consumido
(rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])) - 0.0005
```

## Alertas Baseados em SLO

### Alerta: SLO Violation

```yaml
groups:
  - name: slo_violations
    rules:
      - alert: AvailabilitySLOViolation
        expr: avg_over_time(up[1h]) < 0.9995
        for: 5m
        annotations:
          summary: "Availability SLO violated"
          description: "Availability is {{ $value }} (SLO: 99.95%)"
          runbook: "https://wiki/runbooks/availability"
      
      - alert: LatencySLOViolation
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 0.4
        for: 5m
        annotations:
          summary: "Latency SLO violated"
          description: "p99 latency is {{ $value }}s (SLO: 0.4s)"
          runbook: "https://wiki/runbooks/latency"
      
      - alert: ErrorRateSLOViolation
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.0005
        for: 5m
        annotations:
          summary: "Error rate SLO violated"
          description: "Error rate is {{ $value }} (SLO: 0.05%)"
          runbook: "https://wiki/runbooks/errors"
```

## Error Budget Burn Rate

### Cálculo de Burn Rate

**Burn Rate:** Quão rápido o error budget está sendo consumido

```promql
# Burn rate de disponibilidade
(1 - avg_over_time(up[1h])) / 0.0005  # 0.05% é o error budget

# Se burn rate > 1, está consumindo mais rápido que o permitido
```

### Ações Baseadas em Burn Rate

- **Burn Rate < 1**: Pode fazer deploys normalmente
- **Burn Rate 1-2**: Cuidado, reduzir frequência de deploys
- **Burn Rate > 2**: Parar deploys, focar em estabilidade

## Exemplo de Dashboard SLO

**Métricas para monitorar:**
1. Disponibilidade atual vs SLO
2. Latência p99 atual vs SLO
3. Taxa de erro atual vs SLO
4. Error budget restante
5. Burn rate

**Grafana Queries:**
```promql
# Disponibilidade
avg_over_time(up[24h])

# Latência p99
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Taxa de erro
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Error budget restante (disponibilidade)
(0.0005 - (1 - avg_over_time(up[24h]))) / 0.0005) * 100

# Error budget restante (latência)
# Requisições abaixo do SLO / Total
rate(http_request_duration_seconds_bucket{le="0.4"}[5m]) / rate(http_request_duration_seconds_count[5m]) * 100
```

## Mensagem Final

> **Sem budget, toda falha vira emergência.**

Com error budget definido:
- Equipe sabe quando parar features
- Foco em estabilidade quando necessário
- Decisões baseadas em dados, não em pânico
