# ADR 041 — Explicit installation operator authority

Status: accepted for implementation; executed acceptance is recorded in ROADMAP.
Date: 2026-10-17.

## Context

Namespace administration is tenant-scoped. Treating it as installation provision
or observation authority lets an otherwise isolated customer create more tenants
and inspect aggregate process activity. Shared and dedicated deployments need the
same generic management boundary, with no separate business authorization service.

## Decision

Retain existing Principals, opaque credentials, policies, Namespace grants and
administrative CAS/audit/receipts. Introduce explicit permissions:

- `installation:provision`: create a Namespace from an authorized source Namespace.
- `installation:observe`: read process-wide `/metrics`.
- `installation:admin`: delegate/remove installation capabilities and manage
  credentials belonging to a Principal holding any installation capability.

Administrative mutations still require `namespace:admin`. Capabilities are bound
to the credential's explicit originating Namespace grant; they grant no implicit
domain access to other Namespaces. Outcome-restricted credentials cannot exercise
them. A Namespace administrator cannot escalate by self-grant, copying an operator
grant, removing a protected grant, or issuing a replacement operator credential.
Both current and requested grants are checked. These checks, including current
credential/parent/policy validation, run inside the administrative transaction
before receipt disclosure. Namespace creation copies only Namespace permissions.

The scoped delivery-health API/MCP query remains the tenant diagnostic contract;
its durable Namespace/Outcome counts are distinct from a single process heartbeat.
Process metrics require installation observation. Neither represents billing.

Fresh bootstrap explicitly grants installation capabilities. Existing grants are
never automatically promoted by migration, environment restart or role names.
The offline host command `wos installation authorize --file FILE` promotes one
explicitly selected existing Namespace administrator. It requires drained-writer
assertion, reason, Namespace CAS and idempotency key; it starts no listener,
performs no bootstrap/migration, issues no token and persists the normal audit and
receipt. Existing JSON permission storage and audit tables need no schema rewrite.
The host asserts writer drainage; the command cannot prove it remotely.

Trusted storage composition methods remain privileged deployment APIs; remote
HTTP/MCP administrative intents cannot set the host-operator flag. The helper
grant/credential methods retain their trusted embedding purpose; remote mutation
uses the transactionally guarded administrative command.

## Compatibility and consequences

This intentionally narrows existing authorization: Namespace-only administrators
lose installation provisioning and aggregate metrics access. Operators must use
the explicit offline upgrade path before resuming a shared service. Ordinary
Namespace grants, domain operations and scoped diagnostics remain unchanged.
All management capabilities are part of WOS and equally available to self-hosted
operators and the private Cloud management backend.

Concurrency, multi-storage, HTTP/MCP, revocation, restart and offline promotion
acceptance are required. This boundary alone does not qualify shared-service
fairness, per-tenant storage quotas, fleet billing or Namespace data transfer.
