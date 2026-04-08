# Roteiro de Aula — Fintech com Datadog e k6

Este documento organiza a aula como uma investigação operacional de uma fintech.

A ideia central não é mostrar ferramenta primeiro. A ideia é mostrar como um time interpreta sinais de produção em ordem:

1. o cliente sentiu problema ou não;
2. o caminho online degradou ou o assíncrono degradou;
3. o gargalo está no serviço ou na dependência;
4. a proteção atuou ou o sistema apenas ficou lento;
5. a normalização ficou visível depois da ação.

## 0. O que é este repositório

Este repositório é um laboratório didático de observabilidade para uma fintech simulada.

Ele existe para demonstrar, em ambiente controlado, como:

- gerar carga com `k6`;
- provocar incidentes reproduzíveis no fluxo de pagamento;
- observar sinais no Datadog (métricas, traces e logs);
- correlacionar sintoma do cliente com causa técnica;
- validar recuperação após desativar o cenário.

Em termos práticos, o laboratório combina:

- serviços em Go (`payment-service`, `antifraud-service`, `notification-service`);
- mensageria com `RabbitMQ`;
- telemetria via OpenTelemetry e Datadog Agent;
- scripts para ativar/desativar cenários de degradação;
- scripts de teste de carga para baseline, hot accounts e spike.

Resultado esperado da aula: o aluno sai com método de diagnóstico, não só com dashboard bonito.

## 1. Narrativa da arquitetura

Comece a aula explicando o papel de cada serviço no fluxo de uma fintech.

### `payment-service`

Este é o serviço principal do caminho crítico.

Ele:

- recebe `POST /payments`;
- representa a autorização do pagamento;
- aplica proteções como `rate limit` e `circuit breaker`;
- simula interação com cache, banco e dependência externa de risco;
- publica no `RabbitMQ` o evento de domínio `PaymentCreated` no exchange `payments` (fanout).

Detalhe importante do evento publicado:

- nome do evento: `PaymentCreated`;
- exchange: `payments` (`fanout`, broadcast para múltiplos consumidores);
- campos principais no payload: `paymentId`, `accountId`, `amount`, `currency`, `correlationId`, `traceId`, `ts`;
- headers de correlação: `X-Correlation-ID` e `X-Trace-ID`.

Por que publicar esse evento:

- para não bloquear a resposta online com tarefas que podem ser assíncronas;
- para permitir que antifraude e notificações evoluam de forma desacoplada;
- para manter rastreabilidade ponta a ponta via `correlationId`/`traceId`;
- para demonstrar o trade-off clássico: resposta rápida no online vs. risco de backlog no assíncrono.

O que isso significa na linguagem de fintech:

- este serviço fica no caminho que o cliente sente imediatamente;
- se ele degrada, o cliente percebe aumento de latência ou erro na autorização;
- é o melhor serviço para explicar RED: rate, errors, duration.

### `antifraud-service`

Este serviço representa o pós-processamento assíncrono de risco.

Ele:

- consome mensagens do exchange `payments` (evento `PaymentCreated`);
- simula enriquecimento ou verificação posterior de risco;
- calcula um `risk_score` e marca uma fração das transações como fraude;
- permite injetar atraso assíncrono.

Na prática do código:

- lê `paymentId` e `amount` do evento;
- processa com latência variável (incluindo cauda);
- pode adicionar atraso extra no cenário `async_lag`;
- emite métricas como `antifraud_messages_processed` e `antifraud_processing_duration`.

O que isso significa na linguagem de fintech:

- o cliente pode receber `201` e mesmo assim a operação estar ruim;
- o fluxo online pode parecer saudável enquanto o risco ou a reconciliação acumulam atraso;
- é o melhor serviço para explicar a diferença entre SLO online e SLO assíncrono.

### `notification-service`

Este serviço representa o envio de notificações.

Ele:

- consome o mesmo evento `PaymentCreated` no RabbitMQ;
- simula envio de notificações em múltiplos canais (`email`, `sms`, `push`, `webhook`);
- executa os envios em paralelo para representar trabalho operacional pós-pagamento.

Na aula ele é menos central, mas ajuda a reforçar a ideia de ecossistema:

- pagar não é só responder `201`;
- há efeitos colaterais de negócio que continuam após a resposta HTTP.

### `rabbitmq`

Ele separa o caminho online do caminho assíncrono.

Na aula, a mensagem principal é:

- nem tudo que importa para a operação precisa acontecer antes do `201`;
- filas ajudam desacoplamento, mas também criam risco de backlog.

Como isso está modelado aqui:

- o `payment-service` publica no exchange `payments` (fanout);
- `antifraud-service` e `notification-service` criam seus próprios consumidores;
- o mesmo evento dispara múltiplos fluxos sem acoplamento direto entre serviços.

### `datadog-agent`

Ele centraliza a observabilidade.

Ele:

- recebe traces via OTLP;
- coleta logs dos containers;
- faz scrape de `/metrics`;
- recebe métricas do `k6` via DogStatsD.

Na aula, ele existe para permitir correlação entre:

- carga;
- métricas;
- traces;
- logs.

### `k6`

Ele representa o cliente sob carga.

Na aula, o `k6` serve para responder:

- o cliente está vendo mais latência;
- o cliente está vendo erro;
- o throughput caiu;
- isso aconteceu antes ou depois da ativação do cenário.

## 2. Ordem mental da análise

Esta deve ser a linha de raciocínio da aula inteira.

### Passo 1: começar pelo sintoma do cliente

Pergunta:

- o cliente está vendo aumento de latência, erro ou ambos?

Ferramenta principal:

- `k6 web dashboard`
- widgets de latência e volume no `k6 Correlation`

O que observar:

- `iterations`
- `http_req_duration`
- `http_req_failed`

Mensagem didática:

- antes de discutir infraestrutura, confirme o que o cliente está sentindo.

### Passo 2: olhar a saúde do caminho online

Pergunta:

- o fluxo de pagamento online continua saudável?

Ferramenta principal:

- dashboard `Payment Health`

O que observar:

- `HTTP Requests`
- `HTTP Duration p95`
- `HTTP Duration p99`
- `Rate Limited`
- `Circuit Breaker State`
- `Payments Processed`

Mensagem didática:

- `p95` e `p99` importam mais do que média em pagamentos;
- em fintech, a cauda longa é mais perigosa do que a média bonitinha;
- se erro não sobe mas a latência sobe, suspeite de desenho ineficiente ou dependência lenta;
- se erro e latência sobem juntos, suspeite de saturação, timeout ou dependência contaminando o online.

### Passo 3: separar serviço próprio de dependência externa

Pergunta:

- o problema está dentro do `payment-service` ou em algo que ele chama?

Ferramenta principal:

- Datadog APM

O que observar:

- traces do `payment-service`;
- waterfall do trace;
- span dominante;
- repetição excessiva de spans;
- duração de spans externos.

Como explicar:

- se vários spans curtos de banco aparecem em série, pense em `chatty_db`;
- se uma chamada externa domina o trace, pense em `risk_dependency`;
- se o endpoint está ok mas o pós-processamento degrada, o problema pode estar fora do caminho síncrono.

### Passo 4: confirmar se há proteção ou apenas sofrimento

Pergunta:

- o sistema está se defendendo ou apenas ficando pior?

Ferramenta principal:

- `Payment Health`
- `k6 Correlation`

O que observar:

- `Rate Limited`
- `Circuit Breaker State`
- respostas `429` ou `503` no `k6`

Como explicar:

- proteção não é cura;
- proteção é mecanismo de contenção;
- se `rate limit` ou `breaker` atuam, o sistema está tentando evitar colapso total.

### Passo 5: verificar o caminho assíncrono

Pergunta:

- a operação piorou mesmo que o endpoint principal pareça aceitável?

Ferramenta principal:

- `Scenario Signals`
- APM do `antifraud-service`

O que observar:

- `Scenario Flags`
- `Antifraud Throughput`
- `Antifraud Processing p95`

Como explicar:

- esse dashboard mostra que online e assíncrono não têm o mesmo critério de saúde;
- o cliente pode receber `201`, mas o risco ou pós-processamento podem estar degradados.

### Passo 6: fechar a história com normalização

Pergunta:

- desligar o cenário normaliza os sinais?

Ferramenta principal:

- rerodar o mesmo teste;
- comparar dashboards antes e depois.

Mensagem didática:

- sem comparação antes e depois, você só tem narrativa;
- com comparação antes e depois, você tem evidência operacional.

## 2.1 Conceitos rápidos para explicar em sala

Use este bloco como "cola" de explicação curta durante a demonstração.

### Percentis (`p95` e `p99`)

- `p95` = 95% das requisições terminaram até este tempo; 5% ficaram acima.
- `p99` = 99% das requisições terminaram até este tempo; 1% ficou acima.
- Em pagamentos, `p95` e `p99` são mais relevantes que média porque mostram a experiência na cauda.

Exemplo simples:

- se `p95 = 800ms`, significa que 95 em cada 100 requisições terminaram em até `800ms`;
- as 5 restantes foram mais lentas que isso.

### Cauda longa de latência

- "Cauda longa" é o grupo pequeno de requisições muito mais lentas que a maioria.
- Mesmo com média "bonita", cauda ruim afeta UX, timeout e risco de abandono.

### Throughput

- Throughput é a vazão: quantas requisições o sistema processa por unidade de tempo.
- Se a latência sobe e o throughput cai, há sinal de saturação ou gargalo.

### RED (Rate, Errors, Duration)

- `Rate`: volume de requisições por segundo.
- `Errors`: proporção de respostas com erro (por exemplo `5xx`, `429`).
- `Duration`: tempo de resposta (idealmente olhando percentis).

Como interpretar RED rapidamente:

- `Rate` sobe + `Duration` sobe: provável saturação ou gargalo;
- `Duration` sobe sem `Errors` subir: degradação silenciosa (cliente sofre mesmo com `201`);
- `Errors` sobem com `429`: proteção (`rate limit`) atuando;
- `Errors` sobem com `5xx`: falha real no serviço ou dependência.

Mapeamento prático no dashboard `Payment Health`:

- `Rate` -> widget `HTTP Requests`;
- `Errors` -> widget `Rate Limited` (complementar com status HTTP e logs/APM);
- `Duration` -> widgets `HTTP Duration p95` e `HTTP Duration p99`.

### SLO (Service Level Objective)

- SLO é uma meta objetiva de qualidade (ex.: "95% das requisições abaixo de 500ms").
- Na aula, faz sentido separar SLO do caminho online e SLO do assíncrono.

### Trace e Span (APM)

- Trace é o "filme" completo de uma requisição atravessando serviços.
- Span é cada etapa dentro desse filme (query no banco, chamada externa, cache etc.).
- O span dominante costuma apontar o principal gargalo.

### `rate limit` e `circuit breaker`

- `rate limit` limita volume para evitar colapso quando há excesso de carga.
- `circuit breaker` abre quando uma dependência está ruim e evita insistir em chamadas caras.
- Ambos são mecanismos de contenção; não resolvem a causa raiz sozinhos.

### Baseline

- Baseline é a referência de funcionamento normal antes do incidente.
- Sem baseline, você compara percepções; com baseline, compara sinais.

## 3. O que abrir na aula

Abra estas telas antes de começar a demonstração:

- dashboard `Payment Health`
- dashboard `k6 Correlation`
- dashboard `Scenario Signals`
- Datadog APM com foco em `payment-service`
- Datadog APM com foco em `antifraud-service`
- Datadog Metrics Explorer
- `http://127.0.0.1:5665`

## 3.0 Configuração de tela antes da demo

Padronize estas configurações antes de rodar qualquer teste:

- time range padrão: `Last 15 minutes`;
- auto-refresh: `10s` (ou `5s` se o ambiente estiver leve);
- mesma janela de tempo em todos os dashboards;
- timezone alinhado com seu horário local.

### Quais dashboards manter abertos

Obrigatórios:

- `k6 Correlation`
- `Payment Health`
- `Scenario Signals`
- APM com foco em `payment-service`
- APM com foco em `antifraud-service`

Opcional (apoio):

- Datadog `Metrics Explorer`
- `http://127.0.0.1:5665` (k6 web dashboard)

### Quando usar cada janela de tempo

- durante execução do bloco (baseline/incidente): `Last 15 minutes`;
- para destacar um pico muito curto: `Last 5 minutes` (temporariamente);
- para mostrar narrativa completa (antes -> durante -> depois): `Last 1 hour`.

Regra prática de apresentação:

1. inicie cada bloco em `15m`;
2. após desativar cenário e rerodar, troque para `1h` para comparar;
3. volte para `15m` antes do próximo bloco.

## 3.1 Como ler cada dashboard (normal vs anomalia)

Use esta regra geral antes de tudo:

1. compare sempre com o baseline mais recente;
2. procure mudança sustentada (não apenas um ponto isolado);
3. confirme o sinal em pelo menos dois lugares (ex.: `k6` + `Payment Health`).

### Dashboard `k6 Correlation` (visão do cliente)

Objetivo: responder se o cliente está sentindo problema.

Olhe primeiro:

- `http_req_duration` (latência);
- `http_req_failed` (falha);
- `iterations` (ritmo de execução/carga).

Sinal normal:

- latência estável, com variação pequena ao longo do teste;
- falhas próximas de zero;
- `iterations` consistente com o perfil de carga esperado.

Sinal de anomalia:

- `p95/p99` sobem de forma contínua por vários pontos;
- `http_req_failed` sai do patamar normal e permanece alto;
- queda de `iterations` junto com aumento de latência (sinal de saturação).

Interpretação rápida:

- latência sobe sem erro subir: provável gargalo de desempenho;
- latência e erro sobem juntos: provável timeout, saturação ou dependência degradada;
- erro sobe com `429`: proteção (`rate limit`) atuando.

### Dashboard `Payment Health` (visão do serviço online)

Objetivo: validar se o caminho síncrono de pagamento está saudável.

Olhe primeiro:

- `HTTP Duration p95/p99`;
- `HTTP Requests`;
- `Rate Limited`;
- `Circuit Breaker State`;
- `Payments Processed`.

Sinal normal:

- `p95/p99` próximos do baseline;
- volume (`HTTP Requests`) compatível com o teste;
- `Rate Limited` baixo/zero fora de picos extremos;
- breaker fechado (sem ficar alternando);
- `Payments Processed` acompanhando a carga.

Sinal de anomalia:

- `p95/p99` muito acima do baseline por janela contínua;
- `Rate Limited` crescendo durante o incidente;
- breaker abrindo com frequência;
- `Payments Processed` caindo enquanto há carga ativa.

Interpretação rápida:

- latência alta + breaker abrindo: dependência problemática contaminando o online;
- `Rate Limited` alto + erros `429`: sistema em contenção;
- volume alto + processamento não acompanha: gargalo interno.

### Dashboard `Scenario Signals` (visão da causa simulada)

Objetivo: confirmar qual cenário está ativo e seu efeito.

Olhe primeiro:

- `Scenario Flags`;
- `Antifraud Throughput`;
- `Antifraud Processing p95`;
- sinais de delay configurado.

Sinal normal:

- flags em zero quando nenhum cenário está ativo;
- antifraude com throughput estável e p95 controlado.

Sinal de anomalia:

- flags ativas quando você ligou cenário;
- piora de `Antifraud Processing p95` com queda de throughput no assíncrono;
- delay aparecendo no painel após ativação.

Interpretação rápida:

- online ok + assíncrono ruim: problema fora do caminho síncrono;
- flags ativas sem impacto: cenário ligado, mas carga pode estar insuficiente;
- impacto sem flag esperada: revisar ativação de cenário ou serviço alvo.

### APM (`payment-service` e `antifraud-service`)

Objetivo: localizar o gargalo técnico no trace.

Olhe primeiro:

- trace waterfall;
- span dominante (maior duração);
- repetição de spans (padrão serial/chatty).

Sinal normal:

- distribuição de tempo sem um único span dominando excessivamente;
- baixa repetição serial de chamadas de banco/externas.

Sinal de anomalia:

- um span externo dominando grande parte do trace;
- muitos spans curtos em série para mesma operação (chatty DB);
- aumento claro de duração total do trace versus baseline.

### Datadog Metrics Explorer (confirmação cruzada)

Objetivo: validar hipóteses fora dos dashboards prontos.

Quando usar:

- quando um painel sugere anomalia e você quer confirmar por métrica bruta;
- quando precisa quebrar por serviço, rota ou status code.

Boa prática:

- use a mesma janela de tempo em todos os painéis;
- compare "antes do cenário", "durante cenário" e "após desligar cenário".

## 3.2 Heurística de decisão em 60 segundos

Se você estiver ao vivo e precisar decidir rápido:

1. `k6 Correlation`: o cliente sentiu? (`p95/p99`, falha, throughput)
2. `Payment Health`: o online confirma?
3. APM: onde exatamente está o tempo?
4. `Scenario Signals`: o comportamento bate com o cenário ativo?
5. desligue cenário e rerode baseline para validar normalização.

Se os sinais não baterem entre si, trate como hipótese incompleta e colete mais evidência antes de concluir.

## 3.3 Explicação detalhada de APM, Metrics Explorer e widgets

### O que é APM (Application Performance Monitoring)

APM é a parte da observabilidade focada em desempenho de aplicação ponta a ponta.

Na prática, ele responde:

- quanto tempo uma requisição levou;
- em qual etapa o tempo foi gasto;
- qual dependência (banco, cache, API externa) está dominando o atraso;
- onde surgem erros e retries no fluxo.

No Datadog, você usa APM para navegar por traces e spans e sair do "sintoma" para a "causa técnica".

### O que é o Datadog Metrics Explorer

Metrics Explorer é a ferramenta para consultar métricas "cruas" fora dos dashboards prontos.

Use quando você precisa:

- validar uma hipótese com filtro específico (`service`, `status`, `scenario`, `env`);
- trocar agregação (sum, avg, max) para confirmar comportamento;
- quebrar por dimensões (tags) e entender qual fatia do sistema está ruim.

Ele é o lugar ideal para confirmar se um sinal visto no dashboard é real, ruído ou recorte inadequado.

### Dashboard `k6 Correlation` — widget por widget

#### `k6 Iterations`

- O que mede: número de iterações executadas no teste de carga.
- Normal: linha estável para o perfil do script.
- Anomalia: queda inesperada de iterações durante carga ativa.
- Leitura: queda aqui junto com latência alta indica saturação.

#### `k6 Request Duration p95`

- O que mede: latência p95 observada pelo cliente no `k6`.
- Normal: próximo ao baseline e sem tendência de alta sustentada.
- Anomalia: subida contínua por vários pontos.
- Leitura: primeiro sinal de degradação percebida pelo cliente.

#### `k6 Failed Requests`

- O que mede: volume de requisições que falharam no teste.
- Normal: próximo de zero no baseline.
- Anomalia: barras crescentes e sustentadas.
- Leitura: combine com tipo de erro (`429`, `5xx`) para separar contenção de falha real.

#### `Payment Service p95`

- O que mede: p95 interno do `payment-service` (métrica da aplicação).
- Normal: acompanha patamar de baseline.
- Anomalia: sobe junto com `k6 Request Duration p95`.
- Leitura: quando os dois sobem, você confirma que o problema não é só no gerador de carga.

#### `Rate Limit vs Traffic`

- O que mede: tráfego (`http_requests`) versus rejeições por rate limit.
- Normal: tráfego sobe, rejeições baixas/zero.
- Anomalia: rejeições aumentam de forma relevante com tráfego alto.
- Leitura: proteção entrou em ação; precisa avaliar se limite está agressivo ou se há saturação real.

### Dashboard `Payment Health` — widget por widget

#### `HTTP Requests`

- O que mede: volume de requisições recebidas pelo `payment-service`.
- Normal: coerente com o perfil de carga ativo.
- Anomalia: queda de volume sem redução no `k6` (sinal de gargalo/erros upstream).

#### `HTTP Duration p95`

- O que mede: latência p95 do serviço no caminho online.
- Normal: perto do baseline.
- Anomalia: descola do baseline por janela contínua.
- Leitura: bom para detectar degradação geral antes de virar erro.

#### `HTTP Duration p99`

- O que mede: cauda extrema de latência.
- Normal: acima do p95, mas sem explosão.
- Anomalia: p99 dispara muito mais que p95.
- Leitura: cauda longa degradando UX e risco de timeout.

#### `Rate Limited`

- O que mede: requisições bloqueadas por `rate limit`.
- Normal: baixo/zero fora de stress.
- Anomalia: aumento sustentado durante incidente.
- Leitura: sistema está em contenção para evitar colapso.

#### `Circuit Breaker State`

- O que mede: estado do breaker por dependência.
- Normal: circuito fechado na maior parte do tempo.
- Anomalia: alternância frequente ou abertura contínua.
- Leitura: dependência instável contaminando a rota principal.

#### `Payments Processed`

- O que mede: pagamentos processados por status.
- Normal: acompanha o tráfego de entrada.
- Anomalia: queda de processados com carga ativa, ou mudança brusca de status.
- Leitura: evidencia impacto de negócio, não só métrica técnica.

### Dashboard `Scenario Signals` — widget por widget

#### `Scenario Flags`

- O que mede: quais cenários simulados estão habilitados.
- Normal: zero quando nada está ativo.
- Anomalia: flag ativa sem você esperar, ou cenário ativo não refletindo no sinal.
- Leitura: primeiro check de consistência da demo.

#### `Configured Intentional Lag`

- O que mede: lag configurado (database/cache/external) no `payment-service`.
- Normal: zero quando cenário desligado.
- Anomalia: valores altos após `enable-lag` ou persistência após `disable/reset`.
- Leitura: confirma que a configuração de atraso foi aplicada.

#### `Observed Lag Histograms`

- O que mede: lag efetivamente observado em runtime (média dos histogramas).
- Normal: próximo de zero sem cenário.
- Anomalia: observado muito acima do baseline.
- Leitura: valida se o lag configurado realmente apareceu durante tráfego.

#### `Antifraud Throughput`

- O que mede: vazão de mensagens processadas no `antifraud-service`.
- Normal: ritmo estável e compatível com entrada de eventos.
- Anomalia: queda com fila crescendo (ou p95 subindo).
- Leitura: sinal clássico de degradação assíncrona.

#### `Antifraud Processing p95`

- O que mede: p95 do tempo de processamento do antifraude.
- Normal: faixa estável no baseline.
- Anomalia: aumento sustentado após ativar `async_lag`.
- Leitura: cliente pode receber `201` enquanto o pós-processamento piora.

### Sequência curta para interpretar qualquer widget

1. compare com baseline;
2. confirme se a mudança é sustentada;
3. valide em outro dashboard/APM;
4. relacione com cenário ativo;
5. desative cenário e confirme normalização.

## 4. Scripts e o que cada um faz

Explique os scripts como alavancas de incidente.

### `./scripts/reset-scenarios.sh`

O que faz:

- desliga todos os cenários do `payment-service`;
- zera o atraso assíncrono do `antifraud-service`.

Quando usar:

- antes de qualquer bloco da aula;
- entre cenários;
- quando quiser voltar ao baseline.

Efeito esperado:

- `Scenario Flags` volta para zero;
- baseline tende a estabilizar.

### `./scripts/run-k6-datadog.sh k6/baseline.js`

O que faz:

- executa um teste de carga com 10 VUs por 2 minutos;
- usa contas diferentes a cada iteração;
- envia métricas para o Datadog;
- abre o dashboard web do `k6`.

O que representa:

- o comportamento normal do caminho online de pagamento.

Efeito esperado:

- latência sob controle;
- erro baixo ou zero;
- `Payment Health` saudável.

### `./scripts/run-k6-datadog.sh k6/hot-accounts.js`

O que faz:

- executa carga mais intensa, com 30 VUs por 3 minutos;
- concentra tráfego em poucas contas `hot-*`.

O que representa:

- concentração de acesso em poucas chaves quentes;
- situação comum em fintech: conta muito acessada, wallet, limite, perfil de risco.

Efeito esperado:

- mais sensível ao cenário `cache_stampede`.

### `./scripts/run-k6-datadog.sh k6/spike.js`

O que faz:

- cria uma rampa forte de carga;
- sobe de 20 para 120 usuários e mantém o pico por 1 minuto.

O que representa:

- burst repentino;
- promoção, horário de pico, lote de transações, ou incidente em dependência sob pressão.

Efeito esperado:

- mais útil para `risk_dependency`.

### `./scripts/run-k6-web.sh <script>`

O que faz:

- roda o `k6` com dashboard web;
- não depende do envio para Datadog como fluxo principal.

Quando usar:

- se quiser demonstrar só a carga localmente;
- se estiver depurando o comportamento do script.

### `./scripts/enable-chatty-db.sh`

O que faz:

- ativa o cenário `chattyDb` no `payment-service`.

O que isso simula:

- round-trips excessivos ao banco;
- consultas pequenas demais, mas em série;
- desenho ineficiente da leitura de dados no caminho crítico.

O que esperar:

- latência sobe;
- erro pode continuar baixo;
- APM mostra vários spans de banco.

### `./scripts/enable-cache-stampede.sh`

O que faz:

- ativa o cenário `cacheStampede` no `payment-service`.

O que isso simula:

- várias requisições concorrentes para poucas chaves quentes;
- miss de cache em massa;
- banco absorvendo a avalanche.

O que esperar:

- cauda de latência piora;
- `hot-accounts.js` expõe melhor este problema.

### `./scripts/enable-risk-dependency.sh`

O que faz:

- ativa o cenário `riskDependency` no `payment-service`.

O que isso simula:

- dependência externa de risco lenta ou instável;
- uma integração síncrona contaminando a autorização online.

O que esperar:

- latência e possivelmente erro sobem juntos;
- APM mostra spans externos dominando o trace;
- `spike.js` ajuda a amplificar o efeito.

### `./scripts/enable-async-lag.sh`

O que faz:

- aplica `1500ms` extras no `antifraud-service`.

O que isso simula:

- consumidor assíncrono ficando atrasado;
- antifraude, reconciliação ou pós-processamento lento.

O que esperar:

- `Payment Health` pode permanecer bom;
- `Scenario Signals` mostra piora no antifraude.

### `./scripts/enable-lag.sh`

O que faz:

- ativa lag intencional em 100% das requisições do `payment-service`;
- injeta:
  - `2000ms` no banco;
  - `500ms` no cache;
  - `1000ms` na chamada externa.

O que isso simula:

- um incidente didático claro no caminho online;
- excelente para mostrar um antes e depois muito visível.

O que esperar:

- `p95` e `p99` disparam;
- throughput cai;
- `Scenario Signals` mostra os delays configurados;
- o teste `baseline.js` costuma falhar no threshold de latência.

### `./scripts/disable-lag.sh`

O que faz:

- desativa o lag intencional do `payment-service`.

Quando usar:

- para mostrar a normalização após o incidente.

## 4.1 Mapa rápido dos problemas simulados

Use esta tabela para explicar cada problema em menos de 1 minuto.

| Problema | Causa raiz típica | Sintoma para o cliente | Onde observar primeiro | O que fazer na demo |
|---|---|---|---|---|
| `chatty_db` | Muitas queries pequenas e sequenciais no banco (excesso de round-trip) | Latência sobe (`p95/p99`), erro pode continuar baixo no início | `Payment Health` + APM (`spans` de banco em série) | Ative `./scripts/enable-chatty-db.sh` e rode `k6/baseline.js` |
| `cache_stampede` | Várias requisições concorrentes para a mesma chave após miss/expiração | Cauda de latência piora, banco sobrecarrega | `k6 Correlation` + APM (mais tempo em banco) | Ative `./scripts/enable-cache-stampede.sh` e rode `k6/hot-accounts.js` |
| `risk_dependency` | Dependência externa lenta/instável no caminho síncrono | Latência alta; sob carga, erro também pode subir | APM (`span` externo dominante) + `Payment Health` | Ative `./scripts/enable-risk-dependency.sh` e rode `k6/spike.js` |
| `intentional_lag` | Delay artificial no cache, banco e chamada externa | Degradação forte e didática no online; `p95/p99` disparam | `k6 Correlation` + `Payment Health` + `Scenario Signals` | Ative `./scripts/enable-lag.sh` e rode `k6/baseline.js` |
| `async_lag` | Consumidor assíncrono (`antifraud`) processando com atraso | Cliente pode receber `201`, mas pós-processamento degrada | `Scenario Signals` + APM do `antifraud-service` | Ative `./scripts/enable-async-lag.sh` e rode `k6/baseline.js` |

Leitura rápida durante a aula:

- problema no `k6` mas não no `Payment Health`: valide assíncrono e configuração de cenário;
- `Payment Health` ruim e APM com `span` dominante: foque na dependência dominante;
- após `disable/reset`, os sinais devem voltar perto do baseline.

## 5. Sequência recomendada de aula

Esta é a melhor ordem para a explicação.

### Bloco 1: baseline saudável

Comandos:

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que dizer:

- primeiro estabelecemos o comportamento normal;
- sem baseline, diagnóstico vira opinião;
- queremos saber qual é a latência normal do pagamento.

O que mostrar:

- `k6 Correlation`: carga regular, poucas ou nenhuma falha;
- `Payment Health`: p95 e p99 controlados;
- APM do `payment-service`: nenhum span dominante;
- `Scenario Signals`: sem incidentes ativos.

### Bloco 2: incidente online muito visível

Comandos:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que dizer:

- agora vamos criar um incidente claro no caminho online;
- a resposta pode continuar sendo `201`, mas a latência vai explodir;
- isso é muito real em fintech: serviço não cai, mas degrada forte.

O que mostrar:

- `k6 Correlation`: `k6 Request Duration p95` sobe forte;
- `Payment Health`: `HTTP Duration p95` e `p99` sobem;
- `Scenario Signals`: delays configurados aparecem.

Lição:

- nem todo incidente é erro;
- muitos incidentes começam como degradação de latência.

### Bloco 3: voltar ao normal

Comandos:

```bash
./scripts/disable-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que dizer:

- agora vamos provar que o sinal normaliza;
- observabilidade boa não só explica falha;
- ela também comprova recuperação.

O que mostrar:

- queda do `p95` e `p99`;
- throughput voltando;
- ausência do sinal de lag.

### Bloco 4: desenho ruim de acesso a dados

Comandos:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-chatty-db.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que dizer:

- aqui o problema não é indisponibilidade;
- o problema é desenho ineficiente no caminho crítico.

O que mostrar:

- `Payment Health`: latência sobe mais do que erro;
- APM: muitos spans de banco em série;
- `Scenario Signals`: `chatty_db=1`.

Lição:

- várias consultas pequenas e sequenciais viram cauda longa sob carga.

### Bloco 5: chave quente e cache stampede

Comandos:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-cache-stampede.sh
./scripts/run-k6-datadog.sh k6/hot-accounts.js
```

O que dizer:

- agora concentramos tráfego em poucas contas;
- isso simula contas quentes, wallets quentes ou perfis muito acessados.

O que mostrar:

- `k6 Correlation`: piora de latência em carga concentrada;
- APM: participação maior de spans de banco;
- `Scenario Signals`: `cache_stampede=1`.

Lição:

- chave quente é um problema de arquitetura, não só de infraestrutura.

### Bloco 6: dependência externa contaminando pagamento

Comandos:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-risk-dependency.sh
./scripts/run-k6-datadog.sh k6/spike.js
```

O que dizer:

- agora o problema vem de uma dependência de risco;
- o endpoint principal sofre por causa de uma chamada externa.

O que mostrar:

- `k6 Correlation`: spike, latência alta e possível erro protegido;
- `Payment Health`: latência e erro podem subir juntos;
- APM: spans externos dominando o trace.

Lição:

- dependência síncrona lenta pode ser mais perigosa do que CPU local alta.

### Bloco 7: caminho assíncrono degradado

Comandos:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-async-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

O que dizer:

- agora o cliente pode continuar recebendo `201`;
- mas a operação continua ruim porque o pós-processamento degrada.

O que mostrar:

- `Payment Health`: pode continuar aceitável;
- `Scenario Signals`: `Antifraud Throughput` e `Antifraud Processing p95` pioram;
- APM do `antifraud-service`: tempo maior no consumidor.

Lição:

- uma fintech madura monitora online e assíncrono separadamente.

## 5.1 Como executar a demo ao vivo (passo a passo detalhado)

Use este roteiro operacional durante a apresentação. A ideia é reduzir improviso.

### Etapa A: preparação (antes de compartilhar a tela)

1. Garanta pré-requisitos:
   - Docker e Docker Compose instalados;
   - `.env` preenchido com `DATADOG_API_KEY`.
2. Suba o ambiente:

```bash
docker compose up --build
docker compose --profile k6 build k6-datadog
```

3. Valide serviços principais:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/admin/scenarios
curl http://localhost:8081/admin/scenarios
```

4. Abra previamente as telas:
   - Datadog dashboards `Payment Health`, `k6 Correlation`, `Scenario Signals`;
   - APM `payment-service`;
   - APM `antifraud-service`;
   - `http://127.0.0.1:5665` (k6 web dashboard).
5. Zere cenários antes de começar:

```bash
./scripts/reset-scenarios.sh
```

### Etapa B: abertura da aula (2-4 minutos)

1. Explique em 30 segundos o objetivo:
   - "vamos simular incidentes e diagnosticar por sinais, do sintoma à causa."
2. Explique o fluxo:
   - cliente chama `payment-service`;
   - serviço publica evento;
   - `antifraud-service` processa assíncrono.
3. Reforce o método:
   - começar no sintoma do cliente (`k6`);
   - confirmar no serviço (`Payment Health`);
   - aprofundar no APM;
   - fechar com normalização.

### Etapa C: baseline ao vivo (primeira execução)

1. Rode baseline:

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

2. Enquanto roda, narre:
   - "este é o comportamento normal de referência."
3. Mostre:
   - `k6 Correlation`: baixa falha e latência controlada;
   - `Payment Health`: `p95` e `p99` estáveis;
   - APM: sem span dominante;
   - `Scenario Signals`: sem flags ativas.
4. Conclusão do bloco:
   - "sem baseline, não existe comparação confiável."

### Etapa D: incidente visível no caminho online

1. Ative lag intencional e rode de novo:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

2. Narração esperada:
   - "o endpoint ainda pode responder, mas agora está lento."
3. Mostre evidências na ordem:
   - `k6 Correlation`: `k6 Request Duration p95` sobe;
   - `Payment Health`: `HTTP Duration p95/p99` sobem;
   - `Scenario Signals`: delays configurados ativos.
4. Mensagem didática:
   - "incidente nem sempre começa por erro; muitas vezes começa por latência."

### Etapa E: provar recuperação

1. Desative o cenário e rerode baseline:

```bash
./scripts/disable-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

2. Mostre comparação antes/depois:
   - queda de `p95/p99`;
   - throughput voltando;
   - flags de cenário retornando ao normal.
3. Mensagem:
   - "observabilidade boa também comprova recuperação."

### Etapa F: cenários adicionais (escolha 1-2 conforme tempo)

Se houver tempo, execute nesta ordem sugerida:

1. `chatty_db` (degradação por desenho de acesso a dados):

```bash
./scripts/reset-scenarios.sh
./scripts/enable-chatty-db.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

2. `cache_stampede` com carga concentrada:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-cache-stampede.sh
./scripts/run-k6-datadog.sh k6/hot-accounts.js
```

3. `risk_dependency` com spike:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-risk-dependency.sh
./scripts/run-k6-datadog.sh k6/spike.js
```

4. `async_lag` para mostrar diferença entre online e assíncrono:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-async-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

### Etapa G: script de fala por incidente (modelo repetível)

Repita sempre este mini-script:

1. "Qual sintoma o cliente sentiria?"
2. "Vamos rodar carga para medir."
3. "No `k6`, o sintoma apareceu?"
4. "No `Payment Health`, o serviço confirma?"
5. "No APM, onde está o gargalo?"
6. "No `Scenario Signals`, o cenário explica o sinal?"
7. "Vamos desativar e provar normalização."

### Etapa H: checklist de contingência (se algo der errado ao vivo)

1. Sem dados no Datadog:
   - validar `.env` e `DATADOG_API_KEY`;
   - reiniciar stack;
   - aguardar ingestão por alguns minutos.
2. `k6` não abre dashboard:
   - checar `http://127.0.0.1:5665`;
   - rerodar script `run-k6-*`.
3. Cenário não ativou:
   - consultar `curl http://localhost:8080/admin/scenarios`;
   - executar `./scripts/reset-scenarios.sh` e tentar novamente.
4. Evite travar a aula:
   - se um bloco falhar, volte para baseline e siga para o próximo cenário.

## 6. Como conduzir cada análise

Para cada incidente, use sempre a mesma ordem de fala:

1. diga qual sintoma o cliente sentiria;
2. rode o `k6`;
3. olhe o `k6 Correlation`;
4. confirme no `Payment Health`;
5. aprofunde no APM;
6. valide no `Scenario Signals`;
7. desligue o cenário;
8. mostre a normalização.

Essa repetição é boa para aula porque ensina método.

## 7. Frases boas para usar em sala

- "Não comecem pela infraestrutura. Comecem pelo sintoma do cliente."
- "Média bonita não salva pagamento com cauda ruim."
- "Proteção não é cura. Breaker e rate limit servem para conter dano."
- "201 não significa operação saudável. O assíncrono ainda pode estar degradado."
- "Sem baseline, a gente conta história. Com baseline, a gente faz engenharia."
- "A pergunta principal não é se está lento. É onde a lentidão nasce."

## 8. Comandos de referência

Baseline:

```bash
./scripts/reset-scenarios.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

Lag intencional:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
./scripts/disable-lag.sh
```

Chatty DB:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-chatty-db.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

Cache stampede:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-cache-stampede.sh
./scripts/run-k6-datadog.sh k6/hot-accounts.js
```

Dependência de risco:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-risk-dependency.sh
./scripts/run-k6-datadog.sh k6/spike.js
```

Lag assíncrono:

```bash
./scripts/reset-scenarios.sh
./scripts/enable-async-lag.sh
./scripts/run-k6-datadog.sh k6/baseline.js
```

## 9. Fechamento da aula

Feche com três mensagens:

1. fintech não quebra apenas por erro; quebra também por latência e atraso operacional;
2. dashboards só fazem sentido quando contam uma história coerente com traces e carga;
3. o valor da observabilidade não é olhar gráfico bonito, e reduzir tempo para diagnóstico e para validação da recuperação.
