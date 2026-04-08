# Troubleshooting de Incidentes

Runbook curto para a aula e para o laboratório.

## 1. Comece pelo sintoma

Perguntas iniciais:

- a latência subiu?
- o erro subiu junto ou depois?
- a proteção começou a rejeitar?
- o problema está no online ou no assíncrono?

## 2. Onde olhar no Datadog

Use esta ordem:

1. dashboard `Payment Health`
2. dashboard `k6 Correlation`
3. dashboard `Scenario Signals`
4. APM do `payment-service`
5. APM do `antifraud-service`
6. logs filtrando por `service` e `trace_id`

## 3. Leitura por padrão de falha

### Caso A: latência sobe sem explosão de erro

Suspeite de:

- `chatty_db`
- `cache_stampede`
- dependência mais lenta sem timeout efetivo

Confirme por:

- p95/p99 em alta;
- waterfall do trace mais comprido;
- spans `db.select.*` ou `external.call.*` dominando duração.

### Caso B: latência e erro sobem juntos

Suspeite de:

- dependência externa lenta;
- breaker abrindo tarde;
- spike pressionando limite do serviço.

Confirme por:

- `rate_limit_rejected_total`;
- `circuit_breaker_state`;
- traces com spans externos dominantes;
- `k6` mostrando burst claro no mesmo instante.

### Caso C: endpoint saudável, operação ruim

Suspeite de:

- degradação assíncrona;
- consumidor lento;
- backlog crescendo fora do caminho online.

Confirme por:

- `antifraud_processing_duration_percentiles`;
- `antifraud_messages_processed_total`;
- cenário `async_lag`;
- divergência entre `payment-service` saudável e `antifraud-service` degradado.

## 4. Ações rápidas por cenário

### `chatty_db`

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

Leitura:

- normalização do p95/p99;
- queda do tempo total do trace.

### `cache_stampede`

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/hot-accounts.js
```

Leitura:

- redução da cauda sob contas quentes;
- menor participação relativa de spans de banco.

### `risk_dependency`

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/spike.js
```

Leitura:

- redução de 429/503;
- queda da duração dos spans externos.

### `async_lag`

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

Leitura:

- recomposição do throughput do antifraude;
- queda do p95 assíncrono.

## 5. Estado dos cenários

```bash
curl http://localhost:8080/admin/scenarios
curl http://localhost:8081/admin/scenarios
```

## 6. Perguntas que fecham o diagnóstico

- qual span domina o trace?
- o p99 piorou antes do erro?
- há proteção atuando ou o serviço só está saturando?
- a degradação está no serviço principal ou na dependência?
- desligar o cenário normaliza os sinais?
