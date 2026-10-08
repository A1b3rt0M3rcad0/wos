# Verificação dos contratos e distribuição 0.2.0 — 2026-10-08

Implementação C01–C12 integrada em PRs separados. Source verificado: `2ebe93fa18cbc2ec3d9570fd2725cec5ff9d3ddf` (C12). O commit seguinte acrescenta este relatório e atualiza estados do ROADMAP, sem alterar implementação. O [plano original](work-contract-implementation-plan.md) permanece integral; seus T01–T88 são requisitos, não uma contagem de testes aprovados.

## Evidência executada

| Verificação | Resultado e escopo |
| --- | --- |
| `go test -race -json ./...` | 509 casos/subcasos aprovados, zero falhas; SQLite e PostgreSQL 18.6 reais, incluindo restore via pg_dump e dataset SQLite produzido pelo source v0.1.0. Dois testes opt-in de carga omitidos nesta execução e executados separadamente abaixo. |
| `go vet ./...`, módulos e geração | Aprovados; `go mod tidy`/verify, PostgreSQL/MCP/OpenAPI/schemas YAML regenerados e formatados sem alterações; catálogo de 90 tipos públicos de eventos verificado; actionlint aprovado. |
| Chromium real | Quatro jornadas aprovadas: contrato/cutover/checkpoint/revisão/revogação; planejamento/agente/humano/restart; contestação/bloqueios/conflitos; resumo/lista/quadro/paginação/mobile. |
| Node | 16 testes aprovados: cinco skills, instalador, integridade, plataformas, preparação/publicador e recuperação de publicação parcial simulada. Simulação não comprova publicação externa. |
| Pacotes reais | Cinco artefatos construídos; instalação npm offline, versões/source, cliente autenticado init/checkout/status/recover, 20 cópias das skills em quatro destinos, HTTP/MCP/UI, shutdown e rejeição de binários corrompidos aprovados. |
| Reprodutibilidade | Artefatos nativos e npm idênticos byte a byte em dois diretórios de source. |
| Linux nativo | Recibo real dos testes race CLI/SDK/aceitação, skills e execução do cliente; SHA-256 do binário `3ca840c495c817b04d62860c950c0d7ae06111b448bdc912f03edcc81f06aab7`, igual ao manifesto daquele source. |
| Docker | Imagem real com source/version corretos, wosctl executável, usuário 10001, filesystem read-only, configuração, health, catálogo autenticado, negociação MCP e CA/notices aprovados. Build exigiu resolver o proxy do sandbox via `--add-host`; nenhuma alteração da aplicação para contornar rede. |
| Carga | [54 combinações completas](work-contract-load-evidence.md), source `6179387b1f95cc0ef3d4e060054a4a5e97e909f4`. SQLite/PostgreSQL, 100/1.000/10.000 tarefas, 2/10/50 consumidores, checkpoint amostrado 0/10/100. Relatório inclui conflitos/retries; fixture local não é SLA de produção. |

## O que foi entregue

Contratos persistentes com exclusividade, fencing decimal exato, TTL e CAS separados; encerramento somente por expiração/revogação/conclusão. Spec imutável, checkpoints, sync documental atômico, submissões imutáveis e revisão vinculada ao material exato. HTTP/MCP/SDK, queries focadas e expansão explícita de progresso/histórico. Cliente API-only com YAML restrito, journal, replay, recover, locks e keepalive foreground. Interface humana, cinco skills portáveis, migração explícita e release coordenada 0.2.0.

O serviço não executa agentes, LLMs, ferramentas ou schedulers. Skills não criam isolamento de contexto por si mesmas: a execução e a delegação dependem do host. Namespaces existentes continuam legacy até operação explícita de manutenção; retire todos os writers antigos antes de confirmar `contracts_v1`.

## Gates externos pendentes

- **Windows nativo:** ZIP cross-compilado e testes de filesystem/junction preparados; execução Windows não disponível aqui. Publicador exige recibos Linux e Windows do mesmo source e dos mesmos hashes antes da publicação. Cross-compilação não substitui esse gate.
- **CI hospedado:** GitHub Actions bloqueado antes dos steps por cobrança/limite de gastos da conta. Verificação local não é CI remoto verde.
- **Publicação:** npm exige identidade autorizada; release/GHCR/assets não publicados nesta execução. 0.2.0 está preparada, sem inferir tag ou release de um merge.
- **Woobe e outros runtimes:** aceitação independente HTTP/MCP/CLI aprovada; piloto real Woobe e execução em Claude/Codex/Hermes/OpenClaw não são certificados pela instalação das skills.

## Melhorias prioritárias

Habilitar Windows CI e publicação autorizada; executar piloto Woobe. Medir carga em Outcomes distribuídos e histórias densas, pois o fixture concentra checkpoint em uma tarefa. PostgreSQL serializable apresentou contenção relevante no mesmo Outcome; manter guards de correção e medir antes de reduzir granularidade. Expandir índices/leituras focadas para históricos e relacionamentos volumosos. Preservar retries limitados com a mesma intenção, chave, CAS e autoridade.
