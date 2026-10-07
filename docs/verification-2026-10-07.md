# Aceite das Waves 01–18 — 2026-10-07

O WOS está pronto como candidato de validação independente por humanos, agentes e produtos consumidores. O escopo aceito é o servidor/Core WOS em Linux amd64, SQLite e PostgreSQL 18.6, HTTP/MCP, cliente web e SDK Go. A integração final é o PR #14, após o merge do planejamento no PR #13. Uma tag/release pública e o deploy do produto consumidor Woobe são ações separadas. O estado de integração é verificável nos PRs; não se usa CI de um head anterior como aprovação do head final.

## Critérios canônicos de Release 0.1

A seção 29.5 do design define treze gates. Esta matriz é o aceite atual; o relatório de 5 de outubro preserva o histórico e recomendações mais amplas de produto.

| Gate | Evidência executável |
| --- | --- |
| 1. Standalone sem Woobe | CLI/config/HTTP runtime tests; jornada independente de 18 etapas; smokes de processo/contêiner no CI |
| 2. Embedded externo | `tests/boundary` executa `examples/embedded` a partir de outro módulo Go |
| 3. Mutações remotas idempotentes | Guards de todos os 78 comandos HTTP/MCP em ambos os bancos; Namespace admin receipts; replay/restart e resposta HTTP perdida reconciliada por MCP |
| 4. Versões obsoletas | Contratos de agregado/draft, matriz de proof e conflito real entre dois participantes na interface web |
| 5. Claims/ciclos concorrentes | Conexões/pools separados para claims, idempotência, assessments e slots; corrida de dependências opostas e oráculo de reachability para ciclos indiretos |
| 6. Paridade de bancos | Contratos gerados compartilhados, runtime HTTP/MCP/SDK/sessão, upgrade com dados, restart e clean restore em PostgreSQL real e SQLite |
| 7. HTTP/MCP mesmos services | Catálogo único tipado, igualdade dos schemas anunciados por clientes reais, modos attestation/evidence_review/external_evaluation, quatro perfis MCP anunciados e reconciliação entre transports |
| 8. Snapshots coerentes/limitados | Cursores vinculados a escopo/revisão/tempo, omissões e orçamento JSON, progresso com denominadores, leitura em lote e grafo sem referências falsas no recorte |
| 9. Planos imutáveis/recuperáveis | Contratos de publicação, histórico/slots, restart/restore, replanejamento em navegador e projeção viva sem alterar publicação |
| 10. Prova preservada | Históricos de critério/assessment/Conclusion/Evidence, bulk/single equivalentes, restore, reviewer policy e contestação sem reopen implícito |
| 11. Sinais duráveis | Rollback de state/event/outbox/receipt, lease/fencing de worker, retries assinados com identidade estável, crash exhaustion/redelivery e interrupção em voo |
| 12. Documentação operacional | Apache 2.0, README, contratos/OpenAPI, ADRs, inventário, operação/backup, limites, perfis MCP e guia para agentes |
| 13. Restart/restore | Consumidor SDK em processo independente e jornada de 18 etapas; dump/restore em banco PostgreSQL vazio e arquivo SQLite limpo |

A matriz exercita cenários funcionais e as fronteiras compartilhadas. Não equivale a enumerar toda combinação possível de payload, rota, plataforma ou consumidor. O schema não substitui invariantes de Domain/Application; os contratos dessas camadas continuam executados.

## Execução local

Go 1.27.1; Node 24.19.0/npm 11.9.0; Playwright 1.62.1; Chromium 151.0.7922.173; Linux amd64; PostgreSQL 18.6 descartável em Docker, bind local. Schemas de teste são isolados. As variáveis `WOS_TEST_POSTGRES_DSN` e `WOS_TEST_POSTGRES_CONTAINER` estavam presentes, incluindo clean restore; nenhum caso PostgreSQL é contado como aprovado por skip.

- `go test -json ./...` e `go test -race ./...` passaram: 437 testes e subtestes, zero falhas e zero testes ignorados. Pacotes sem arquivos de teste são declarados como tal.
- `go vet ./...`, módulos verificados e tidy sem alteração de go.mod/go.sum; geração PostgreSQL/HTTP/MCP/SDK/OpenAPI e catálogo de eventos verificados.
- Duas jornadas Playwright passaram: planejamento/MCP/replanejamento/restart e revisão/obrigação/impedimento/conflito/contestação/archive/resume. Teclado, retorno de foco, nomes dos campos e viewport de 390 px são verificados; não se alega auditoria WCAG completa por esses testes.
- Jornada independente passou em todas as 18 etapas sobre o mesmo Outcome, com humano, dois agentes, lease real, arquivo de comprovação, webhook duplicado e restore limpo.
- Fuzz de reachability do grafo, escopo de cursor e serialização pública; resultados medidos em `audit/release-2026-10-07-fuzz.txt`.
- Benchmarks SQLite/PostgreSQL, contagens SQL e throughput com quatro leitores: [método e limites](benchmarks.md).

Saídas de execução estão em `docs/audit/release-2026-10-07-*`. O build local do Dockerfile encontrou DNS indisponível dentro do builder ao baixar Alpine; o gate de contêiner/build reproduzível deve passar no CI do head final antes do merge. Esse problema do ambiente não é registrado como aprovação local.

## Reprodução

```sh
export WOS_TEST_POSTGRES_DSN='postgres://postgres@127.0.0.1:15432/wos_test?sslmode=disable'
export WOS_TEST_POSTGRES_CONTAINER=wos-release-postgres
go mod verify
go vet ./...
go test ./...
go test -race ./...
go build -trimpath -buildvcs=false -o bin/wos ./packages/wos-api/cmd/wos
python3 tests/acceptance/journey.py --binary bin/wos --client http
cd tests/web
npm ci
npx playwright install --with-deps chromium
npm test
```

O DSN acima é apenas um banco de testes local descartável com autenticação trust em loopback. Em implantação remota use credenciais e HTTPS conforme `operations.md`. Nunca use banco de produção para a suíte de restore. Sem DSN, `go test` pula os testes PostgreSQL e não prova esse gate.

## Limites da entrega

A resposta compacta é limitada, mas o custo de reconstruir todo o estado/histórico não é constante. As listagens de Objectives/WorkItems usam bulk reads; outras coleções e o detalhe de histórico individual ainda têm custos proporcionais ao dataset. Benchmarks são fixtures, não metas de produção. Métricas são cumulativas e reiniciam com o processo; não há histogramas/OTel. O perfil OAuth é futuro; esta versão usa token opaco e sessões. A aceitação de um cliente independente não certifica o deploy completo de Woobe nem corrige vulnerabilidades de sua imagem. A referência Woobe documenta a dependência do PR #178 de compatibilidade. Essas limitações estão expostas para a validação do consumidor.
