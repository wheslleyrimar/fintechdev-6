# Investigando Problemas de Latência

[← Voltar ao README](../README.md)

---

### Passo 1: Detectar o Problema

**Sinais de alerta (métricas RED):**

```promql
# Latência p99 aumentando (sinal mais importante)
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 0.5

# Comparar p50 vs p99 (cauda longa indica problema)
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

**Alertas sugeridos:**
- **Crítico**: p99 latência > 500ms por 5 minutos
- **Warning**: p95 latência > 300ms por 5 minutos

### Passo 2: Identificar o Gargalo

**Via Métricas:**
```promql
# Ver se lag intencional está ativo
intentional_lag_enabled

# Ver qual componente tem mais lag
intentional_lag_database_duration_seconds
intentional_lag_cache_duration_seconds
intentional_lag_external_duration_seconds
```

**Via Traces (Jaeger):**
1. Acesse: http://localhost:16686
2. Selecione: `payment-service`
3. Filtre por: `Duration > 1s`
4. Analise a árvore de spans:
   - Identifique o span mais lento
   - Verifique atributos `lag.intentional` e `lag.type`

**Via Logs:**
```bash
# Buscar logs de lag
docker compose logs payment-service | grep "intentional_lag"

# Exemplo de log esperado:
# {
#   "level": "warn",
#   "msg": "intentional_lag_database",
#   "lag.type": "database",
#   "lag.duration": "2s",
#   "correlation_id": "test-123"
# }
```

### Passo 3: Quantificar o Impacto

```promql
# Quantas requisições afetadas?
count(http_requests_total{endpoint="/payments"})

# Qual a latência média vs p99?
avg(rate(http_request_duration_seconds_sum[5m])) / avg(rate(http_request_duration_seconds_count[5m]))
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Taxa de erro aumentou?
rate(http_requests_total{status="5xx"}[5m]) / rate(http_requests_total[5m])
```

### Passo 4: Ações Imediatas

**Se lag é intencional (demonstração):**
```bash
# Verificar se lag está ativo
docker compose exec payment-service printenv INTENTIONAL_LAG_ENABLED

# Desativar lag (se necessário)
./scripts/disable-lag.sh
# OU edite docker-compose.yml e comente as variáveis INTENTIONAL_LAG_*
# Depois: docker compose up -d payment-service
```

**Se lag NÃO é intencional (problema real):**

**Gargalo no Banco de Dados:**
- Verificar conexões do banco
- Verificar queries lentas
- Verificar índice faltando
- Escalar para time de banco de dados

**Gargalo no Cache:**
- Verificar tamanho do cache
- Verificar TTL
- Verificar conectividade com Redis/cache
- Escalar para time de plataforma

**Gargalo em Serviços Externos:**
- Verificar saúde do serviço externo
- Verificar rede
- Escalar para time do serviço externo

### Checklist de Investigação Rápida

**Fase 1: Detecção (2 minutos)**
- [ ] Verificar métricas RED no Prometheus
- [ ] Confirmar p99 > threshold
- [ ] Verificar taxa de erro

**Fase 2: Identificação (5 minutos)**
- [ ] Verificar `intentional_lag_enabled`
- [ ] Analisar traces no Jaeger
- [ ] Verificar logs estruturados
- [ ] Identificar componente mais lento

**Fase 3: Diagnóstico (5 minutos)**
- [ ] Quantificar impacto (quantas requisições afetadas)
- [ ] Identificar causa raiz
- [ ] Documentar evidências

**Fase 4: Ação (variável)**
- [ ] Ação imediata (se aplicável)
- [ ] Escalar para time correto (se necessário)
- [ ] Monitorar recuperação
- [ ] Documentar incidente

---

[← Voltar ao README](../README.md)

