# Tenant schema qualification

This record covers the schema-per-Tenant and signed-protocol-3 change in
[PR #114](https://github.com/A1b3rt0M3rcad0/wos/pull/114). It supplements the
[operating guide](tenants.md). The [earlier scaling measurements](validation.md)
retain their original sources; they do not establish capacity for this change.
No merge, release publication, production SLA or automatic Cloud orchestration
is implied by these checks.

## Executed boundaries

The laboratory uses Linux amd64, Go 1.27.2, bounded package execution and real
PostgreSQL 18.6. Native Go commands use `GOMAXPROCS=2`. PostgreSQL has 512 MiB shared
memory, 1,024 transaction locks and 4,096 predicate locks. Test databases, schemas,
containers and credentials are disposable.

The implementation at `790dcc2402ce68507d65664974414feec1705272` introduces
the catalog, fixed-schema runtimes, host routing, strict v3 mutations and explicit
historical readers. Commit `97b1978` adds pre-migration refusal for unsettled
signed authority. Commit `6f085c5` extends the preflight to unsigned
contracts and legacy leases before any intermediate migration advances the source.
Final production source `f21d454f8d84da398bc9e908054b2b20e581544f` also waits
for PostgreSQL to release initializer sessions before completing provisioning or
migration. Destructive lifecycle operations still refuse live sessions immediately.

The final test harness is `e81c20643ac0a63ea59cf6d72e4c787f22f40434`.
Its 465 production Go/SQL inputs are identical to `f21d454`; subsequent changes
only correct test shutdown observation and reduce repeated fixture initialization.
Business tests copy an empty migrated SQLite snapshot into independent private
files. Existing files, cold migration/upgrade checks and direct startup tests
still execute their original initialization paths. Server snapshots contain no
Tenant, credentials or issuer identity. PostgreSQL tests retain native isolated
schema initialization. No production initialization shortcut is introduced.
The cold-start regression uses the existing 120-second startup default and still
checks explicit cancellation plus an independent one-millisecond writer limit.
A diagnostic run exceeded that test's earlier explicit 30-second allowance under
race instrumentation; no production timeout is increased by its correction.
The diagnostic SQLite package took 667.43 seconds, exceeding Go's default
ten-minute package limit. CI now applies a bounded twenty-minute package timeout
to its full normal/race commands, retaining the cold historical migration checks.

Focused checks exercise:

- Separate Tenant credentials, identity, current state, immutable history and
  queues, including reused connections and unknown-host refusal.
- Live API/data-pool lifecycle exclusion, lost provisioning connections,
  interrupted initialization, cross-schema dependency refusal, suspended-only
  migration, deletion and permanent tombstones.
- Exact compiled migration-ledger verification without active-runtime DDL;
  future-schema refusal and unchanged recovery sentinels.
- Offline shared-schema import, complete scoped state, RLS refusal, unknown
  objects, atomic rollback and an always-enabled retained-source write fence.
- Unchanged protocol-v2 proof bytes and cryptographic verification; explicit
  profile migration and refusal of unresolved frozen operations.
- Current signed execution, independent review, approval and requested changes.

Both final integrated Go suites pass **1,676 test/subtest entries in 22 packages**.
Each has 14 explicitly skipped optional fixtures and ten packages without tests.
Normal execution uses package parallelism one; race execution uses two, with
separate PostgreSQL databases and actual backup/restore enabled. Both use a
twenty-minute package timeout. The committed CI runs both suites serially.

The [verification receipt](evidence/tenant/validation.json) pins the production
source, final test harness and hashes of 795 Go/SQL/module inputs. It identifies
valid Go package-cache results rather than treating cached entries as fresh
executions. Skips cover opt-in capacity fixtures, subprocess-child entry points,
native-keyring/full-disk prerequisites and unsupported SQLite worker cases.
A failed diagnostic run is never counted as a passing qualification.

## Real published-version upgrades

The current candidate uses production source `f21d454` for both published-version
scenarios. The v0.2 producer uses pinned source
`b75aa3e0ca6d649cafadf8450d9b5d606367dda4`. The executed SQLite and PostgreSQL
scenarios advance schema 19 to 33 after settling authority with the old binary.
They preserve the original contract, submission, two criteria and ten command
receipts. Seven original requests replay before cutover and are rejected under
the new contract afterward. PostgreSQL explicitly imports the old shared schema
into a suspended catalog placement, verifies it, then resumes it.

The old v0.2 migrator does not reject an unknown future schema. Its unsuccessful
readiness and mutation attempts, and the absence of writes, are recorded
separately. This is a breaking offline transition; mixed-version serving is not
supported.

The v0.4.2 producer is pinned to
`b572269085494ddd4d6370716ff1cb360b46065f`. Its acceptance runner preserves
the original production sources, independently verifies historical DSSE, and
requires refusal of an active-authority probe before any source-row or migration
ledger change. Settlement uses the old authenticated API before backup and
cutover. Each backend preserves 67 original tables and verifies 39 historical
DSSE proofs without re-signing. Its final report is recorded separately from v0.2.

| Boundary | Public receipt |
| --- | --- |
| Published v0.2 to schema 33, SQLite and PostgreSQL | [v0.2 cutover](evidence/tenant/v020-cutover.json) |
| Published v0.4.2 to schema 33, SQLite and PostgreSQL | [v0.4.2 cutover](evidence/tenant/v042-cutover.json) |
| Two APIs, failed acknowledgements and whole-database restore | [Recovery](evidence/tenant/two-api-recovery.json) |
| TLS, API/worker replicas and drain-first lifecycle | [Compose lifecycle](evidence/tenant/compose-lifecycle.json) |

## Browser and operational recovery

Eighteen real-server Chromium checks pass: five authority journeys, five canvas
race regressions, two desktop/mobile workspace journeys, four inspector checks,
and two signed review journeys. The signed tests use a writable `XDG_CACHE_HOME`
in this managed environment. All 38 web unit checks and 15 release/compatibility
checks also pass. These local browser/unit logs originate from the Tenant
implementation worktree before or at `790dcc2`; their executable commit metadata
was not separately captured, so they are not relabelled as final-head executions.
The hosted Browser acceptance job independently passes on exact source
`6ac6f17eaac4398abfc6d56497f371301ce3e99c`:
[run 39083660857](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/39083660857).
These checks do not replace representative-user acceptance.

The two-API failover pilot at `f21d454` completes 100 deliveries after killing the
original API while external effects await acknowledgement. The receiver observes
104 attempts and 100 unique effects, deduplicating four repeats by integration
event ID. There are zero duplicate authoritative success acknowledgements.
Delivery remains at least once; the receiver's deduplication is necessary.

A separate eight-message recovery scenario stops both APIs, dumps and restores
the whole database, and compares every catalog and Tenant table hash. Issuer
identity, credentials and queue state survive. Restore plus verification takes
2.418 seconds in this small fixture; this is not a production recovery objective.

The Compose pilot records the actual initializer, complete-to-replicated
transition, TLS/session persistence, API/worker restarts and lifecycle operations.
The final eight-check pilot uses the same complete production source `f21d454`.
Its receipt distinguishes a normal Dockerfile network build from the temporary
verified-module-cache build used when the managed network rejects a module
download redirect. TLS verification, checksum verification and inherited proxy
configuration remain enabled.

Four ten-second fuzz campaigns cover canonical payloads, envelopes, signed-return
resolution and work-payload YAML. They complete 105,524 executions without a
crash. Fuzzing is a bounded robustness check, not an exhaustive proof.

## Remaining deployment responsibilities

Operators still stop and drain legacy writers before upgrading. Lifecycle guards
protect cooperating runtimes; they do not remotely stop old processes. Suspension
preserves data, and schema deletion does not erase retained sources or backups.
Production capacity, database HA, storage quotas, automatic placement, online
movement, infrastructure autoscaling and commercial billing require separate
operational qualification or external management.

Direct PostgreSQL/session-pooling connections are required. Transaction or
statement pooling is outside the tested lifecycle-lock contract.
