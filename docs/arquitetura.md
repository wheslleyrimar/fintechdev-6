# Arquitetura do Laboratório

Este laboratório simula um fluxo simplificado de fintech com observabilidade orientada a incidente.

## Componentes

- `payment-service`
  - recebe `POST /payments`
  - aplica rate limit e circuit breaker
  - simula cache, banco e dependência externa de risco
  - publica evento no RabbitMQ
- `antifraud-service`
  - consome eventos assíncronos
  - permite simular degradação no pós-processamento
- `notification-service`
  - consome eventos para notificação
- `rabbitmq`
  - intermedia o fluxo assíncrono
- `datadog-agent`
  - recebe traces OTLP
  - coleta logs de containers
  - recebe métricas DogStatsD do `k6`
  - faz scrape OpenMetrics dos serviços
- `k6`
  - gera baseline, hot keys e spike
  - pode exibir dashboard web e enviar métricas ao Datadog

## Fluxo principal

```text
Cliente -> payment-service -> RabbitMQ -> antifraud-service / notification-service
```

No caminho síncrono, o `payment-service` concentra os cenários de:

- `chatty_db`
- `cache_stampede`
- `risk_dependency`
- `intentional_lag`

No caminho assíncrono, o `antifraud-service` concentra:

- `async_lag`

## Fluxo de observabilidade

```text
services -> OTLP HTTP -> datadog-agent -> Datadog APM
services -> /metrics -> datadog-agent -> Datadog Metrics
containers -> logs -> datadog-agent -> Datadog Logs
k6 -> DogStatsD -> datadog-agent -> Datadog Metrics
```

## Papel de cada ferramenta

- Datadog:
  - ferramenta principal da aula
  - usada para correlação entre carga, latência, traces e logs
- `k6 web dashboard`:
  - mostra o comportamento da carga durante a demo
  - é complementar ao Datadog

## Objetivo didático

O laboratório foi desenhado para responder quatro perguntas de produção:

1. o sintoma aparece no cliente ou só internamente?
2. o gargalo está no serviço ou na dependência?
3. a degradação está no online ou no assíncrono?
4. a normalização depois da correção é visível?
