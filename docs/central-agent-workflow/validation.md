# Branch validation

This record covers `feat/central-agent-workflow`, based on `80074f470d44a55c33c2ea297e509f9e14871d96`. Operational code is in commits `c9aa10c` and `1ed35b1`; subsequent documentation/CI commits carry this record. It records development evidence, not publication or stable 1.0 acceptance. The PR's checks are the current hosted-source evidence.

## Environment

Native Linux amd64, stable Go 1.27.2, Node.js 24.19.0, Chromium, SQLite, and real PostgreSQL 18. PostgreSQL runs in owned disposable loopback containers; final normal/race runs use independent instances and scoped fixtures. The fixture uses `max_locks_per_transaction=1024`, `max_pred_locks_per_transaction=4096`, and 512 MiB shared memory for the dense serializable workloads and catalog dumps/restores. The CI service uses the same tested initdb settings. This is a test-fixture setting, not a changed product default.

Initial combined runs exhausted PostgreSQL shared lock capacity (`SQLSTATE 53200`); server diagnostics specifically identified `max_pred_locks_per_transaction`, and a dump timed out. Raising only heavyweight-lock capacity did not resolve predicate-lock exhaustion. Those runs were stopped, and the complete suites restarted against fresh independent instances with the settings above. Concurrent dense race execution also caused a schema-open deadline and dump timeout. The final complete normal and race commands run sequentially with `GOMAXPROCS=2` and `-p 1`; package scheduling changes neither test assertions nor the concurrent actors inside the concurrency tests. No application authorization or isolation was relaxed.

## Recorded checks

| Check | Evidence and scope |
| --- | --- |
| Full Go suite | `go test -p 1 -json ./...` passed: 1,501 test/subtest pass events and nine explicitly classified skips |
| Full race suite | `go test -p 1 -race -json ./...` passed: 1,501 test/subtest pass events and the same nine explicitly classified skips |
| Vet, formatting | `go vet -p 2 ./...`, changed-source `gofmt`, and `git diff --check` passed |
| Generated contracts | Signed/PostgreSQL/MCP/OpenAPI generators reproduced their current bytes; 103 public integration event mappings checked |
| Public compatibility | Three Node checks passed; original released schemas/manifest retained; exact reviewed optional route additions and two current fingerprints recorded |
| Official Skills | Six Node installer/instruction tests passed; native Go Skills normal/race tests passed |
| Distribution/release unit contracts | 15 Node tests passed; publication adapters use fixtures and do not publish artifacts |
| Report tooling | Seven context-cost report tests and four execution-report tests passed |
| Browser | 42 real-server Playwright journeys and 14 browser unit checks passed; desktop/mobile, matte modals, editorial layout, accessibility, review, uncertain responses and role restrictions |
| Native central workflow | Real binary/server/CLI passed SQLite and PostgreSQL: protected-reference enrollment, persisted selection from another CWD, typed material, independent review, accepted receipt cleanup, historical context and durable activity after restart |
| Real disk-full cleanup | Existing unmodified gate passed both receipt and rejected-return-resolution cleanup tests, including four resolution interruption points, in a disconnected read-only container with a 1 MiB writable tmpfs |
| Active fuzzing | Four ten-second campaigns passed: canonical JSON, envelopes, return resolution and restricted YAML |
| Central migration | Native process-death stages, exact frozen-copy recovery, 150-file sibling directory, conflicting edited prefix, read-only dry run, aliases/paths, active selection pinning, history preservation and busy-source rejection passed |
| Platform | Windows amd64 client test binary cross-compiled; this is compilation evidence only |

Native process acceptance uses `node tools/acceptance/central-agent-workflow.mjs bin/wos`. `WOS_CENTRAL_TEST_POSTGRES_DSN` selects a fresh disposable PostgreSQL database. The runner provisions two distinct authorized Principals; its example documentary URLs are synthetic registered references, not fetched external proof. Reviewers in production must inspect actual delivery material.

The signed public transport journey additionally exercises HTTP append, MCP replay, SDK reads, ascending/descending fixed-upper-bound pagination, changed-order cursor rejection, attention parity and history after contract closure on memory, SQLite and PostgreSQL. Storage/app tests cover immutable chronology, maximum uint64 ordering, rollback, current keys/scope/leases/fencing, inline documents, cache-independent replay and real backup/restore.

## Limits

- The server remains passive; no model provider, agent scheduler, external execution sandbox or separate deployed-host fleet is qualified here.
- Actual local native keyring execution and Windows runtime execution are not inferred from headless protected-reference tests or cross-compilation. Dedicated hosted platform jobs report those gates.
- Optional large load matrices are separate from ordinary unit/race runs; previous load reports do not certify a new source automatically. Child-process-only helper skips are not missing parent crash coverage.
- Rejected final returns have authenticated terminal resolution. A frozen activity append that becomes definitively rejected/expired has no corresponding terminal-closure protocol yet. Preserve its exact pending request for authorized recovery/diagnosis; never delete it or substitute a new profile as resolution.
- Central import accepts signed schema-2 sources and refuses mixed unsigned history or unfinished source-publication pairs. All old writers must be drained; an old binary cannot honor the new retirement fence.
- A declared validation/result activity or registered evidence is not proof of execution, assessment or approval. Delivered-for-review does not mean Task completed, and Task completion does not achieve its parent Outcome.

## Reproduce the package

Run `python3 tools/documentation/package-central-plan.py` and then `--check`. The ZIP contains the plan, this record, byte-preserved owner briefs and a SHA-256 manifest, with deterministic entry metadata. It excludes secrets, databases, generated local state and test logs.
