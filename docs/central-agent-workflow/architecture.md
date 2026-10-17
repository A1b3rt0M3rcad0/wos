# Architecture and component boundaries

## Existing foundation

The system already has Outcome/Objective/WorkItem aggregates, current-authorization checks, immutable signed specifications, execution/review contracts, signed returns, durable receipts/resolution, SQL migrations, and bounded continuity queries. Extend these components; do not build a parallel task system.

| Responsibility                         | Existing implementation to inspect/change                                                                                                   |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| CLI dispatch and error/output contract | `packages/wos-cli/client.go`                                                                                                                |
| Local files, publication, locks        | `workspace.go`, `workspace_v2.go`, `publication_v2.go`, `locks_v2_unix.go`, `locks_v2_windows.go` under `packages/wos-cli/`                 |
| Profile binding, enrollment, trust     | `profiles.go`, `profile_commands_v2.go`, `profile_trust_v2.go` under `packages/wos-cli/`                                                    |
| Contract verification and return       | `contract_v2.go`, `work_commands_v2.go`, `return_preflight_v2.go`, `return_send_v2.go`, `return_resolution_v2.go` under `packages/wos-cli/` |
| Immutable contract/material model      | `packages/wos-core/domain/signed_material.go`, `signed_fact.go`, `signed_review.go`                                                         |
| Acquisition/authority/return use cases | `packages/wos-core/application/signed_acquisition.go`, `signed_next.go`, `signed_return.go`, `signed_review_return.go`, `signed_lease.go`   |
| Continuity and reference discovery     | `packages/wos-core/application/continuity*.go`, `readiness_query.go`; `packages/wos-api/http/references.go`                                 |
| SQL and memory transactions            | `packages/wos-core/storage/{memory,sqlite,postgres}/`                                                                                       |

Paths in a shared row are relative to the named package. New implementation files can split responsibilities, but these existing paths are the integration boundaries and regression anchors.

## Proposed flow

```text
External agent runtime
        |
        v
CLI: resolve installation data -> freeze selected Profile -> validate intent
        |                                      |
        |                                      v
        |                           protected local recovery state
        v
HTTP/SDK -> Application -> Domain + Ports -> Memory/SQLite/PostgreSQL
MCP --------------------^
```

The CLI exposes small operational commands to the model. It owns protected persistence, signing, request identity, and receipt reconciliation. Server business state remains authoritative. MCP and SDK continue to expose equivalent use cases for authorized non-agent integrations; official agent Skills do not instruct agents to bypass the CLI.

## Steps and dependencies

1. Record the accepted central-storage/CLI-only direction in an ADR. Retain Tenant isolation and Principal/ActorRef separation. This unblocks Profiles and activity work.
2. Move default storage resolution to installation scope without removing explicit legacy recovery entry points. Keep location resolution separate from protected file operations.
3. Select one Profile once per invocation, construct its destination/identity binding, and never consult mutable global selection during the command.
4. Add typed activity use cases through Core and ports, then adapters. Do not write SQL from HTTP/MCP or the CLI.
5. Compose CLI context/status from verified local state and bounded server queries. Report source and omission semantics rather than claiming a coherent snapshot from unrelated reads.
6. Update Skills and public docs after command and response shapes settle.

## Invariants and acceptance

- No Domain import of CLI, HTTP, SQL, runtime, or provider-specific agent packages.
- No network fetch, model call, or agent execution inside a domain transaction.
- Activity table/event append and durable signed-operation completion share a transaction; failed appends do not leave partial history.
- A Profile's current directory cannot change destination, Tenant, credential, or signing key.
- Coordination still uses expected versions, Outcome guards, leases, quotas and fencing. Convenience references resolve to exact IDs before freezing intent.
- A prepared-only return is not silently transmitted by recovery; accepted signed proof is required before cleanup.

Verification: dependency-boundary tests, existing signed contract regression suites, activity rollback/storage tests, and any-directory/pinned-selection CLI tests. Merely moving files does not establish crash recovery or Windows behavior.

## Implemented file map

The baseline map above explains the existing boundaries. These are the concrete new implementation entry points on this branch:

| Area | Files |
| --- | --- |
| Installation and selection | `packages/wos-cli/central.go`, `central_permissions_unix.go`, `central_permissions_windows.go` |
| Protected onboarding | `packages/wos-cli/central_provision.go`, with existing `profile_commands_v2.go` |
| Exact migration and crash recovery | `packages/wos-cli/central_migration.go`, `central_migration_process_test.go`, `central_test.go` |
| Typed draft authoring | `packages/wos-cli/operations_material_v2.go`, `operations_draft_query_v2.go` |
| Activity persistence and recovery | `packages/wos-cli/operations_activity_v2.go` |
| Focal/accepted context and references | `packages/wos-cli/operations_query_v2.go`, `operations_resolution_v2.go` |
| Domain and application | `packages/wos-core/domain/work_activity.go`, `packages/wos-core/application/work_activity.go`, `work_attention.go` |
| Optional storage port | `packages/wos-core/ports/work_activity.go` |
| Concrete stores | `packages/wos-core/storage/memory/work_activity.go`, `packages/wos-core/storage/sqlite/work_activity.go`, `packages/wos-core/storage/postgres/work_activity.go` |
| Forward migrations | `packages/wos-core/storage/sqlite/migrations/0030_work_activity.sql`, `packages/wos-core/storage/postgres/migrations/0030_work_activity.sql` |
| Public adapters | `packages/wos-api/http/work_activity.go`, `packages/wos-api/mcp/work_activity.go`, `packages/wos-sdk-go/work_activity.go` |
| Protocol and native acceptance | `tests/acceptance/work_activity_transport_test.go`, `tools/acceptance/central-agent-workflow.mjs`, `packages/wos-cli/operations_v2_test.go` |
| Public generation | `tools/openapigen/work_activity.go`, existing MCP/PostgreSQL/command generators |
| Reproducible plan packaging | `tools/documentation/package-central-plan.py` |

Paths shortened after an explicit directory remain in that directory. Current dispatch, return cleanup, schemas, capabilities, browser command exposure and generated registrations integrate these entry points. The associated tests live beside their owning implementation; [verification](testing.md) and [observed validation](validation.md) define the evidence and limits.
