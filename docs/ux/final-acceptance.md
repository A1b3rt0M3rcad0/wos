# Human workspace UX/DX acceptance — October 9, 2026

The executable implementation of the owner's [original plan](original-plan-2026-10-09.md) is complete. Representative-user usability, human baseline/comprehension/timing and manual screen-reader validation remain pending by the owner's explicit decision. Native browser zoom is also pending; equivalent viewport reflow is automated evidence, not native zoom or WCAG certification.

The accepted executable source is `315fcadbcb4a6acc614cf908e04b5d55d54e9db9`. The final documentation/capture PR adds no application behavior. [Machine-readable evidence](final-acceptance.json) binds tests, artifact manifests and source hashes to that revision. Foundation, query, language, registry, forms, navigation and operation changes are integrated through PRs [#79](https://github.com/A1b3rt0M3rcad0/wos/pull/79), [#80](https://github.com/A1b3rt0M3rcad0/wos/pull/80), [#81](https://github.com/A1b3rt0M3rcad0/wos/pull/81), [#82](https://github.com/A1b3rt0M3rcad0/wos/pull/82), [#83](https://github.com/A1b3rt0M3rcad0/wos/pull/83), [#84](https://github.com/A1b3rt0M3rcad0/wos/pull/84) and [#85](https://github.com/A1b3rt0M3rcad0/wos/pull/85).

## Plan coverage

| Plan findings | Delivered behavior and evidence |
| --- | --- |
| UX-01–04, UX-10, UX-14 | Explicit 100-command exposure contract; contextual permission/state/protocol registry; dedicated human forms; hidden authority/CAS fields; technical schema console restricted to Developer tools |
| UX-05–06, UX-12 | All eleven reference kinds searched within authorized scope; bounded pages and exact-ID resolution; duplicate context, keyboard and selected-ID preservation; All Tasks server title/priority search |
| UX-07, UX-15, UX-17 | Six purpose areas, persisted Draft preparation, explicit activation and scoped URL/reload/Back/Forward restoration |
| UX-08, UX-13 | Single en-US official-copy catalog, English shell, typed terms and UTC presentation; authored/wire values preserved |
| UX-09 | Visual Roadmap phases/milestones/references, hierarchy/order, draft diff and reason, immutable publication and separate activation |
| UX-11 | Local edits retained for explicit comparison and new CAS intent; uncertain retries preserve original bytes/key; revoked or changed identity cannot redispatch |
| UX-16 | Real-server tests select human labels/intents; 1,001-item fixtures and actual grants/protocols rather than invented accounts |
| UX-18 | Feature modules separate transport, actions, copy, navigation, forms, references, immutable intents, advanced editors and contract views; [ADR-027](../adr/0027-human-workspace-client.md) selects ES modules with zero production JS dependencies |
| UX-19 | Current execution/review/material/expiry presentation and actual copied host-profile guidance; no browser signing or keys; independent signed review regression |
| UX-20 | Automated axe scans with zero serious/critical findings in covered states, keyboard confirmation, focus recovery, 390px and equivalent 200%/400% reflow; manual limitations stated above |

F01–F08 are covered by creation, Objective/Task preparation, visual planning, assessment, Issue/Blocker handling, protocol execution/review and conflict/replay tests. Assessment binds the exact criterion definition. Evidence registration alone does not certify a criterion; waiver is not verified. Explicit Issue resolution retains separate Blockers unless the selected compound operation is requested. Tasks, Objectives, Outcomes and reviews preserve independent completion/authority. Moving/removing a Roadmap reference does not mutate operational work.

## Reexecuted verification

| Gate | Actual result |
| --- | --- |
| Real embedded-server browser journeys | **31 passed**, no skipped tests; [execution log](final-browser.log) |
| Action/copy/navigation unit contracts | **14 passed**, including single copy source and authored-value preservation; [execution log](final-web-unit.log) |
| Go unit/contracts with real PostgreSQL | **712 test/subtest pass events**, 18 packages passed, 7 opt-in/environment-specific test skips and 8 packages with no tests |
| Go race detector with real PostgreSQL | Same 712 passes, 18 passing packages and explicit skips; no failures |
| Storage/reference scope, pagination, priority and revocation | Memory, SQLite and real PostgreSQL included; 1,001 Tasks and Evidence per store, all eleven types, exact-ID, cursor and HTTP/MCP parity |
| Static/generated contracts | `go vet`, module tidy, signed/PostgreSQL/MCP/OpenAPI regeneration with no drift, event catalog and `actionlint` passed |
| Reporting contracts | Seven context/cost-report and four scenario-report checks passed |
| Package/release contracts | **17 passed**; [execution log](final-package-tests.log) |
| Actual full-disk recovery | Nonroot disconnected read-only container with bounded disposable tmpfs passed ENOSPC receipt recovery; [execution log](final-full-disk.log) |
| Distribution | Offline install of actual service/skill tarballs; Linux/Windows archive metadata/hashes; unsigned checkout/recovery, signed protected profiles and independent review; four skill destinations; HTTP/MCP/UI; graceful shutdown; corruption rejection; [execution log](final-distribution.log) |
| Reproducibility | All five native/npm archives and manifest/checksums byte-identical across two source paths; [execution log](final-reproducible.log) |
| Docker Compose | Real Dockerfile build, UID/GID 10001:10001, copied `.env.example` unchanged, HTTP/MCP and authorized query readiness, English shell/all embedded assets, CSP and Draft/Task persistence across restart; [exact scope](final-compose.json) |
| Screenshots | Ten actual isolated-server captures, public API demonstration data, no credentials; [gallery](../ui-workspace/README.md) |

The normal/race skips include the separately executed full-disk fixture, native OS-keyring tests requiring a desktop session, and opt-in throughput matrices. They are not quietly counted as passes. Native Windows CLI checks passed in hosted CI; the headless local sandbox does not claim native desktop keyring validation. Historical load/fuzz reports remain separate acceptance records; no new throughput claim is made by this UX program.

The Docker smoke used port 18109 to preserve an existing service on 8080 and a build-only system trust secret for this cloud's HTTPS proxy. The ordinary user workflow remains `cp .env.example .env` followed by `docker compose up --build -d`. The base Compose, Dockerfile and example file were not changed for that smoke. Default local image metadata is `dev/unknown`; the tested distribution archives carry the precise accepted source revision and version 0.3.0. Verification does not publish that version to npm, GHCR or GitHub Releases.

## Hosted checks and integration

All five hosted checks passed for the foundation/query/language/registry slices. For PRs #83–85, browser, packages, Ubuntu and native Windows checks passed. The primary backend job was interrupted by Docker Hub's anonymous image-pull limit (`toomanyrequests`), including a failed-job rerun; #84/#85 failed during PostgreSQL service initialization. Local backend/race/PostgreSQL/full-disk/container gates completed successfully. The owner explicitly allowed work to continue despite refused hosted jobs. No application failure is inferred from the image registry response, and the failed hosted check is not reported as green.

## Remaining human validation

Recruit representative users and record outcome/objective/task comprehension, unaided creation, reference selection and recovery timing against an explicitly unmeasured baseline. Manually inspect screen-reader announcements, focus and high native-browser zoom. These pending activities are validation, not unimplemented product behavior, and are retained in ROADMAP rather than manufactured percentages. The screenshots show executable flows, not adoption or usability success metrics.

To recreate demonstration captures after building `bin/wos` and installing `tests/web` dependencies:

```sh
node tools/acceptance/workspace-screenshots.mjs "$PWD" /tmp/wos-captures
```

The helper creates its own temporary SQLite fixture through public commands, captures ten screens and cleans its own service/database. It does not modify an existing instance.
