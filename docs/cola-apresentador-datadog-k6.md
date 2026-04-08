# Cola do Apresentador

Roteiro operacional e de fala para 2 horas.

## 0:00 - 0:05

Objetivo:

- posicionar a aula no nível certo;
- deixar claro que o foco é incidente, não ferramenta.

Fale:

- “Hoje eu não quero mostrar dashboard bonito. Eu quero mostrar como um time sênior separa sintoma, causa e contenção.”
- “Em fintech, latência é também problema de risco, reconciliação e confiança.”

Na tela:

- abrir [`aula-fintech-datadog-k6-keynote.html`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/aula-fintech-datadog-k6-keynote.html)

## 0:05 - 0:12

Objetivo:

- montar o modelo mental de leitura.

Fale:

- “Se eu começo por CPU e memória, eu posso ver ruído e errar causa.”
- “A ordem é: sintoma do cliente, p95/p99, erro ou proteção, trace dominante, backlog assíncrono.”

Perguntas para a sala:

- “Quem aqui já viu média saudável com usuário reclamando?”
- “Quem aqui já viu `201` com operação degradada?”

## 0:12 - 0:20

Objetivo:

- apresentar rapidamente o laboratório.

Fale:

- “O `payment-service` é o online. O `antifraud-service` é o assíncrono. O `k6` injeta a pressão. O Datadog fecha a leitura.”

Na tela:

- mostrar [`docs/arquitetura.md`](/Users/wheslley/Desktop/Fintech%20Dev/Aula%206/fintechdev-6/docs/arquitetura.md)

## 0:20 - 0:30

Objetivo:

- preparar o ambiente e abrir tudo o que será usado.

Comandos:

```bash
cp .env.example .env
docker compose up --build
docker compose --profile k6 build k6-datadog
./scripts/reset-scenarios.sh
```

Abrir:

- Datadog APM
- Datadog Metrics Explorer
- dashboard `Payment Health`
- dashboard `k6 Correlation`
- dashboard `Scenario Signals`
- `http://127.0.0.1:5665`

Fale:

- “Eu quero uma linha contínua de evidência. Não quero trocar de stack para explicar o mesmo incidente.”

## 0:30 - 0:42

Objetivo:

- estabelecer baseline.

Comando:

```bash
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que apontar:

- `k6 web`: throughput e latência estáveis;
- `Payment Health`: p95/p99 comportados;
- APM: nenhum span dominando tempo total;
- `Scenario Signals`: tudo desligado.

Fale:

- “Sem baseline, eu não tenho regressão. Eu só tenho impressão.”

## 0:42 - 0:58

Objetivo:

- demonstrar `chatty_db`.

Comandos:

```bash
./scripts/enable-chatty-db.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

Hipótese a defender:

- a latência subiu por multiplicação de round-trips, não por indisponibilidade.

O que apontar:

- p95 e p99 subindo;
- waterfall com vários `db.select.*`;
- erro não necessariamente acompanha no mesmo ritmo.

Frases úteis:

- “Isso é um problema de desenho transacional.”
- “Várias queries baratas em série viram cauda longa sob carga.”
- “Escalar CPU aqui mascara causa.”

Feche o bloco:

```bash
./scripts/reset-scenarios.sh
```

## 0:58 - 1:14

Objetivo:

- demonstrar `cache_stampede`.

Comandos:

```bash
./scripts/enable-cache-stampede.sh
./scripts/run-k6-datadog.sh k6/hot-accounts.js
```

Hipótese a defender:

- o problema nasce em poucas chaves quentes e se transfere para o banco.

O que apontar:

- cauda cresce primeiro;
- spans de banco ganham participação depois de `cache.lookup`;
- o comportamento aparece com poucas contas, não com distribuição ampla.

Frases úteis:

- “Esse é o padrão clássico de conta quente, limite quente, carteira quente.”
- “Quando o cache erra junto, o banco paga a conta.”

Correções para citar:

- `single-flight`
- jitter no TTL
- `stale-while-revalidate`

Feche o bloco:

```bash
./scripts/reset-scenarios.sh
```

## 1:14 - 1:30

Objetivo:

- demonstrar `risk_dependency`.

Comandos:

```bash
./scripts/enable-risk-dependency.sh
./scripts/run-k6-datadog.sh k6/spike.js
```

Hipótese a defender:

- a dependência externa lenta está sequestrando o caminho crítico durante burst.

O que apontar:

- burst nítido no `k6 web`;
- aumento conjunto de latência e erro;
- `external.call.*` dominando o trace;
- 429 ou 503 como mecanismo de proteção.

Frases úteis:

- “O endpoint é a vítima. A dependência é a causa.”
- “Timeout, retry e breaker são política operacional.”

Feche o bloco:

```bash
./scripts/reset-scenarios.sh
```

## 1:30 - 1:43

Objetivo:

- demonstrar `async_lag`.

Comandos:

```bash
./scripts/enable-async-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

Hipótese a defender:

- o online está razoável, mas a operação está degradando no assíncrono.

O que apontar:

- `Payment Health` pode continuar aceitável;
- `Scenario Signals`: `async_lag=1`;
- métricas do antifraude pioram;
- APM mostra degradação fora do endpoint principal.

Frases úteis:

- “Esse é o incidente que engana time júnior.”
- “Receber `201` não encerra a responsabilidade operacional.”

Feche o bloco:

```bash
./scripts/reset-scenarios.sh
```

## 1:43 - 1:52

Objetivo:

- provar normalização.

Comando:

```bash
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que apontar:

- `k6 web` voltando ao padrão;
- p95/p99 recompondo;
- spans dominantes do incidente desaparecendo;
- flags de cenário zeradas.

Fale:

- “Diagnóstico sem prova de recuperação é só metade do trabalho.”

## 1:52 - 2:00

Objetivo:

- fechar a tese da aula.

Fale:

- “Time sênior não confunde proteção com cura.”
- “Time sênior separa online de assíncrono.”
- “Time sênior usa observabilidade para reduzir incerteza, não para produzir narrativa pós-falha.”

Checklist final:

- começamos pelo sintoma do cliente;
- analisamos p95/p99;
- localizamos o span dominante;
- distinguimos contenção de causa;
- mostramos normalização.
