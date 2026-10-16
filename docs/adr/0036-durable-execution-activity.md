# ADR-036: Durable execution activity separate from proof and assessment

Status: accepted for implementation in the centralized-agent-workflow branch; exact public contracts and verification are recorded with the implementation.

## Context

Agents need a chronological record of discoveries, changes, checks, and results that survives local cleanup and a change of runtime. Existing signed submissions preserve deliveries, but legacy checkpoint synchronization does not authorize mutations under the signed-v2 protocol. Reusing it as an unsigned substitute would break the authority boundary.

## Decision

Add an append-only execution activity command through the existing Application pipeline. Its exact Scope, current holder, credential, signer, immutable issuer grant, execution, expiry, fencing, and applicable versions are validated using the signed protocol. Activity is persisted with server-assigned identity, ordering, timestamp, actor, contract and WorkItem references. Transactional Outcome ordering and idempotency prevent duplicate logical appends and ambiguous chronology.

An activity may reference explicitly registered Evidence, Artifacts, and current criterion revisions in its Outcome. Registering documentary material does not assess a criterion. Recording an activity does not prove the external action occurred, complete a Task, renew authority, or alter an issued specification. Those remain separate explicit operations.

Reads expose bounded pages with declared Scope, revision, cursor and omissions. Work-level history remains authoritative after contract expiry, completion, or removal of a local file. Read authorization continues to apply; holding an old issuer document does not grant access after revocation. Filtering by contract does not allow escaping the owning WorkItem/Outcome.

Compact attention/context projections reuse current criteria, operational dependencies, blockers, review findings, and recorded material. They describe known obligations without asking the agent to duplicate a manual `pending[]`. A missing assessment is not evidence that material is absent or poor. A bounded projection must declare truncation and never imply global absence from a partial page.

Memory, SQLite and PostgreSQL preserve equivalent semantics. HTTP, MCP and the Go SDK call the same Application service. Additive routes, DTOs, capability declarations, schemas and reviewed compatibility fingerprints evolve together. No transport writes storage directly and no new scheduler, agent runtime, inference service or remote tool execution is introduced.

## Concurrency and recovery

Each local CLI mutation freezes its exact request before transmission and reconciles its original receipt after an uncertain response. New log text, a new idempotency key, changed version or another identity is not an automatic retry. Profile/contract locks protect local material while server authorization, the Outcome guard and durable idempotency remain decisive for shared state.

The issuer acceptance receipt for delivery is distinct from Task completion when independent review is required. Cleanup can close the executor's local obligation while the Task awaits review; history, activity and evidence remain on the server. An auxiliary subagent uses the holder's delegated host workflow and does not become an independent reviewer merely by changing a profile or process name.

## Acceptance

Exercise append/replay/conflict, cross-Namespace denial, revoked/expired/stale authority, concurrent append ordering, pagination/cursor scope, retained history after completion/cleanup, documentary references, unmet/corrected obligations, SQL restore, and HTTP/MCP/SDK parity. Preserve all existing signing, independent-review and exact-intent regression gates.
