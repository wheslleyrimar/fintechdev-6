# Arquitetura do Sistema

[← Voltar ao README](../README.md)

---

## Diagrama de Arquitetura

```mermaid
graph TB
    Client[Cliente/Usuário]
    Payment[Payment Service<br/>:8080]
    RabbitMQ[RabbitMQ<br/>Message Broker]
    Antifraud[Antifraud Service<br/>:8081]
    Notification[Notification Service<br/>:8082]
    
    Prometheus[Prometheus<br/>:9090]
    Grafana[Grafana<br/>:3000]
    Jaeger[Jaeger<br/>:16686]
    
    Client -->|POST /payments| Payment
    Payment -->|Rate Limiting| Payment
    Payment -->|Circuit Breaker| Payment
    Payment -->|Cache Lookup| Payment
    Payment -->|Database Query| Payment
    Payment -->|External Calls x3| Payment
    Payment -->|Publica Evento| RabbitMQ
    
    RabbitMQ -->|Consome| Antifraud
    RabbitMQ -->|Consome| Notification
    
    Payment -.->|Métricas| Prometheus
    Antifraud -.->|Métricas| Prometheus
    Notification -.->|Métricas| Prometheus
    
    Prometheus -->|Dados| Grafana
    
    Payment -.->|Traces| Jaeger
    Antifraud -.->|Traces| Jaeger
    Notification -.->|Traces| Jaeger
    
    style Payment fill:#4CAF50
    style Antifraud fill:#2196F3
    style Notification fill:#FF9800
    style Prometheus fill:#E91E63
    style Grafana fill:#9C27B0
    style Jaeger fill:#00BCD4
```

## Fluxo de Processamento de Pagamento

```mermaid
sequenceDiagram
    participant C as Cliente
    participant PS as Payment Service
    participant RL as Rate Limiter
    participant CB as Circuit Breaker
    participant Cache as Cache
    participant DB as Database
    participant Ext as External Services
    participant RMQ as RabbitMQ
    participant AF as Antifraud
    participant NS as Notification
    
    C->>PS: POST /payments
    PS->>RL: Verifica limite
    RL-->>PS: OK (ou 429)
    PS->>CB: Verifica estado
    CB-->>PS: Closed (ou Open)
    PS->>Cache: Busca dados
    Cache-->>PS: Cache hit/miss
    PS->>DB: Query dados
    DB-->>PS: Resultado
    loop 3 vezes
        PS->>Ext: Chamada externa
        Ext-->>PS: Resposta
    end
    PS->>RMQ: Publica evento
    PS-->>C: 201 Created
    RMQ->>AF: Consome evento
    RMQ->>NS: Consome evento
```

## Fluxo de Observabilidade

```mermaid
graph LR
    Services[Serviços Go]
    Metrics[Métricas<br/>Prometheus]
    Logs[Logs<br/>Estruturados]
    Traces[Traces<br/>Jaeger]
    
    Services -->|Expõe /metrics| Metrics
    Services -->|Gera JSON| Logs
    Services -->|Envia spans| Traces
    
    Metrics -->|Query| Prometheus[Prometheus<br/>:9090]
    Prometheus -->|Visualiza| Grafana[Grafana<br/>:3000]
    
    Traces -->|Armazena| Jaeger[Jaeger<br/>:16686]
    
    style Services fill:#4CAF50
    style Metrics fill:#E91E63
    style Logs fill:#FF9800
    style Traces fill:#00BCD4
```

## Observabilidade

- Todos os serviços expõem métricas para Prometheus
- Todos os serviços enviam traces para Jaeger
- Todos os serviços geram logs estruturados

---

[← Voltar ao README](../README.md)
