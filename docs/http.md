# WOS HTTP — Waves 05–08

Waves 05–08 expose the M1 domain plus dependency/readiness, Issue/Blocker coordination and hardened WorkItem lease/fencing coordination through the standalone WOS process. The HTTP transport is an adapter over the existing Application services; it does not duplicate lifecycle, criteria, dependency, blocking, readiness, lease, idempotency or concurrency rules.

## Start the local server

The local profile uses SQLite, local authentication and loopback binding by
default:

```bash
go run ./packages/wos-api/cmd/wos server
```

Default runtime values:

```text
listen:             127.0.0.1:8080
API prefix:         /api/v1
SQLite file:        ./data/wos.db
local principal:    local-user
request deadline:   15s
migrate on start:   true
```

Supported environment overrides:

```bash
export WOS_LISTEN=127.0.0.1:8080
export WOS_SQLITE_PATH=./data/wos.db
export WOS_LOCAL_PRINCIPAL_ID=local-user
# Optional and disabled by default:
export WOS_LOCAL_ADMIN_OVERRIDES=true
go run ./packages/wos-api/cmd/wos server
```

MCP is disabled in the standalone default until its own implementation wave.

## Health

```bash
curl -i http://127.0.0.1:8080/livez
curl -i http://127.0.0.1:8080/readyz
```

`/livez` checks process liveness. `/readyz` checks that the SQLite storage
can report a compatible migrated schema.

## Addressing

Every domain route carries the Namespace explicitly:

```text
/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}
```

The server never derives a Namespace from external metadata.

All WOS IDs in this contract are normalized UUIDv7 values.

## Mutation contract

Every HTTP mutation requires an `Idempotency-Key` between 16 and 128 safe
ASCII characters. Retries of the same logical command must reuse the same key.

Versioned mutations also require a persisted aggregate version. Supply it
through either:

```http
If-Match: "work_item:<uuid>:v5"
```

or a JSON `expected_version` field. When both are supplied, they must match.

Successful individual aggregate responses return a strong ETag:

```http
ETag: "outcome:<uuid>:v2"
```

`X-Correlation-ID` is optional. WOS preserves a supplied value or creates one
for the response.

## Reproducible local flow

Set a UUIDv7 Namespace first:

```bash
NS=0199e100-0000-7000-8000-000000000001
BASE=http://127.0.0.1:8080/api/v1/namespaces/$NS
```

Create an Outcome:

```bash
curl -i -X POST "$BASE/outcomes" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-create-outcome-0001' \
  -d '{
    "title": "Validar a fatia HTTP do WOS",
    "desired_state": "Estado M1 pode ser criado e retomado via HTTP",
    "description": "Fluxo local sem dependência de Agent",
    "priority": "normal"
  }'
```

The response is `201 Created`, includes `Location`, a strong `ETag`, the
created entity, `command_id` and `outcome_revision`.

Before activation, the current domain contract requires at least one required
active criterion. Using the Outcome ETag returned above:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/criteria" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-add-criterion-0001' \
  -H "If-Match: $OUTCOME_ETAG" \
  -d '{
    "title": "Validação humana",
    "required": true,
    "verification_mode": "attestation"
  }'
```

The criterion mutation returns the new ETag of its owner aggregate. Use that
ETag to activate the Outcome:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/actions/activate" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-activate-outcome-0001' \
  -H "If-Match: $OUTCOME_ETAG" \
  -d '{}'
```

Create a WorkItem directly under the Outcome:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-create-work-0001' \
  -d '{
    "title": "Executar prova HTTP",
    "priority": "normal",
    "lifecycle": "todo"
  }'
```

Claim it with the WorkItem ETag:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/claim" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-claim-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{"lease_ttl_seconds":300}'
```

Renew the live lease before it expires. Renewal preserves both the claim ID and fencing token while extending `expires_at` from the evaluation time:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/renew-lease" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-renew-work-lease-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{
    "claim_id": "<claim-id>",
    "fencing_token": 1,
    "lease_ttl_seconds": 300
  }'
```

Operational state is time-derived and therefore returned with `Cache-Control: no-store`:

```bash
curl -i "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/operational-state"
```

An expired lease keeps the persisted WorkItem in `in_progress`, but the projection reports `lease_status=expired` and `display_state=attention_needed`. It can then be reclaimed explicitly:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/reclaim" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-reclaim-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{"lease_ttl_seconds":300}'
```

A successful reclaim returns a new claim ID and a strictly higher fencing token. The previous claimant cannot renew, release or complete using the old claim/fencing pair.

Complete it using the returned claim ID, fencing token and newest WorkItem
ETag:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/complete" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-complete-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{
    "claim_id": "<claim-id>",
    "fencing_token": 1,
    "result_summary": "Prova concluída",
    "reason": "Execução confirmada pelo consumidor HTTP"
  }'
```

Create two `todo` WorkItems and register a hard dependency from the dependent
item to its prerequisite:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/relations" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-add-dependency-0001' \
  -d '{
    "source_ref": {"kind": "work_item", "id": "<dependent-work-id>"},
    "relation_type": "depends_on",
    "target_ref": {"kind": "work_item", "id": "<prerequisite-work-id>"},
    "strength": "hard",
    "satisfaction": "target_completed",
    "reason": "O trabalho dependente exige o pré-requisito"
  }'
```

The dependency is Outcome-local, versioned and audited. `hard` dependencies
participate in readiness and completion gates; `advisory` dependencies remain
visible but do not block execution. Adding an edge that would create a direct
or indirect cycle returns `dependency_cycle`.

Query the deterministic ready-work projection:

```bash
curl -i "$BASE/outcomes/$OUTCOME_ID/ready-work"
```

The response uses `Cache-Control: no-store`, includes one `evaluated_at`
instant and the current `outcome_revision`. A `todo` WorkItem is omitted
while a hard dependency is unsatisfied or while `not_before` is in the
future.

Relations can be listed or read directly:

```bash
curl -i "$BASE/outcomes/$OUTCOME_ID/relations"
curl -i "$BASE/outcomes/$OUTCOME_ID/relations/$RELATION_ID"
```

Remove a dependency with its strong ETag and an audit reason:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/relations/$RELATION_ID/actions/remove" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-remove-dependency-0001' \
  -H "If-Match: $RELATION_ETAG" \
  -d '{"reason":"O pré-requisito não é mais necessário"}'
```

Read the current state projection:

```bash
curl -i "$BASE/outcomes/$OUTCOME_ID/state"
```

The state response is `Cache-Control: no-store` and contains the persisted
Outcome, Objectives, WorkItems, Relations, Issues and Blockers plus derived
`blocking_states`, `work_item_operational_states`, one `evaluated_at` instant
and the current `outcome_revision`. Lease expiration can therefore change the
projection without changing persisted lifecycle or `outcome_revision`.


## Administrative WorkItem overrides

Administrative override is intentionally separate from normal lease ownership.
The Application layer calls an explicit `Authorizer` port and defaults to
deny for privileged operations. The two Wave 08 permissions are:

```text
work:admin_cancel
work:admin_complete
```

In the standalone local profile these permissions remain disabled unless
`WOS_LOCAL_ADMIN_OVERRIDES=true` is set. This is an explicit local-development
opt-in; the Namespace grant/token model remains Wave 15.

Normal `CancelWorkItem` no longer cancels an `in_progress` leased WorkItem.
An operator with `work:admin_cancel` may use:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/admin-cancel" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-admin-cancel-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{"reason":"Operador encerrou execução abandonada"}'
```

Administrative completion is restricted to `in_progress` leased work. It
overrides lease ownership/expiry only; it still enforces active Blockers, hard
dependencies, SuccessCriteria and result requirements:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/admin-complete" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-admin-complete-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{
    "result_summary":"Resultado externo verificado pelo operador",
    "reason":"Executor original ficou indisponível após produzir o resultado"
  }'
```

Both operations are idempotent, version-checked and produce Outcome-scoped
Domain Events (`work_item.admin_cancelled` or `work_item.admin_completed`)
containing the authenticated principal, declared actor, command ID and audit
reason in the command payload.

## Issues and Blockers

Issues and Blockers are independent aggregates. An Issue records a problem or
observation; it does not block work by itself. A Blocker records an active
impediment against exactly one Outcome, Objective or WorkItem. Outcome and
Objective Blockers may propagate through their subtree; WorkItem Blockers are
always direct.

Create an Issue and its first Blocker atomically:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/issues/actions/report-with-blocker" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-report-issue-blocker-0001' \
  -d '{
    "issue": {
      "title": "Dependência externa indisponível",
      "severity": "major",
      "affected_refs": [{"kind": "work_item", "id": "<work-id>"}]
    },
    "blocker": {
      "blocked_ref": {"kind": "work_item", "id": "<work-id>"},
      "description": "O trabalho não pode prosseguir"
    }
  }'
```

This is one idempotent command and one `outcome_revision`, but it produces
separate Issue and Blocker aggregates and separate Domain Events. Retrying with
the same key returns the original pair.

The normal endpoints are:

```text
GET|POST  /outcomes/{outcome_id}/issues
GET|PATCH /outcomes/{outcome_id}/issues/{issue_id}
POST      /outcomes/{outcome_id}/issues/{issue_id}/actions/investigate
POST      /outcomes/{outcome_id}/issues/{issue_id}/actions/resolve
POST      /outcomes/{outcome_id}/issues/{issue_id}/actions/reopen
POST      /outcomes/{outcome_id}/issues/{issue_id}/actions/wont-fix
POST      /outcomes/{outcome_id}/issues/{issue_id}/actions/mark-duplicate

GET|POST  /outcomes/{outcome_id}/blockers
GET|PATCH /outcomes/{outcome_id}/blockers/{blocker_id}
POST      /outcomes/{outcome_id}/blockers/{blocker_id}/actions/resolve
POST      /outcomes/{outcome_id}/blockers/{blocker_id}/actions/cancel
```

Resolving an Issue alone intentionally leaves every Blocker active. To resolve
an Issue and selected Blockers in one transaction, the caller must enumerate
each Blocker and confirm its release explicitly:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/issues/$ISSUE_ID/actions/resolve-with-blockers" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-resolve-issue-blockers-0001' \
  -d '{
    "expected_issue_version": 3,
    "issue_resolution_summary": "Causa corrigida e verificada",
    "blockers": [{
      "blocker_id": "<blocker-id>",
      "expected_version": 2,
      "resolution_summary": "Liberação confirmada",
      "release_confirmed": true
    }]
  }'
```

The command accepts between 1 and 64 explicit Blockers. Every Blocker must have
that Issue as its persisted `cause_ref`; stale versions, a missing release
confirmation or a non-matching cause abort the whole transaction. Unlisted
Blockers are never inferred or silently resolved.

A Blocker may instead point to a persisted Objective or WorkItem cause, or to
an explicit external cause:

```json
{
  "blocked_ref": {"kind": "work_item", "id": "<work-id>"},
  "external_cause": {
    "provider": "vendor",
    "id": "incident-42",
    "description": "Vendor outage"
  },
  "description": "Waiting for recovery"
}
```

A persisted `Decision` may also be used as a Blocker cause. The reference is
validated in the same Namespace/Outcome scope before the Blocker is committed.

`blocking_states` explains whether each Outcome, Objective and WorkItem is
blocked, which active Blockers apply and whether each one is direct or inherited.
The ready-work query applies the same projection, so blocked WorkItems are not
returned as ready.


## Documentary records and Decisions

Wave 09 adds durable documentary state without turning WOS into a blob store or
a truth oracle. Artifact URIs are stored as opaque references; WOS does not
dereference them. Evidence preserves the observation and provenance supplied by
the caller, while `EvidenceLink` records how that Evidence relates to a specific
target.

The HTTP resources are:

```text
GET|POST /outcomes/{outcome_id}/artifacts
GET       /outcomes/{outcome_id}/artifacts/{artifact_id}
POST      /outcomes/{outcome_id}/artifacts/{artifact_id}/actions/withdraw

GET|POST /outcomes/{outcome_id}/evidence
GET       /outcomes/{outcome_id}/evidence/{evidence_id}
POST      /outcomes/{outcome_id}/evidence/{evidence_id}/actions/retract

GET|POST /outcomes/{outcome_id}/evidence-links
GET       /outcomes/{outcome_id}/evidence-links/{evidence_link_id}
POST      /outcomes/{outcome_id}/evidence-links/{evidence_link_id}/actions/retract

GET|POST  /outcomes/{outcome_id}/decisions
GET|PATCH /outcomes/{outcome_id}/decisions/{decision_id}
POST      /outcomes/{outcome_id}/decisions/{decision_id}/actions/accept
POST      /outcomes/{outcome_id}/decisions/{decision_id}/actions/reject
POST      /outcomes/{outcome_id}/decisions/{decision_id}/actions/supersede
```

Registering an Artifact records documentary metadata and an external URI only.
WOS does not fetch the URI or store arbitrary Artifact bytes. Withdrawal requires
an explicit reason and preserves the original metadata.

Evidence content is immutable after registration. Retraction changes lifecycle
and records the reason while preserving the original observation, source,
producer, timestamp, checksum and optional Artifact reference.

The Evidence stance belongs to `EvidenceLink`, not to Evidence itself. The same
Evidence can therefore support one target and contradict another. Supplying a
`criterion_id` identifies which SuccessCriterion the link concerns, but creating
the link never creates a CriterionAssessment and never marks the criterion as
met.

A Decision starts as `proposed`. Its editable content may be patched only while
it remains proposed. Once accepted or rejected, the documentary content is
immutable. Rejection requires an explicit reason.

Supersession is a dedicated Decision operation:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/decisions/$DECISION_ID/actions/supersede" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-wave09-decision-supersede-0001' \
  -H "If-Match: $DECISION_ETAG" \
  -d '{
    "title":"Storage v2",
    "proposal":"Use PostgreSQL",
    "alternatives":["SQLite","PostgreSQL"],
    "chosen_alternative":"PostgreSQL",
    "rationale":"Remote multi-user operation requires shared durable storage"
  }'
```

The command atomically creates the accepted successor and marks the accepted
predecessor `superseded`. A predecessor may have at most one accepted direct
successor. A stale version or conflicting supersession aborts the transaction.

All Wave 09 remote mutations require `Idempotency-Key`. Versioned lifecycle
transitions and Decision edits use the same strong ETag / `If-Match` contract as
the other WOS aggregates.

`GET /outcomes/{outcome_id}/state` is the canonical Outcome-scoped continuity
projection. In addition to Outcome/Objectives/WorkItems/Relations/Issues/Blockers,
it includes `artifacts`, `evidence`, `evidence_links` and `decisions`.
Consumers can therefore reconstruct the current documentary context and Decision
history from one WOS state read without depending on prior chat/session history.

## Verifiable assessments and Conclusions

Wave 10 separates immutable validation history from mutable current bindings.

Record a criterion assessment with the owner's strong ETag:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/criteria/$CRITERION_ID/assessments" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-wave10-assessment-0001' \
  -H "If-Match: $OUTCOME_ETAG" \
  -d '{
    "criterion_revision": 2,
    "result": "met",
    "rationale": "External benchmark satisfied the criterion",
    "evidence_ids": ["<evidence-id>"]
  }'
```

`evidence_review` requires at least one active Evidence from the same Outcome.
`external_evaluation` requires an `evaluator_ref` containing provider, ID and
version. `waived` stays visibly waived and requires the privileged
`assessment:waive` authorization; it is never rewritten as `met`.

The current assessment is only a binding. Every assessment and every semantic
criterion revision remains in immutable history. Read one criterion's complete
validation history with:

```text
GET /outcomes/{outcome_id}/criteria/{criterion_id}/history
GET /outcomes/{outcome_id}/objectives/{objective_id}/criteria/{criterion_id}/history
GET /outcomes/{outcome_id}/work-items/{work_item_id}/criteria/{criterion_id}/history
```

A positive terminal transition records a Conclusion with its own public UUIDv7,
the owner/version and lifecycle result, the exact assessment IDs used, criterion
revision obligations and structural obligations such as required Objectives.
Reopening removes only the current binding and preserves that Conclusion in
history.

Conclusions are directly addressable:

```text
GET /outcomes/{outcome_id}/conclusions
GET /outcomes/{outcome_id}/conclusions/{conclusion_id}
GET /outcomes/{outcome_id}/objectives/{objective_id}/conclusions
GET /outcomes/{outcome_id}/work-items/{work_item_id}/conclusions
```

Later contradictory assessments or retracted Evidence do not mutate terminal
lifecycle. `GET /outcomes/{outcome_id}/state` exposes
`conclusion_contested` plus concrete `conclusion_contestations`, preserving
the historical claim separately from the current contradictory facts.

## Errors

Errors use a stable envelope:

```json
{
  "error": {
    "code": "version_conflict",
    "message": "version_conflict: expected version 3 but current version is 4",
    "retryable": false,
    "correlation_id": "request-42"
  }
}
```

Current mappings are:

| HTTP | Meaning |
| --- | --- |
| 400 | malformed JSON, invalid identifiers/format or invalid idempotency key |
| 403 | authenticated principal lacks a required privileged permission |
| 404 | entity absent from the explicit Namespace/Outcome scope |
| 409 | lifecycle, dependency cycle/graph limit, precondition, lease or idempotency conflict |
| 412 | stale or mismatched `If-Match` / `expected_version` |
| 413 | request body exceeds the supported limit |
| 422 | syntactically valid request violates a domain field constraint |
| 428 | required aggregate version precondition is absent |
| 503 | request deadline or transient infrastructure timeout |
| 408 | caller/request cancellation |

Local auth intentionally uses the configured local Principal and does not
pretend to provide multi-user authentication. Token/OAuth authorization remains
a later wave.

## OpenAPI

The initial machine-readable contract is in
[`packages/wos-api/openapi.yaml`](../packages/wos-api/openapi.yaml).

The canonical architecture remains
[`docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`](./WOS_Design_Arquitetura_Planejamento_Atualizado.md).
