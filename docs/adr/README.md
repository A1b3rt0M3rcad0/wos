# WOS Architecture Decision Records

ADRs in this directory record accepted architectural decisions that constrain implementation.

The canonical design document proposed ADR-001 through ADR-015. They are materialized here as their corresponding implementation areas become active. A later ADR may supersede an earlier one, but accepted history is not rewritten silently.

Initial Wave 01 records:

- [ADR-001 — Outcome-oriented domain model](./0001-outcome-oriented-domain-model.md)
- [ADR-002 — Agent-agnostic Core](./0002-agent-agnostic-core.md)
- [ADR-003 — Roadmap ownership and versioning](./0003-roadmap-ownership-and-versioning.md)
- [ADR-010 — Public Core and internal Server boundary](./0010-public-core-server-boundary.md)

- [ADR-011 — Product package topology](./0011-product-package-topology.md)

- [ADR-012 — Documentary records and explicit decision history](./0012-documentary-records-and-decision-history.md)

- [ADR-013 — Verifiable conclusions and immutable assessment history](./0013-verifiable-conclusions-and-assessment-history.md)

## Completion records — 2026-10-07

- [ADR-004 — Separate Issues and Blockers](./0004-issues-and-blockers.md)
- [ADR-005 — Relational state with an immutable event log](./0005-relational-state-and-event-log.md)
- [ADR-007 — Aggregate versions and Outcome coordination](./0007-optimistic-versions-and-outcome-guard.md)
- [ADR-008 — Namespace isolation and consumer context](./0008-namespace-and-consumer-context.md)
- [ADR-009 — MCP as an Application adapter](./0009-mcp-application-adapter.md)
- [ADR-016 — PostgreSQL lease time authority](./0016-postgres-lease-time-authority.md)

The original specification proposed numbers, not immutable file identities. Accepted ADR-011 records package topology; criteria/conclusions are materialized in ADR-013, leases in ADR-016 and the public contracts, triggers and identity in ADR-014/015. These records preserve the implemented decisions rather than renumbering accepted history.

- [ADR-027 — Durable signed operation results](./0027-durable-signed-operation-results.md)
