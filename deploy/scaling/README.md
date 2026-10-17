# Start small and scale one WOS installation

These opt-in Compose profiles run the complete public WOS product. The small profile uses one complete WOS process and PostgreSQL. The replica overlay uses two API processes and two independent delivery workers against the same database, with discovery of all API replicas at the TLS ingress. All replicas of the selected Tenant use its catalog-resolved schema and the same product image. There is no Redis or Kubernetes dependency.

The root [local quick start](../../compose.yaml) remains the SQLite option. SQLite is an embedded single-writer backend; never scale API writers by mounting its file into multiple containers. Starting a production installation on PostgreSQL avoids requiring a later SQLite-to-PostgreSQL cutover.

## Prepare one installation

PostgreSQL runtimes require a provisioned catalog entry and Tenant schema before
startup. These profiles include the shared reference's one-shot `tenant-init`
service, which runs `wos storage tenant provision --from-env --if-absent` after
PostgreSQL becomes healthy and before API startup. Existing active placements
must match the configured ID/name exactly. Suspension, interrupted provisioning
and deletion require explicit operator handling; rerunning Compose never resumes
or recreates them silently.

Custom deployments can provision with `wos storage tenant provision --file
REQUEST.json` under the protected database environment, then select the Tenant
with `WOS_TENANT_ID`. The JSON contains `{tenant_id, name, reason}` and no secrets.
A configuration change does not adopt a legacy `public` schema automatically.
See [Tenant operations](../../docs/scaling/tenants.md).


Follow the [shared profile's secret provisioning and TLS instructions](../shared/README.md#provision-and-start). Provision an external `database_password`, `bootstrap_token` and persistent `issuer_seed`; preserve the approved public identity and separate protected recovery material. Copy [`.env.example`](.env.example) outside the repository and replace the Tenant and secret-directory settings. Leave the optional expected-process override unset: the small profile declares one process and the replica overlay declares four.

```sh
docker compose -p wos-production --env-file /path/to/production.env \
  -f deploy/scaling/compose.yaml up --build -d

docker compose -p wos-production --env-file /path/to/production.env \
  -f deploy/scaling/compose.yaml run --rm --no-deps wos config validate --deployment=shared
```

Only the loopback HTTPS ingress is published, at `https://wos.localhost:8443/app/` by default. API, worker and database ports remain private. The reference uses Caddy's local CA; trust only its public certificate for local validation. Configure real DNS, ingress exposure and an appropriate certificate policy before public hosting. The ingress uses Caddy's built-in `network_proxy none` only on its private WOS upstream transport, so resolved container IPs remain local even when the container inherits an outbound HTTP proxy. Ambient proxy settings and TLS verification remain available for other traffic. The internal database network is private to this project; a managed/external database requires its own verified TLS configuration. The supplied secret entrypoint currently addresses the Compose `postgres` service; replace it deliberately when using an external database.

API and worker health probes bypass proxies locally. `wos config validate --deployment=shared` checks configuration and operator declarations, not the actual firewall, certificate policy, backup system or capacity.

## Add API and delivery capacity

Keep the same Compose project, database volume, issuer, configuration and qualified product version. Apply both files; their default connection-budget declaration accounts for four processes:

```sh
docker compose -p wos-production --env-file /path/to/production.env \
  -f deploy/scaling/compose.yaml -f deploy/scaling/compose.replicas.yaml \
  up --build -d
```

A worker role starts deliveries automatically. If the existing installation has integrations, configure the reviewed `WOS_WEBHOOK_ENDPOINTS` JSON and matching `WOS_WEBHOOK_SECRETS` JSON map in protected external configuration **before** starting the workers. These optional variables are forwarded by both profiles; APIs in the replica profile receive only endpoint metadata, while workers receive the signing secrets. Empty values produce idle workers on a fresh installation; they do not remove persisted endpoints or pending deliveries from an existing database. Workers connected to an existing queue need its complete destination allowlist and signing secrets. Keep rendered Compose configuration private when it contains secret values; a production secret manager can supply the equivalent configuration through a protected override.

The resulting roles are:

| Service | Role | Initial replicas | Responsibility |
| --- | --- | ---: | --- |
| `wos` | `api` | 2 | Browser, HTTP, MCP and domain operations |
| `worker` | `worker` | 2 | Durable integration delivery; health and operator metrics only |
| `postgres` | Database | 1 | Authoritative shared state, coordination and queue |
| `ingress` | TLS ingress | 1 | Discover and balance all API addresses through Compose DNS |

Workers consume the same durable delivery queue with leases and fencing. They do not execute Tasks, run agents or call an LLM. The worker reference receives only the database password; it receives neither the bootstrap credential nor the issuer's private seed. All worker replicas must receive the same configured endpoint definitions and access to the appropriate webhook signing secrets before real deliveries are enabled. Endpoint secret resolution must work on every worker eligible to claim that delivery.

The profile admits at most 32 concurrent authenticated requests per Tenant **per API process**. This is not a cluster-wide rate limit or a subscription quota. Durable delivery caps default to four active claims per Tenant and two per endpoint across cooperating workers, so a slow destination cannot consume every lane. Every worker must use the same policy settings; zero disables each optional cap. These caps bound simultaneous claims, not requests per second or storage consumption.

A worker's private `/readyz` checks its database/schema and local worker availability. Its listener has no application, HTTP domain, session or MCP routes. Process health, durable backlog health and fleet health have different meanings: a healthy API does not prove that delivery workers are running.

To choose another process count, set `WOS_DEPLOYMENT_EXPECTED_PROCESSES` in the external environment file to the sum of **all** API, worker and complete processes, then specify the replica counts:

```sh
# First set WOS_DEPLOYMENT_EXPECTED_PROCESSES=6 and an actually available
# WOS_DEPLOYMENT_DATABASE_CONNECTION_BUDGET of at least 86 in the environment file.
docker compose -p wos-production --env-file /path/to/production.env \
  -f deploy/scaling/compose.yaml -f deploy/scaling/compose.replicas.yaml \
  up -d --scale wos=4 --scale worker=2
```

Apply the same file set on subsequent `up`, `logs`, `stop` and `down` operations. Returning to the small profile requires a deliberate worker drain and removal of the extra worker service; do not discard pending deliveries or change the logical installation identity. `down -v` destroys persistent volumes and is not a routine scale-down command.

## Budget the database before adding processes

The reference uses eight maximum PostgreSQL data connections per Tenant runtime.
A single-Tenant process additionally reserves one held lifecycle-guard connection
and a catalog pool of two. The example reserves 20 other connections and declares
an 80-connection database budget:

```text
single-Tenant process ceiling = data pool + one lifecycle guard + catalog pool
four processes: 4 × (8 + 1 + 2) + 20 = 64 connections
six processes:  6 × (8 + 1 + 2) + 20 = 86 connections
```

Six processes therefore exceed the default 80-connection declaration. Increase
actual PostgreSQL capacity and the declared budget, or reduce pool/process counts,
before deploying that configuration. Changing the declaration does not allocate
connections or change PostgreSQL's real limit.

With `WOS_TENANT_HOSTS`, each process has one data pool and held guard per Tenant,
plus a shared catalog pool of four:

```text
hosted process ceiling = hosted Tenants × (data pool + 1) + 4
five Tenants, two processes, pool 4 and reserve 20:
2 × (5 × (4 + 1) + 4) + 20 = 78 connections
```

`wos config validate --deployment=shared` reports `hosted_tenants`,
`catalog_pool_limit`, `lease_connections` and `process_connection_limit`. Multiply
by all deployed processes and add operational reserve. With unequal runtimes,
calculate the exact sum or conservatively declare the largest pool. These are
configuration checks, not a cluster-wide connection limiter or discovery of
unregistered clients. Account for administrators, backups, migration jobs and
other services. Never declare more capacity than the database actually has.

Watch query latency, lock waits, pool waits, PostgreSQL CPU/I/O/WAL, table/index growth, oldest pending delivery and exhausted retries. More processes cannot remove contention on one hot Outcome or create more write capacity than the primary database supports. Increase worker concurrency only when the database and destinations have spare capacity. A slow external destination must not be mistaken for insufficient API capacity.

The [scalability qualification procedure](../../docs/scaling/qualification.md) compares independently pooled application replicas, Outcome/Tenant contention and real signed history. Its measurements are a laboratory reference, not supported customer counts or an SLA.

## Recovery, updates and operational limits

The supplied PostgreSQL service has persistent storage but **is not a high-availability database cluster**. Replica APIs do not make a single database or single ingress highly available. Use a qualified external PostgreSQL HA setup when required, with a documented replication/durability policy, WAL/PITR, independently protected issuer material, restore drills, recovery-time objectives and acceptable data-loss boundaries. A read replica must never authorize work from stale state.

Follow [backup/restore and upgrade cutover](../../docs/operations.md). For a catalog-managed Tenant, pause mutations and stop all its API/worker runtimes, suspend its placement, apply `wos storage tenant migrate`, verify the suspended data, then resume and start one qualified version. Use a fresh catalog version for every lifecycle action. This profile does not certify mixed-version rolling upgrades. An additive migration or an old binary's startup refusal does not fence a process already running.

For delivery recovery, preserve consumer deduplication records: a crash after the destination commits but before WOS receives its acknowledgement can repeat the external request. Database restoration does not reverse external effects or extend leases.

These profiles scale one Tenant runtime within **one installation**. The optional
`WOS_TENANT_HOSTS` composition adds explicit hostname-bound Tenant runtimes, each
with its own schema, issuer and credentials; ingress and secret provisioning must
match those approved hostnames. See [shared and dedicated placement](../../docs/scaling/tenants.md). Additional installations can share infrastructure or use dedicated infrastructure. Tenant transfer, cross-installation trust migration, automatic storage sharding, automated failover and fleet billing require their own qualified capabilities; copying a Tenant's tables is not a safe transfer protocol. The private WOS Cloud manager can provision and operate these same public installations without duplicating the WOS domain.
