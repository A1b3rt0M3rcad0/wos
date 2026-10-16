# Public contracts

HTTP and MCP call the same Application Service. The Go client in `packages/wos-sdk-go` transports authentication, deadlines, versions, fencing, pagination, and typed errors. It does not implement another state machine or automatic retries.

## Commands and recovery

The catalog is `GET /api/v1/commands`. Each entry has a public snake_case name and JSON Schema. `POST /api/v1/commands/{name}` accepts `{"command": ...}` and requires `Idempotency-Key`. `commands.openapi.json` is generated from the same typed registrations used by HTTP/MCP and the Go client. `openapi.yaml` includes REST endpoints and references those schemas. Regenerate with `go run ./tools/mcpgen` and `go run ./tools/openapigen`; SQL variants use `python3 tools/postgresgen/generate.py`.

MCP uses `wos_{name}` with `{idempotency_key,command,correlation_id?}`. Remote TTL values use seconds; embedded Go uses `time.Duration`. Arguments do not select the Principal or override the authenticated Actor. Administration uses `wos_administer_namespace` / `POST /api/v1/security/commands` through `SecurityService`, with Namespace CAS.

Recover an uncertain response using the original intent and key: timeout/disconnection can occur after commit. Reusing a key with another intent returns `idempotency_conflict`. A stale version requires a fresh read and user reconciliation; versions do not increase automatically. Retain `claim_id` and `fencing_token` during renewal/completion. Secret administration returns receipts without re-exposing tokens; see [operations](operations.md).

Transport commands are limited to 256 KiB. An oversized mutation result returns a receipt with `command_id`, `outcome_revision`, `idempotent_replay`, and `result_omitted=true`: the commit succeeded. Use continuity to expand the result. `transaction_conflict` is transient; retry does not alter versions or intents.

## Continuity and addressing

Discovery uses `GET /namespaces/{namespace_id}/outcomes`, with `text`, `lifecycle`, `archived`, `priority`, `creator_principal_id`, `owner_kind/provider/id`, `external_provider/kind/id`, and `external_context` (a flat JSON object) filters. Listings contain metadata with `detail_omitted=true`, ordered by `(created_at,id)`. Live keyset cursors bind the Namespace and filters; changed filters require a new search. External context addresses consumer data; `tenant_id`/`user_id` do not grant access.

ExternalContext allows up to 64 keys and 8 KiB, with string, boolean, interoperable numeric, or null values and no nested objects/lists. The `wos.` prefix is reserved. Search combines pairs with AND, preserves types, and normalizes equivalent JSON numbers; numbers outside the interoperable ±(2^53−1) range are rejected. Indexes and aggregates change in the same transaction. External references `(provider,kind,external_id)` are unique within a Namespace and may contain reference-only HTTP(S) URLs.

`GET .../continuity` returns `snapshot_schema_version: 1`, `outcome_revision`, `evaluated_at`, `consistency=transactional`, Outcome metadata, sections, `counts`, `omitted`, `section_cursors`, `truncated`, `section_limit`, and `progress`. Each section allows 1–100 items, default 25; maximum response size is 256 KiB. Documentary detail and complete plans are explicitly omitted. `GET .../continuity/{section}` expands a section. Section/graph cursors pin the revision and evaluation time; changed revisions cause an explicit error and require a new snapshot. IDs and cursors do not grant access.

Metrics distinguish completed work, achieved Objectives, achieved required Objectives, required criteria met with usable evidence, and waived criteria. A zero denominator returns null with a reason. Waiver is not verified proof; metrics never conclude results automatically.

`GET .../timeline` exposes immutable public metadata history (`schema_version: 1`) with Principal, command, entity, and type filters, paginated append-only by `(outcome_revision,event_index)`. It does not expose internal Domain Event payloads. `GET .../graph` accepts a root, kinds, direction, depth 0–8, limit, and cursor; `source_of_truth` distinguishes hierarchy, dependency, and documentary links. `GET .../work-items/{id}/context` combines work/criterion context, owning Objective, dependencies, blockers, evidence links, and accepted decisions. Decisions belong to the Outcome; there is no exclusive focal WorkItem association, so that collection declares Outcome scope and explicit truncation.

Some compact projections coherently read complete collections. Objective/WorkItem listings load base state and histories in batches, rather than one query per entity/assessment/conclusion. Blockers and dependencies reuse the loaded collections. SQL discovery avoids hydration. Response bounds do not imply bounded read/history cost; see [measurements](benchmarks.md).

## Proof and planning

Roadmap publication freezes criteria, references, labels, and the hash. `active_plan_references` is paginated and distinguishes published plan/reference labels from current versions, lifecycles, and readiness. It uses the same transactional read and never rewrites a published revision. Draft/aggregate version, revision_number, and outcome_revision are distinct concepts. Active slots are explicitly replaced and serialized without CAS on the previous slot, preserving both activation histories; see ADR-0014.

Conclusions are immutable. Revised criteria do not inherit earlier assessments; retracting Evidence can contest a conclusion without deleting or reopening it. Structural changes forbidden by the domain on a terminal Outcome require explicit reopen. Changed-obligation projections also cover incompatible legacy/restored state. Independent review is a deployment policy based on Principal and Outcome history.

## MCP and integrations

The exhaustive public v1 fact catalog is [integration-events-v1.json](integration-events-v1.json). [Integration events](integration-events.md) defines semantics, evolution, and deduplication. CI checks its correspondence with the Application mapper.

The official Go SDK is v1.8.0, with local stdio and remote stateless/JSON Streamable HTTP. The current profile is 2026-07-28; the SDK also negotiates 2025-11-25, 2025-06-18, and 2025-03-26. The official client exercises the current profile; the independent/browser journey exercises 2025-06-18. Catalog tests exercise all four profiles against both databases. The current profile uses `server/discover` and per-request `_meta`, with `MCP-Protocol-Version` and `Mcp-Method`; legacy clients use `initialize`. Prefer the official SDK to construct these envelopes. Real client tests cover both transports. The inspected Woobe profile uses 2025-06-18; complete deployed-consumer acceptance remains separate. OAuth is not implemented, and WOS does not depend on Woobe. Resources include `wos://namespaces/{namespace_id}/outcomes/{outcome_id}/continuity`; reads cover discovery, snapshots/sections, graphs, context, readiness, timelines, criterion/conclusion history, Roadmap revisions/slots/activations, and signals/deliveries.

IntegrationFact `schema_version: 1` contains the ID, public fact type, Namespace/Outcome/revision/index, entity, authorship, command, time, and correlation. TriggerFiring `schema_version: 1` contains stable identity, Trigger/version, signal_type, and source fact. Neither exposes internal Go struct/command names or payloads. Predicates use allowlisted public metadata, `eq|neq|in|exists|all|any`, up to three levels and twenty nodes. They do not execute scripts, LLMs, tools, or work. Signals have at-least-once delivery; consumers deduplicate by ID. See operations for signatures and limits.

Semantic review and existing tests remain required. A generated schema establishes field correspondence; it does not replace behavior tests or PostgreSQL/browser acceptance.

The schema distinguishes optional Evidence metadata (`checksum`, `source_version`) without changing the Core serialization used by existing idempotency fingerprints. Severity, Blocker propagation, assessment mode/result, and Evidence kind advertise the actual Domain enums.

## Execution contracts v1 (2026-10-08)

ADRs 018/019 and the supplied implementation plan govern the additive evolution. Namespace activation is explicit; `contracts_v1` support in capabilities does not mean every existing task is migrated. WorkItem identity and lifecycle remain separate from contract state.

New generated commands: acquire_work_contract, acquire_next_work_contract, renew_work_contract, resume_work_contract, sync_work_contract, submit_work_result, finalize_work_contract, revoke_work_contract. New authority arguments use `authority: {execution_id, fencing_token, spec_digest}` with fencing_token a canonical positive uint64 **string**. TTL is integer seconds, omitted uses policy default. expected_work_item_version, expected_contract_version and expected_lease_version protect separate intent. No release_work_contract exists.

HTTP and MCP use the generated catalogue and same Application service. MCP read tools use namespace_id/outcome_id, entity_id for contract/submission, principal_id for holder filtering. REST contract routes and capabilities/receipt/available-work routes are generated in commands.openapi.json. SDK exposes matching typed methods without automatic mutation retries. Contract history omits specs explicitly; expand immutable spec or current contract separately. Contract records use limit and scope/section-bound cursors; reducing limit resolves an oversized page.

Acquire-next orders priority (critical/high/normal/low), creation time, ID; scans at most limit candidates in one explicit Outcome. A false result has search_complete and next_cursor: incomplete scan is not proof of no eligible task. Repeating a successful empty intent returns the original result, even after new work appears. Use a new key for a new search intention. Available-work is read-only and grants no authority.

Receipt lookup at `/namespaces/{namespace_id}/commands/{command_id}` is restricted to the authenticated Principal and current authorization. Absent/expired retention does not establish that a mutation failed. A large HTTP receipt exposes command/revision with result_omitted; reconcile canonical entities or replay the original intent. Contract effective_status invalidates exact expiry immediately without a sweeper; GET never records expiration.

Specifications are bounded at 128 KiB and operational commands at 256 KiB. Use documentary references rather than embedding an entire Outcome. A digest verifies semantic integrity under RFC8785 and does not authenticate the file holder. Progress, immutable submission, exact review binding, finalization, API-only client and explicit Namespace cutover are implemented. Actual Windows and registry publication remain separate gates.

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

Namespace phase/policy is readable through `GET /namespaces/{namespace_id}/work-protocol` and `wos_get_namespace_work_protocol`. `set_namespace_work_protocol` and `reconcile_expired_work_contracts` require Namespace administration. ADR-020 defines guard ordering, auditing, old-writer drain acknowledgement and no downgrade. `get_work_context` retains current proof and declares omitted histories. Oversized latest progress/material expands through work-checkpoints/{id} or work-submissions/{id}; MCP has matching individual tools.
