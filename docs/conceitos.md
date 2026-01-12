# Conceitos Fundamentais

[← Voltar ao README](../README.md)

---

## 1. Escalabilidade: Três Dimensões

**Definição:** Capacidade de um sistema manter SLOs aceitáveis sob aumento de carga, falha e mudança.

### 1.1 Escala de Tráfego (mais requisições)
- Adicionar mais instâncias do serviço
- Load balancing
- **Implementado**: Rate limiting e backpressure

### 1.2 Escala de Estado (mais dados)
- Particionamento (sharding)
- Cache distribuído
- Banco de dados escalável
- **Desafio**: Estado compartilhado é difícil de escalar

### 1.3 Escala Organizacional (mais times e deploys)
- Deploy independente por serviço
- Baixo acoplamento
- **Implementado**: Microsserviços independentes

> **Frase-chave:** Escalar tráfego é fácil. Escalar estado e pessoas é difícil.

## 2. Escala Vertical vs Horizontal

**Escala Vertical (Scale Up):**
- Aumenta CPU/memória do servidor
- Simples, mas limitado pelo hardware
- **Quando usar**: Carga previsível e estável

**Escala Horizontal (Scale Out):**
- Adiciona mais nós/servidores
- Exige arquitetura stateless e idempotência
- **Quando usar**: Carga variável, alta disponibilidade necessária

> **Mensagem:** Horizontal escala exige arquitetura. Vertical exige dinheiro.

## 3. Observabilidade: Três Sinais

**Observabilidade** lida com sinais, não com logs isolados.

### 3.1 Métricas Numéricas (Prometheus)
- **Contadores**: Total de requisições, erros
- **Gauges**: CPU, memória (valores atuais)
- **Histogramas**: Distribuição de latência

### 3.2 Eventos Estruturados (Logs)
- Logs em JSON com campos consistentes
- Correlation ID e Trace ID
- Campos semânticos de negócio

### 3.3 Traces Correlacionados (Jaeger)
- Árvore de spans conectados
- Contexto propagado entre serviços
- Identifica gargalos e dependências

## 4. Métricas RED vs USE

**RED (Request/Error/Duration)** - Para Serviços:
- **Rate**: Taxa de requisições por segundo
- **Errors**: Taxa de erros
- **Duration**: Latência das requisições
- **Uso**: Entender experiência do usuário

**USE (Utilization/Saturation/Errors)** - Para Infra:
- **Utilization**: Uso do recurso (0-100%)
- **Saturation**: Quão "cheio" está o recurso
- **Errors**: Erros do recurso
- **Uso**: Entender causa dos problemas

> **Mensagem:** RED mostra experiência. USE mostra causa.

## 5. Percentis e Latência de Cauda

**Por que não usar média?**

**Exemplo:**
- 99 requisições: 10ms cada
- 1 requisição: 2000ms
- **Média**: ~29ms (parece OK!)
- **p99**: 2000ms (usuário sente lentidão extrema)

**Percentis importantes:**
- **p50** (mediana): 50% das requisições são mais rápidas
- **p90**: 90% das requisições são mais rápidas
- **p95**: 95% das requisições são mais rápidas
- **p99**: 99% das requisições são mais rápidas (cauda)

> **Frase-chave:** Usuário não sente média. Sente cauda.

## 6. Backpressure e Controle de Carga

**Princípio:** Sistema precisa saber dizer "não"

**Técnicas implementadas:**
- **Rate Limiting**: Rejeita requisições além do limite (HTTP 429)
- **Circuit Breaker**: Abre quando há muitas falhas
- **Timeouts Agressivos**: Previne requisições "zombie"

> **Mensagem:** Rejeitar cedo é melhor que falhar tarde.

## 7. SLO, SLA e Error Budget

**SLA (Service Level Agreement):**
- Promessa externa com cliente
- Exemplo: "99.9% uptime"
- Consequência: Penalidade se violado

**SLO (Service Level Objective):**
- Objetivo interno da equipe (mais rigoroso que SLA)
- Exemplo: "99.95% uptime" (para garantir 99.9% SLA)

**Error Budget:**
- Quanto pode falhar: 100% - SLO
- Exemplo: SLO 99.9% = 0.1% pode falhar
- **Uso**: Decidir quando parar features e focar em estabilidade

> **Mensagem:** Sem budget, toda falha vira emergência.

---

[← Voltar ao README](../README.md)
