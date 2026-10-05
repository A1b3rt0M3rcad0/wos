# ADR 0015 — Namespace administration, access snapshots and integration facts

Status: accepted for implementation; release acceptance remains separate.
Date: 2026-10-05.

## Decision

Remote administration is an Application operation with a Namespace version guard, principal-bound idempotency receipt and immutable administrative audit. Grants, credential issue/revocation and Namespace creation commit with the administrative version. Creating a Namespace copies only the creator's current grants. Credential tokens are opaque, persisted only as digests and disclosed once; replay discloses the receipt and credential metadata without recovering a token. Lost tokens require explicit revocation and another issuance.

Outcome mutations check the credential and grants before starting a transaction, and again in the UnitOfWork before looking up an idempotent result. PostgreSQL read locks on the Namespace and credential coordinate that check with administrative revocation; SQLite uses its writer transaction. Reads authenticate per request; an already running read is not promised to be cancelled by a later revocation.

Operator-only composition helpers may bootstrap or set grants without this public administrative command surface. They are trusted deployment entry points and are not exposed as an unguarded remote API. Bootstrap is one-time and restart must not restore revoked privileges.

ExternalContext is a bounded flat map of typed JSON scalars, separately indexed. External references are unique by Namespace/provider/context-kind/external-ID. Consumer metadata never grants permission.

Public integration facts and trigger firings have explicit schema version 1 and omit internal Go payloads. Source events, facts, firings and outbox records commit together. Delivery has stable identity, expiring leases and fencing, bounded retry and signed webhooks. It is at least once. An expired started attempt consumes the attempt budget, including after worker crashes; exhausted deliveries require explicit authorized redelivery. No trigger executes WorkItems, tools or models.

## Consequences

Clients must preserve one idempotency key for the same uncertain intent and reload versions before a new intent. SQL storage carries authorization and integration contracts without importing transport code into Domain. PostgreSQL concurrency and deployment restore acceptance still require real PostgreSQL execution; generated code alone is insufficient.
