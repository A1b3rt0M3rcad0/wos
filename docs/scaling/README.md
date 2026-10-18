# Start small, scale the complete WOS installation

WOS persists and coordinates work across people, external agents and sessions.
The same product runs on a small self-hosted server, a shared hosted installation
or dedicated infrastructure. External runtimes still select and execute agents.

This guide describes the schema-per-Tenant branch architecture. Follow the
[Tenant lifecycle and breaking cutover](tenants.md) before upgrading an existing
installation. [Tenant qualification](tenant-validation.md) records the new
schema boundary. The [earlier qualification record](qualification.md) retains the exact
source of executed measurements; earlier runs do not qualify the new storage
boundary, and configuration alone does not establish production capacity.

## Deployment progression

| Stage | Run | Grow by |
| --- | --- | --- |
| Local evaluation | One complete process with SQLite | Keeping the simple single-host workflow |
| Small production | One complete process with PostgreSQL | Database sizing and workload measurement |
| Replicated installation | API and worker roles against the same PostgreSQL database | Adding capacity to the role under pressure within a database connection budget |
| Shared service | Several Tenant runtimes, each with a dedicated schema, in one database | Resource budgets and placement across databases/installations |
| Dedicated service | The same WOS product with dedicated application/database resources | Independently sizing the installation and its infrastructure |

Use the [public Compose profiles](../../deploy/scaling/README.md) for a small
PostgreSQL installation and an API/worker replica overlay. Redis and Kubernetes
are not required. The ordinary [local quick start](../../README.md#quick-start)
continues to use SQLite. A SQLite database remains single-writer; mounting its
file in more containers does not create a replicated database.

[SQLite-to-PostgreSQL migration](mobility.md) is an explicit offline operation.
It is distinct from adopting the Tenant catalog and cannot split a legacy
multi-customer database automatically. Starting a new production Tenant on
PostgreSQL avoids the SQLite backend cutover.

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

All replicas of a Tenant use compatible binaries, its catalog-resolved schema
and its persistent issuer identity. A process can host several independent
Tenant runtimes through the explicit hostname map, each with its own fixed SQL
pool and authentication state. Browser sessions and pending work are durable;
load balancing must retain the selected Tenant host but needs no sticky routing
to one replica. Open MCP streams reconnect after process loss; they do not own
work state.

## Persistent state and concurrency

Coherent queries use a single database snapshot and evaluation time without
locking the Outcome for mutation. PostgreSQL retains shared authorization
barriers so concurrent revocation and administrative changes remain ordered.
SQLite has one writer and a bounded four-reader pool using WAL snapshots.

Ordinary signed operations share the Tenant protocol barrier. Acquisition
coordinates the relevant Principal quota and any explicitly configured Tenant
ceiling before taking the Outcome mutation guard. Protocol/security changes
retain exclusive coordination. One disputed Outcome, one Principal's acquisition
quota and an enabled shared Tenant ceiling remain genuine shared resources.

PostgreSQL commands retain Serializable isolation. Conflicts can still occur;
clients reconcile and repeat the same exact intention when appropriate. WOS
does not silently change an expected version, sign a replacement intention or
extend a lease to make a retry succeed. Configured signed authority ceilings
retain their existing 1–100 validation; absent Tenant policy has no
Tenant-wide ceiling and keeps the existing Principal defaults.

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
| `WOS_HTTP_MAX_INFLIGHT_PER_TENANT` | Optional authenticated-Tenant admission shared by HTTP/MCP in one process; zero disables |
| `WOS_DELIVERY_WORKER_CONCURRENCY` | Local delivery lanes, with one lease per lane and no prefetch |
| `WOS_DELIVERY_MAX_INFLIGHT_PER_TENANT` | Live delivery lease ceiling across workers using the same policy; zero disables |
| `WOS_DELIVERY_MAX_INFLIGHT_PER_ENDPOINT` | Live lease ceiling for a configured endpoint across those workers; zero disables |
| `WOS_POSTGRES_MAX_OPEN_CONNECTIONS` | Each Tenant runtime's data-pool ceiling |

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
Tenant runtimes own separate delivery queues and endpoint caps protect eligible
work from a slow destination. The per-Tenant API setting is local, so its aggregate
capacity grows with API replicas. These are resource controls, not commercial
billing meters or a distributed request-rate contract.

Choose budgets from representative load. Count every Tenant runtime in every
process: five hosted Tenants mean five pools per replica. The sum of all pools,
held lifecycle guards and catalog connections plus
maintenance, backup and administrator connections must fit the actual database
budget. Deployment diagnostics check the operator's declared process/budget
values, including one held lifecycle connection per Tenant runtime and the
catalog pool (two for a single Tenant, four for a hosted group); they cannot
discover unregistered processes or enforce the database's global connection
ceiling. With pool eight and reserve twenty, four single-Tenant processes need
64 connections and six need 86. See [deployment budgeting](../../deploy/scaling/README.md#budget-the-database-before-adding-processes).

Storage quotas, query-cost classes and plan-specific commercial limits are not
added by this branch. Provision shared capacity conservatively and measure its
growth; the implemented authority quotas are not storage or CPU quotas.

## Operator and tenant authority

Tenant administration does not grant installation management. WOS uses its
existing authorization system for explicit `installation:provision`,
`installation:observe` and `installation:admin` capabilities. Fresh bootstrap
grants operator capabilities; existing installations must explicitly promote
their selected operator through the
[offline upgrade procedure](../operations.md#shared-service).
Restarting the server never silently elevates old grants.

`/metrics` requires installation-operator authority and reports the selected
Tenant runtime. In hosted mode, poll each configured hostname; one response does
not aggregate every runtime or replica. Scoped delivery diagnostics remain
available through the existing authorized HTTP/MCP contracts.
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
For a catalog-managed PostgreSQL Tenant, stop all its API/worker runtimes,
suspend its catalog entry, apply `storage tenant migrate`, verify, then resume
and start a compatible version. Legacy shared storage requires the explicit
[import and cutover](tenants.md#upgrade-an-existing-installation).
Mixed-version rolling updates, automatic failover and multi-region writes are
not qualified by adding replica processes.

## Fleet growth and mobility boundaries

Use **Account** for the customer or organization and **Tenant** for its complete
WOS state boundary. One Account can initially map to one Tenant. Current API and
CLI contracts use `tenant_id`; the browser calls the working context a Workspace.
The Tenant ID records authenticated scope and provenance inside a schema. It no
longer partitions several customers' domain rows in a shared schema.

Each PostgreSQL Tenant owns its identities, credentials, issuer history, domain
state, receipts and delivery queues. The shared catalog stores only placement and
lifecycle metadata with administrative audit. A runtime resolves a fixed schema;
requests cannot select SQL schemas or alter pooled connection routing. The
[Tenant guide](tenants.md) documents the hostname mapping, provisioning and the
stop-before-suspend/delete lifecycle.

A shared database can contain five Tenant schemas, or another operator-selected
number. Resource usage and recovery requirements determine capacity. Dedicated
customers use the same product against an exclusive database; dedicated compute
and storage additionally require exclusive infrastructure. An idle schema consumes
storage but is not a running process. Suspend access and stop unused runtimes to
reduce compute; delete data only through an explicit retention-aware operation.

WOS Cloud remains a separate private management repository. It provisions and
operates this product through generic administrative capabilities and external
infrastructure, maintaining its own organizations, plans and billing. It does not
reproduce Outcomes, contracts, domain authorization or persistence.

New Tenant placement is supported independently of moving an existing customer's
signed history. Whole-state movement must preserve issuer identity, exact signed
bytes, accepted receipts, active authority and delivery obligations. The
[read-only Tenant preflight](mobility.md) cannot authorize transfer. Online
relocation, automatic sharding, mixed-version rolling migration, cross-cluster
scheduling and Cloud autoscaling remain separate work. No schema or replica count
can make a single hot Outcome or PostgreSQL primary grow without bounds.

## Decisions and verification

- [Scoped coordination and snapshots](../adr/0038-scoped-coordination-and-read-snapshots.md).
- [Incremental persistence and cache lifecycle](../adr/0039-incremental-persistence-and-disposable-cache.md).
- [Independent process roles](../adr/0040-independent-wos-process-roles.md).
- [Installation operator authority](../adr/0041-installation-operator-authority.md).
- [Storage growth and migration](../adr/0042-installation-mobility-and-tenant-transfer-boundaries.md).
- [Shared budgets and delivery time](../adr/0043-shared-resource-budgets-and-delivery-time.md).
- [Tenant storage and breaking cutover](tenants.md).
- [Reproducible qualification](qualification.md): source, workload, measurements and unexecuted boundaries.
