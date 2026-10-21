# PR #116 CI qualification

The failed hosted runs on 2026-10-21 tested `de15e7277b58da30d3800873d3aa2178173b53ff`. Six subsequent local commits and the operational implementation/fixture corrections had not reached the PR. A successful local run did not change those hosted check results.

## Corrections

- Policy fixtures update the credential policy created at issuance using its actual revision. Transport, installed-package smoke and central workflow fixtures do not attempt to recreate it at revision zero.
- Central workflow Tasks belong to a real, started Objective, and fixture credentials carry the canonical permissions for their actual operations.
- PostgreSQL test generation passes `testing.TB` to every direct store open, preserving schema teardown for tests, benchmarks and partial migration failures.
- Signed storage generation excludes the unsigned-only credential binding columns introduced by migration 0044; signed-v2 retains its existing schema and signature protocol.
- The integration event catalogue explicitly includes `objective.completion_policy_authorized`.
- `go mod tidy` classifies `github.com/google/jsonschema-go` as a direct dependency because the current command schema implementation imports it.
- The existing browser fixes preserve exact command versions, the focused Issue/Task after relation removal, and the actual lease lifecycle and visible action scope.

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

Hosted native platform acceptance, packaging reproducibility, race/fuzz/load/upgrade gates and the final browser workflow are tracked against the new pushed source. Local checks do not substitute for those hosted results. No merge, release, tag or package publication is part of this work.
