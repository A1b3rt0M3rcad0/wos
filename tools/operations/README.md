# Opt-in backup and recovery procedures

These tools are run by an operator or an external cron/systemd/Kubernetes job. They are not installed as a scheduler, service, agent runtime or automatic task in WOS. Python 3 is required; PostgreSQL also needs matching `psql`, `pg_dump` and `pg_restore` clients on PATH. Backups cover the whole WOS database/schema, including all internal Namespaces, not an agent's ordinary API scope. Use an operator database account and protect the output directory.

`backup_restore.py` takes a consistent online SQLite backup or exports one read-only repeatable-read PostgreSQL snapshot shared with `pg_dump`. It writes new private files, a manifest and checksums. Metadata includes operator-declared deployed source/version, storage schema version, server identity/current issuer fingerprint, Outcome revision watermark, snapshot time and per-table counts/logical digests. Source/version declarations identify the deployment supplied by the operator; they do not cryptographically attest which binary wrote every historical row. No issuer seed, token environment or CLI private key is collected. The database backup itself can contain private application data and credential digests; keep it private and apply your external encryption/access/retention policy.

## Backup

Use a **new** output directory under an existing protected backup parent. Never put passwords/DSNs in command arguments. The PostgreSQL URI is read from the named environment variable and passed to client tools through libpq environment fields; database names must be plain explicit names. The tool supports the listed libpq URI options and refuses unsupported options rather than dropping them.

```sh
umask 077
python3 tools/operations/backup_restore.py backup --backend sqlite --sqlite /data/wos.db \
  --output /private/backups/new-snapshot --source-commit DEPLOYED_GIT_SHA --product-version DEPLOYED_VERSION

python3 tools/operations/backup_restore.py backup --backend postgres --dsn-env WOS_POSTGRES_DSN --schema public \
  --output /private/backups/new-snapshot --source-commit DEPLOYED_GIT_SHA --product-version DEPLOYED_VERSION
```

Record the returned `manifest_sha256` independently of the backup bundle. Bundled checksums detect corruption; an independently approved manifest digest also detects replacement of both data and its bundled checksum. The tools do not replace trusted storage, signatures, approved issuer pins or access control. A failed/incomplete directory is not a completed backup; inspect the error and use a new destination for the next attempt.

## Inspect, restore and verify

Stop writers and pause agents for the explicit cutover procedure. Keep the old database/service and the original snapshot intact. SQLite restore requires a nonexistent destination file; PostgreSQL restore requires an already provisioned **new empty database** and refuses any existing user object. The tool never drops, cleans, overwrites or downgrades a running database. `--writers-stopped` records the operator's cutover precondition; it does not remotely stop processes.

```sh
python3 tools/operations/backup_restore.py inspect --backup /private/backups/snapshot \
  --manifest-sha256 INDEPENDENT_APPROVED_DIGEST

python3 tools/operations/backup_restore.py restore --backup /private/backups/snapshot \
  --sqlite /private/restore/new-wos.db --writers-stopped --manifest-sha256 INDEPENDENT_APPROVED_DIGEST

python3 tools/operations/backup_restore.py restore --backup /private/backups/snapshot \
  --dsn-env WOS_RESTORE_POSTGRES_DSN --writers-stopped --manifest-sha256 INDEPENDENT_APPROVED_DIGEST \
  --expected-server-id APPROVED_SERVER_ID --issuer-fingerprint APPROVED_CURRENT_ISSUER_PIN

python3 tools/operations/backup_restore.py verify --backup /private/backups/snapshot \
  --dsn-env WOS_RESTORE_POSTGRES_DSN --manifest-sha256 INDEPENDENT_APPROVED_DIGEST \
  --expected-server-id APPROVED_SERVER_ID --issuer-fingerprint APPROVED_CURRENT_ISSUER_PIN
```

A snapshot containing a server identity requires the independently approved server ID and current issuer fingerprint for restore, including SQLite signed-state snapshots. Obtain them from the existing approved deployment/CLI trust records, not by treating an untrusted backup as its own approval. A wrong pin fails before creating/writing the destination. An unsigned legacy snapshot has no issuer pin to check.

`verify` is read-only and compares every table's count and stable logical digest, schema migration data, identity/public-key history, grants/revocations, contracts, leases/fences, immutable signed facts/results, terminal return resolutions and outbox state. It checks preservation of those exact records; hashes alone do not certify their signatures, current authorization or work quality. After verification, migrate the **new** destination with the reviewed WOS binary, start it with the **same protected issuer**, confirm the approved public pin, and run the signed recovery/authorization pilot before reconnecting writers. Existing 0.4.2 snapshots are verified before forward migration. An old binary must refuse the new schema; rollback uses a separately restored compatible backup, never in-place table deletion.

The restore result reports `read_only=false`, `validation_read_only=true` and measured `rto_restore_and_verify_seconds`. That interval covers only this tool's restore plus logical verification. Full operational RTO also includes provisioning, issuer recovery, migration, signed validation and resuming agents. RPO is the gap between the incident/cutover watermark and the consistent snapshot; record both times/versions in the actual environment. A rehearsal's numbers are not a production SLA.

## Issuer and CLI workspace recovery

The issuer seed lives outside the database and outside this backup bundle. Protect a separate recovery copy and approved public-key history. Restore the original seed through the deployment's secret manager and verify its public fingerprint. A missing/wrong seed must cause an explicit unavailable/failure state; do not silently create another identity. If the original key is irrecoverable, use the existing explicit `wos issuer recover --file FILE` procedure and independently approve the new pin/history transition. Keep old public keys and receipts so old DSSE proofs remain verifiable.

Back up each CLI workspace separately, while its writer/keepalive process is stopped: `.wos` configuration, named profile bindings, pending-operation journal, original frozen payload/envelope, edited drafts, immutable archives and receipts. Protect any external key-provider/OS-keyring recovery material separately; a database snapshot does not recover a host keyring or missing local journal. Preserve exact bytes, IDs, expected versions, lease/fencing bindings and the original idempotency key. On restart authenticate/inspect the selected profile, run doctor and read-only status/receipt reconciliation. Never delete pending files or sign again just because a request timed out. An accepted return reconciles to the same proof; an officially closed rejected return uses its negative resolution receipt before a corrected new intent is prepared.

A stale backup can predate a credential/key revocation or external effect. Before cutover, reconcile security changes after the snapshot and reapply required revocations/pin approvals. Database restore does not undo receiver/tool effects. Receivers must deduplicate by stable `integration_event_id` and retain their own durable effect ledger independently of WOS. The two-server pilot deliberately kills APIs after the receiver commits its effect but before ACK, then demonstrates authoritative queue recovery and external dedupe, including restore into a new PostgreSQL database.

## External monitoring and rehearsal

Use the [external schedule examples](examples/) as templates; they are not activated by installation. Install this complete directory at `/opt/wos-backup`, make `wrapper.sh` executable and set the protected environment values in `examples/protected.env.example` to the actual deployment. The wrapper uses an exclusive new UTC-named destination and records the deployed source/version; cron should load the environment through an operator-owned launcher, systemd uses its protected EnvironmentFile, and Kubernetes uses its Secret. Recommended reference retention remains 24 hourly / 7 daily / 4 weekly copies with a monthly new-destination restore rehearsal. Your backup system implements retention, encryption and alerts externally.

Record/alert on:

- age of the newest completed/checksummed manifest and of the newest independently validated restore report;
- authenticated scoped delivery-health `oldest_due_age_seconds`, `oldest_lease_age_seconds`, `expired_leased`, `exhausted`, attempts and worker status (default due-age reference 5 minutes or any exhausted signal);
- allocated database/volume bytes against the operator's quota, filesystem free space and PostgreSQL pool wait/capacity;
- growth between successive snapshot `tables` row counts/digests for domain events, assessments, conclusions, signed facts/results/resolutions and delivery attempts; retain snapshot timestamps/watermarks with the growth series;
- issuer/credential/key-provider recovery availability and explicit approval of any trust transition.

Use database/volume monitoring for storage quota and growth, plus the manifest/restore timestamps for backup freshness. Do not turn an absent backup, a receiver outage or a large immutable history into a false claim that Core readiness certifies operational continuity. Rehearse execution, independent review, correction and terminal-resolution recovery with current authorization, exact proofs/versions/fences, revoked-token rejection and original receiver dedupe state. The acceptance reports distinguish these laboratory checks from production guarantees.
