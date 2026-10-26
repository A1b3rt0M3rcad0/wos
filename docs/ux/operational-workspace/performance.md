# Operational workspace performance

The workspace has a reproducible F08 backend benchmark and bounded read contracts. The measurements below describe this development runner; the [plan's](owner-plan-2026-10-19.md) **400 ms p95 target on 2 vCPU / 4 GiB, SSD and controlled 50 ms RTT remains unqualified**. These are engineering observations, not a product SLA.

The final quiet run was sequenced after the full database, race, upgrade and browser checks stopped. Earlier measurements remain available below, including runs that exceeded the target.

## Source and environment

The measured core implementation is committed through [`505f2e6`](https://github.com/A1b3rt0M3rcad0/wos/commit/505f2e637ec37a189c82365fcc7102cdaf038490). The checkout was [`076550b`](https://github.com/A1b3rt0M3rcad0/wos/commit/076550b2fbf8e2f9ddede5b1efc440568c6e0edf), on `fix/workspace-operational-density` (base `4d33dced5cd8e17036f94f3b1401a86f2c433899`). The core and canonical harness files remained unchanged throughout both runs. The successful PostgreSQL run used the separately recorded, setup-only overlay described below. The source fingerprint, canonical harness hashes and executed overlay hash are recorded in [source-and-runner.json](performance/source-and-runner.json) and [quiet-run-environment.json](performance/quiet-run-environment.json). The fingerprint includes non-test Go/SQL under the core application, domain, ports and adapters, the module files and PostgreSQL generator; documentation changes do not change it.

| Condition | Measured runner |
| --- | --- |
| CPU | Intel Xeon Platinum 8573C; affinity exposes 5 CPUs; cgroup quota permits 4 CPU equivalents |
| Go scheduler | `GOMAXPROCS=2`; this does not limit PostgreSQL to two CPUs |
| Memory limit | 32 GiB for the runner, not the reference 4 GiB |
| Toolchain | Go 1.27.2, Linux amd64 |
| SQLite | SQLite 3.53.4 via `go-sqlite3` v0.35.6; local temporary database under `/tmp` on tmpfs; one pooled connection |
| PostgreSQL | PostgreSQL 18.6 (Debian); real local database over TCP; 16 pooled connections |
| Schema | Migrations through 0032, including indexed timeline and acquisition-time reads |
| Cache | Fixture preloaded and `ANALYZE` completed before timing; no OS/database cache eviction |
| Transport | Application service calls plus JSON serialization; no HTTP, browser rendering, credential authentication or artificial RTT |
| Concurrency | 20 batches of 10 concurrent reads of the same Task; not ten independently authenticated browser sessions |

Database/runtime versions were read directly and retained in [database-versions.json](performance/database-versions.json).

The PostgreSQL tracer reopens its connection pool before the timed cases. Single-read cases can warm one connection; the concurrent case can open more. Connection establishment is included when it occurs.

The SQLite statement tracer uses `TRACE_STMT`; PostgreSQL uses pgx query tracing. Counts include the operations seen by each adapter's tracer and should not be interpreted as identical network round trips across adapters.

## Fixture and measurement

[The canonical harness](../../../packages/wos-core/storage/sqlite/workspace_f08_benchmark_test.go) constructs and validates one Outcome with:

- 100 Objectives, including five required for the Outcome.
- 1,000 Tasks distributed across those Objectives, with one current criterion and explicit assessment per Task.
- 5,000 Evidence records; 1,000 are referenced by the assessments.
- 10,000 notes, each with its canonical `entity_note.recorded` event: 10,000 timeline entries, rather than two entries for the note and its event.

This is a legacy-protocol read fixture. It does not acquire or execute Tasks, create a signed review, or turn an assessment into Task completion. Signed execution and independent review are covered by separate functional tests; their throughput is not measured here.

Fixture construction, migrations and `ANALYZE` are outside the timed section. Each single-read case has 20 observations. The concurrent case has 200 per-request observations. Percentiles use the nearest observed rank, so p99 of a 20-observation case is its maximum; these short runs do not establish statistical service guarantees. Timing includes application projection and JSON serialization, with response size measured before HTTP framing or compression.

The first inspector loads recent activity and current facts, with declared previews and pagination. It does not load all 10,000 note bodies. The card case performs one batch for 100 explicit references rather than opening 100 inspectors. Structural navigation asks for 50 direct children. The timeline case reads five recent entries for one Task.

### PostgreSQL setup failure and retry

The first final PostgreSQL attempt failed before any timed read. Its untimed, unqualified `ANALYZE` reached the adapter's 30-second statement limit. The PostgreSQL server log identified that exact statement; a later catalog check found 469 retained test schemas with 26,578 tables. Unqualified PostgreSQL `ANALYZE` covers the database rather than only the fixture's `search_path`. No active client blocker remained after the failure, and no timeout setting or production code was changed.

One retry used a Go source overlay that analyzes only the fixture schema's 71 tables, with quoted identifiers. The timed benchmark loops, application queries and metrics are unchanged. Statistics preparation remains outside timing. The executed [harness copy](performance/postgres-scoped-setup-harness.go.txt), [exact diff](performance/postgres-scoped-setup.patch), [failed output](performance/postgres-setup-timeout-076550b.txt) and [original attempt metadata](performance/quiet-attempt-076550b-setup-timeout.json) are retained. The successful retry's source hash and invocation are recorded alongside its results. This setup failure produces no PostgreSQL read-latency measurement.

## Quiet candidate measurements

These adapters ran sequentially, SQLite then PostgreSQL, after the delegated database, race, upgrade, build and browser jobs stopped. The runner was not hardware-isolated. The start/end conditions and exact source fingerprint are recorded in [quiet-run-environment.json](performance/quiet-run-environment.json).

### SQLite

| Case | p50 (ms) | p95 (ms) | p99 (ms) | Traced statements/read | JSON bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| 100 card summaries | 21.11 | 36.28 | 76.46 | 8 | 66,648 |
| 50 structural children | 8.89 | 16.00 | 16.77 | 10 | 32,628 |
| Task inspector | 17.65 | 28.39 | 41.78 | 42 | 18,456 |
| Objective inspector | 32.11 | 72.48 | 466.88 | 43 | 19,569 |
| Outcome inspector | 24.23 | 38.16 | 41.97 | 31 | 31,627 |
| Five timeline entries | 2.22 | 4.68 | 5.75 | 4 | 6,985 |
| Task read, 10 concurrent requests | 98.05 | 216.09 | 257.59 | 42 | 18,456 |

Raw output: [sqlite quiet candidate](performance/sqlite-candidate-quiet.txt).

### PostgreSQL

| Case | p50 (ms) | p95 (ms) | p99 (ms) | Traced statements/read | JSON bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| 100 card summaries | 34.47 | 48.34 | 80.44 | 10 | 66,648 |
| 50 structural children | 18.57 | 30.06 | 37.33 | 12 | 32,628 |
| Task inspector | 49.95 | 62.08 | 66.63 | 44 | 18,456 |
| Objective inspector | 51.65 | 75.48 | 79.30 | 45 | 19,569 |
| Outcome inspector | 37.55 | 61.91 | 64.49 | 33 | 31,627 |
| Five timeline entries | 3.33 | 6.02 | 9.74 | 6 | 6,985 |
| Task read, 10 concurrent requests | 101.51 | 183.16 | 234.38 | 44 | 18,456 |

Raw output: [postgres quiet candidate](performance/postgres-candidate-quiet.txt).

The largest observed p95 in this quiet harness was **216.09 ms**. All sampled p95 values were below the local 400 ms comparison point. The specified 2-vCPU / 4-GiB / SSD / 50-ms-RTT environment and end-to-end HTTP target remain **pending qualification**, regardless of this local result.

SQLite and the PostgreSQL retry passed all timed benchmark cases without a read error or an oversized first response. The setup failure, sample counts, warm-cache conditions and synthetic concurrency limits above still apply.

## Previous quiet run

The earlier quiet run used [`a8571c5`](https://github.com/A1b3rt0M3rcad0/wos/commit/a8571c551c2be1de195ecd3c40672d4f13f16445), before the conclusion and structural impact-review additions. It used the same runner and timed harness logic, but a different core source and the original unqualified PostgreSQL statistics setup. Preserve it as observed telemetry, not a controlled comparison.

| Case | SQLite p95 (ms) | PostgreSQL p95 (ms) |
| --- | ---: | ---: |
| 100 card summaries | 81.76 | 182.60 |
| 50 structural children | 17.27 | 37.98 |
| Task inspector | 60.64 | 75.75 |
| Objective inspector | 194.39 | 180.44 |
| Outcome inspector | 38.02 | 148.89 |
| Five timeline entries | 2.90 | 9.23 |
| Task read, 10 concurrent requests | 277.27 | 834.39 |

The PostgreSQL ten-request p95 was **834.39 ms**, above the local 400 ms comparison point. The final rerun does not erase that result. Full percentile, statement and byte measurements remain in [previous results](performance/quiet-results-a8571c5.json), [SQLite output](performance/sqlite-candidate-quiet-a8571c5.txt), [PostgreSQL output](performance/postgres-candidate-quiet-a8571c5.txt), and [source/environment metadata](performance/quiet-run-environment-a8571c5.json).

## Earlier shared-run measurements

These runs overlapped other builds, browser checks or real database tests. The first run was an earlier in-progress implementation; subsequent changes added required-Objective coverage and other facts, so payloads and query counts differ. They are useful retained telemetry, not a controlled before/after comparison.

| Case | In-progress SQLite p95 (ms) | Candidate SQLite p95 (ms) | In-progress PostgreSQL p95 (ms) | Candidate PostgreSQL p95 (ms) |
| --- | ---: | ---: | ---: | ---: |
| 100 card summaries | 37.61 | 48.36 | 168.43 | 97.14 |
| 50 structural children | 25.89 | 30.78 | 44.06 | 66.88 |
| Task inspector | 35.39 | 41.42 | 112.23 | 98.54 |
| Objective inspector | 82.52 | 69.31 | 112.38 | 370.00 |
| Outcome inspector | 40.36 | 55.49 | 37.72 | 328.68 |
| Five timeline entries | 2.36 | 6.90 | 10.32 | 22.48 |
| Task read, 10 concurrent requests | 473.03 | 278.33 | 970.65 | 495.31 |

The candidate PostgreSQL shared run exceeded 400 ms p95 for ten concurrent Task reads. Keep that result visible even if a quieter rerun is faster.

Raw outputs: [SQLite in progress](performance/sqlite-in-progress-shared.txt), [PostgreSQL in progress](performance/postgres-in-progress-shared.txt), [SQLite candidate](performance/sqlite-candidate-shared.txt), [PostgreSQL candidate](performance/postgres-candidate-shared.txt).

## Bounds and correctness evidence

The request limit is 100 explicit card references or a structural page of at most 100. Inspector relationships and criteria are paged, recent activity is limited to five entries, note previews are limited to 512 bytes, and serialized workspace responses have a 256 KiB hard limit. Large canonical records remain available through their authorized detail resources. The fixture does not prove constant cost for every graph validation or for arbitrary criteria counts.

The final run returned 66,648 bytes for 100 summaries, 32,628 for a 50-child page, 18,456 for a Task inspector, 19,569 for an Objective inspector, 31,627 for an Outcome inspector and 6,985 for five timeline entries. All were below the response budget. Traced statements per call were respectively 8/10/42/43/31/4 on SQLite and 10/12/44/45/33/6 on PostgreSQL. The Objective and Outcome reads include bounded conclusion-impact metadata. Exact count queries and hierarchy traversal can examine more metadata than the returned sample; the response bound does not imply constant database work.

The following executable regressions supplement the timing evidence:

- `TestWorkspaceCardsBatchAndStructuralPagesAvoidHydration`: one bounded batch, no full Task/Outcome hydration per card, exact large versions and scoped cursors.
- `TestWorkspaceViewHasOneSnapshotAndDoesNotPromoteLinkedEvidence`: facets share one revision and evaluation time; linked Evidence does not become approval.
- `TestWorkspaceNamedRelationshipsCriterionLinksAndTerminalCapabilities`: real endpoint names, a Task link beyond the first 25, global nested required Objectives, pagination and command/capability parity.
- `TestWorkspaceCapabilitiesPinContractExpiryWhileActualMutationsRecheckTime`: five capabilities use the pinned frame time across an expiry boundary without another clock read; actual mutations still acquire their guard and recheck current authoritative time. Passed against memory, SQLite and PostgreSQL with the race detector.
- `TestPostgresWorkspaceFrameKeepsRevisionFactsAndCapabilitiesWhileWriterCommits`: a real concurrent writer commits while the old read frame remains coherent; a new frame sees the new state. Passed with the race detector.
- `TestWorkspaceRetractedProofAgreesAcrossAttentionCriteriaCardsAndConclusionsAfterRestart`: proof applicability agrees across reads and mutation gates after retraction and restart.
- `TestWorkspaceConclusionScopeCountsOpenDescendantsAcrossTerminalParentsWithoutCrossingScope`: exact open-descendant counts and bounded named samples traverse terminal parents without crossing Outcome or Namespace boundaries.
- `TestWorkspaceBlockerPreviewShowsCanonicalDirectAndSubtreeScopeWithoutWriting`: optional blocker previews retain target-specific propagation rules, true totals and partial sample coverage without changing revision or state.

These tests passed against memory and SQLite where applicable, and a real PostgreSQL server. Database query-plan tests also verify the indexed acquisition-time lookup, including nanoseconds and UTC offsets. These checks are separate from browser and human usability qualification.

## Reproduce

Use the repository's Go toolchain and a disposable PostgreSQL database. The PostgreSQL helper creates an isolated test schema. Supply the DSN through the environment without storing it in the report.

```sh
export GOMAXPROCS=2

go test ./packages/wos-core/storage/sqlite \
  -run '^$' -bench '^BenchmarkWorkspaceF08$' -benchtime=20x -count=1 -v

# Set WOS_TEST_POSTGRES_DSN through your local secret/environment mechanism.
go test ./packages/wos-core/storage/postgres \
  -run '^$' -bench '^BenchmarkWorkspaceF08$' -benchtime=20x -count=1 -v
```

To replay the successful PostgreSQL run's schema-scoped setup at the recorded checkout, prepare the same external overlay from the retained source. This leaves repository source files unchanged:

```sh
export WOS_F08_REPLAY_DIR="$(mktemp -d)"
python3 - <<'PY'
import json, os, pathlib, shutil
root = pathlib.Path.cwd()
target = pathlib.Path(os.environ["WOS_F08_REPLAY_DIR"])
source = root / "packages/wos-core/storage/postgres/workspace_f08_benchmark_test.go"
overlay = target / "workspace_f08_benchmark_test.go"
shutil.copyfile(root / "docs/ux/operational-workspace/performance/postgres-scoped-setup-harness.go.txt", overlay)
(target / "overlay.json").write_text(json.dumps({"Replace": {str(source): str(overlay)}}))
PY
go test -overlay "$WOS_F08_REPLAY_DIR/overlay.json" \
  ./packages/wos-core/storage/postgres \
  -run '^$' -bench '^BenchmarkWorkspaceF08$' -benchtime=20x -count=1 -v
```

Record CPU affinity and quota, memory, database and Go versions, cache preparation, concurrent workload and code identity. Run the adapters sequentially after other heavy jobs stop. A passing benchmark means the fixture ran without read errors or an oversized response; percentile targets are reported explicitly and are not asserted by the benchmark.

The reference-environment p95, first usable map within two seconds, local selection feedback within 100 ms, controlled-network HTTP behavior and signed-protocol performance remain separate measurements to complete. This application-service harness does not establish those results.
