# CI repair — historical migration replay

The WOS verification job on PR #107 failed in **Unit and contract tests** for source `28fac98c6bc208a1c273aeb5941ca21f03e12316`; the browser, packages, Linux CLI, and Windows CLI checks passed. See the [failed job](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/38846766801/job/116798956036).

The failure was reproduced locally in `TestActualV010DatasetMigrationsPreserveProofAndCommandFingerprints`:

```text
original 0.1 claim fingerprint changed
precondition_failed: work item is not ready: lifecycle_not_ready
```

The legacy fixture's claim receipt expired at `2026-10-15T19:16:36.279570Z`. The test attempted to replay that receipt against the execution date. SQLite correctly expired the cached result, so execution reached the already-completed WorkItem. Injecting an application Clock alone does not control SQLite's internal wall-clock retention check.

## Correction

The existing test now runs inside the standard library's `testing/synctest` time-isolated environment. It reads the last command timestamp and earliest receipt expiry from the loaded fixture, verifies that they share a valid retention window, and advances the virtual clock to that historical command. Both the application and SQLite therefore evaluate the unmodified receipts within their original retention window. The virtual interval advances immediately; it does not sleep in real time or change the system clock.

The compressed database, metadata, checksum verification, command fingerprints, replay assertions, historical proof/version checks, and before/after cutover checks remain unchanged. Production code, the seven-day retention rule, and public contracts are unchanged. No CI gate is disabled or made optional.

## Local validation

With Go 1.27.2:

- The isolated historical migration test passed.
- `go test -p 4 -count=1 -json ./...` passed: **746 test/subtest outcomes passed, 75 skipped, zero failed**. Skips remain service/platform/load-gated checks, including PostgreSQL tests without a local PostgreSQL service.
- `go vet -p 4 ./...`, repository-wide `gofmt` verification, and `git diff --check` passed.

The [PR checks](https://github.com/A1b3rt0M3rcad0/wos/pull/107/checks) record execution on the pushed source, including the PostgreSQL service, complete race detector, fuzz campaigns, historical upgrade/restore, acceptance journeys, build/container checks, and load reports. Local SQLite execution does not establish PostgreSQL parity. The earlier failed run is retained as historical evidence; its skipped downstream steps are not counted as passed.
