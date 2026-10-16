# Shared PostgreSQL deployment reference

The root quick-start remains `cp .env.example .env && docker compose up --build -d`: one local SQLite instance, persisted in its own volume. This optional reference runs one API with PostgreSQL, scoped API-token authentication, a persistent externally provisioned issuer, TLS ingress, health checks and explicit resource limits. It does not activate signed-v2 Namespaces or install an agent runtime.

## Provision and start

1. Provision an external secret directory containing three **single-line** files: `database_password` (at least 64 random hexadecimal characters), `bootstrap_token` (at least 32 random characters) and `issuer_seed` (base64 of one independently generated 32-byte Ed25519 seed). Never reuse these between sold instances. Keep the same issuer seed for the lifetime of this logical server, including all its API replicas. Keep an independently protected recovery copy and record the approved public fingerprint.
2. Grant container UID/GID `10001:10001` read access to the three secret files through your secret manager, ownership/group or an ACL. The files need not be world-readable. Compose local file secrets use bind mounts and do not reliably remap ownership/mode; verify access in the target environment. The image includes its public entrypoint with explicit readable/executable permissions.
3. Copy this directory's `.env.example` to an external environment file and set `WOS_SHARED_SECRETS_DIR` to the absolute secret directory and `WOS_BOOTSTRAP_NAMESPACE_ID` to this instance's UUIDv7. No seed, credential or database password belongs in Git, a public manifest or a shell argument.
4. From the repository root, run:

```sh
docker compose -p wos-shared --env-file /path/to/shared.env -f deploy/shared/compose.yaml up --build -d
docker compose -p wos-shared --env-file /path/to/shared.env -f deploy/shared/compose.yaml run --rm --no-deps wos config validate --deployment=shared
```

The runnable reference binds HTTPS to `127.0.0.1:8443` and uses Caddy's local CA for `wos.localhost`. Export **only the public root certificate** for local validation:

```sh
docker compose -p wos-shared --env-file /path/to/shared.env -f deploy/shared/compose.yaml cp ingress:/data/caddy/pki/authorities/local/root.crt ./wos-reference-ca.crt
curl --cacert ./wos-reference-ca.crt --resolve wos.localhost:8443:127.0.0.1 https://wos.localhost:8443/readyz
```

Trust that approved local CA in your test browser to open `https://wos.localhost:8443/app/`. For a public instance, replace `tls internal` with your managed/ACME certificate policy, configure DNS and your ingress/firewall, and publish HTTPS deliberately. Keep the API and database ports off the public network. The database link is isolated inside this Compose project; an external database needs its own verified TLS policy and CA. Internal health probes bypass HTTP proxies explicitly; ingress-to-API proxy bypass covers only internal service names.

The `wos config validate --deployment=shared` command is read-only. Its warnings remain warnings for legitimate small single-instance deployments. External TLS, issuer persistence, instance credentials and managed backups are **operator declarations**, not remote verification or certification. `managed_backups` defaults to false until the external procedure is actually installed and restore-tested. No configuration validation opens a database or generates a server identity.

## Capacity, workers and replicas

`WOS_POSTGRES_MAX_OPEN_CONNECTIONS` is a per-API pool limit (default 16; explicit range 1..256). Budget every replica's pool plus maintenance and other clients against PostgreSQL capacity. Domain/security transactions remain Serializable with Outcome/Namespace guards. PostgreSQL transaction time governs remote lease evaluation; keep host/database clocks supervised. Large imports need ordinary PostgreSQL statistics maintenance; the qualified cold proof path also avoids repeated broad scans before statistics warm.

The optional webhook worker is disabled by default. The reference lane count is 4 when enabled; the product default remains 1. Configure endpoints and protected signing secrets explicitly before enabling it. Read `/readyz` for Core availability and the authenticated scoped delivery-health resource for backlog, expired claims, exhaustion and worker supervision. Consumer deduplication by stable delivery identity is required after timeout, crash or restore. A third-party webhook outage does not make the Core unready.

For multiple API replicas, use PostgreSQL, the same logical server identity/issuer and current public pin, a properly configured load balancer/service discovery, persistent database storage and an explicit migration rollout. The reference declares one API replica; `--scale` alone does not qualify balancing or capacity. Stateless MCP is the default. Stateful MCP streams have bounded lifetimes and reconnect; protocol sessions are not durable domain state. Multiple API processes must not share a SQLite file to claim multiple writers.

Signed acquisition Principal defaults remain 3 work/1 review, with independent credential and Namespace ceilings. Administrators can set Namespace-only Principal ceilings 1..100; profiles/CIDs of one Principal share that quota and do not create independent review identity. `auth status`/doctor report own activity and effective limits. Neither quota increases nor deployment declarations grant permissions.

The opt-in measured contention reference is:

```sh
WOS_TEST_POSTGRES_DSN=... WOS_TEST_POSTGRES_CONTAINER=... python3 tools/acceptance/shared-contention.py --output /private/new-contention-report
```

It executes real signed work, independent review, explicit assessment/conclusion, exact replay and one-winner version collisions for 1/4 Outcomes and 2/8/50 consumers on SQLite and PostgreSQL. Reports include sample counts, p95, conflicts/exact retries, throughput, pool wait and whole-test-process CPU/peak RSS. Resource usage includes fixture setup/compilation and excludes database CPU/RSS. This bounded fixture is not an SLA or a 1.0 certification; measure your database, network, longer histories and representative load separately. Keep the report source immutable while it runs; an explicit Git archive mode records its exact source and hashes.

Backups, recovery pins and CLI workspaces are covered in [operations](../../docs/operations.md). Migrations are forward-only. Stop writers for cutover, restore into a new destination, verify authority/fencing and preserve receiver dedupe state. Rollback after an additive schema upgrade means a separately restored compatible backup and an explicit reconciliation of later effects, never running an old binary against the new schema or deleting new tables in place.
