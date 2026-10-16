# Controlled v2 rich-read measurement

Released baseline b572269085494ddd4d6370716ff1cb360b46065f versus immutable candidate aa7ccc3 (full SHA and production/harness hashes in report.json), same host, Go 1.27.2 and GOMAXPROCS=3. Both archives have identical reviewed test-only harness files; no baseline production file changed. Both run ANALYZE outside timing. PostgreSQL uses an isolated new fixture database per source/size, removed after measurement; SQLite uses isolated files. This does not replace the separately retained cold unanalysed query failures and their corrected primary-key plans.

Three repetitions each: 80 real requests at 100/1,000 rich items, 40 at 10,000. The comparison uses median p95 across every repetition, as declared before running. SQL counts use actual SQLite statement/pgx query tracing; pool deltas are actual database/sql counters. CPU/peak RSS cover the whole precompiled child including validated domain setup/statistics, excluding PostgreSQL process resources. All samples/allocations/response sizes and whole-child boundaries remain in raw logs/report; no production percentile or SLA is inferred.

| Adapter | Case, 10,000 rich items | Before p95 ms | After p95 ms | Before MB/op | After MB/op |
| --- | --- | ---: | ---: | ---: | ---: |
| sqlite | section_work_items | 568.16 | 9.38 | 125.832 | 0.085 |
| sqlite | snapshot_with_proof | 661.24 | 247.32 | 129.596 | 44.505 |
| postgres | section_work_items | 566.46 | 8.36 | 129.970 | 0.094 |
| postgres | snapshot_with_proof | 615.38 | 424.59 | 133.737 | 45.371 |

All predeclared simple/full 10,000 targets pass (p95 <=70% and allocation <=50% of baseline). Focal 1,000 p95 <=110% also passes on both adapters. Simple sections retain scoped counts/LIMIT and no full aggregate hydration. Full derived snapshots still classify scoped global state/proof and allocate about 44–45 MB at 10,000; they are not constant-work views. The report does not certify thousands of writers or a production user count. Final PR source gates and dense-history index-plan qualification remain separate.

Reproduce with tools/acceptance/operational-read-report.py, explicit baseline/candidate archives, their full source commits, disposable WOS_TEST_POSTGRES_DSN/CONTAINER and the same harness copies. The runner precompiles all children before timing and refuses partial/failed/mismatched harness reports. Do not run other local builds/load tests concurrently with timing.
