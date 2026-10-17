# ADR 043 — Shared resource budgets and authoritative delivery time

Status: accepted implementation decision; workload qualification is separate.

## Context

A shared PostgreSQL installation can run multiple APIs and delivery workers.
Process-local concurrency bounds alone let one Namespace or slow destination
occupy the delivery pool. Worker host-clock differences must not decide whether
another worker's durable lease has expired. Authentication and fleet management
remain responsibilities of the existing WOS product and its external operator.

## Decision

All SQL delivery claims acquire a short dedicated Namespace scheduling row,
separate from domain/security Namespace coordination. Eligible Namespaces take
monotonically advancing turns. PostgreSQL skips locked scheduling/delivery rows;
SQLite retains its immediate single-writer transaction. The scheduler examines
at most 64 candidate Namespaces per selection and bounds crashed-delivery
recovery to the existing 25-record pass. Saturated Namespaces and endpoints are
excluded before candidate limiting. No external delivery holds a scheduling lock.

Optional Namespace and endpoint live-lease limits are checked in the same
transaction as lease acquisition. They apply across processes using this schema
and matching worker policy, rather than a local semaphore. Zero preserves an
unlimited dimension. All workers must use the same limits; changing them requires
coordinated configuration and does not revoke existing leases. An endpoint is a
configured destination ID, not a grouping of unrelated URLs by hostname.

Workers claim only deliveries whose secret references they can resolve to
configured signing material. Missing secret material leaves work pending and
observable instead of consuming its retry budget. SSRF, signature, retry and
stable event-identity rules remain in effect. A worker must also receive the
appropriate destination allowlist; operator configuration is part of deployment.

Official SQL workers use an optional durable delivery port: PostgreSQL reads
`clock_timestamp()` for claims and, after locking the delivery, for ACK deadlines
and relative retry schedules. Worker host time does not grant fleet lease
authority. SQLite uses its single-host clock. The original explicit-clock storage
methods remain available for embedding and deterministic tests. Third-party
adapters without the new ports retain their previous behavior; they cannot claim
the optional shared-capacity guarantee. A configured shared cap fails closed on
an adapter that lacks the supporting port.

HTTP/MCP optionally share a per-process authenticated-Namespace admission budget.
It uses the resolved credential scope, has no waiting queue, releases on panic or
cancellation, and retains only active entries. Authentication remains bounded by
the existing process-wide transport limit. This budget is explicitly local:
adding API replicas increases its installation-wide total. It is neither a
distributed rate limiter nor billing. Static pages, login and reserved health
probes retain their existing transport paths.

Process metrics add fixed latency bucket counts and connection-pool pressure.
They contain no tenant payloads or high-cardinality tenant labels, require
installation observation authority, and reset on process restart. External
monitoring aggregates them. Durable scoped delivery health remains separate.

## Product and fleet boundaries

WOS is the complete source-available product for standalone and shared operation.
The private WOS Cloud manages installations, placement, capacity and commercial
accounts through generic WOS management capabilities plus external infrastructure.
It does not implement another Outcome domain or business-authorization service.

Namespaces can share an installation; installations can share infrastructure.
Dedicated deployment can isolate the application, logical database, database
cluster or infrastructure according to the operator's policy. Neither replicas
nor a broker removes the primary database's write limit or one Outcome's genuine
coordination boundary. Storage sharding, multi-region writes, Namespace trust
transfer and mixed-version rolling updates require separate qualified protocols.

## Verification

SQL contracts exercise Namespace turns, cross-connection simultaneous claims,
shared caps, endpoint isolation, secret eligibility, lease expiry and fencing.
PostgreSQL additionally exercises extreme caller-clock skew and database-clock
retry/ACK decisions. Admission tests check isolation, authentication, supplied
tenant-header rejection, cancellation/panic release and shared transport wiring.
Metrics tests verify bucket boundaries and pool diagnostics. Normal, race and
real PostgreSQL suites remain required before acceptance.
