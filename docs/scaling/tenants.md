# Tenant schemas and lifecycle

This branch replaces the shared-row `namespace_id` model with one complete Tenant
per PostgreSQL schema. It is a breaking storage and public-contract transition.
Use a server, CLI, SDK and Skills built from the same source. The recorded
[earlier scaling results](validation.md) describe their pinned commits; they do
not qualify this transition automatically.

## Account, Tenant, runtime and database

An Account identifies the customer or organization in your application or WOS
Cloud. Its Tenant owns the WOS state. People and agents receive explicit
credentials and grants in that Tenant; customer metadata never grants access.
Account signup, cross-account membership, commercial plans and billing belong to
the consumer or private Cloud manager.

| Deployment | Data placement | Application capacity |
| --- | --- | --- |
| Local evaluation | One Tenant in one SQLite file | One complete process; SQLite remains single-writer |
| Small production | One Tenant schema in PostgreSQL | One complete WOS runtime |
| Shared Cloud | Several Tenant schemas in the same database | Separate Tenant runtimes, optionally routed by hostname in one process |
| Database exclusive | One Tenant in its own database | The same runtime and replica model |
| Infrastructure dedicated | One Tenant database on dedicated infrastructure | Independently allocated API, delivery and database resources |

Five accounts can therefore use five schemas in one database. Five is a placement
policy, not a hard product limit or a capacity guarantee. Size placement using
actual database connections, load, storage and recovery requirements.

Each data schema contains the Tenant's domain entities, Principals, credentials,
grants, browser sessions, public issuer history, immutable signed material,
command receipts, integration queues and delivery attempts. The catalog contains
placement, lifecycle and administrative audit metadata outside those data
schemas. A deleted Tenant leaves a catalog tombstone rather than a reusable
identity.

`tenant_id` remains an authenticated identifier and a provenance/foreign-key
constraint. It does not select rows belonging to different customers in a common
domain schema. A runtime resolves the approved catalog entry and binds its SQL
pool to that schema for its entire lifetime. It does not accept a schema name
from an HTTP/MCP payload or change a pooled connection's `search_path` per request.

Use direct PostgreSQL connections or a pooler with session pooling. Transaction
and statement pooling do not preserve the session-level lifecycle locks and
fixed connection binding required by this runtime.

A schema is logical separation within PostgreSQL. Schemas in one database share
CPU, memory, I/O, WAL and the database failure boundary. A dedicated database on
the same PostgreSQL server still shares that server's resources. Database roles,
network permissions and secret access remain part of the deployment's isolation
boundary.

## Provision and select a Tenant

Use the native operator interface from a protected server host:

```sh
wos storage tenant --help
wos storage tenant list
wos storage tenant provision --file /private/tenant-provision.json
wos storage tenant show --tenant TENANT_UUID
```

These are local storage-administration commands. Their PostgreSQL credentials
come from protected operator configuration; they do not use a browser session or
an agent Profile. Provisioning accepts `{tenant_id, name, reason}`. Other mutation files
accept `{tenant_id, expected_version, reason}`; read the catalog entry before
choosing its current version. `list` supports `--after UUID` and `--limit N`
for bounded pagination. `--dsn-env` selects the protected environment variable
containing the PostgreSQL URI (default `WOS_POSTGRES_DSN`); `--catalog-schema`
selects the catalog (default `WOS_POSTGRES_CATALOG_SCHEMA`, then `wos_control`).
Never place a database password or issuer seed in an intent file or command argument.

The supplied shared/scaling Compose profiles run a one-shot initializer with
`provision --from-env --if-absent`. It reads `WOS_TENANT_ID` and
`WOS_BOOTSTRAP_TENANT_NAME` (defaulting to the ID when omitted), and accepts only
an existing active placement with the same ID and name. It does not change
versions or grants, reactivate suspension, or recycle a tombstone. For explicit
operator JSON, `--file -` reads standard input.

The catalog defaults to `wos_control`; configure
`WOS_POSTGRES_CATALOG_SCHEMA` when another approved catalog name is needed.
Provisioning records the data-schema binding and initializes the Tenant's storage.
For a provisioning record that fails to become active, inspect its current
version and use `wos storage tenant retry-provision --file REQUEST.json
--writers-stopped`. Starting the API does not turn an arbitrary schema into an
approved Tenant. First API startup initializes credentials and issuer in the
provisioned Tenant; provisioning itself does not issue a login secret.

For a single runtime, configure `WOS_STORAGE_DRIVER=postgres`,
`WOS_POSTGRES_DSN`, and `WOS_TENANT_ID`. `WOS_POSTGRES_SCHEMA` is an optional
assertion that the catalog-resolved schema matches the intended deployment; it is
not a request-controlled routing override. Existing bootstrap settings initialize
the selected Tenant's first administrator and issuer only when their normal
first-initialization conditions are met. Restarting with a different bootstrap
secret cannot recover a revoked credential.

Replicas use the same Tenant ID, catalog, schema and compatible binaries. API and
complete roles need the Tenant's issuer configuration; delivery-only workers
operate with the applicable database and integration secrets. Each runtime holds
a lifecycle guard while running so destructive administration cannot race it.

## Route several Tenants in one process

`WOS_TENANT_HOSTS` is a strict JSON array of at most 32 bindings. Each entry names
one unique hostname and Tenant, and references protected environment variables
for the bootstrap token and signing seed:

```json
[
  {
    "host": "acme.example.com",
    "tenant_id": "0199d000-0000-7000-8000-000000000001",
    "bootstrap_token_env": "ACME_BOOTSTRAP_TOKEN",
    "signing_seed_env": "ACME_SIGNING_SEED",
    "principal_id": "acme-operator"
  },
  {
    "host": "example.example.com",
    "tenant_id": "0199d000-0000-7000-8000-000000000002",
    "bootstrap_token_env": "EXAMPLE_BOOTSTRAP_TOKEN",
    "signing_seed_env": "EXAMPLE_SIGNING_SEED",
    "principal_id": "example-operator"
  }
]
```

The JSON carries secret references, not secret values. Hosted API Tenants must
use different bootstrap credential values; configuration rejects reuse between
Tenants. The child runtimes share
the configured PostgreSQL destination but own separate fixed-schema pools,
authentication, application services and queues. Hostname selection chooses the
runtime; it never authenticates the caller. Unknown hostnames have no default
Tenant fallback. TLS ingress must preserve the intended host and restrict traffic
to approved routes. Health probes must also send an approved Host header; an
unknown host receives `421 unknown_tenant_host`, including on health paths.
Monitor each configured Tenant endpoint rather than treating one child's health
as the state of every Tenant. Do not expose direct SQL credentials to customers.

The hostname mapping is startup configuration, not a dynamic registration API.
Changing it requires restarting the process. Removing one binding and restarting
releases that Tenant's runtime guard while the remaining configured Tenants can
continue after the restart. A worker-only deployment does not require bootstrap
tokens; it still needs explicit approved Tenant bindings and delivery secrets.

Count every child runtime and every replica when sizing connection pools. A
process with five Tenant runtimes does not have just one database pool. Include
one held lifecycle connection per Tenant runtime, the catalog pool (two in
single-Tenant mode, four in hosted mode), migrations, backups and operator reserve
in the actual PostgreSQL budget. Monitor each Tenant and the shared database; schema
separation does not impose storage or CPU quotas.

## Suspend, migrate, resume and delete

An idle schema is not a running process and cannot be paused like a container.
Its retained tables still consume storage. Suspension stops access and activity
while preserving that data; capacity savings come from stopping runtimes and
reducing unused infrastructure.

1. Stop new traffic and external work for the Tenant. Reconcile uncertain commands
   and plan for in-flight deliveries and absolute authority expiry.
2. Stop **all** of its API, complete, delivery and maintenance runtimes on every
   replica. Catalog lifecycle operations require their guards to be released.
3. Submit the exact lifecycle intent through the protected operator command.

```sh
wos storage tenant suspend --file /private/tenant-suspend.json --writers-stopped
wos storage tenant migrate --file /private/tenant-migrate.json --writers-stopped
wos storage tenant resume --file /private/tenant-resume.json --writers-stopped
```

Create each request with the current catalog version; every successful transition
advances it. To delete instead of resuming a suspended Tenant, use:

```sh
wos storage tenant delete --file /private/tenant-delete.json --writers-stopped
```

Suspension preserves the schema. `migrate` applies the compiled schema chain only
to a suspended Tenant, checks its version, records an audit entry and leaves it
suspended at a new version. Active runtime startup only verifies schema
compatibility; it does not apply migrations. Verify the result before resuming. Resume makes the
Tenant eligible to serve again; restart
the appropriate runtimes afterward. Deletion requires a suspended Tenant or an
incomplete provisioning entry, and removes its data schema while retaining the
catalog tombstone and lifecycle audit. A live runtime prevents the operation;
`--writers-stopped` records the operator's assertion and does not remotely drain
or stop processes. PostgreSQL may briefly retain session locks after a process
closes its sockets. If a lifecycle operation refuses during that interval, wait
for the sessions to close and verify that placement and version remain unchanged
before retrying the same intent. After an uncertain response, inspect the catalog state and
audit before choosing another operation; do not change the expected version just
to force a destructive retry.

Deleting a schema does not delete earlier backups, a retired shared source
retained after import, external Artifacts, receiver deduplication records,
object-store content or billing records. Handle those under
the applicable operator retention policy. Removing live data is distinct from
expiring backups. Never delete an idle customer's data merely to scale down
compute.

## Upgrade an existing installation

This is a coordinated breaking upgrade. Current contracts use `tenant_id`,
`TenantID`, `/tenants/` routes and Tenant-specific command/permission names.
The previous `namespace_id` request fields are not aliases for new mutations.
Historical migrations 0001–0032 retain their original checksums; migration 0033
introduces the live Tenant naming and explicit storage binding.

Before the cutover, stop new acquisitions, finish or explicitly resolve active
execution/review authority and open review cases, reconcile pending client
intentions, and preserve the
original receipts and CLI recovery material. Fence every old API, worker and
direct SQL writer. Take a consistent protected backup with independently approved
server and issuer pins, and rehearse a restore before changing live state.

The offline `import-shared` operation partitions a legacy shared PostgreSQL
schema into catalog-managed Tenant schemas **within the same database**. It
preserves signed bytes and source issuer trust; it does not transfer customers
to an unrelated installation.

Create a protected request file:

```json
{
  "source_schema": "public",
  "reason": "Approved offline migration to separate Tenant schemas"
}
```

With every old API, worker and direct SQL writer stopped, run:

```sh
wos storage tenant import-shared \
  --file /private/tenant-import.json --writers-stopped
```

The command upgrades the stopped source through migration 0033 before partitioning.
That migration is a separate committed step, so retain the pre-upgrade backup even
if the later partition fails. The import copies Tenant-owned rows and their
referenced Principals, public issuer history, credentials, proofs, receipts and
queues into generated `wos_t_<UUID-without-hyphens>` schemas. It verifies table
counts and ordered content digests. Target data, catalog placement and source
write fences commit atomically; a failed import rolls that partition transaction
back without leaving a partial set of targets or fences.

Imported Tenants start **suspended**. Review the result and each catalog entry,
verify original accepted receipts and issuer pins, and configure the correct
existing issuer and integration secrets. Resume each Tenant with its current
catalog version, then start only compatible runtimes bound to that Tenant ID.
A runtime refuses an incompatible ID rather than rebinding its data silently.

The original schema stays intact for inspection and contains the original data
for every migrated Tenant. Apply an explicit retention/disposal policy to that
retired copy after verifying all migrated Tenants and the approved recovery
window. Deleting one new Tenant schema does not erase its rows from this retained
source. Do not delete individual source rows to circumvent its write fence.

The importer refuses unsupported source objects, row-level security policies and
table inheritance rather than copying a filtered or incomplete view of customers.
Database triggers reject its
`INSERT`, `UPDATE`, `DELETE` and `TRUNCATE` operations after import. They do not
remove a privileged operator's ability to alter DDL or disable protections.
Revoke retired writer credentials and DDL privileges; an owner/superuser cannot
be universally fenced by an application. Do not route old binaries to the source
or manually remove its fences to reopen competing authority.

Imported historical Tenants retain the source installation's public issuer
identity. Preserve the approved seed under the same trusted operator; never give
a customer the former shared installation's private issuer material merely
because its tables now occupy a separate schema. A future customer-owned
installation requires a separately verified trust transition.

New signed mutations use **protocol 3**, whose signed fields bind `tenant_id`.
Historical protocol-2 envelopes and receipts retain their exact original bytes,
including `namespace_id`, for read-only verification. They are never reserialized,
renamed or signed again to appear current. Historical verification does not grant
new authority or permit a protocol-2 mutation against the new service. Update
clients and approved Profiles together; `wos profile migrate-tenant NAME` handles
eligible legacy local Profiles only after pending operations have been resolved.
Do not automatically reinterpret an unresolved old signed intention as a new one.

Before new destination writes, rollback can restore the original binary and
pre-upgrade state under an explicit routing cutover. After new writes or delivery
side effects, a stale snapshot cannot safely undo the cutover without
reconciliation. Mixed old/new writers and SQL downgrades are unsupported.

## Move a customer to dedicated resources

Provisioning a new customer on a dedicated database uses the same Tenant runtime.
Moving an existing customer's state additionally requires a fenced, verified
cutover preserving its issuer identity, signed bytes, original receipts, authority
and delivery obligations. Separate schemas make the storage boundary clearer;
they do not implement that trust-transfer protocol or live migration themselves.

WOS Cloud can manage placement, provision resources and call generic WOS operator
capabilities. It remains a separate private fleet manager. Automatic relocation,
autoscaling, cross-cluster scheduling, billing, external backup retention,
PostgreSQL HA and online mixed-version migration are separate capabilities.
