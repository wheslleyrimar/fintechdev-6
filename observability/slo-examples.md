# SLOs para o Laboratório

Este arquivo continua útil porque ajuda a fechar a aula com critério operacional, não só com incidente.

## SLOs sugeridos

### SLO 1: Caminho online de pagamento

- disponibilidade lógica: `POST /payments` com resposta protegida ou sucesso
- alvo sugerido: `99.9%`
- latência alvo:
  - p95 < `400ms`
  - p99 < `900ms`

## SLO 2: Pós-processamento antifraude

- processamento assíncrono sem atraso excessivo
- alvo sugerido:
  - p95 do antifraude < `2s`
  - throughput compatível com o volume publicado

## Sinais no Datadog

Para o online:

- `fintechdev.http_request_duration_percentiles`
- `fintechdev.http_requests_total`
- `fintechdev.rate_limit_rejected_total`
- `fintechdev.circuit_breaker_state`

Para o assíncrono:

- `fintechdev.antifraud_processing_duration_percentiles`
- `fintechdev.antifraud_messages_processed_total`
- `fintechdev.antifraud_scenario_enabled`

Para a carga:

- `k6.http_req_duration`
- `k6.http_req_failed`
- `k6.iterations`

## Monitores que valem para a aula

### Monitor 1: p95 do payment-service

Condição:

- p95 sustentado acima do objetivo do online

Leitura:

- indica degradação antes de colapso total;
- útil para `chatty_db` e `cache_stampede`.

### Monitor 2: erro protegido ou falha real

Condição:

- aumento de `rate_limit_rejected_total` ou respostas de falha

Leitura:

- separa proteção ativa de indisponibilidade silenciosa.

### Monitor 3: antifraude degradado

Condição:

- p95 assíncrono alto com throughput menor do que o esperado

Leitura:

- mostra sistema online aparentemente saudável, mas operação degradada.

## Burn rate na prática

O laboratório não precisa de cálculo completo de burn rate para a demo, mas a mensagem final deve ser:

- erro ou latência sustentada consome budget;
- quando o budget é queimado rápido, feature para e estabilidade vira prioridade;
- observabilidade boa não serve só para explicar falha passada, serve para governar mudança.
