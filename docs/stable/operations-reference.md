# WOS operational reference

This reference completes the deployment procedures requested by the stable plan. It is an operating policy and measured fixture envelope, not a managed hosting SLA. Local installation remains `cp .env.example .env && docker compose up --build -d`; the public example credential is only for that loopback demonstration.

## Capacity and context

The initial reference envelope separates reads from writers:

| Operation | Demonstrated scope | Boundary |
| --- | --- | --- |
| Dense continuity / focal work reads | 1,000 WorkItems in an Outcome; one required criterion and one explicit assessment per rich WorkItem; 100 sequential samples | Full continuity reconstructs the full graph even when only 25 items are returned |
| Signed acquisition / return / independent review / correction | Existing load matrix: 1 or 4 Outcomes, 2 or 8 consumers, 0 or 16 historical completions per Outcome, with required criteria | This does not certify 1,000 simultaneously signed writers or a 10,000-item signed history |
| Same-Outcome contention | Existing work-contract matrix including 50 consumers and measured conflicts/retries | CAS and Outcome guards remain enforced; many consumers do not imply parallel writes inside one Outcome |

Local reference host: Linux amd64, Go 1.27.2, Intel Xeon Platinum 8573C, GOMAXPROCS=3. At 1,000 items, PostgreSQL rich continuity mean/p95/p99 was 82.48/120.33/136.92 ms, 32,643 response bytes, 12.73 MB allocated per call. Focal context was 15.35/20.91/28.78 ms, 2,225 bytes, 108.6 KB allocated per call. Three concurrent readers measured 24.36 snapshots/s. SQLite rich/focal means were 68.95/12.63 ms, with 32/34 SQL statements per call. PostgreSQL SQL statement count was unavailable. These local measurements preceded the clean-source profiling receipt and are not that receipt.

Reference read targets for this host/fixture are rich snapshot p95 ≤150 ms, focal context p95 ≤35 ms and three-reader rich throughput ≥20 snapshots/s at 1,000 tasks. The local unprofiled measurements above met these targets on both stores. These targets are deliberately limited to the measured reads and hardware; they are not a signed-write or deployment SLA. Profiled runs include instrumentation overhead.

Measurements use the trusted embedded Application query fixture, not HTTP/MCP authentication or a remote production ingress.

For agents, start with the issued obligation and `show --for-agent`, then request focal context and exact immutable references when needed. Avoid a full continuity snapshot on every action. Paging a continuity section still reconstructs the full Outcome; it is not a reconstruction-cost workaround. Focal reads already avoid loading unrelated task graphs. Dense full-state queries remain a known scaling cost; optimize their SQL projection only if the deployment actually needs frequent full snapshots. Do not weaken guards, CAS, policy or fencing to improve a chart.

Reproduce and profile both stores on a clean checkout, with a dedicated PostgreSQL test database configured through `WOS_TEST_POSTGRES_DSN`:

```sh
GOMAXPROCS=3 python3 tools/acceptance/continuity-capacity.py \
  --history 100 1000 --samples 100 --output dist/continuity-capacity
```

CI records the immutable source, actual four read cases per store/history, allocations, bytes, empirical quantiles, pool wait, optional SQL count, Go-process CPU and RSS high watermark, and CPU/heap profiles. CPU/RSS include fixture construction and compilation and exclude the independent PostgreSQL process. Use an external host/database collector for whole-deployment resource use. A 100-sample p99 is a sample quantile, not a production tail estimate. `--history 10000` is an explicit larger experiment; 10,000 is not accepted capacity until measured. Set deployment limits from its own hardware and workload.

## Retention and backup policy

| Data class | Reference policy |
| --- | --- |
| Immutable criteria, assessments, conclusions, submissions, findings, signed specs/authorities/acceptances, public key/trust history | Retain while the Outcome or any accepted proof is retained; no generic TTL purge |
| Aggregate state, leases, current protocol/policy versions, credentials/grants/revocations, sessions, admin audits/receipts | Include in every consistent backup; preserve revocation and fencing through restore |
| Domain/Integration events, trigger firings and delivery attempts/outbox | Preserve stable identities and pending delivery state; receiver deduplication must cover retry/redelivery and restore horizons |
| Checkpoints and documentary history | Retain with the obligation; archiving is not deletion; external artifact storage has its own version/checksum retention |
| Ordinary Outcome idempotency cache | Existing default seven days; expiry does not delete durable signed operation results or establish remote failure |
| Local pending intentions, drafts and receipts | Host-owned protected workspace; only exact verified acceptance permits cleanup; back up separately if the host requires it |
| Private signing keys / bootstrap secrets | Separate protected secret-manager backup and rotation history; never copy them into public DB receipts or this repository |

For the reference installation, schedule a consistent backup hourly, retain 24 hourly snapshots, 7 daily and 4 weekly snapshots, and perform a clean restore monthly and before upgrades. These are recommended operating targets, not a new background scheduler inside WOS. Encrypt snapshots, restrict access to the instance operator and protect at least one independent copy. Do not purge the last verified restore point. Record the last successful backup and age, failed jobs and restore verification.

An hourly schedule has an intended RPO of at most one hour **only while successful backups and their delivery remain within that interval**. Restore target RTO is one hour for the reference envelope; this is a target to validate per installation, not a measured promise. Missed/slow backups increase RPO. Fixtures restoring in seconds do not establish production RTO.

Use the same tested binary/configuration as the service. SQLite supports consistent live `VACUUM INTO`, not copying a WAL database's main file:

```sh
umask 077
snapshot="/srv/wos-backups/wos-$(date -u +%Y%m%dT%H%M%SZ).db"
wos db backup "$snapshot"
sha256sum "$snapshot" > "$snapshot.sha256"
```

The destination must be new. Configuration selects the real SQLite path; use the operator's protected environment rather than putting credentials in command arguments. The host's cron/systemd/backup service schedules this command; WOS does not launch it. Keep schedule and retention outside the Core. PostgreSQL uses matching `pg_dump --format=custom` and `pg_restore --single-transaction` clients with protected libpq configuration; include the entire database/schema, not only current entities. A dump contains authorization and history and requires protected storage.

## Clean restore and upgrade

1. Withdraw ingress and stop all service/worker writers for the affected installation. Preserve the current database and all post-backup external effects for reconciliation.
2. Verify the snapshot checksum and source/version manifest. Restore into a **new** database/file. SQLite: `wos db restore SOURCE DESTINATION`. PostgreSQL: restore the custom dump with a single transaction into a new database. Never overwrite the live installation by approximation.
3. Configure the preserved server identity, current issuer and protected secret references. A DB backup stores public trust, not the lost private signer. If unavailable, use the documented host issuer recovery; never generate an unannounced identity.
4. Run `wos db migrate` with the intended binary against the restored target, then start on a private temporary port. Verify `/readyz`, authenticated HTTP/MCP reads, the same server/Namespace identity and existing revocations.
5. Check pending execution, review and correction authority: versions/fencing, immutable submission/criterion/findings digests and local pending intentions. Database time determines expired authority; restore does not renew a lease. Recover the original accepted receipt/intention before making another mutation.
6. Inspect outbox pending/delivered states and replay a controlled acceptance. Receivers deduplicate by stable Integration Event identity. Restore may repeat external delivery; it cannot undo effects outside the database or provide exactly-once delivery.
7. Record backup timestamp, observed data loss/reconciled effects and elapsed restore-to-ready time. Reopen ingress only after these checks; keep the previous installation isolated until reconciliation finishes.

Numbered migrations are checksummed and forward-only. Back up before upgrading, withdraw old writers before signed cutover, and preserve legacy unsigned history explicitly. Prefer a forward fix once new-schema facts exist. Rolling back to an older binary uses a pre-upgrade backup in another database and may lose/reconcile later facts; an old writer must never write a v2 authority schema it does not understand.

Existing executable evidence includes `TestSignedReviewCorrectionJourneyWithoutExecutorAuthority`, which restores clean SQLite/PostgreSQL databases at execution, review and correction stages, verifies exact authority bytes/version/fence and continues the authenticated journey. Issuer recovery/history, durable cacheless replay, credentials/revocations, planning/criteria/history and outbox restore have separate contract/runtime tests. Memory has no database restore contract. Full Go CI binds executed cases to the source; it does not certify the operator's schedule or RPO/RTO.

## Diagnostics and reference alerts

Poll authenticated JSON `/metrics`, readiness and structured logs with an external collector. Counters/maxima/sums reset on process restart: detect resets and derive rates from nonnegative deltas only. The service does not export production percentiles. Compute histograms/percentiles from collected durations externally. Never label by token, payload, arbitrary documentary content, credential value or webhook URL.

| Symptom / reference alert | Evidence and next step |
| --- | --- |
| `/readyz` fails twice in 60 seconds | Check configuration, migration checksum/version, database reachability and persistent issuer/trust; `/livez` alone does not establish readiness |
| Transaction conflicts exceed 5% over five minutes with at least 100 commands | Correlate Outcome and operation, expected versions, transaction/guard acquisition and SQL pool wait; retain the caller's exact intent on retry |
| p95 observed command latency exceeds the installation's measured target for five minutes | Use externally collected durations; distinguish wait/commit/query costs before changing pool/concurrency |
| Outbox has an eligible undelivered event older than five minutes, or any exhausted delivery | Inspect classified attempt outcomes and worker heartbeat; verify allowed destination/TLS and receiver deduplication before authorized redelivery |
| No successful backup in 65 minutes | Report increasing RPO and backup failure; never advertise the one-hour target as met |
| Repeated 401/403 or issuer mismatch | Distinguish revoked/expired credential, permission/policy, profile destination and issuer trust; do not retry with a broader Principal automatically |
| Committed receipt but local cleanup failure | Host filesystem/lock/ENOSPC problem; preserve the draft and run exact `wos work recover` / `wos review recover` |

Transaction/guard timing is not a pure blocked-time metric. SQLite waits mainly when acquiring a transaction; PostgreSQL also has database/connection wait. Service JSON metrics alone do not expose all PostgreSQL pool/SQL/backlog gauges; add external SQL statistics/log collection when needed and mark missing data unavailable. No empty metric is proof of an empty outbox or no contention.

Shared instances require fresh per-instance credentials, separately enrolled signing keys, protected durable storage, TLS at the ingress, private database networking and restricted admin metrics. Compose remains bound to loopback; terminate TLS in the installation's existing ingress and forward only to that private port. Set container memory/CPU limits from the measured workload and retain health checks, non-root UID 10001, persistent volumes and graceful shutdown. Do not reuse `.env.example` credentials or secrets between sold instances. Test the actual ingress with authorized and unauthenticated requests before exposing it; this reference does not assert that a production ingress was deployed here.

The clean-source profiling receipt is [capacity-2026-10-12.json](capacity-2026-10-12.json), produced at source `09de362daaed7696970f760ba6fa7cdd7ab25e60`. All 16 store/history/query cases completed with 100 samples each. The largest reported child RSS high watermark was 267,044 KiB; it includes Go/fixture work and is not whole-database memory. Profiled PostgreSQL 1,000-item rich/focal p95 was 97.32/26.20 ms with 26.24 parallel rich snapshots/s. SQLite rich/focal p95 was 117.73/21.66 ms with 22.83 snapshots/s. Empirical p99 spikes remain visible in the receipt; none is removed to make a target pass.
