# Aula 6 — Laboratório Fintech com Datadog e k6

Laboratório prático para aula técnica de sistemas de fintech com foco em:

- carga controlada com `k6`;
- traces, métricas e logs no Datadog;
- incidentes reproduzíveis no caminho crítico de pagamentos;
- análise antes, durante e depois da correção.

## Stack principal

- Go 1.22
- RabbitMQ
- Datadog Agent
- OpenTelemetry OTLP HTTP
- `k6` com dashboard web
- `k6` com envio de métricas para Datadog via DogStatsD

`Prometheus` e `Grafana` continuam no ambiente como apoio local, mas o fluxo principal da aula agora é Datadog-first.

## Como executar

Pré-requisitos:

- Docker e Docker Compose
- conta trial do Datadog
- `.env` com `DATADOG_API_KEY`

Setup:

```bash
cp .env.example .env
docker compose up --build
docker compose --profile k6 build k6-datadog
```

Validação rápida:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/admin/scenarios
```

Interfaces:

- Datadog: trial web
- RabbitMQ: `http://localhost:15672`
- k6 web dashboard: `http://127.0.0.1:5665`

## Fluxo operacional da aula

1. Subir o ambiente e validar ingestão no Datadog.
2. Rodar baseline com `k6`.
3. Ativar um incidente reproduzível.
4. Correlacionar `k6 web`, APM, métricas e logs.
5. Aplicar correção operacional.
6. Rodar novamente e mostrar normalização.

## Cenários disponíveis

`payment-service`:

- `chatty_db`
- `cache_stampede`
- `risk_dependency`
- `intentionalLagEnabled`

`antifraud-service`:

- `async_lag`

Estado atual:

```bash
curl http://localhost:8080/admin/scenarios
curl http://localhost:8081/admin/scenarios
```

## Scripts principais

Ativação de cenários:

- `./scripts/enable-chatty-db.sh`
- `./scripts/enable-cache-stampede.sh`
- `./scripts/enable-risk-dependency.sh`
- `./scripts/enable-async-lag.sh`
- `./scripts/enable-lag.sh`
- `./scripts/disable-lag.sh`
- `./scripts/reset-scenarios.sh`

Carga:

- `./scripts/run-k6-web.sh k6/baseline.js`
- `./scripts/run-k6-datadog.sh k6/baseline.js`
- `./scripts/run-k6-datadog.sh k6/hot-accounts.js`
- `./scripts/run-k6-datadog.sh k6/spike.js`

## Documentação útil

- [`docs/datadog-setup.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/datadog-setup.md)
- [`docs/casos-praticos-datadog.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/casos-praticos-datadog.md)
- [`docs/roteiro-demo-datadog.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/roteiro-demo-datadog.md)
- [`docs/arquitetura.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/arquitetura.md)
- [`docs/conceitos.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/conceitos.md)
- [`docs/troubleshooting.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/troubleshooting.md)
- [`observability/slo-examples.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/observability/slo-examples.md)

## Dashboards Datadog

JSONs base para importação:

- [`observability/datadog/dashboards/payment-health.json`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/observability/datadog/dashboards/payment-health.json)
- [`observability/datadog/dashboards/k6-correlation.json`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/observability/datadog/dashboards/k6-correlation.json)
- [`observability/datadog/dashboards/scenario-signals.json`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/observability/datadog/dashboards/scenario-signals.json)
