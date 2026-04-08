# Conceitos para a Aula

Este documento só mantém conceitos que ajudam a interpretar os incidentes do laboratório.

## 1. O que muda em sistemas de fintech

Em fintech, latência não é só desconforto de UX. Ela impacta:

- autorização no caminho crítico;
- janelas de risco e antifraude;
- reconciliação;
- consistência percebida pelo cliente.

O sistema pode responder `201` e ainda assim estar operacionalmente degradado se o pós-processamento estiver atrasado.

## 2. Escalabilidade de verdade

Escalar não é apenas subir mais réplicas.

O que normalmente limita uma fintech:

- estado compartilhado;
- round-trips excessivos em leituras críticas;
- dependências síncronas demais;
- burst de carga em chaves quentes;
- filas que crescem mais rápido do que o consumo.

## 3. O que observar primeiro

Ordem recomendada de leitura:

1. sintoma do cliente;
2. latência p95/p99;
3. taxa de erro e proteção ativada;
4. span dominante;
5. sinais de saturação ou backlog.

Essa ordem evita começar a investigação pela infraestrutura errada.

## 4. RED e USE

RED responde se o serviço está saudável para o usuário:

- `Rate`
- `Errors`
- `Duration`

USE responde por que a degradação está acontecendo:

- `Utilization`
- `Saturation`
- `Errors`

Na aula:

- `Payment Health` é a leitura RED;
- `Scenario Signals` e traces ajudam a fechar a leitura USE.

## 5. Cauda importa mais do que média

Se algumas transações sofrem 2 segundos extras por dependência externa, a média pode parecer aceitável e mesmo assim o fluxo estar ruim.

Para o laboratório, priorize:

- p95;
- p99;
- waterfall do trace.

## 6. Proteção não é cura

Rate limit, breaker e timeout existem para conter dano.

Eles não resolvem:

- `chatty_db`
- `cache_stampede`
- dependência externa lenta

Eles apenas impedem que a degradação vire colapso total.

## 7. Online e assíncrono têm SLOs diferentes

Em fintech madura, o endpoint de pagamento e o pós-processamento não compartilham o mesmo critério de saúde.

Exemplo:

- online saudável: autorização em poucos centenas de ms;
- assíncrono degradado: antifraude ou notificação com backlog crescente.

Por isso existe o cenário `async_lag`.

## 8. O que é uma boa hipótese operacional

Hipótese útil é específica e testável.

Exemplos:

- "o p99 subiu porque a dependência de risco está dominando o trace"
- "o banco está absorvendo cache miss em massa em poucas contas quentes"
- "o caminho síncrono está saudável, mas o consumidor assíncrono acumulou atraso"

Hipótese ruim:

- "o sistema ficou lento"
