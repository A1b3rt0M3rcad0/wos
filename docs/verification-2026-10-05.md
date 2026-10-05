# Verificação da implementação — 2026-10-05

Escopo: branch do PR #14, baseada no PR #13; Go 1.27.1, Linux amd64. Resultados de execução local são separados dos gates de integração e release. PostgreSQL não estava disponível localmente: testes que exigem `WOS_TEST_POSTGRES_DSN` são ignorados explicitamente, não aprovados. O CI configura PostgreSQL com essa variável obrigatória. O CI #232 executou PostgreSQL 18.6 real e passou integralmente; os skips locais não são usados como evidência desse resultado.

## Evidência executada

| Verificação | Resultado | Alcance |
| --- | --- | --- |
| `go test ./...` | Passou | Domain, Application, memória/SQLite, HTTP, SDK, MCP oficial por HTTP e processo stdio, autorização, sessões, contextos indexados, sinais duráveis e webhook real. PostgreSQL ignorado sem DSN. |
| `go test -race ./...` | Passou no checkpoint de instrumentação e no fechamento | Inclui reautorização transacional e recuperação de integrações; PostgreSQL ignorado sem DSN. |
| `go vet ./...` | Passou | Análise estática; não prova contratos funcionais. |
| Fuzz `FuzzCursorRejectsForeignScope`, 2 s | Passou, 38.210 execuções | Decoder e vínculo do cursor ao escopo; não equivale ao fuzz completo de grafos. |
| Fuzz `FuzzPublicCommandRoundTrip`, 2 s | Passou, 22.778 execuções | Serialização pública/duração em segundos e decode de comando; não é paridade de todos os comandos. |
| Benchmark SQLite, 500 ms por caso | Passou | Saída original em `audit/benchmark-sqlite.txt`, método/limites em `benchmarks.md`. |
| Jornada em navegador real | Passou, 1 teste em 4,6 s | Playwright 1.62.1; Chromium 153.0.8010.0 obtido via pacote temporário `@sparticuz/chromium` 153.0.0, sem desabilitar segurança web. Sessão, criação, plano inicial/revisão 2, agente MCP, avaliação humana e novo processo de navegador após restart. Viewport 390 px sem overflow; não equivale a auditoria de acessibilidade. |
| Drift de geração | Passou, 63 arquivos sem mudança após regenerar | PostgreSQL, catálogo HTTP, MCP, SDK e OpenAPI de comandos. |
| Build Linux standalone com `-trimpath` | Passou | Sem runtime Woobe; matriz/reprodutibilidade ainda pendentes. |

## Testes com significado de produto

- `TestIndependentSDKConsumerContinuesAfterRestart`: consumidor Go independente descobre e continua o mesmo estado após restart.
- `TestRealMCPClientSharesDurableStateWithHTTP` e `TestRealStdioMCPSubprocess`: cliente do SDK oficial exercita transporte real e estado compartilhado com HTTP.
- `TestRemoteMCPRevalidatesGrantsAndAuthenticatedAuthorship`, `TestBrowserSessionCatalogAndRevocation`, `TestPermissionRevokedWhileAwaitingTransactionCannotMutate`: identidade, sessões e revogação antes de replay/mutação.
- `TestSecurityAdministrationCASReceiptsSecretsAndRevocation`: CAS entre conexões reais SQLite, receipt idempotente, token somente uma vez, auditoria e revogação após restart.
- `TestNamespaceCreationCopiesOnlyGrantedPermissions`: novo contexto não amplia permissões nem aceita a credencial em outro Namespace.
- `TestIndexedExternalContextPreservesTypeAndRestart`: filtro AND indexado, número/string/bool/null distintos, autoria, substituição e restart.
- `TestContinuationFromDiscoverySnapshotAndCursors`: descoberta, omissões, cursor por revisão e escopo, timeline completa por páginas.
- `TestTerminalOutcomeRequiresExplicitReopenBeforeChangingObligations`: alterações estruturais rejeitadas preservam Conclusion e lifecycle.
- `TestDurableSignalsAtomicRollbackRestartAndDeliveryFencing`: em SQLite inclui backup consistente e restore em arquivo vazio, credencial/sessão com parent, grants, audit e recibos. PostgreSQL compartilha a recuperação por restart, sem alegar pg_dump/restore. Também prova source/event/outbox/idempotência atômicos, rollback injetado, pending após restart e worker antigo impedido de confirmar.
- `TestDeliveryCrashBudgetRequiresExplicitRedelivery`: seis leases expiram, a entrega fica esgotada após restart, o worker antigo é rejeitado; redelivery idempotente preserva identidade e incrementa fencing.
- `TestProofProgressDoesNotTreatWaiverOrRetractedEvidenceAsProof`: waiver e revisão antiga não contam como prova; retração retira comprovação e denominador zero não inventa percentual.
- `TestRealWebhookRetryKeepsEventIdentityAndSignature`: HTTP real 503 → 204, identidade estável e assinatura.
- `TestWave11UpgradeExistingVersionSevenDatabase`: schema antigo com dados preservado nas migrations novas; não é prova de upgrade PostgreSQL.

## Gates ainda abertos

Matriz exaustiva de bancos/transports; jornada visual completa e acessibilidade; restore com todas as coleções; desligamento com requisições/entregas em voo; otimização e contagem de queries; todos os exemplos de produto/Woobe; matriz de plataformas/containers; comportamento de todas as rotas OpenAPI; demonstração integral da auditoria. Jobs configurados, testes existentes e código compilado não substituem execuções ausentes.

## Reprodução final

```sh
go test ./...
go test -race ./...
go vet ./...
python3 tools/postgresgen/generate.py
go run ./tools/mcpgen
go run ./tools/openapigen
gofmt -w packages/wos-core/storage/postgres packages/wos-api/mcp packages/wos-api/commands packages/wos-sdk-go
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
bin/wos version
bin/wos config validate
cd tests/web
npm ci
npx playwright install --with-deps chromium
npm test
```

No ambiente local, o download padrão do Chromium falhou; o mesmo teste foi executado com um binário real alternativo por `WOS_TEST_CHROMIUM_PATH`. Apenas flags necessárias ao processo isolado foram usadas (`--no-sandbox`, `--no-zygote`, `--single-process`, `--disable-dev-shm-usage`). O CI usa o Chromium padrão do Playwright. O mesmo cenário passou remotamente com Chrome for Testing 151.0.7922.34 em Browser acceptance #1 e #2.

Saídas finais preservadas em `docs/audit/`: testes Go, race, vet, navegador e benchmark. Arquivos de saída `ok` do pacote PostgreSQL representam compilação e skips no ambiente sem DSN; essa ressalva é parte da evidência.

## Execução remota após sincronização

Commit `ffc6e7fbe1acde41c3c9b27f87e03cc12a406146`: workflow Browser acceptance #1, run `37380121824`, job `111999646696`, **sucesso**, inclusive instalação do Chromium padrão e jornada completa do teste. Assim, essa fatia foi verificada também no navegador padrão do CI.

CI #231, run `37380121402`, job `111999644305`: containers PostgreSQL 18.6, higiene, formatação, vet e geração passaram; sete testes de reconstrução de avaliações/conclusões falharam em PostgreSQL com `driver: bad connection`. A causa foi consulta filha com resultado pai ainda aberto na mesma conexão transacional. A correção compartilha o fechamento/materialização do resultado entre SQLite e PostgreSQL antes de hidratar Evidence e assessment refs. Esse primeiro run falhou; a correção foi aprovada pelo CI #232 descrito abaixo.


## Checkpoint remoto aprovado e ampliação de recuperação

Commit `62658054adb6c556aab940e1bada4a5416cbf423`: [CI #232](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/37380724897), job `112001706861`, **sucesso em todas as etapas**. PostgreSQL 18.6 real: testes de contrato 7,209 s; detector de corridas 12,798 s. SQLite: 2,481 s e 22,217 s. Higiene, formatação, vet, drift, build, versão, configuração e smokes HTTP também passaram. Excerto original: `audit/postgres-ci-passed.txt`.

[Browser acceptance #2](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/37380724938), job `112001707479`, **sucesso**, jornada 4,7 s, total 6,2 s, Chromium padrão 151.0.7922.34. Excerto: `audit/browser-ci-second-passed.txt`.

A revisão seguinte amplia quatro cenários de runtime (MCP/HTTP compartilhados, revogação/autoria, SDK após restart e sessão web) com schemas PostgreSQL isolados; reutiliza exatamente os mesmos asserts em SQLite/PostgreSQL. O contrato durável de integração passa a restaurar pg_dump em banco vazio antes de continuar, com clientes da própria imagem PostgreSQL, transação única e limpeza do banco de teste. Essa ampliação passou no [CI #233](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/37382265559), commit `5d0829799dad9edbd9a9ca26bd1439a5313909fe`, job `112006923940`, com suíte completa, detector de corridas e todos os smokes. Excerto: `audit/postgres-runtime-restore-ci-passed.txt`. Browser acceptance #3 (`37382265550`) também passou. O restore cobre grants, sessão/credencial parent, auditoria, receipts, histórico e outbox do cenário; não se generaliza para planning, conclusões ou todos os leases.


## Catálogo e consumidor embedded

`python3 tools/eventcatalog/check.py`: passou, 81 tipos públicos v1. O catálogo explicita a projeção sem payload Go interno, estabilidade do envelope, múltiplos fatos por revisão e deduplicação at-least-once.

`go run ./examples/embedded`: passou, `lifecycle=achieved revision=5 certified=true`. `go test ./tests/boundary`: passou; o teste copia esse consumidor para outro módulo Go e executa criação, critério, ativação, avaliação e certificação pelo Core público.

As fronteiras de recuperação dos testes Wave 10 de conclusões/histórico, Wave 11 de revisões imutáveis e leases agora usam backup em arquivo vazio SQLite e, no CI, pg_dump/pg_restore em banco PostgreSQL vazio. Os asserts existentes incluem continuação com um novo Service, preservação dos snapshots de avaliação e conclusão, revisão histórica legível e renovação/reclaim/fencing após restore. A fatia SQLite passou; PostgreSQL desta ampliação aguarda CI.
