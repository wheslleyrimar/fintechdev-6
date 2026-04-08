# Casos Práticos Datadog + k6

## Visualização web do k6

O k6 possui um **web dashboard embutido** que pode ser ativado em tempo real durante a execução do teste. Ele é útil para a aula porque permite:

- ver throughput, latência e erro enquanto o teste roda;
- exportar um relatório HTML no final;
- comparar rapidamente o comportamento do teste com os dashboards do Datadog.

### Rodar com dashboard web

```bash
./scripts/run-k6-web.sh k6/baseline.js
```

Para enviar também as métricas do teste ao Datadog:

```bash
./scripts/run-k6-datadog.sh k6/baseline.js
```

Alternativa por container, sem instalar `k6` localmente:

```bash
docker compose run --rm --service-ports k6-web run /work/k6/baseline.js
```

Dashboard padrão:

```text
http://127.0.0.1:5665
```

Relatório HTML:

```text
reports/baseline-report.html
```

No modo `docker compose run`, o relatório padrão fica em:

```text
reports/compose-k6-report.html
```

### Observação

Para correlação com o Datadog, o web dashboard do k6 é complementar, não substituto:

- **k6 web** mostra o comportamento do teste;
- **Datadog** mostra o comportamento da aplicação, dependências, traces e logs.
- **k6 + Datadog** permite correlacionar o instante do pico de carga com p95/p99, rate limit e degradação assíncrona.

## 1. Chatty DB

Ativar:

```bash
./scripts/enable-chatty-db.sh
```

Carga:

```bash
./scripts/run-k6-web.sh k6/baseline.js
```

Observar no Datadog:

- p95/p99 de `payment-service`
- waterfall dos spans `db.select.*`
- aumento de duração total sem necessariamente elevar erro

## 2. Cache Stampede

Ativar:

```bash
./scripts/enable-cache-stampede.sh
```

Carga:

```bash
./scripts/run-k6-web.sh k6/hot-accounts.js
```

Observar:

- queda de cache hit
- aumento de spans `database.query`
- p99 subindo antes de erro

## 3. Dependência de Risco Lenta

Ativar:

```bash
./scripts/enable-risk-dependency.sh
```

Carga:

```bash
./scripts/run-k6-web.sh k6/spike.js
```

Observar:

- spans `external.call.*`
- erros 503 ou proteção 429
- percentis do endpoint `/payments`

## 4. Lag Assíncrono em Antifraude

Ativar:

```bash
./scripts/enable-async-lag.sh
```

Carga:

```bash
./scripts/run-k6-web.sh k6/baseline.js
```

Observar:

- antifraud mais lento
- divergência entre online saudável e pós-processamento degradado

## Reset

```bash
./scripts/reset-scenarios.sh
```
