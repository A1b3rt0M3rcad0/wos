# Verificação da implementação — 2026-10-05

Escopo: branch do PR #14, baseada no PR #13; Go 1.27.1, Linux amd64. Resultados de execução local são separados dos gates de integração e release. PostgreSQL não estava disponível localmente: testes que exigem `WOS_TEST_POSTGRES_DSN` são ignorados explicitamente, não aprovados. O CI configura PostgreSQL 18 com essa variável obrigatória; os runs anteriores foram cancelados sem etapas.

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

PostgreSQL real e matriz de bancos/transports; jornada visual completa e acessibilidade; restore com todas as coleções; desligamento com requisições/entregas em voo; otimização e contagem de queries; todos os exemplos de produto/Woobe; matriz de plataformas/containers; comportamento de todas as rotas OpenAPI; demonstração integral da auditoria. Jobs configurados, testes existentes e código compilado não substituem execuções ausentes.

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

No ambiente local, o download padrão do Chromium falhou; o mesmo teste foi executado com um binário real alternativo por `WOS_TEST_CHROMIUM_PATH`. Apenas flags necessárias ao processo isolado foram usadas (`--no-sandbox`, `--no-zygote`, `--single-process`, `--disable-dev-shm-usage`). O CI usa o Chromium padrão do Playwright. Os dois modos não são presumidos idênticos sem execução remota.

Saídas finais preservadas em `docs/audit/`: testes Go, race, vet, navegador e benchmark. Arquivos de saída `ok` do pacote PostgreSQL representam compilação e skips no ambiente sem DSN; essa ressalva é parte da evidência.
