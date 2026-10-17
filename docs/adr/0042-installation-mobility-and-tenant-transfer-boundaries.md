# ADR 042: Installation mobility and tenant transfer boundaries

Status: Accepted for the owner-requested progressive self-hosted and Cloud scaling implementation on 2026-10-17.

## Context

WOS must grow from a small self-hosted deployment to shared and dedicated installations without losing persistent work. The installation owns a singleton public server identity; signed authority, accepted-operation receipts and client trust are bound to it. Namespace isolation alone does not define a complete export boundary because Principals are global and several delivery/history tables are scoped indirectly.

The existing operator backup/restore procedures preserve whole installations within a backend. They do not convert SQLite to PostgreSQL or transfer one Namespace to a new issuer identity.

## Decision

Provide explicit, offline, operator-controlled `wos storage migrate-sqlite` and `wos storage namespace-preflight` commands in the shipped native executable. A public non-domain operations package owns the implementation; an optional source-tree wrapper delegates to the same code, so self-hosted operators need no Go toolchain for migration or preflight. The migration moves a whole SQLite installation to PostgreSQL. Require a pinned immutable backup, all source writers fenced, the independently approved public issuer pins for signed installations, matching compiled source migrations and a new empty destination database. Import all application tables transactionally, validate every foreign key, restore original constraint flags, and verify logical row digests before committing. Retain the destination's canonical PostgreSQL migration metadata. Normalize only documented structural unsigned-counter representations; preserve signed and documentary strings verbatim.

The migration command never changes routing or collects private issuer seeds. Resuming the same logical installation requires the operator to retain its existing identity and secret-management configuration. Actual storage operations are qualified on Linux server hosts and fail before storage access on other platforms; help remains available. Windows filesystem mode bits do not establish protected DACLs, so no Windows staging-security claim is made. The tool is a public generic operator capability and has no Cloud control-plane dependency.

Do not expose an unsafe Namespace row-copy operation. Provide a coherent read-only preflight with measured reference closure, shared-Principal, credential, lease, delivery and public-issuer inventory. Always report Namespace transfer as unavailable until a dedicated domain/trust/cutover protocol satisfies its acceptance criteria.

## Consequences

- Production can start with a simple PostgreSQL deployment or explicitly move a SQLite installation using the [mobility procedure](../scaling/mobility.md).
- Application-state copy failure rolls back the complete copy. Schema initialization may remain in the new unused destination; retry requires another empty database.
- This is an offline conversion with bounded execution, not replication, sharding, automatic failover or a zero-downtime upgrade promise.
- The source must remain fenced after the destination accepts writes. Rollback after new work requires reconciliation rather than restoring an old snapshot over acknowledged state.
- Shared-to-dedicated transfer requires source fencing, complete domain reference closure, historic issuer verification, destination enrollment/trust transition, receipt/delivery reconciliation and interruption recovery. A shared issuer private key is never transferred to an unrelated tenant installation; signed server bindings are never rewritten.
- Self-hosted and WOS Cloud consume the same future transfer contract. The private Cloud remains fleet/infrastructure management rather than a second WOS domain implementation.
