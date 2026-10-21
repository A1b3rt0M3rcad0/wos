# E01 — Baseline and reconciliation

## Execution context

- Repository: `A1b3rt0M3rcad0/wos`
- Pull request: [#116](https://github.com/A1b3rt0M3rcad0/wos/pull/116), verified as **open and Draft** from GitHub on 2026-10-21.
- Branch: `fix/workspace-operational-density`
- Baseline source SHA: `2aab252129f548ab54ad4232095e1bc83d76d372`
- Worktree at baseline inspection: clean.
- The PR remains Draft. No merge, release, tag, publication or version change is part of this execution.

The owner authorized implementation of the supplied plan on 2026-10-21 and explicitly permitted subagents. That current instruction supersedes the source plan's historical sentence in §1.2 saying that the request was only to deliver the Markdown document. The plan's architectural boundaries and delivery restrictions remain in force. The complete source plan is preserved in [`implementation-plan.md`](implementation-plan.md).

## Reconciliation of the previous 44-stage plan

The previous hierarchy, Issue and dependency plan is implemented through Stage 26. The roadmap's “Stage 26 is next” line was stale: the current HEAD contains separate Stage 25 and 26 commits. Stages 27–44 remain unimplemented and are mapped to the consolidated E01–E84 plan below; they are not duplicated or claimed complete.

| Previous stage | State at baseline | Commit / consolidated mapping |
|---|---|---|
| 01 | Implemented | `402ab4d` |
| 02 | Implemented | `f80bdf9` |
| 03 | Implemented | `1435b5e` |
| 04 | Implemented | `48de104` |
| 05 | Implemented | `165abfb` |
| 06 | Implemented | `b3c284f` |
| 07 | Implemented | `b6255fa` |
| 08 | Implemented | `889af1a` |
| 09 | Implemented | `213ef20` |
| 10 | Implemented | `757342c` |
| 11 | Implemented | `7bd353e` |
| 12 | Implemented | `46f5b7c` |
| 13 | Implemented | `21d04af` |
| 14 | Implemented | `24fc800` |
| 15 | Implemented | `0c4e8bc` |
| 16 | Implemented | `89f0144` |
| 17 | Implemented | `f471b5b` |
| 18 | Implemented | `2376e0a` |
| 19 | Implemented | `b813cce` |
| 20 | Implemented | `aaed622` |
| 21 | Implemented | `6df2139` |
| 22 | Implemented | `152e6af` |
| 23 | Implemented | `0062e6f` |
| 24 | Implemented | `e021fca` |
| 25 | Implemented | `2eb2e26`; Task Objective ownership and migration 0035 |
| 26 | Implemented | `2aab252`; Issue Objective ownership and migration 0036 |
| 27 | Planned | API/OpenAPI generation → E49 |
| 28 | Planned | HTTP parity → E50 |
| 29 | Planned | MCP parity → E51 |
| 30 | Planned | Go SDK parity → E52 |
| 31 | Planned | CLI → E53–E55 |
| 32 | Planned | Issue read model → E69 |
| 33 | Planned | Issue Kanban context → E69–E71 |
| 34 | Planned | Issue cards and filters → E69–E70 |
| 35 | Planned | Issue inspector and assignment → E42, E70 |
| 36 | Planned | Issue drag/drop and close decisions → E69–E72 |
| 37 | Planned | Issue plus Blocker creation → E44, E70 |
| 38 | Planned | Bidirectional dependency editing → E67 |
| 39 | Planned | Canvas relation mode → E68 |
| 40 | Planned | Dependency batch UI → E47, E67 |
| 41 | Planned | Invalid placement prevention → E09, E56–E59 |
| 42 | Planned | Concurrency, storage and restore → E74–E75 |
| 43 | Planned | E2E, onboarding and accessibility → E72, E77, E83 |
| 44 | Planned | Documentation, parity and acceptance → E76, E79–E84 |

Migration 0035 and 0036 are present at the baseline; this plan does not recreate them. The current executor did not have `WOS_TEST_POSTGRES_DSN`, so this run does not independently re-qualify a live PostgreSQL server. Prior stage receipts remain historical evidence, not a result from this executor.

## Existing implementation and next gaps

The Board/Kanban already uses backend capabilities and actual domain operations. Cards have no status selector: dragging prepares an authorized operation, and the lane changes only after the persisted command and refreshed projection. The Kanban modal, return context, proof checklist, Backlog-only definition editing, zoom and partial-page messaging also exist. These approved behaviors must remain intact.

Backend inventory found these concrete gaps for the consolidated plan:

- There is no persisted operational profile or shared Objective/ancestor readiness gate across every acquisition, activation, and ready query (E04–E13).
- Legacy `WorkLease` is not bound to a credential. Signed contracts already bind their authority and must retain their existing checks (E22–E26).
- Signed `WorkSubmission`/`ReviewCase` are protocol-bound. There is no native unsigned `TaskDelivery`/verification queue, so the later delivery stages must add the bounded model described in the plan rather than weaken or reuse signed review records incorrectly (E27–E34).
- Objective achievement is explicit and manual; no authoritative completion eligibility report or policy-driven deterministic completion exists (E35–E40).
- Objective ownership, Issue gates, dependency impact, batch dependency editing and migrations 0035/0036 are already implemented. Their next work is transport/read-model/UX parity, not a second ownership or graph model (E41–E48, E69–E70).
- The contextual Kanban still uses a separate inspector beside its lanes. New item-focused work should preserve the Kanban as the main surface and avoid recreating this space loss (E64–E72).

## Baseline defects found and corrected

The first full Go run exposed three SQLite fixture violations introduced by the now-required Objective owner contract:

1. `tests/boundary/reference_search_test.go` created an Issue without its Objective owner.
2. `tests/acceptance/signed_transport_test.go` inserted a Task without an Objective.
3. `tests/acceptance/work_contract_protocol_test.go` inserted Tasks without an Objective; its first fixture correction reused a later Task ID and was corrected to a unique Objective ID.

The domain/schema enforcement was retained. Fixtures now create an Objective and explicitly attach the Tasks/Issue to it. The initial test attempts and corrections are retained here as baseline evidence; no test assertion was weakened.

## Verification on the baseline source

The source code under `packages/` remained at baseline SHA `2aab252`. Three test fixtures were corrected in the worktree before the successful complete run. Toolchain was Go 1.27.2, provisioned under `/tmp` after identifying that `/usr/bin/go` was the unrelated two-player Go game. Module download through the default proxy redirected one dependency to a host blocked by the environment; using the same official GitHub source directly resolved it. No repository dependency or version changed.

| Check | Result | Notes |
|---|---|---|
| `go test -p 1 -count=1 ./...` | Pass | Full repository; 1m28s observed. PostgreSQL tests requiring a live DSN were skipped because `WOS_TEST_POSTGRES_DSN` was unset. |
| `go vet ./...` | Pass | Full repository; no diagnostics. |
| `gofmt -l` on the three corrected fixtures | Pass | No files reported after formatting. |
| `npm run test:unit` | Pass | 79 tests, 0 failures, 0.759s. |
| Playwright: login, contextual Kanban, Kanban workspace and simplification specs | Pass | 47 tests, 2.1m, Chromium `/usr/bin/chromium`; includes 1280×720 through 3840×2160, 320px mobile, keyboard, 400% zoom, proof, drag-only movement and partial pagination. |
| Playwright login capture run | Pass | 12 tests, 19.1s. Captured 10 viewport compositions plus expanded-login desktop/mobile variants. |
| Live PostgreSQL acceptance | Not qualified | `WOS_TEST_POSTGRES_DSN` unset; package skips are not counted as passes for real PostgreSQL. |
| Hosted CI on the new execution commits | Not run | Must be checked against the final pushed HEAD in a later qualification stage. |

The source-bound login captures and their served-asset hashes are under [`baseline/screenshots/login-2aab252/2aab252129f5/`](baseline/screenshots/login-2aab252/2aab252129f5/). Desktop and mobile images were visually inspected after capture. Existing design screenshots remain historical; these new captures bind the current baseline SHA and browser run.

## E01 outcome

The 44-stage crosswalk, real baseline failures, their fixture-only corrections, current implementation gaps and unqualified environments are recorded. The machine-readable 84-stage manifest and complete pending acceptance matrix are in [`manifest.json`](manifest.json) and [`acceptance.json`](acceptance.json). The PR is still open/Draft. The next eligible stage is E02; E03 must follow E02 before domain implementation E04.
