# Contratos públicos do WOS 0.1

HTTP e MCP chamam o mesmo Application Service. O cliente Go em `packages/wos-sdk-go` é fino: transporta autenticação, deadlines, versões, fencing, paginação e erros tipados, sem implementar outra máquina de estados e sem retries automáticos.

## Comandos e recuperação

O catálogo é `GET /api/v1/commands`. Cada entrada possui nome público snake_case e JSON Schema. `POST /api/v1/commands/{name}` recebe `{"command": ...}`, com `Idempotency-Key` obrigatório. `commands.openapi.json` é gerado das mesmas 78 inscrições tipadas utilizadas por HTTP/MCP e pelo cliente Go. `openapi.yaml` inclui os endpoints REST e referencia esses schemas. Regeneração: `go run ./tools/mcpgen`, `go run ./tools/openapigen`; variantes SQL: `python3 tools/postgresgen/generate.py`.

MCP usa `wos_{name}` com `{idempotency_key,command,correlation_id?}`. TTL é em segundos no contrato remoto e `time.Duration` no Go embedded. Os argumentos não escolhem Principal nem sobrescrevem Actor autenticado. Administração usa `wos_administer_namespace` / `POST /api/v1/security/commands` sobre `SecurityService`, com Namespace CAS.

Não se usa uma nova chave para recuperar uma resposta incerta. Timeout/desconexão pode ocorrer após commit. Repita a mesma intenção e chave para obter replay; uma chave com outra intenção retorna `idempotency_conflict`. A versão obsoleta exige leitura e reconciliação do usuário; não há incremento automático. `claim_id` e `fencing_token` devem ser mantidos durante renovação/conclusão. Administração de segredos tem recibo sem reexposição do token, conforme [operations.md](operations.md).

Comandos têm máximo de 256 KiB no transport. Um resultado de mutação maior que o limite retorna um recibo com `command_id`, `outcome_revision`, `idempotent_replay` e `result_omitted=true`: é commit realizado, não falha de domínio. Consulte continuidade para expandir o resultado. `transaction_conflict` é transitório; versões/intenções não são alteradas em retries.

## Continuidade e endereço

Descoberta: `GET /namespaces/{namespace_id}/outcomes`, filtros `text`, `lifecycle`, `archived`, `priority`, `creator_principal_id`, `owner_kind/provider/id`, `external_provider/kind/id` e `external_context` (objeto JSON plano). A listagem contém metadados com `detail_omitted=true`, ordenados por `(created_at,id)`; cursor de keyset é live e vinculado ao Namespace/filtros. Mudanças de filtros exigem nova busca. O contexto é endereço de consumidor; `tenant_id`/`user_id` não concedem acesso.

ExternalContext: até 64 chaves e 8 KiB; valores string, boolean, número interoperável ou null, sem objetos/listas. Prefixo `wos.` é reservado. Busca combina pares com AND, preserva tipo e normaliza números JSON equivalentes; números fora do intervalo interoperável de ±(2^53−1) são rejeitados. Índices e agregado são alterados na mesma transação. Referências externas `(provider,kind,external_id)` são únicas dentro do Namespace e podem apontar para URL HTTP(S) reference-only.

`GET .../continuity` retorna `snapshot_schema_version: 1`, `outcome_revision`, `evaluated_at`, `consistency=transactional`, resumo do Outcome, seções, `counts`, `omitted`, `section_cursors`, `truncated`, `section_limit` e `progress`. Limite por seção: 1–100, default 25; resposta máxima 256 KiB. Detalhes documentais e planos completos são omitidos explicitamente. `GET .../continuity/{section}` expande uma seção. Cursores de seção/grafo fixam a revisão e o instante de avaliação; mudança de revisão causa erro explícito e exige novo snapshot. IDs/cursores não são concessões de acesso.

Métricas separam trabalho done, Objectives achieved, Objectives obrigatórios achieved, critérios obrigatórios met com evidências utilizáveis e critérios waived. Denominador zero retorna valor null e motivo. Waiver não é contado como prova met; nenhuma métrica conclui automaticamente um resultado.

`GET .../timeline` é histórico público de metadados imutáveis (`schema_version: 1`), com filtros de Principal, comando, entidade e tipo; paginação append-only por `(outcome_revision,event_index)`. Não retorna payload interno de Domain Event. `GET .../graph` aceita raiz, tipos, direção, profundidade 0–8, limite e cursor; `source_of_truth` distingue hierarquia, dependência e vínculo documental. `GET .../work-items/{id}/context` agrega trabalho/critério, Objective proprietário, dependências, bloqueios, vínculos de evidência e decisões aceitas. Decisions são do Outcome; o domínio atual não contém associação focal exclusiva por WorkItem, portanto essa coleção tem escopo declarado de Outcome e truncamento explícito.

Consultas compactas usam uma leitura coerente de coleções completas para algumas projeções. Objectives/WorkItems carregam base e históricos em conjuntos, sem uma consulta por entidade/assessment/conclusão nessas listagens. Bloqueios e dependências reutilizam as coleções lidas. Descoberta SQL evita hidratação. Limite de resposta não representa limite de custo de leitura ou de histórico; benchmarks e limites estão em `benchmarks.md`.

## Prova e planejamento

Publicação de Roadmap fixa critérios, referências, rótulos e hash. `active_plan_references` é uma seção paginada com rótulo do plano e da referência publicada separados da versão, lifecycle e disponibilidade atuais; essa projeção usa a mesma leitura transacional e nunca reescreve a revisão publicada. Draft/aggregate version, revision_number e outcome_revision são conceitos separados. Slots ativos são substituídos explicitamente e serializados, sem CAS do slot anterior; o contrato preserva a política existente e o histórico de ambas as ativações. Ver ADR 0014.

Conclusões são imutáveis. Critério revisado não herda avaliação anterior; retrair Evidence pode contestar conclusão sem apagar ou reabrir o resultado. Alterações estruturais que o domínio já proíbe em Outcome terminal exigem reopen explícito. Projeção de obrigações alteradas também cobre estado legado/restaurado incompatível. Revisor independente é política configurável por deployment, baseada em Principal e histórico do Outcome.

## MCP e integrações

O catálogo público exaustivo de tipos de fatos v1 está em [integration-events-v1.json](integration-events-v1.json); [integration-events.md](integration-events.md) define semântica, evolução e deduplicação. O CI verifica sua correspondência com o mapeamento Application.

SDK oficial Go v1.8.0; stdio local e Streamable HTTP remoto stateless/JSON. O perfil atual é 2026-07-28; o SDK também negocia 2025-11-25, 2025-06-18 e 2025-03-26. O cliente oficial exercita o perfil atual e a jornada independente/browser exercita 2025-06-18. O teste de catálogo exercita os quatro perfis em ambos os bancos. O perfil atual usa `server/discover` e `_meta` por request, com `MCP-Protocol-Version` e `Mcp-Method`; clientes legados usam `initialize`. Prefira o SDK oficial para construir esses envelopes. Cliente oficial SDK executado em testes reais nos dois transports. Perfil Woobe inspecionado usa `2025-06-18`; compatibilidade de deployment Woobe completo ainda precisa de demonstração. WOS não implementa OAuth nem depende de Woobe. Recursos: `wos://namespaces/{namespace_id}/outcomes/{outcome_id}/continuity`; leituras incluem descoberta, snapshot/seções, grafo, contexto, readiness, timeline, histórico de critérios/conclusões, revisões/slots/ativações de Roadmap e sinais/deliveries.

IntegrationFact `schema_version: 1` contém ID, tipo de fato público, Namespace/Outcome/revisão/índice, entidade, autoria, comando, tempo e correlação. TriggerFiring `schema_version: 1` contém identidade estável, Trigger/versão, signal_type e fato fonte. Não contém nome de struct/comando Go ou payload interno. Predicados usam whitelist de metadados públicos, `eq|neq|in|exists|all|any`, até três níveis e vinte nós; não executam scripts, LLM, Tools ou trabalho. Sinais são entregues pelo menos uma vez; consumidores deduplicam pelo ID. Consulte operações para assinatura e limites.

A revisão semântica e os testes existentes continuam obrigatórios. Um schema gerado comprova a correspondência dos campos, não substitui testes de comportamento nem aceite de PostgreSQL/browser.

O schema distingue metadados opcionais de Evidence (`checksum`, `source_version`) sem alterar a serialização Core usada em fingerprints idempotentes existentes. Severity, propagação de Blocker, modo/resultado de avaliação e tipo de Evidence anunciam os enums efetivos do Domain.

## Execution contracts v1 (2026-10-08)

ADRs 018/019 and the supplied implementation plan govern the additive evolution. Namespace activation is explicit; `contracts_v1` support in capabilities does not mean every existing task is migrated. WorkItem identity and lifecycle remain separate from contract state.

New generated commands: acquire_work_contract, acquire_next_work_contract, renew_work_contract, resume_work_contract, sync_work_contract, submit_work_result, finalize_work_contract, revoke_work_contract. New authority arguments use `authority: {execution_id, fencing_token, spec_digest}` with fencing_token a canonical positive uint64 **string**. TTL is integer seconds, omitted uses policy default. expected_work_item_version, expected_contract_version and expected_lease_version protect separate intent. No release_work_contract exists.

HTTP and MCP use the generated catalogue and same Application service. MCP read tools use namespace_id/outcome_id, entity_id for contract/submission, principal_id for holder filtering. REST contract routes and capabilities/receipt/available-work routes are generated in commands.openapi.json. SDK exposes matching typed methods without automatic mutation retries. Contract history omits specs explicitly; expand immutable spec or current contract separately. Contract records use limit and scope/section-bound cursors; reducing limit resolves an oversized page.

Acquire-next orders priority (critical/high/normal/low), creation time, ID; scans at most limit candidates in one explicit Outcome. A false result has search_complete and next_cursor: incomplete scan is not proof of no eligible task. Repeating a successful empty intent returns the original result, even after new work appears. Use a new key for a new search intention. Available-work is read-only and grants no authority.

Receipt lookup at `/namespaces/{namespace_id}/commands/{command_id}` is restricted to the authenticated Principal and current authorization. Absent/expired retention does not establish that a mutation failed. A large HTTP receipt exposes command/revision with result_omitted; reconcile canonical entities or replay the original intent. Contract effective_status invalidates exact expiry immediately without a sweeper; GET never records expiration.

Specifications are bounded at 128 KiB and operational commands at 256 KiB. Use documentary references rather than embedding an entire Outcome. A digest verifies semantic integrity under RFC8785 and does not authenticate the file holder. Progress/result/review/finalization are delivered in C06, and client/cutover in subsequent waves; this section does not announce their completion.


Contract result mutations use the same transactional command pipeline on HTTP and MCP:
`sync_work_contract`, `submit_work_result`, and `finalize_work_contract`. Sync accepts
at most 100 documentary records and a checkpoint within 256 KiB, returns immutable
server IDs mapped to unique `local_key` values, and commits all records, events and
receipt together. Nested documentary scopes must match the outer scope. Existing
artifact/evidence references are scoped and checked; no external URI is fetched.

Submission requires exact registered artifact source versions/checksums and explicit
supersession of the latest result. Criterion assessments on contract work require
`submission_id`; the server binds the canonical digest and verifies criterion evidence.
Finalization rechecks current eligibility, dependencies, blockers and registered evidence,
then atomically closes the contract and WorkItem with material-bound conclusion. A new
submission invalidates the usefulness of earlier assessments for finalization. Reads
include only the latest checkpoint/submission plus bounded historical pages. Administrative
revocation remains explicit; specification/dependency edits and parent closure cannot
silently bypass a valid contract. Independent reviewer policy includes contract acquisition,
takeover and submission as execution facts.
