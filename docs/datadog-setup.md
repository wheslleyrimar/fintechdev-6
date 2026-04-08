# Datadog Setup

Este repositório usa o Datadog como stack principal de observabilidade do laboratório.

## Pré-requisitos

1. Criar conta trial no Datadog.
2. Obter uma API key.
3. Copiar `.env.example` para `.env`.
4. Preencher:

```bash
DATADOG_API_KEY=...
DATADOG_SITE=datadoghq.com
```

## O que está integrado

- **Traces**: os serviços Go exportam via OTLP HTTP para `datadog-agent:4318`.
- **Métricas**: o Agent faz scrape dos endpoints `/metrics` dos serviços.
- **Logs**: o Agent coleta logs de todos os containers do Compose.
- **Carga k6**: há um runner `k6` com dashboard web e envio de métricas para o Datadog via DogStatsD.

## Subir o ambiente

```bash
docker compose up --build
```

Nota para Docker Desktop e versões recentes do Datadog Agent:

- o `datadog-agent` deste laboratório define `HOST_PROC=/proc` no Compose;
- isso evita o problema em que o receiver OTLP HTTP sobe como `Closed` e os traces falham com `connect: connection refused` em `4318`.

Para incluir o runner `k6` com extensão `StatsD`:

```bash
docker compose --profile k6 build k6-datadog
```

## Como validar

No Datadog, valide:

1. **APM > Services**
   - `payment-service`
   - `antifraud-service`
   - `notification-service`
2. **Metrics Explorer**
   - `fintechdev.http_requests_total`
   - `fintechdev.antifraud_messages_processed_total`
   - `fintechdev.notifications_sent_total`
3. **Logs**
   - buscar por `service:payment-service`

## Rodar k6 com dashboard web + Datadog

Com o ambiente já de pé:

```bash
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que esse comando faz:

- abre o dashboard web do `k6` em `http://127.0.0.1:5665`;
- exporta o relatório HTML em `reports/compose-k6-datadog-report.html`;
- envia métricas do teste para o `datadog-agent` em `8125/udp`.

No Datadog, procure por:

1. **Metrics Explorer**
   - `k6.http_req_duration`
   - `k6.http_req_failed`
   - `k6.iterations`
2. **Tags úteis**
   - `scenario:baseline`
   - `scenario:hot_accounts`
   - `scenario:spike`
   - `env:dev`

## Dashboards base

Os JSONs base para importação estão em:

- `observability/datadog/dashboards/payment-health.json`
- `observability/datadog/dashboards/k6-correlation.json`
- `observability/datadog/dashboards/scenario-signals.json`

## Próximos passos

- enriquecer métricas de negócio do fluxo de pagamento;
- revisar traces por dependência externa;
- refinar dashboards da aula com SLO e queries de incidente;
- correlacionar carga com p95/p99, spans lentos e lag assíncrono.
