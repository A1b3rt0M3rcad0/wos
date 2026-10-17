# Installation growth and mobility

WOS can start on SQLite and move the **whole installation** to PostgreSQL with an explicit maintenance window. The native `wos storage` operator commands preserve all Namespaces, global Principals, credentials, public issuer identity/history, immutable signed bytes, durable command receipts and integration delivery state. It never changes routing or collects private signing keys.

Run actual storage operator commands on the **Linux WOS server host**. The shipped Windows client exposes their help but refuses storage operations before accessing source data or destination credentials. Private staging relies on Linux/POSIX filesystem permissions; Windows DACL protection and Windows/macOS host operations are not qualified. The command does not copy data or imply protected staging on unsupported hosts.

The existing [backup and restore procedures](../../tools/operations/README.md) support whole-installation SQLite-to-SQLite and PostgreSQL-to-PostgreSQL moves. The cross-backend command adds SQLite-to-PostgreSQL. PostgreSQL-to-SQLite conversion, online replication, tenant extraction, automatic failover and transfer between unrelated installation identities are not provided by this command.

## SQLite to PostgreSQL

Requirements: the native `wos` binary matching the upgraded SQLite schema, Python 3 for the backup procedure, a PostgreSQL operator account that owns a **new empty database**, and the approved public server identity/issuer fingerprint if the installation has signed state. Migration and preflight are included in the shipped binary; no Go toolchain is needed to run them. They do not start the server, bootstrap credentials or load an agent workspace. Use `wos storage --help` or `wos storage migrate-sqlite --help` for local discovery.

1. Stop and fence **all** source APIs, delivery workers, maintenance jobs and other writers. Pause clients and external integrations for the cutover. `--writers-stopped` records this operator precondition; it does not remotely stop anything.
2. Take a new consistent SQLite [backup](../../tools/operations/README.md#backup) after stopping writers. Retain the backup, its independently recorded manifest digest, the original database and the existing deployment configuration. Keep the issuer seed and integration secrets under the operator's existing secret management, separately from the backup.
3. Provision a new empty PostgreSQL database. Keep all destination WOS processes stopped. Set its URI in `WOS_MOBILITY_POSTGRES_DSN` through protected environment configuration; never place a password or URI in command arguments.
4. Run the migration using the independently approved backup digest and public identity pins:

```sh
umask 077
wos storage migrate-sqlite \
  --backup /private/backups/stopped-installation \
  --manifest-sha256 INDEPENDENT_APPROVED_DIGEST \
  --dsn-env WOS_MOBILITY_POSTGRES_DSN \
  --expected-server-id APPROVED_SERVER_ID \
  --issuer-fingerprint APPROVED_CURRENT_ISSUER_PIN \
  --writers-stopped > /private/migration-result.json
```

For an installation that has never established a signed issuer identity, omit both public identity flags. Signed state without an installation identity is refused. The destination is always the `public` schema of a new empty database. An existing destination is refused, including an earlier migration attempt that initialized its schema. A failed attempt can leave the new database's canonical empty schema, but cannot commit a partial application-state copy; use a new database for a retry.

The tool validates and parses the same manifest bytes against the independent pin. It copies the database through one opened stream into a private temporary directory while hashing, validates that exact copy, and opens it immutable/read-only. Adjacent WAL files and later source-path replacement cannot supply unapproved state. The private staging copy is removed after use. Allow temporary disk capacity for one full SQLite snapshot. It then validates SQLite integrity/foreign-key closure, every compiled SQLite migration checksum, and the complete source table/column inventory. Older or newer schemas are refused: upgrade using a compatible WOS binary and take another backup first.

The destination uses the compiled PostgreSQL migration chain. Dialect-specific migration checksums and application timestamps remain PostgreSQL metadata rather than being replaced by SQLite migration records. All other tables are included automatically, including new delivery scheduling tables. Unknown or incompatible column types fail closed.

The application-state import runs in one transaction with exclusive destination table locks. Foreign keys are temporarily made deferrable **inside that transaction**, every reference is validated, and the original constraint flags are restored before commit. Per-table row counts and ordered SHA-256 digests compare source and destination before committing. Memory use is bounded by the current row; database sorting and the destination transaction still require capacity appropriate to the installation. This is an offline copy, with a 30-minute command deadline, and has no production throughput or recovery-time guarantee.

The structural representations of unsigned counters differ: PostgreSQL `NUMERIC(20,0)` receives exact decimal values, including values above signed 64-bit range. SQLite's fixed-width activity sequence becomes its equivalent numeric value. Noncanonical authority counters are refused. Signed JSON, envelopes, receipts, specs and documentary text remain byte-for-byte strings; signed `server_id` bindings are never rewritten.

5. Review `completed: true`, `verified: true`, the public pins and all table results. Configure the destination WOS deployment for PostgreSQL with the **same logical installation identity and existing issuer seed**, verify issuer readiness and restore required integration secrets. This is an operator move of the same installation; never distribute a shared installation's seed to a tenant-owned installation.
6. Start the destination, verify authentication and representative persisted state, then resume clients and deliveries. Ensure clocks satisfy the deployment's lease policy. Time elapsed during the maintenance window does not extend contracts or delivery leases. Keep the source fenced permanently while the destination accepts writes.

Before any destination business write, rollback can return to the intact source under an explicit routing cutover. After destination writes or external delivery effects, returning to the old snapshot can lose acknowledged work or repeat effects. Pause and reconcile that state; the command does not perform an automatic reverse migration or merge divergent databases.

## Shared-to-dedicated Namespace preflight

Tenant relocation is a separate trust protocol. A Namespace references global Principals, public installation issuers, exact-intent receipts and indirectly scoped delivery records. Its signed facts bind to the source installation. Copying rows selected only by `namespace_id`, rewriting signed payloads, or giving a tenant the shared private issuer key is unsafe.

The command offers a **read-only preflight**, not an export/import operation:

```sh
wos storage namespace-preflight \
  --sqlite /private/snapshots/installation.db \
  --namespace NAMESPACE_UUID > /private/namespace-preflight.json

wos storage namespace-preflight \
  --dsn-env WOS_MOBILITY_POSTGRES_DSN \
  --namespace NAMESPACE_UUID > /private/namespace-preflight.json
```

Each preflight reads one coherent snapshot and reports scoped table counts; SQL foreign-key closure and cross-Namespace references; the number of referenced Principals also used elsewhere; revoked credentials, browser sessions and invalid parent links; unexpired execution/review authority; delivery states and indirect attempt/firing counts; public issuer history; and tables that require global or indirect closure.

`ready_for_transfer` is **always false**. Exit status zero means the inspection completed, never that a Namespace can be moved. The output contains counts and public trust metadata, not tokens, signed payload contents or private keys. It still belongs in protected operator storage. SQL foreign keys cannot establish closure for every signed payload, documentary reference, polymorphic reference or external side effect; the manifest explicitly records that limit.

Actual shared-to-dedicated transfer remains blocked on these executable protocol capabilities:

- A durable source write fence and bounded cutover, including authorization and active-contract handling.
- Complete domain-level reference closure, with explicit treatment of global Principals and any references outside the selected Namespace.
- Preservation and verification of historical source issuer trust, destination enrollment and client trust-pin transition.
- Exact-intent receipt, pending delivery and already-executed external-effect reconciliation.
- Destination validation, interruption recovery and rollback that never allows both installations to grant conflicting authority.

These are generic WOS responsibilities available to self-hosted operators. WOS Cloud can later orchestrate them alongside provisioning and infrastructure placement without reproducing the domain or embedding tenant transfer into a private-only service.

## Optional source-tree wrapper

Developers can still build `tools/operations/mobility` as a separate wrapper. It delegates to the same public non-domain package, `packages/wos-api/operations/mobility`, as `wos storage`. The previous `migrate-sqlite-to-postgres` wrapper spelling remains an alias for `migrate-sqlite`; there is one migration implementation. Normal self-hosted operation uses the native commands above.

## Verification

```sh
python3 -m unittest discover -s tools/operations -p 'test_*.py'
WOS_TEST_POSTGRES_DSN=YOUR_ISOLATED_TEST_DATABASE_URI \
  go test ./packages/wos-api/operations/mobility ./packages/wos-api/cmd/wos -count=1
```

The Go tests create and drop only uniquely named isolated PostgreSQL databases using a test account with database-creation privileges. PostgreSQL tests skip explicitly when the DSN is absent; such a run does not qualify cross-backend migration. The fixtures cover the full compiled table inventory, real Ed25519 verification after migration, exact durable receipt reconstruction, authentication/authorization, shared Principals, maximum uint64 values, padded activity sequence conversion, pending deliveries and indirect attempts, original foreign-key flags, tampered backup/identity/migration-chain refusal, unapproved adjacent WAL and source-path replacement, and transaction rollback after invalid authority. They qualify these fixtures rather than every possible production dataset or an online tenant-transfer protocol.
