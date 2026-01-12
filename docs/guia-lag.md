# Guia Completo: Testando e Analisando LAG

[← Voltar ao README](../README.md)

---

### O que é LAG?

**LAG (Latência Artificial Gerada)** é um mecanismo que simula problemas de latência no sistema. Ele permite:

1. **Demonstrar** como problemas de latência aparecem na prática
2. **Treinar** equipes a identificar e diagnosticar problemas
3. **Validar** que a observabilidade está funcionando corretamente

### Conceito: Por que simular LAG?

Em produção, problemas de latência podem ser causados por:
- Banco de dados lento (queries sem índice, conexões esgotadas)
- Cache miss (dados não encontrados no cache)
- Serviços externos lentos (timeout, rede lenta)
- Sincronismo excessivo (múltiplas chamadas em sequência)

O LAG intencional simula esses problemas de forma controlada para aprendizado.

---

## Passo a Passo: Testando LAG

### Fase 1: Baseline - Sistema sem LAG

**Objetivo:** Estabelecer uma linha base de latência normal.

#### Passo 1.1: Verificar estado inicial do LAG

```bash
# Verificar se LAG está desativado
docker compose exec payment-service printenv INTENTIONAL_LAG_ENABLED
```

**Resultado esperado:** (vazio ou não definido)

#### Passo 1.2: Fazer requisições de baseline

Execute 5 requisições para estabelecer a latência normal:

```bash
# Requisição 1
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: baseline-001" \
  -d '{
    "accountId": "acc-001",
    "amount": 100.50,
    "currency": "BRL"
  }' \
  -w "\nTempo total: %{time_total}s\n" \
  -o /dev/null -s

# Requisição 2
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: baseline-002" \
  -d '{
    "accountId": "acc-002",
    "amount": 200.75,
    "currency": "BRL"
  }' \
  -w "\nTempo total: %{time_total}s\n" \
  -o /dev/null -s

# Requisição 3
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: baseline-003" \
  -d '{
    "accountId": "acc-003",
    "amount": 50.25,
    "currency": "BRL"
  }' \
  -w "\nTempo total: %{time_total}s\n" \
  -o /dev/null -s

# Requisição 4
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: baseline-004" \
  -d '{
    "accountId": "acc-004",
    "amount": 300.00,
    "currency": "BRL"
  }' \
  -w "\nTempo total: %{time_total}s\n" \
  -o /dev/null -s

# Requisição 5
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: baseline-005" \
  -d '{
    "accountId": "acc-005",
    "amount": 150.90,
    "currency": "BRL"
  }' \
  -w "\nTempo total: %{time_total}s\n" \
  -o /dev/null -s
```

**O que observar:**
- Tempo total de cada requisição (deve ser rápido, < 100ms)
- Response code (deve ser 201)
- Headers `X-Correlation-ID` e `X-Trace-ID` na resposta

**Anotar:** Latência média do baseline: ______ ms

#### Passo 1.3: Verificar métricas no Prometheus (Baseline)

1. Acesse: http://localhost:9090
2. Na aba "Graph", execute a query:

```promql
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))
```

**O que observar:**
- p99 latência deve estar baixo (< 0.1s ou 100ms)
- Anotar o valor: ______ segundos

---

### Fase 2: Ativando LAG Intencional

**Objetivo:** Simular problemas de latência no sistema.

#### Passo 2.1: Ativar LAG usando script (Recomendado)

```bash
./scripts/enable-lag.sh
```

**O que o script faz:**
1. Cria backup do `docker-compose.yml`
2. Ativa variáveis de ambiente de LAG:
   - `INTENTIONAL_LAG_ENABLED=true`
   - `INTENTIONAL_LAG_DATABASE_MS=2000` (2 segundos de delay no banco)
   - `INTENTIONAL_LAG_CACHE_MS=500` (0.5 segundos de delay no cache)
   - `INTENTIONAL_LAG_EXTERNAL_MS=1000` (1 segundo de delay em cada chamada externa)
3. Reinicia o serviço `payment-service`

**Resultado esperado:**
```
✓ Backup criado: docker-compose.yml.backup
✓ Variáveis de ambiente de lag ativadas no docker-compose.yml
Reiniciando payment-service...
✓ Lag intencional ativado!
```

#### Passo 2.2: Verificar que LAG está ativo

```bash
# Verificar variável de ambiente
docker compose exec payment-service printenv INTENTIONAL_LAG_ENABLED
```

**Resultado esperado:** `true`

#### Passo 2.3: Verificar métrica de LAG no Prometheus

1. Acesse: http://localhost:9090
2. Execute a query:

```promql
intentional_lag_enabled
```

**Resultado esperado:** `1` (LAG ativo)

**O que isso significa:**
- `0` = LAG desativado
- `1` = LAG ativo

---

### Fase 3: Gerando Requisições com LAG

**Objetivo:** Observar como o LAG afeta as requisições.

#### Passo 3.1: Fazer requisições e medir latência

Execute 10 requisições e observe a latência:

```bash
# Script para gerar múltiplas requisições
for i in {1..10}; do
  echo "=== Requisição $i ==="
  correlation_id="lag-test-$(date +%s)-$i"
  
  curl -X POST http://localhost:8080/payments \
    -H "Content-Type: application/json" \
    -H "X-Correlation-ID: $correlation_id" \
    -d "{
      \"accountId\": \"acc-lag-$i\",
      \"amount\": $((RANDOM % 1000 + 10)).$((RANDOM % 100)),
      \"currency\": \"BRL\"
    }" \
    -w "\nHTTP Code: %{http_code}\nTempo total: %{time_total}s\n" \
    -o /dev/null -s
  
  echo "Correlation ID: $correlation_id"
  echo ""
  sleep 0.5
done
```

**O que observar:**
- **Tempo total:** Deve estar alto (> 3 segundos)
- **HTTP Code:** Deve ser 201 (sucesso)
- **Correlation ID:** Anotar para rastrear depois

**Anotar:** Latência média com LAG: ______ segundos

**Comparação:**
- Baseline: ~0.05s
- Com LAG: ~5.5s (2000ms banco + 500ms cache + 3000ms externas = 5500ms)

#### Passo 3.2: Entender de onde vem a latência

Com LAG ativo, cada requisição tem:
- **Database delay:** 2000ms (2 segundos)
- **Cache delay:** 500ms (0.5 segundos)
- **External calls delay:** 1000ms × 3 = 3000ms (3 segundos)
- **Total esperado:** ~5.5 segundos

**Por que 3 chamadas externas?**
O serviço simula 3 chamadas externas em sequência, cada uma com 1 segundo de delay.

---

### Fase 4: Analisando LAG nas Ferramentas de Observabilidade

Agora vamos ver como o LAG aparece em cada ferramenta.

#### 4.1 Analisando no Prometheus

**Passo 4.1.1: Verificar se LAG está ativo**

1. Acesse: http://localhost:9090
2. Na aba "Graph", execute:

```promql
intentional_lag_enabled
```

**Resultado esperado:** `1`

**O que fazer:**
- Clique em "Execute"
- Veja o valor no gráfico (deve ser 1)
- Anotar: LAG está ativo? Sim/Não

**Passo 4.1.2: Ver latência p99 (deve estar alta)**

```promql
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[2m]))
```

**O que observar:**
- Valor deve estar alto (> 5 segundos)
- Compare com o baseline (era < 0.1s)

**Passo 4.1.3: Ver latência de cada componente com LAG**

**Database delay:**
```promql
intentional_lag_database_duration_seconds
```

**Resultado esperado:** `2` (2 segundos)

**Cache delay:**
```promql
intentional_lag_cache_duration_seconds
```

**Resultado esperado:** `0.5` (0.5 segundos)

**External calls delay:**
```promql
intentional_lag_external_duration_seconds
```

**Resultado esperado:** `1` (1 segundo por chamada)

**Passo 4.1.4: Comparar p50 vs p99 (cauda longa)**

Execute ambas as queries:

```promql
# p50 (mediana)
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[2m]))

# p99 (cauda)
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[2m]))
```

**O que observar:**
- p50 e p99 devem estar altos (ambos > 5s)
- Isso indica que TODAS as requisições estão lentas (não apenas algumas)

**Passo 4.1.5: Ver taxa de requisições**

```promql
rate(http_requests_total{endpoint="/payments"}[2m])
```

**O que observar:**
- Taxa de requisições por segundo
- Deve estar baixa se você fez poucas requisições

---

#### 4.2 Analisando no Grafana

**Passo 4.2.1: Acessar Grafana**

1. Acesse: http://localhost:3000
2. Login: `admin` / `admin`
3. Vá em "Dashboards" → "Browse"

**Passo 4.2.2: Abrir Dashboard RED Metrics**

1. Procure por "RED Metrics - Payment Service"
2. Clique para abrir

**O que você verá:**

**Painel 1: Rate - Requests per Second**
- Mostra taxa de requisições
- **O que analisar:** Linha deve mostrar suas requisições

**Painel 2: Errors - Error Rate**
- Mostra taxa de erros
- **O que analisar:** Deve estar em 0% (requisições estão sucedendo)

**Painel 3: Duration - Latency Percentiles**
- Mostra p50, p90, p95, p99
- **O que analisar:**
  - Todas as linhas devem estar altas (> 5s)
  - p99 deve ser a mais alta
  - Compare com baseline (se tiver)

**Painel 4: Intentional Lag - Status**
- Mostra se LAG está ativo
- **O que analisar:**
  - Deve mostrar "Enabled" (vermelho)
  - Se mostrar "Disabled" (verde), LAG não está ativo

**Painel 5-7: Intentional Lag - Database/Cache/External Duration**
- Mostra duração do LAG em cada componente
- **O que analisar:**
  - Database: ~2s
  - Cache: ~0.5s
  - External: ~1s

**Passo 4.2.3: Abrir Dashboard USE Metrics**

1. Procure por "USE Metrics - Infrastructure"
2. Clique para abrir

**O que você verá:**

**Painel 1: Utilization - CPU**
- Uso de CPU
- **O que analisar:** Deve estar variando (simulado)

**Painel 2: Utilization - Memory**
- Uso de memória
- **O que analisar:** Deve estar variando (simulado)

**Painel 3: Saturation - Queue Depth**
- Profundidade da fila
- **O que analisar:** Quantas mensagens estão na fila

**Painel 4: Errors - Infrastructure**
- Erros de infraestrutura
- **O que analisar:** Deve estar baixo

---

#### 4.3 Analisando no Jaeger

**Passo 4.3.1: Acessar Jaeger**

1. Acesse: http://localhost:16686
2. Na página inicial, você verá um formulário de busca

**Passo 4.3.2: Buscar traces com LAG**

1. **Service:** Selecione `payment-service`
2. **Operation:** Deixe vazio ou selecione `payment.process`
3. **Tags:** Adicione `lag.intentional=true`
4. **Lookback:** Selecione "Last 1 hour"
5. **Max duration:** Deixe vazio ou coloque `>5s`
6. Clique em **"Find Traces"**

**O que você verá:**
- Lista de traces (cada trace = uma requisição)
- Traces devem ter duração alta (> 5 segundos)

**Passo 4.3.3: Analisar um trace específico**

1. Clique em um trace da lista
2. Você verá uma árvore de spans (operações)

**Estrutura esperada:**
```
payment.process (5.5s)
├── database.query (2.0s) [lag.intentional=true, lag.type=database]
├── cache.lookup (0.5s) [lag.intentional=true, lag.type=cache]
└── external.call.0 (1.0s) [lag.intentional=true, lag.type=external]
    └── external.call.1 (1.0s) [lag.intentional=true, lag.type=external]
        └── external.call.2 (1.0s) [lag.intentional=true, lag.type=external]
```

**O que analisar em cada span:**

1. **payment.process** (span raiz)
   - Duração total: ~5.5s
   - **Tags importantes:**
     - `correlation_id`: ID da requisição
     - `trace_id`: ID do trace

2. **database.query**
   - Duração: ~2.0s
   - **Tags importantes:**
     - `lag.intentional=true`
     - `lag.type=database`
     - `lag.duration_ms=2000`

3. **cache.lookup**
   - Duração: ~0.5s
   - **Tags importantes:**
     - `lag.intentional=true`
     - `lag.type=cache`
     - `lag.duration_ms=500`

4. **external.call.0, .1, .2**
   - Duração: ~1.0s cada
   - **Tags importantes:**
     - `lag.intentional=true`
     - `lag.type=external`
     - `lag.duration_ms=1000`

**Passo 4.3.4: Filtrar por Correlation ID**

Se você anotou um `correlation_id` de uma requisição:

1. Na busca de traces, em **Tags**, adicione:
   - Key: `correlation_id`
   - Value: `lag-test-XXXXX-1` (seu correlation ID)
2. Clique em "Find Traces"
3. Você verá apenas o trace daquela requisição específica

---

#### 4.4 Analisando nos Logs

**Passo 4.4.1: Ver logs em tempo real**

```bash
docker compose logs -f payment-service
```

**O que você verá:**
- Logs estruturados em JSON
- Cada requisição gera múltiplos logs

**Passo 4.4.2: Filtrar logs de LAG**

```bash
docker compose logs payment-service | grep "intentional_lag"
```

**O que você verá:**
- Logs específicos sobre LAG
- Exemplo:

```json
{
  "level": "warn",
  "ts": 1234567890,
  "msg": "intentional_lag_database",
  "service": "payment-service",
  "lag.type": "database",
  "lag.duration_ms": 2000,
  "correlation_id": "lag-test-1234567890-1"
}
```

**Passo 4.4.3: Buscar logs por Correlation ID**

```bash
docker compose logs payment-service | grep "lag-test-1234567890-1"
```

**O que você verá:**
- Todos os logs relacionados àquela requisição específica
- Inclui logs de database, cache, external calls

**Passo 4.4.4: Ver logs estruturados formatados**

```bash
docker compose logs payment-service | jq 'select(.msg | contains("intentional_lag"))'
```

**O que você verá:**
- Logs formatados em JSON bonito
- Filtrados apenas para mensagens de LAG

---

### Fase 5: Diagnóstico e Documentação

**Objetivo:** Documentar o que foi encontrado.

#### Passo 5.1: Criar relatório de diagnóstico

**Template de relatório:**

```
=== RELATÓRIO DE DIAGNÓSTICO DE LAG ===

Data: [DATA]
Analista: [SEU NOME]

1. BASELINE (sem LAG)
   - Latência média: ______ ms
   - p99 latência: ______ s
   - Status: Normal

2. COM LAG ATIVO
   - Latência média: ______ s
   - p99 latência: ______ s
   - Status: Degradado

3. COMPONENTES AFETADOS
   - Database delay: 2000ms ✓ Confirmado
   - Cache delay: 500ms ✓ Confirmado
   - External calls delay: 1000ms × 3 = 3000ms ✓ Confirmado

4. IMPACTO
   - Total de requisições testadas: ______
   - Requisições afetadas: ______ (100%)
   - Taxa de erro: ______%

5. EVIDÊNCIAS
   - Prometheus: intentional_lag_enabled = 1 ✓
   - Grafana: Dashboard mostra LAG ativo ✓
   - Jaeger: Traces mostram spans com lag.intentional=true ✓
   - Logs: Logs mostram intentional_lag_* ✓

6. CONCLUSÃO
   - LAG intencional está ativo e funcionando corretamente
   - Observabilidade está capturando todos os sinais
   - Sistema está detectando e reportando o problema
```

---

### Fase 6: Desativando LAG

**Objetivo:** Voltar o sistema ao estado normal.

#### Passo 6.1: Desativar LAG

```bash
./scripts/disable-lag.sh
```

**O que o script faz:**
1. Comenta as variáveis de ambiente de LAG no `docker-compose.yml`
2. Reinicia o serviço `payment-service`

**Resultado esperado:**
```
✓ Variáveis de ambiente de lag desativadas no docker-compose.yml
Reiniciando payment-service...
✓ Lag intencional desativado!
```

#### Passo 6.2: Verificar que LAG está desativado

```bash
docker compose exec payment-service printenv INTENTIONAL_LAG_ENABLED
```

**Resultado esperado:** (vazio ou não definido)

#### Passo 6.3: Verificar métrica no Prometheus

```promql
intentional_lag_enabled
```

**Resultado esperado:** `0` ou métrica não existe

#### Passo 6.4: Fazer requisições para confirmar

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: test-after-lag" \
  -d '{
    "accountId": "acc-test",
    "amount": 100.50,
    "currency": "BRL"
  }' \
  -w "\nTempo total: %{time_total}s\n" \
  -o /dev/null -s
```

**Resultado esperado:** Tempo total < 0.1s (sistema voltou ao normal)

---

[← Voltar ao README](../README.md)

