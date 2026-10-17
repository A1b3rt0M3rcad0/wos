# Reproducible scalability qualification

WOS must be qualified as a persistent coordination flow: acquisition, delivery, independent assessment, acceptance, replay and subsequent reads. An HTTP health check or increased API replica count is insufficient evidence of capacity.

The runner [`tools/acceptance/scalability-report.py`](../../tools/acceptance/scalability-report.py) exercises the actual signed Application and SQL storage implementations. It preserves the existing signed workload's independent-review, original-intent retry, version-collision and outbox checks while varying Namespace distribution, independently pooled service replicas and accepted historical work. PostgreSQL is the shared production backend; SQLite remains separately measurable as a single-writer reference.

## What the fixture measures

Each case creates a disposable schema/file, real credentials and signing keys, explicit criteria and a persistent issuer. It creates historical Tasks through the signed work/review journey before the timed interval. Timed consumers then perform two further complete journeys each, concurrently with bounded `work_items` continuity pages.

| Dimension | Values |
| --- | --- |
| Contention topology | One Outcome; four Outcomes in one Namespace; four Outcomes across four Namespaces |
| Application/store replicas | 1, 2 or 4, each with its own SQL connection pool against the same state (PostgreSQL fixture default: 16 connections per replica; SQLite: one) |
| Accepted historical Tasks per Outcome | 0, 16, 100 or 1,000 |
| Concurrent executing/reviewing consumers | 2, 4, 8 or 50 |
| Repetitions | Default 3; explicitly bounded 1..10 |
| Read load | One closed-loop reader per consumer, bounded page of 25, 10 ms between attempts |

Every timed delivery must finish independent review, retain one completion event and one stable outbox identity, and replay the exact original approval receipt. A separate deliberately conflicting acquisition must retain one winner and reject each losing original version. Only transaction conflicts retry an unchanged mutation; the fixture does not advance expected versions to force success. Read transaction conflicts are recorded separately from successful-read quantiles and are retried as the same read query on the next loop; non-conflict read failures fail the fixture.

The runner records complete-journey and successful-read p50/p95/p99, accepted throughput, read attempts/conflicts, transaction/guard waits, total pool waits, exact mutation retries and response sizes. Whole-child CPU/RSS includes seeding and fixture setup but excludes the database process. Sample counts remain explicit: eight accepted journeys do not establish a production p99.

**Boundary:** service replicas are independent Application/Store instances in one precompiled Go process. This isolates shared SQL coordination and pool behavior; it does not measure independent API processes, load balancing, TLS, browser rendering, network latency or delivery-worker throughput. The [two-server delivery pilot](../../tools/acceptance/two-server-delivery.py), transport tests and actual deployment smoke cover different boundaries. Do not label service replicas as measured HTTP workers.

## Run against immutable sources

Requirements are Linux, the repository's Go version and a disposable real PostgreSQL database. Set `WOS_TEST_POSTGRES_DSN` through protected environment configuration; logs/reports do not record the DSN or credentials. The Go fixture creates uniquely named test schemas. Do not target a production database. Control host/database resources and keep the same statistics/configuration between compared sources.

From one clean committed source:

```sh
python3 tools/acceptance/scalability-report.py \
  --output /private/wos-scale-report \
  --backend postgres --replicas 1 2 4 --history 0 16 \
  --consumers 4 --repetitions 3
```

For a bounded first check, use `--replicas 1 --history 0 --repetitions 1`. Add 100/1,000-history and 8/50-consumer cases deliberately: seeding and contention can take substantially longer. Each case has a default ten-minute bound; failure or timeout retains an incomplete report and its private raw logs. A skipped SQL fixture, partial result, non-finite metric, wrong active topology or missing invariant cannot qualify.

For before/after measurement, create immutable Git archives outside the working tree, copy exactly the same reviewed `signed_load_test.go` harness and runner into both, and supply `--root /archive/path --archive-commit EXACT_40_CHARACTER_SHA`. This overlay is recorded explicitly. A report contains hashes of actual measured Go/SQL/tool inputs as well as the claimed source commit; an overlaid harness is not misrepresented as existing in an earlier release. Never change either archive while it runs.

The explicit `--allow-dirty` development mode records actual source hashes and marks immutable qualification false. It must not be presented as final committed-source evidence. Output must remain outside the repository, is created with private-directory permissions, and is rejected if source inputs change during measurement.

## Compare the right things

Compare each topology/replica/history combination separately, using all raw repetitions. Include read conflict rate alongside successful-read latency, and mutation conflicts alongside accepted throughput. Do not compare a four-Namespace candidate to a one-Namespace baseline, hide failed runs, or treat faster success-only quantiles as evidence of fewer errors.

Capture PostgreSQL version, assigned CPU/memory, active competing workloads, storage/network properties and tuning separately with the report. The runner records host, Go and `GOMAXPROCS`; it does not infer database CPU/RSS, WAL pressure, disk I/O, storage growth or a database's resource allocation from the Go process. Use PostgreSQL observation alongside load tests for those measurements.

The existing [operational read harness](../../tools/acceptance/operational-read-report.py) separately controls 100/1,000/10,000-item rich reads, allocations and SQL statements. This new fixture grows real signed accepted history while holding the number of timed Tasks constant; completed Task rows also remain in current relational storage, so this is not an isolated single-aggregate history benchmark. Neither substitutes for a long-duration storage-growth test, external failover/PITR drill or concurrent security-race tests.

Required correctness remains independent of performance: no duplicate authority, no stale fence acceptance, no revocation bypass, no quota oversubscription, no cross-Namespace disclosure, no implicit criterion approval, and no loss of an accepted original receipt after restart/recovery. Runtime and storage contract tests enforce those rules alongside the load fixture.

## Capacity and delivery decisions

A production capacity statement needs a representative workload and declared latency, error, queue-age and recovery objectives. Record at least:

- Distribution of active Namespaces, Outcomes and hot aggregates.
- Active versus historical Tasks, criteria/proof size and retained receipt volume.
- Read/write mix, payload sizes, polling frequency and integration destinations.
- API/worker replica counts and the sum of every connection pool plus operational reserve.
- PostgreSQL query/lock/pool waits, I/O/WAL, storage growth and maintenance behavior.
- Crash, restart, receiver-deduplication, backup/restore and upgrade results.

Use the [public deployment profiles](../../deploy/scaling/README.md) to grow from one complete PostgreSQL process to API and worker replicas. PostgreSQL HA, rolling binary/schema compatibility, Namespace mobility and storage sharding have separate acceptance requirements; these measurements do not certify them.
