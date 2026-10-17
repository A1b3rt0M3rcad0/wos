# Server and protocol implementation

## Boundary

The Server authorizes and persists shared coordination state. It does not launch, schedule, supervise or execute agents, fetch arbitrary artifacts as domain work, choose a model or run a tool. The CLI's simple commands map to existing or additive Application services.

## Existing component map

| Layer                           | Files/components                                                                                                                                   |
| ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Pure material and authority     | `packages/wos-core/domain/signed_material.go`, `signed_fact.go`, `work_item.go`, `documentary.go`                                                  |
| Work queries/mutations          | `packages/wos-core/application/signed_queries.go`, `signed_acquisition.go`, `signed_next.go`, `signed_return.go`, `work_item_operational_query.go` |
| Readiness/continuity            | `packages/wos-core/application/readiness_query.go`, `continuity_projected.go`, `continuity_direct.go`, `work_attention.go`                         |
| Ports                           | `packages/wos-core/ports/events.go`, `signed_contracts.go`, `signed_operations.go`, `continuity_read.go`                                           |
| SQL/memory                      | `packages/wos-core/storage/memory/`, `sqlite/`, `postgres/`; adapter-local migrations                                                              |
| HTTP                            | `packages/wos-api/http/commands.go`, `signed_queries.go`, `work_contracts.go`, `references.go`                                                     |
| MCP                             | `packages/wos-api/mcp/commands_generated.go`, `signed_queries.go`, `queries.go`                                                                    |
| SDK                             | `packages/wos-sdk-go/commands_generated.go`, `signed_queries.go`, `queries.go`                                                                     |
| Public contract/code generation | `packages/wos-api/openapi.yaml`, `tools/signedgen/generate.py`, repository command catalog generators                                              |

## Ordered changes

1. Add typed append/history activity service and storage contract with authorization, idempotency, stable order and bounded input/output. Use the activity plan's durable representation decision.
2. Map commands to the Application service from HTTP/MCP and generate/update SDK wrappers and OpenAPI/JSON schemas. Both transports must produce equivalent scope checks, errors and events.
3. Keep `GetWorkContext` as the focal read and add scoped `GetWorkAttention` for derived obligations. Compose compact CLI context from immutable specification and current relevant authority/blocker/material/review state. Preserve evaluated-at/revision/omission semantics; do not return the entire Outcome graph.
4. Provide obligations as projections from required criteria, current assessment, dependencies, blockers and original review findings. Do not persist a second manually maintained business `pending[]` list.
5. Reuse authorized reference discovery for exact titles/prefixes. A bounded incomplete result must not be called a unique match.
6. Provide authorized coordinator reads through existing scope queries. Avoid privileged all-tenant scans disguised as convenience output.
7. Add matching `0030_work_activity.sql` forward migrations and restore/upgrade tests for both SQL stores. Dedicated immutable activity history uses the `work_activity` table; Domain Events retain mutation auditing. Validate the adapter indexes and revision-bound cursor against long-history paging.

## Transaction and security contract

Authenticate scope, reserve idempotency, acquire the Outcome guard, verify current grants/CAS/authority, mutate/append, persist events/outbox/result, then commit. Authenticated Principal controls access; supplied ActorRef, Profile name, Tenant UUID and external metadata do not grant it. Fencing and database-authoritative lease time remain intact.

Never retry by changing the caller's expected version, resolved target or payload. An accepted historical receipt does not prove current execution authority. Reviewer independence uses distinct authenticated Principals under policy, not display labels.

## Acceptance

Application tests cover invalid/stale authority and rollback; shared Memory/SQLite/PostgreSQL tests cover persistence/idempotency/order; HTTP/MCP/SDK integration covers equivalent result and error shapes; real process restart/restore tests cover durability. Re-run schema generation/public contract fingerprint checks. No new query's existence establishes production scale or complete multi-host deployment qualification.
