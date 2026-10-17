# Scalable-installation validation

> [!NOTE]
> This is historical evidence for the pre-schema commits identified below. The
> schema-per-Tenant and protocol-3 transition changes the tested executable and
> storage boundary. These results are not relabelled as validation of that change.
> See [Tenant operations](tenants.md) and current PR checks for the new scope.

This record belongs to `feat/scalable-wos-operation`, based on `master` at
`cba58d6d609377df141d1369dccaaaf3b3e331e3`. [PR #114](https://github.com/A1b3rt0M3rcad0/wos/pull/114)
is a draft. Creating and testing this branch does not merge it, publish a release,
qualify production capacity or complete the separate stable-1.0 program.

Backend Go and schema changes are recorded through
`aaa447520f9236bd0a6b929fa1aeb7932ec0e246`; subsequent evidence/documentation commits
must not be confused with a differently tested executable. The pinned
[deployment receipt](evidence/compose-smoke.json) and
[contention comparison](qualification.md) identify their actual subjects.
The browser route-restoration correction is recorded separately at `59f19cd`.

## Verification scope

The integrated Go suites use Go 1.27.2 on Linux amd64, `GOMAXPROCS=3`, serial
package execution and real PostgreSQL 18.6. The PostgreSQL test service uses
512 MiB shared memory, `max_locks_per_transaction=1024` and
`max_pred_locks_per_transaction=4096`, matching the CI fixture's lock settings.
Normal and race runs use separate disposable databases, with actual container
backup/restore tests enabled. Targeted worker/core correction runs use
`GOMAXPROCS=2` or `3`, recorded separately from timed qualification. A skipped
capacity fixture is not measured load.

The final full normal suite at `aaa4475` passed **1,591 test/subtest entries in
21 tested packages**, with 14 explicitly skipped optional fixtures and ten
packages without tests. It includes every production Go/SQL correction and real
PostgreSQL backup/restore.

Earlier full normal and race suites each passed **1,579 entries**. Those full
local runs predate the final worker/core corrections. After the worker-policy
correction, all three affected packages (`integration`, server and ports) were
rerun normally and with race detection: **349 passing entries each**, with four
explicit optional skips. The final core corrections have additional passing race
runs for signed access/enrollment failures, actual PostgreSQL serialization
faults, persistence/history/cache maintenance, Next acquisition and issuer
recovery. Restoring the old streamed-error return in an isolated overlay fails
the new regression as expected. No earlier whole-suite race result is relabelled
as a final-head execution. [Machine-readable counts, source scopes and private-log
fingerprints](evidence/validation.json) preserve that distinction.

The optional skips include load fixtures without their opt-in environment,
subprocess-child fixtures, real full-disk/native-keyring prerequisites and SQLite
worker cases outside the supported PostgreSQL worker profile. The separate load
comparison explicitly enables its own fixture. [Live PR checks](https://github.com/A1b3rt0M3rcad0/wos/pull/114/checks)
identify the hosted source and status; local results do not imply that a queued
or running hosted check has passed.

Other executed checks include:

- Generated PostgreSQL adapters, MCP/OpenAPI contracts, the 103-type event
  catalogue and packaged central-plan archive remain current.
- Go formatting/vet, native Linux build/configuration/storage-help smoke and
  a Windows amd64 cross-build. Cross-compilation is not native Windows execution.
- All 38 browser unit checks; five real-server Chromium authority journeys;
  and the desktop/mobile details, native forms and administrative-modal journey.
  The authority checks include Namespace administrators without installation
  authority and the denied installation-metrics route.
- All nine Canvas journeys after the final route correction, plus the
  Namespace-administrator browser journey again against the rebuilt embedded
  application. The deterministic regression holds graph expansion during reload
  and verifies dismissal, URL/focus, late-response suppression and Back navigation.
- Five complete journey/specialist race tests after correcting test readiness.
  The held post-commit discovery response proves one persisted update, intentional
  busy rejection, and subsequent mobile keyboard/menu/Escape/focus behavior.
  These changes affect tests only; the product's submission guard is preserved.
- Six backup-manifest tests and three scalability-report evidence-contract tests.
- Local Markdown targets, anchors and preserved incoming README anchors.

The [actual Compose smoke](evidence/README.md) separately checks TLS ingress,
one complete process transitioning to two APIs/two workers, private worker
routes, browser-session continuity, issuer identity, idempotent-result continuity
and restarts. It does not infer delivery throughput from health checks.

## Failures found and corrected

Qualification retained failed/interrupted development runs separately from
successful evidence:

- SQLite delivery tests exposed an application-clock regression when the new
  durable-clock port was advertised on SQLite. Only PostgreSQL implements that
  optional port; deterministic SQLite clock behavior remains intact.
- Concurrent terminal returns exposed a uniqueness race and an authorization
  path that masked storage transaction conflicts. The losing original intent
  now receives its precise conflict without a partial durable success; unrelated
  SQL uniqueness mapping and revocation checks remain strict.
- SQLite writer contention returned an unclassified busy error. BUSY/LOCKED now
  maps to the existing transaction-conflict contract, with a held-writer test.
- A reused PostgreSQL fixture exhausted shared memory during early parallel
  development stress; a later cleanup exceeded its existing deadline in that
  heavily reused catalogue. The interrupted run was not counted as a pass.
  Final integrated suites use fresh isolated databases and the CI-sized lock
  configuration; assertions and cleanup deadlines were not weakened.
- Review found that an approved migration path could still read adjacent WAL
  or replaced source bytes. Hashing and import now share a privately staged,
  immutable source, with WAL/path-replacement regression coverage.
- The deployment pilot exposed ambient proxy routing of private dynamic API
  addresses. Caddy's built-in proxy bypass is restricted to that private
  transport; inherited proxy settings and verified TLS remain intact.
- Review found eligible-secret policies that passed configuration validation
  but stopped a worker on its first tick. The same policy is now checked before
  storage initialization, construction and worker startup. More than 256 eligible
  references is rejected explicitly; opaque long reference identifiers remain
  supported and no eligibility set is silently truncated.
- Hosted browser verification found one Canvas deep-link test failing while
  the other 80 passed. A held graph response reproduced the actual race: the
  inspector closed but route restoration suppressed the user's cleared URL.
  Explicit dismissal now survives that guard; the original assertion remains
  unchanged and all nine Canvas journeys pass against the rebuilt application.
- The multi-Tenant workload exposed transactional SQL failures that escaped
  nested reads as raw errors or were disguised as missing signed policies.
  Work, enrollment and rotation policy reads now preserve infrastructure
  failures; actual query/scan/iteration boundaries classify database conflicts
  consistently. Missing or disallowed policies remain forbidden. Real PostgreSQL
  fault tests cover both an immediate lookup and a failure after one valid row;
  restoring the old streamed-error return in an isolated Go overlay makes the
  new regression fail. The original database isolation and exact-intent retry
  rules remain unchanged.
- A subsequent hosted browser run passed 81 tests and exposed a premature
  interaction in the human-journey helper. Modal closure acknowledges commit;
  the command remains busy until discovery/refresh completes. The helper now
  waits for the existing enabled-state readiness signal, matching the shared
  fixture. The original mobile keyboard assertions remain intact; a deterministic
  held-refresh regression verifies the intended guard and subsequent readiness.

## Measured candidate

The [recorded matrix](evidence/qualification/README.md) enables the signed load
fixture and passes **18/18 candidate cases, 144 accepted journeys** at `aaa4475`.
The unchanged baseline completes 12 cases and fails its first multi-Tenant case;
its failure and all completed repetitions are retained. Only four groups with
three completed repetitions on both sources receive performance ratios. Observed
median accepted throughput improves 18–51% in those groups, with fixed workload
and important sample/order/host limits. This is not a full successful before/after
matrix or production capacity guarantee.

## Deliberately unqualified boundaries

This branch does not establish an SLA, customer/agent count, unrestricted scaling
or a production p99. The comparison uses a small declared workload and independently
pooled services in one Go process; the actual-process pilot covers a different
boundary. PostgreSQL I/O/WAL and long-duration storage growth need their own
production workload observations.

Full signed Namespace transfer, automatic sharding, mixed-version rolling
upgrades, historical archival, storage/CPU subscription quotas and automated
database HA/PITR are not implemented. Namespace preflight always reports
`ready_for_transfer=false`. Whole-installation migration is an offline operation.
The reference database and ingress remain single points of failure; a complete
in-memory aggregate still grows with its history even though unchanged historical
records no longer cause per-record SQL writes. See the [operating guide](README.md)
and [mobility boundaries](mobility.md).
