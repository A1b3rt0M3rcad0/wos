# Continuity measurements — 2026-10-07

Go 1.27.1, Linux amd64, Intel Xeon Platinum 8573C, GOMAXPROCS=4. SQLite uses the pinned WASM driver; PostgreSQL 18.6 runs in a local container. Fixture: 101 Outcomes, two with 100 WorkItems; the second has one required criterion and attestation assessment per WorkItem. Page size: 25. Each operation includes Application execution and JSON serialization. Minimum benchmark duration: 500 ms. No external concurrent queries/tests ran during SQL counting.

| Database / query                       | Mean (ms) | JSON (bytes) | SQL/op | Allocated bytes/op | Allocations/op |
| -------------------------------------- | --------: | -----------: | -----: | -----------------: | -------------: |
| SQLite / discovery                     |     0.410 |        7,402 |      1 |             35,549 |            381 |
| SQLite / simple snapshot               |     5.459 |       32,238 |     23 |            934,443 |          5,953 |
| SQLite / snapshot with assessments     |     7.376 |       32,627 |     32 |          1,320,365 |         12,515 |
| PostgreSQL / discovery                 |     1.423 |        7,505 |      3 |             43,449 |            565 |
| PostgreSQL / simple snapshot           |    46.099 |       32,238 |     25 |            974,290 |          7,092 |
| PostgreSQL / snapshot with assessments |    92.224 |       32,627 |     34 |          1,403,446 |         15,812 |

Batch reads preserve base state, authorship, criteria, revisions, Evidence, assessments, and conclusions. Nine history queries replace per-owner/assessment/conclusion queries in Objective/WorkItem listings. Before optimization, the simple fixture measured on this same machine used 822 SQLite statements. The SQL reduction is verifiable; timings vary with hardware/load and are not compared with the October 5 machine.

| Four readers, same Outcome with assessments |        Throughput | Mean latency/call | Mean pool wait/op |
| ------------------------------------------- | ----------------: | ----------------: | ----------------: |
| SQLite, single-connection pool              | 172.6 snapshots/s |         22.876 ms |         15.301 ms |
| PostgreSQL, pool up to 16 connections       | 34.35 snapshots/s |        109.612 ms |              0 ms |

The parallel benchmark reads the same Outcome. PostgreSQL serializes that Outcome's guard, so additional connections do not guarantee improvement in this scenario. Zero pool wait does not mean zero guard contention. Percentiles, write throughput, multiple Outcomes, and production capacity were not measured.

SQLite counts statements through TRACE_STMT. PostgreSQL sums `public.pg_stat_statements` calls, subtracting the statistics query; the extension must be enabled and accessible. Without it, the benchmark omits this metric. Timings/allocations include instrumentation overhead. Serial reads did not wait for a connection. Runtime guard metrics include preparation/acquisition as well as blocked time.

Responses are bounded, but snapshots still reconstruct complete collections; individual history cost and some projections grow with the dataset. These figures describe an acceptance fixture, not performance targets or guarantees for arbitrary datasets.

```sh
GOMAXPROCS=4 go test ./packages/wos-core/storage/sqlite ./packages/wos-core/storage/postgres -run '^$' -bench BenchmarkContinuity -benchmem -benchtime=500ms
```

Configure `WOS_TEST_POSTGRES_DSN` with a test database. Raw output: `docs/audit/release-2026-10-07-benchmarks.txt`. `BenchmarkContinuity` and its PostgreSQL variant are generator-maintained and checked for drift.

## Historical measurements — 2026-10-05

These local measurements are not performance targets or production results. Go 1.27.1, Linux amd64, AMD EPYC 9V74 (80-Core Processor), `GOMAXPROCS=8`, project WASM SQLite driver, temporary directory. Fixture: 100 Outcomes, the first with 100 WorkItems; pages of 25. Each iteration includes Application query execution and JSON serialization. Minimum duration per benchmark: 500 ms.

| Query                   |                 Mean time | JSON response | Memory/operation | Allocations |
| ----------------------- | ------------------------: | ------------: | ---------------: | ----------: |
| Discovery, 100 Outcomes |     195,194 ns (0.195 ms) |   7,402 bytes |     35,626 bytes |         380 |
| Snapshot, 100 WorkItems | 65,329,583 ns (65.330 ms) |  32,156 bytes |  2,768,710 bytes |      53,759 |

Discovery reads metadata and indexes. The snapshot bounds its response but still reconstructs complete collections and performs per-entity reads. The measurement identifies optimization work; it does not establish bounded cost for large datasets. At this historical revision, query counts, guard wait, concurrent throughput, percentiles, and PostgreSQL measurements remained pending.

Reproduce:

```sh
go test ./packages/wos-core/storage/sqlite -run '^$' -bench BenchmarkContinuity -benchmem -benchtime=500ms
```

Fixture: `packages/wos-core/storage/sqlite/continuity_benchmark_test.go`. Original output: `docs/audit/benchmark-sqlite.txt`.
