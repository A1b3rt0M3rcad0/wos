# ADR 044: Tenant schemas, complete state ownership and protocol cutover

Status: Accepted by the repository owner for implementation in PR #114; qualification is tracked in ROADMAP.md.

## Decision

Tenant replaces Namespace as the product's isolation identity. A customer Account
normally owns one Tenant; Account signup, subscriptions and billing remain outside
the WOS domain. A Tenant ID addresses and authenticates the boundary. It is never
an authorization grant by itself.

In PostgreSQL, each Tenant owns one schema containing its complete WOS state:
domain records, identity and credentials, policies, immutable signed history,
receipts, integration queues and recovery metadata. The schema also owns its
database migration ledger. A fixed connection pool and application runtime bind
to exactly one Tenant. A schema cannot contain multiple Tenant identities.
Remaining `tenant_id` fields are identity/provenance and reference-integrity
guards; they do not distribute customers across shared operational tables.

An installation may host several such runtimes behind an explicit hostname map.
Hostnames select a configured runtime; its existing authentication and permission
checks authorize access. Unknown hosts have no fallback. No request field can set
a schema name or alter a pooled connection's search path. Pool and worker budgets
account for every hosted Tenant and replica.

A shared database hosts several Tenant schemas. A dedicated installation uses
the same code with a separate database. Dedicated CPU, memory and storage require
dedicated infrastructure; a separate database on the same PostgreSQL server does
not provide those resources by itself.

## Lifecycle and external management

A separate installation catalog records generated schema names, Tenant IDs,
placement state, versions and operation audit. It contains no second business
domain or competing authorization service. Provisioning becomes active only
after schema initialization succeeds. Interrupted provisioning is explicit and
can be resumed with a matching version.

Each live runtime and its data connections hold shared database lifecycle locks.
Suspend, resume and delete require exclusive ownership. Operators first drain
and stop all participating API and worker replicas. Suspension preserves data
and blocks new starts; deletion requires suspended or incomplete-provisioning
state and retains an audit/tombstone in the catalog. Tenant/schema identities are
not silently recycled. Automatic online draining and infrastructure autoscaling
remain external orchestration work.

## Public contract and existing data

Current HTTP, MCP, CLI, SDK and stored relational terminology use `tenant_id`,
Tenant and tenant administration. This is a breaking contract change. Signed
protocol 3 uses distinct v3 payload purposes, `tenant_id` and writer epoch 3.
New mutations never reinterpret a protocol-v2 request as protocol 3.

Historical migrations and signed protocol-v2 artifacts retain their original
bytes, spelling, digests and signatures. Explicit historical readers verify the
original purpose/version and project typed identity fields only after verification.
They do not rename customer metadata or change evidence. Mutable CLI profiles
require explicit migration; frozen pending operations must be reconciled using
the compatible prior client before cutover.

An append-only schema migration renames relational identifiers and advances
mutable protocol metadata after checking that active signed contracts and open
reviews have been settled. Existing multi-Tenant storage requires an explicit
offline placement transition; binding a new runtime must never silently discard
other Tenants or choose the first row. Migration, backup and restoration must
include all identity, signed-history and queue state belonging to the boundary.

## Verification

Required checks cover schema/credential/hostname isolation, connection reuse,
replica lifecycle exclusion, interrupted provisioning, deletion retention,
historical signed-byte preservation, v2 mutation rejection, current v3 journeys,
CLI profile migration, old-schema upgrades, and shared/dedicated storage parity.
Earlier PR scaling measurements remain evidence for their original source; they
do not qualify this architectural change.
