# Start small, scale the complete WOS installation

WOS persists and coordinates work across people, external agents and sessions.
The same product runs on a small self-hosted server, a shared hosted installation
or dedicated infrastructure. External runtimes still select and execute agents.

This guide describes this branch's implementation. Use the
[qualification record](qualification.md) for executed measurements and their
boundaries; configuration alone does not establish production capacity.

## Deployment progression

| Stage | Run | Grow by |
| --- | --- | --- |
| Local evaluation | One complete process with SQLite | Keeping the simple single-host workflow |
| Small production | One complete process with PostgreSQL | Database sizing and workload measurement |
| Replicated installation | API and worker roles against the same PostgreSQL database | Adding capacity to the role under pressure within a database connection budget |
| Shared service | Several authorized Namespaces in one installation | Resource budgets and placement across installations |
| Dedicated service | The same WOS product with dedicated application/database resources | Independently sizing the installation and its infrastructure |

Use the [public Compose profiles](../../deploy/scaling/README.md) for a small
PostgreSQL installation and an API/worker replica overlay. Redis and Kubernetes
are not required. The ordinary [local quick start](../../README.md#quick-start)
continues to use SQLite. A SQLite database remains single-writer; mounting its
file in more containers does not create a replicated database.

Whole-installation [SQLite-to-PostgreSQL migration](mobility.md) is an explicit
offline operation with source/destination verification. Starting on PostgreSQL
avoids that operational cutover when production growth is already expected.

## One product, independently sized processes

`WOS_PROCESS_ROLE` or `wos server --role` selects:

- `complete` (default): browser/API and optional delivery processing together.
- `api`: browser, HTTP, MCP and domain operations; no delivery worker.
- `worker`: existing-installation delivery and bounded maintenance, with a private
  operational listener exposing health and operator metrics only.

A worker requires PostgreSQL and API-token authentication. It cannot create or
migrate an installation, bootstrap its administrator or replace its issuer.
Workers do not need the issuer private key. They need database access and the
webhook secret/destination configuration appropriate to their queue.

All processes in an installation use compatible binaries, one schema and the
same installation identity. Browser sessions and pending work are durable; API
load balancing requires no process-local business session or sticky routing.
Open MCP streams reconnect after a process is lost; they do not own work state.

## Persistent state and concurrency

Coherent queries use a single database snapshot and evaluation time without
locking the Outcome for mutation. PostgreSQL retains shared authorization
barriers so concurrent revocation and administrative changes remain ordered.
SQLite has one writer and a bounded four-reader pool using WAL snapshots.

Ordinary signed operations share the Namespace protocol barrier. Acquisition
coordinates the relevant Principal quota and any explicitly configured Namespace
ceiling before taking the Outcome mutation guard. Protocol/security changes
retain exclusive coordination. One disputed Outcome, one Principal's acquisition
quota and an enabled shared Namespace ceiling remain genuine shared resources.

PostgreSQL commands retain Serializable isolation. Conflicts can still occur;
clients reconcile and repeat the same exact intention when appropriate. WOS
does not silently change an expected version, sign a replacement intention or
extend a lease to make a retry succeed. Configured signed authority ceilings
retain their existing 1–100 validation; absent Namespace policy has no
Namespace-wide ceiling and keeps the existing Principal defaults.

Persistence writes compare transaction-local immutable snapshots, append new
historical facts and update changed current links. Scoped set reads remove
per-historical-record child queries. These changes bound SQL statement growth
for tested updates; full aggregate DTO hydration, validation and hashing still
depend on the history returned. They do not establish constant-memory history
or automatic archival. Prefer bounded continuity/focal/page APIs for interactive
traffic; full snapshots and exports retain their documented cost.

## Isolate shared capacity

| Setting | Meaning |
| --- | --- |
| `WOS_HTTP_MAX_INFLIGHT_REQUESTS` | Process-wide transport admission |
| `WOS_HTTP_MAX_INFLIGHT_PER_NAMESPACE` | Optional authenticated-Namespace admission shared by HTTP/MCP in one process; zero disables |
| `WOS_DELIVERY_WORKER_CONCURRENCY` | Local delivery lanes, with one lease per lane and no prefetch |
| `WOS_DELIVERY_MAX_INFLIGHT_PER_NAMESPACE` | Live delivery lease ceiling across workers using the same policy; zero disables |
| `WOS_DELIVERY_MAX_INFLIGHT_PER_ENDPOINT` | Live lease ceiling for a configured endpoint across those workers; zero disables |
| `WOS_POSTGRES_MAX_OPEN_CONNECTIONS` | This process's database pool ceiling |

All workers must use matching shared limits. Lowering a limit leaves existing
leases intact and stops new claims until capacity is available. Worker secret
eligibility prevents consuming retries for a secret unavailable in that worker.
Each delivery worker supports at most 256 eligible signing secret references
(secrets containing at least 32 bytes). This bounds SQL query parameters.
Configuration validation and direct worker construction/pool startup reject a
larger eligible set before claiming any delivery; references are never silently
truncated. Reference identifiers retain their existing opaque-string contract.
Configure additional workers with explicit disjoint secret-reference subsets
and the corresponding endpoint allowlists when the installation needs more
references; keep shared capacity limits equal.
Namespace turns and endpoint caps protect other eligible work from a busy tenant
or slow destination. The per-Namespace API setting is local, so its aggregate
capacity grows with API replicas. These are resource controls, not commercial
billing meters or a distributed request-rate contract.

Choose budgets from representative load. The sum of all process pools plus
maintenance, backup and administrator connections must fit the actual database
budget. Deployment diagnostics check the operator's declared process/budget
values; they cannot discover unregistered processes or enforce the database's
global connection ceiling.

Storage quotas, query-cost classes and plan-specific commercial limits are not
added by this branch. Provision shared capacity conservatively and measure its
growth; the implemented authority quotas are not storage or CPU quotas.

## Operator and tenant authority

Namespace administration does not grant installation management. WOS uses its
existing authorization system for explicit `installation:provision`,
`installation:observe` and `installation:admin` capabilities. Fresh bootstrap
grants operator capabilities; existing installations must explicitly promote
their selected operator through the
[offline upgrade procedure](../operations.md#shared-service).
Restarting the server never silently elevates old grants.

`/metrics` is installation-operator-only and process-local. Scoped delivery
diagnostics remain available through the existing authorized HTTP/MCP contracts.
Cloud management uses the same capabilities available to a self-hosted operator.

## Retention, recovery and updates

Complete/worker processes clean completed, expired short-idempotency cache
records in 256-row batches once per minute, yielding on a partial batch, a
five-second pass deadline or a 4,096-batch safety cap. Durable signed receipts,
proofs, revocations, active authority and pending deliveries are not deleted.
Historical audit/evidence archival and table partitioning need separate policies
and compatibility work; a generic age-based TTL would break recovery contracts.

Production PostgreSQL can use independently operated HA and WAL/PITR. The public
Compose reference is a persistent single PostgreSQL instance, not an HA cluster.
Define acceptable data loss and recovery time, test failover/restore and preserve
issuer material separately. Read replicas must not authorize or acquire work from
lagging state. The implementation does not route reads to replicas.

Delivery workers use PostgreSQL time for fleet lease and ACK/retry decisions.
Delivery remains at least once: recipients must deduplicate stable event IDs.
Recovery does not undo external side effects or restore post-backup revocations.

Use the [operations guide](../operations.md) for backup and coordinated upgrade.
Stop old API/worker writers before migrating, then start a compatible version.
Mixed-version rolling updates, automatic failover and multi-region writes are
not qualified by adding replica processes.

## Fleet growth and mobility boundaries

One Namespace can serve a customer, team, environment or another operator-chosen
authorization scope. Several Namespaces can share an installation; several
installations can share infrastructure. Replicas add capacity to an installation;
new installations add environments and failure/isolation boundaries.

WOS Cloud remains a separate private management repository. It provisions and
operates this same product through administrative capabilities and external
infrastructure, maintaining its own commercial organizations/plans/billing. It
does not reproduce Outcomes, contracts, domain authorization or persistence.

Whole-installation movement preserves the installation identity. Shared-to-
dedicated Namespace movement is a different trust protocol: use the executable
[Namespace preflight](mobility.md) to inspect references, Principals, leases and
outbox obligations. It deliberately cannot authorize transfer. Full Namespace
transfer, generic shard routing and online migration remain planned; never copy
the shared installation's private issuer into a tenant-owned service or rewrite
historical signed bytes.

The next storage-distribution boundary to evaluate is Namespace, whose quotas,
authorization and idempotency already cross Outcomes. Add routing or another
broker only when measured workload justifies it. No component can make a single
hot Outcome or PostgreSQL primary grow without bounds.

## Decisions and verification

- [Scoped coordination and snapshots](../adr/0038-scoped-coordination-and-read-snapshots.md).
- [Incremental persistence and cache lifecycle](../adr/0039-incremental-persistence-and-disposable-cache.md).
- [Independent process roles](../adr/0040-independent-wos-process-roles.md).
- [Installation operator authority](../adr/0041-installation-operator-authority.md).
- [Storage growth and migration](../adr/0042-installation-mobility-and-tenant-transfer-boundaries.md).
- [Shared budgets and delivery time](../adr/0043-shared-resource-budgets-and-delivery-time.md).
- [Reproducible qualification](qualification.md): source, workload, measurements and unexecuted boundaries.
