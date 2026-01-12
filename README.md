# Aula 6 — Escalabilidade e Observabilidade

## 📚 Índice

1. [Visão Geral](#visão-geral)
2. [Como Executar](#como-executar)
3. [Documentação Completa](#documentação-completa)
4. [Endpoints Disponíveis](#endpoints-disponíveis)
5. [Checklist Técnico](#checklist-técnico)

---

## Visão Geral

Este projeto demonstra **escalabilidade e observabilidade** em sistemas distribuídos, implementando:

- ✅ **Métricas RED** (Rate, Errors, Duration) e **USE** (Utilization, Saturation, Errors)
- ✅ **Logs estruturados** com correlation ID e trace ID
- ✅ **Tracing distribuído** com Jaeger
- ✅ **Backpressure** e **rate limiting**
- ✅ **Circuit breaker** para resiliência
- ✅ **Percentis de latência** (p50, p90, p95, p99)
- ✅ **Lag intencional** para simular problemas reais

### Stack Tecnológica

- **Go 1.22**: Serviços de alta performance
- **Prometheus**: Coleta de métricas
- **Grafana**: Visualização de métricas
- **Jaeger**: Distributed tracing
- **RabbitMQ**: Message broker
- **Docker Compose**: Orquestração

---

## Como Executar

### Pré-requisitos

- Docker e Docker Compose instalados
- Portas disponíveis: 8080, 8081, 8082, 5672, 15672, 9090, 3000, 16686

### Passo 1: Subir o Ambiente

```bash
cd "/Users/wheslley/Desktop/Fintech Dev/Aula 6/fintechdev-6"
docker compose up --build
```

### Passo 2: Aguardar Inicialização

Aguarde até ver nos logs:
```
payment-service    | payment-service listening on :8080
antifraud-service  | antifraud-service ready
notification-service | notification-service ready
```

### Passo 3: Verificar Saúde

```bash
curl http://localhost:8080/health
```

Resposta esperada: `{"status":"healthy"}`

### Passo 4: Acessar Interfaces de Observabilidade

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)

---

## Documentação Completa

A documentação completa está organizada em documentos separados para facilitar a navegação:

### 📖 Conceitos Fundamentais
**[docs/conceitos.md](docs/conceitos.md)**
- Escalabilidade: três dimensões
- Escala vertical vs horizontal
- Observabilidade: três sinais
- Métricas RED vs USE
- Percentis e latência de cauda
- Backpressure e controle de carga
- SLO, SLA e Error Budget

### 🏗️ Arquitetura do Sistema
**[docs/arquitetura.md](docs/arquitetura.md)**
- Diagrama de arquitetura (Mermaid)
- Fluxo de processamento de pagamento
- Fluxo de observabilidade
- Componentes e dependências

### 🧪 Guia Completo: Testando e Analisando LAG
**[docs/guia-lag.md](docs/guia-lag.md)**
- O que é LAG e por que simular
- Passo a passo completo (6 fases):
  - Fase 1: Baseline - Sistema sem LAG
  - Fase 2: Ativando LAG Intencional
  - Fase 3: Gerando Requisições com LAG
  - Fase 4: Analisando LAG nas Ferramentas de Observabilidade
  - Fase 5: Diagnóstico e Documentação
  - Fase 6: Desativando LAG
- Exemplos práticos de requisições
- Análise detalhada em Prometheus, Grafana, Jaeger e Logs

### 📊 Guia Completo: Analisando Observabilidade
**[docs/guia-observabilidade.md](docs/guia-observabilidade.md)**
- **Prometheus**: Conceitos, queries essenciais, como analisar
- **Grafana**: Dashboards RED e USE, análise de painéis
- **Jaeger**: Busca de traces, análise de spans, identificação de gargalos
- **Logs**: Logs estruturados, filtros, formatação com jq
- Exemplos práticos e didáticos para cada ferramenta

### 🔍 Investigando Problemas de Latência
**[docs/troubleshooting.md](docs/troubleshooting.md)**
- Detecção de problemas
- Identificação de gargalos
- Quantificação de impacto
- Ações imediatas
- Checklist de investigação rápida

### 🧪 Testes e Demonstrações
**[docs/testes.md](docs/testes.md)**
- Requisição básica
- Teste de carga
- Simulação de gargalos
- Demonstração de lag intencional
- Análise de métricas
- Análise de traces

---

## Endpoints Disponíveis

### Payment Service (porta 8080)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `POST` | `/payments` | Criar pagamento |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Métricas Prometheus |

### Exemplo de Requisição

**Request:**
```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: meu-pagamento-123" \
  -d '{
    "accountId": "acc-1",
    "amount": 100.50,
    "currency": "BRL"
  }'
```

**Response (201 Created):**
```json
{
  "paymentId": "pay-abc123xyz",
  "status": "PROCESSED",
  "processedAt": "2024-01-15T10:30:00Z"
}
```

**Headers de Response:**
- `X-Correlation-ID`: ID de correlação usado
- `X-Trace-ID`: ID do trace distribuído

---

## Checklist Técnico

### ✅ Esse serviço é observável?
- [ ] Expõe métricas (Prometheus)
- [ ] Logs estruturados (JSON)
- [ ] Traces distribuídos (Jaeger)
- [ ] Correlation ID e Trace ID

### ✅ Esse gargalo é detectável?
- [ ] Métricas mostram latência alta
- [ ] Traces identificam operação lenta
- [ ] Logs indicam causa (ex: "slow query")
- [ ] Percentis (p99) revelam cauda

### ✅ Esse alerta é acionável?
- [ ] Aponta violação de SLO
- [ ] Indica impacto no usuário
- [ ] Dispara ação clara
- [ ] Tem runbook associado

---

## Suporte

Em caso de dúvidas:

1. Verifique logs: `docker compose logs -f`
2. Verifique métricas: http://localhost:9090
3. Verifique traces: http://localhost:16686
4. Consulte a [documentação completa](#documentação-completa)

---

**Desenvolvido para demonstrar escalabilidade e observabilidade em sistemas distribuídos.**
