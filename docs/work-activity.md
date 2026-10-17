# Work activity and attention

Work activity records what a participant observed while working under a live signed
execution Contract. Attention explains what the current Task still requires. Both
are shared server state, available through HTTP, MCP, and the Go client. Neither
executes work or replaces an independent review.

## Record an observation

`POST /api/v1/commands/append_work_activity` and MCP tool
`wos_append_work_activity` call the same application command. Supply an
`Idempotency-Key` for HTTP or `idempotency_key` for MCP, together with:

- Exact `scope`, `contract_id`, and the credential's enrolled `signer_key_id`.
- Current issued `authority` (`execution_id`, `fencing_token`, `spec_digest`) and
  `authority_digest`.
- `expected_contract_version`, as a quoted decimal string.
- `type`: `progress`, `discovery`, `problem`, `correction`, `validation`, or `result`.
- A nonempty `description` of at most 16 KiB.

The complete immutable activity record, including captured artifact versions, is
limited to 128 KiB so that it can always be read through a bounded history page.
Keep large material in referenced artifacts.

Optional `artifact_refs`, `evidence_refs`, and `criterion_refs` attach scoped records
and frozen criterion revisions. Inline `artifacts`, `evidence`, and `evidence_links`
use the same typed inputs as contract sync, with unique local keys; creation and
the activity record commit together. The response maps those keys to permanent
server IDs. Artifact references retain their exact source versions and checksums.
No artifact URL is fetched by the server.

The credential must still own the signed Contract and hold current work-write
permission. Expired, revoked, superseded, or submitted authority cannot append a
new record. Recording activity never renews a lease, changes a Contract version,
completes a Task, approves a criterion, or modifies an earlier observation.

If the response is uncertain, replay the identical command with its original key.
A committed observation returns its original result, including after the Contract
closes, while the caller retains the required authorization. A new key describes a
new intention and must pass current authority checks.

## Read activity history

```text
GET /api/v1/tenants/{tenant_id}/outcomes/{outcome_id}/work-activity
  ?work_item_id={optional_task_id}&contract_id={optional_contract_id}&limit=25
```

MCP: `wos_list_work_activity` accepts `tenant_id`, `outcome_id`, optional
`work_item_id`, optional `contract_id`, `descending`, `limit`, and `cursor`.

The result contains `items`, optional `next_cursor`, `order`, `outcome_revision`,
and `history_through_revision`. Revisions and each activity's `sequence` are quoted
decimal strings. By default, `order: "asc"` follows chronological Outcome sequence.
Set `descending=true` for `order: "desc"` and newest records first; that cursor
continues toward older records. HTTP accepts only `true` or `false` for this flag.
Activity is immutable, so the first page freezes an upper revision for that
traversal. Subsequent writes appear in a fresh traversal, without shifting the
remaining pages. Cursors bind the Tenant, Outcome, identity filters, and order.

The default page size is 25; supported limits are 1–100. Responses remain bounded
by the shared snapshot byte limit and may contain fewer records. Follow
`next_cursor` until absent. Current Outcome read authorization is checked on each
page; possession of a cursor does not grant access. HTTP responses use
`Cache-Control: no-store`.

## Read remaining obligations

```text
GET /api/v1/tenants/{tenant_id}/outcomes/{outcome_id}/work-items/{work_item_id}/attention
```

MCP: `wos_get_work_attention` accepts `tenant_id`, `outcome_id`, and
`work_item_id`.

Attention is computed from current Task, Contract, readiness, required criteria,
assessments, linked evidence, and review/correction state. It returns `status`,
`execution_allowed`, `awaiting_review`, `correction_required`, `obligations`,
`outcome_revision`, and `evaluated_at`, with explicit omission metadata when
necessary. `evidence_search_complete=false` means missing links cannot be inferred
from that bounded response. An observation labeled `validation` or `result` does
not satisfy a criterion: its current revision still needs an assessment.

This is a read projection, not a second pending list or execution grant. Participants
must still acquire or resume current authority before acting.

## Discover current execution headers

`GET /api/v1/tenants/{tenant_id}/outcomes/{outcome_id}/signed-state/executions`
and `wos_read_signed_state` with `resource: "executions"` expose bounded Contract
headers for rediscovery after local files are removed. Optional filters are
`holder_principal_id`, `work_item_id`, and persisted `status` (`active`, `delivered`,
`completed`, `revoked`, or `expired`), together with `limit` and `cursor`.

Each header carries Contract/Task IDs, holder and credential IDs, persisted and
effective status, expiry, and the latest submission ID when present. Expand one
authorized execution separately to obtain its current metadata. Headers contain
no full specifications and grant no authority. Unlike immutable activity history,
these pages describe current state; expiry may change between reads. Inspect
`search_complete` and `next_cursor` before inferring that discovery is complete.

## Go client

The typed client exposes:

```go
client.AppendWorkActivity(ctx, idempotencyKey, command)
client.ListWorkActivity(ctx, scope, filter, limit, cursor)
client.GetWorkAttention(ctx, scope, workItemID)
```

`filter` is `ports.WorkActivityFilter`; callers select `WorkItemID`, `ContractID`,
and `Descending`. Traversal bounds are server-owned. The client does not retry mutations
automatically. Capabilities advertise `work_activity_version: 1` and
`work_attention_version: 1`; signed protocol setup and permissions remain required
for writes.

Machine-readable requests and read responses are in the generated
[OpenAPI extension](../packages/wos-api/commands.openapi.json). See
[HTTP conventions](http.md), [transport contracts](contracts.md), and the
[signed contract design](signed-contracts-design.md) for the surrounding protocol.
