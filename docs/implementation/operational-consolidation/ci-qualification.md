# PR #116 CI qualification

The failed hosted runs on 2026-10-21 tested `de15e7277b58da30d3800873d3aa2178173b53ff`. Six subsequent local commits and the operational implementation/fixture corrections had not reached the PR. A successful local run did not change those hosted check results.

## Corrections

- Policy fixtures update the credential policy created at issuance using its actual revision. Transport, installed-package smoke and central workflow fixtures do not attempt to recreate it at revision zero.
- Central workflow Tasks belong to a real, started Objective, and fixture credentials carry the canonical permissions for their actual operations.
- PostgreSQL test generation passes `testing.TB` to every direct store open, preserving schema teardown for tests, benchmarks and partial migration failures.
- Signed storage generation excludes the unsigned-only credential binding columns introduced by migration 0044; signed-v2 retains its existing schema and signature protocol.
- The integration event catalogue explicitly includes `objective.completion_policy_authorized`.
- `go mod tidy` classifies `github.com/google/jsonschema-go` as a direct dependency because the current command schema implementation imports it.
- The oversized HTTP batch receipt fixture uses the runtime's 15-second request budget. Its inherited two-second small-fixture deadline reproducibly canceled the valid batch under `-race`; the complete HTTP package passes with race instrumentation after this correction.
- Integration load fixture setup has an explicit two-minute migration allowance for race instrumentation; the production startup deadline is unchanged. The killed worker retains its deliberately short persisted lease, while recovery uses the real worker's 30-second lease instead of expiring successful replacement ACKs after 500 ms.
- The standalone eighteen-step journey grants the precise claim/accept permissions captured by issued credential policies and gives its Issue an explicit Objective owner. The two-server recovery pilot likewise creates its signal Tasks under an explicit Objective.
- The existing browser fixes preserve exact command versions, the focused Issue/Task after relation removal, and the actual lease lifecycle and visible action scope.

- The concurrent acquisition/cancellation test uses an atomic fixture ID counter; concurrent domain calls remain concurrent. The full application package passes with `-race`, and the regression passes 100 repetitions.
- Browser relation navigation waits for the selected Issue before expanding its details, avoiding clicks against a detached Task inspector. The relation scenarios pass ten repeated browser runs.
- Historical upgrade fixtures explicitly establish Objective ownership before signing or acquisition. Legacy grants are upgraded by an independent administrator through public CAS commands; unbound protected receipts remain refused, while their stored history is preserved. Policy revisions use the JSON string representation required by the API.
- The v0.4.2 upgrade manifest accepts reserved migration ordinals while still requiring exact adapter ordering, filenames, checksums and applied rows. A regression rejects omitted applied migrations.
- Signed load returns bind the authenticated credential policy revision rather than a stale hardcoded revision.
- The instrumented full Go suite has a finite 30-minute per-package deadline; SQLite migration-heavy race tests exceed the default ten minutes on constrained runners. Production deadlines remain unchanged.

## Local evidence

- Complete Go suite with live PostgreSQL: passed in `/tmp/wos-go-tests-verified.log` before this CI follow-up. Those runtime changes are preserved.
- Complete browser suite: 83 JavaScript units and 238 Playwright scenarios passed in `/tmp/wos-playwright-system-chromium.log`, using the installed Chromium through the existing `WOS_TEST_CHROMIUM_PATH` setting.
- `go vet ./...`: passed after generation.
- Exact signed public transport acceptance with `-race`: Memory and SQLite passed; PostgreSQL was skipped in that specific invocation. The prior complete Go run included live PostgreSQL.
- `node --test packages/wos-skill/test/*.test.mjs tools/distribution/test.mjs tools/distribution/native-go-tests.test.mjs tools/releases/test.mjs tools/releases/compatibility.test.mjs`: 33 passed.
- Complete signed/PostgreSQL/MCP/OpenAPI regeneration: repeated without byte changes.
- Event catalogue, implementation-plan archive and context/cost/upgrade report checks: passed.
- Central CLI workflow with independent review: passed on SQLite and an isolated live PostgreSQL database.
- Installed-binary signed smoke: passed, including controlled rejected approval, terminal resolution, corrections and independent review.
- Complete offline package build/verification and separate-source-path reproducibility: passed; native binaries and npm assets are byte-identical.
- All four active fuzz campaigns (canonical JSON, signed envelope, restricted YAML and return resolution): passed.
- Complete HTTP package with `-race`: passed in 9.955 seconds.
- The subsequent full local race run exposed migration setup and replacement delivery-lease timing failures in integration; it was stopped for correction rather than counted as passing. The complete integration package then passed with live PostgreSQL and `-race` in 89.776 seconds.
- The standalone eighteen-step HTTP journey and all six operator backup/restore unit tests pass after the fixture corrections.
- Actual pinned v0.2 upgrades passed for SQLite and PostgreSQL, preserving original contracts, submissions, criteria and receipt bytes; five safe historical commands replay and two unbound protected receipts explicitly refuse replay.
- Actual pinned v0.4.2 signed-state producer, backup, restore and migration passed on SQLite and PostgreSQL (schema 26 to 51), preserving all historical table columns and verifying historical DSSE with the current codec.
- Hosted `20791ba` package, native Linux and native Windows checks passed. Its browser annotation identified the detached-inspector wait corrected above; the main check was still running during this qualification.
- Hosted `dd4cfa3` package, native Linux and native Windows jobs also passed; main and browser qualification remained running before the integration-fixture correction was prepared.
- Hosted `193d5e0` package, native Linux and native Windows jobs passed. Its full CI and browser jobs were still executing when the independently reproduced HTTP race-fixture correction was prepared.

Hosted native platform acceptance, packaging reproducibility, race/fuzz/load/upgrade gates and the final browser workflow are tracked against the new pushed source. Local checks do not substitute for those hosted results. No merge, release, tag or package publication is part of this work.
