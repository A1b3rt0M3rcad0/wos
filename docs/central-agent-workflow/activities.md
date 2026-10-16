# Durable chronological activity history

## Record semantics

An activity is a participant's statement about execution: a change made, problem observed, correction, test attempt, integration step or result. It is not an automatically verified event and not a criterion assessment. Preserve incorrect/failed attempts and append a correction rather than rewriting them.

The public `AppendWorkActivityCommand` carries stable activity identity and scope; the persisted record contains stable activity ID, Namespace/Outcome/WorkItem, originating contract/execution when applicable, authenticated Principal and authorized ActorRef, type, description, server-recorded timestamp/order, and explicit evidence/artifact/activity references. Types are `progress`, `discovery`, `problem`, `correction`, `validation` and `result`; descriptions are at most 16 KiB. A complete immutable activity is capped at 128 KiB so that one record fits a bounded page. Optional exact ArtifactRefs, EvidenceRefs and CriterionRefs bind existing records; inline Artifacts/Evidence/EvidenceLinks reuse documentary registration and return canonical IDs. Activity declarations never approve a criterion.

```sh
wos work log add --type progress --description "Implemented the endpoint"
wos work log add --type problem --description "Observed an expired-token failure"
wos work log add --type correction --description "Corrected expiry handling"
wos work history --limit 10 --output json
```

Each type in runnable documentation must be one accepted by the actual CLI. Activity claims must describe checks actually run; WOS cannot establish the truth of an external claim.

## Existing components to extend

- `packages/wos-core/domain/work_activity.go`: typed activity and input validation. `event.go` remains the immutable domain event envelope; an activity declaration is not verified external truth.
- `packages/wos-core/application/work_contract_service.go`, `signed_queries.go`, `signed_return.go`: current scope/authority and result boundaries.
- `packages/wos-core/application/work_activity.go` and `packages/wos-core/ports/work_activity.go`: append/query contract and bounded scoped reads.
- `packages/wos-core/storage/{memory,sqlite,postgres}/work_activity.go` and adapter migration `0030_work_activity.sql`: immutable `work_activity` table and indexed ordering; durable signed operation records prevent replay duplication after command-cache expiry.
- `packages/wos-api/http/work_contracts.go`, `signed_queries.go`; `packages/wos-api/mcp/signed_queries.go`; `packages/wos-sdk-go/signed_queries.go`.

## Implementation sequence

1. Add the typed append response and scoped bounded history query over the immutable `work_activity` table. Retain Domain Events for mutation facts and use the dedicated table for efficient durable execution-history reads. The ADR records this representation.
2. Validate type, nonempty bounded description and references. Derive identity from authentication; do not trust supplied owner names or arbitrary cross-Namespace references.
3. Require signed-v2 live-holder authority, the same credential/registered signing key, and exact AuthorityDigest, ExecutionID, fencing, spec and version. Do not imply a completed contract grants perpetual mutation rights.
4. Persist the activity, domain event and idempotent result in the same UnitOfWork under existing Outcome coordination. Replay returns the original ID/order.
5. Query by authorized WorkItem/contract or authorized Outcome-wide coordination scope, ordered by durable Outcome-revision sequence. First page freezes ThroughRevision; cursors bind scope/filter/revision. Limits are 1–100 (default 25) and pages have a 256 KiB byte cap. Server-recorded time and revision sequence establish chronology; do not invent a user-supplied occurrence-time field.
6. Expose HTTP `GET /api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-activity` with optional work_item_id/contract_id and limit/cursor, MCP `wos_list_work_activity`, and SDK `ListWorkActivity(ctx, scope, ports.WorkActivityFilter, limit, cursor)`. Mutation uses SDK `AppendWorkActivity(ctx, key, cmd)`. CLI log add resolves its contract and registers immediately; do not keep the sole journal locally.

## Failure and acceptance cases

- Same idempotency intent yields one activity; changed payload with same identity conflicts.
- Cross-Namespace/other-holder/stale-fence/expired/revoked append is rejected with no partial event.
- Concurrent appends yield deterministic ordering with no dropped records; a page boundary neither repeats nor skips a committed record.
- Process restart, completed return, receipt-driven local cleanup, and removal of accepted local contracts leave server history readable by an authorized Principal.
- Memory, SQLite and real PostgreSQL run the same contract tests; HTTP/MCP/SDK return equivalent order, scope and errors.
- Bounds and omission semantics remain explicit for a long history; Skills do not request the entire journal each iteration.
