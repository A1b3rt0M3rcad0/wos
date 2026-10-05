# WOS Roadmap

**Status document:** live and mandatory  
**Canonical design:** `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`  
**Last reviewed:** 2026-10-05
**Current target:** Release 0.1  
**Current wave:** Wave 11 integration and completion audit E00–E11 (in progress)

This file records the real implementation state of WOS. It must be kept synchronized with the repository by every agent that changes planned work.

The architecture document describes what should exist. This ROADMAP records what actually exists.

## Status legend

- ✅ **Done** — acceptance criteria satisfied and relevant verification passes.
- 🚧 **In progress** — implementation has started but acceptance criteria are not yet complete.
- ⬜ **Planned** — not started in a meaningful implementation sense.
- ⛔ **Blocked** — cannot proceed because of an explicit dependency or decision.
- 🧭 **Decision required** — an administrative/architectural decision is still open.

## Repository baseline

| Item | Status | Notes |
| --- | --- | --- |
| GitHub repository | ✅ | `A1b3rt0M3rcad0/wos`, default branch `master` |
| Canonical architecture specification | ✅ | Stored under `docs/`; revision 2 |
| Public README | ✅ | Executable features, draft PR status and remaining acceptance explicitly separated |
| Go module | ✅ | `github.com/A1b3rt0M3rcad0/wos` |
| Repository ignore rules | ✅ | Go, local DB/data, environment/secrets, IDE files |
| AGENTS development contract | ✅ | Root `AGENTS.md` |
| Live implementation roadmap | ✅ | Root `ROADMAP.md` |
| Product package topology | ✅ | `packages/wos-core` + `packages/wos-api`; ADR-011 |
| Concrete Open Source license | ✅ | Apache 2.0 chosen by maintainer; LICENSE present |

## Milestones

| Milestone | Status | Definition |
| --- | --- | --- |
| M1 — Local vertical slice | ✅ | Core + SQLite + HTTP with Outcome, Objective, WorkItem and criteria |
| M2 — Full coordination | ✅ | Dependencies, Blockers, Issues, leases and documentary records |
| M3 — Planning and continuity | 🚧 | Roadmaps, snapshot/graph/timeline and MCP |
| M4 — Complete standalone | 🚧 | PostgreSQL parity, triggers/outbox, auth and packaging |
| Release 0.1 | ⬜ | Contracts, CI, examples, recovery and release criteria validated |

---

# Implementation waves

## Wave 01 — Public Core Foundation

**Status:** ✅ Done

**Goal:** establish a Go project usable as a public library and standalone server, with verifiable boundaries.

### Completed

- [x] Create standalone `wos` repository.
- [x] Define Go module path.
- [x] Add public architecture README.
- [x] Store canonical architecture and implementation plan.
- [x] Add repository `.gitignore`.
- [x] Add root `AGENTS.md`.
- [x] Add live `ROADMAP.md`.

### Implementation checklist

- [x] Establish `packages/wos-core/domain`.
- [x] Establish `packages/wos-core/ports`.
- [x] Add foundational ID / UUIDv7 value object.
- [x] Add `Scope`.
- [x] Add `EntityRef`.
- [x] Add `ActorRef`.
- [x] Add aggregate `Version` and Outcome revision types.
- [x] Add `Clock` port.
- [x] Add `IDGenerator` port.
- [x] Add stable domain/application error model.
- [x] Add transport-neutral `CommandContext` separating authenticated Principal, declared `ActorRef`, and optional external execution correlation.
- [x] Add `packages/wos-api/cmd/wos/main.go`.
- [x] Add `packages/wos-api/internal/server` bootstrap/configuration skeleton.
- [x] Add version command/output.
- [x] Add configuration validation skeleton.
- [x] Create `docs/adr/`.
- [x] Materialize foundational ADRs relevant to Wave 01.
- [x] Add static dependency-boundary verification.
- [x] Add unit tests for value objects and enum/ID serialization.
- [x] Enforce canonical flat `EntityRef` JSON addressing (`namespace_id`, `outcome_id`, `kind`, `id`).
- [x] Add an external compile test/example proving that public Core can be imported outside the module's internal packages.
- [x] Add GitHub Actions CI using the Go version declared by `go.mod`.
- [x] Ensure `go test ./...` passes.
- [x] Ensure process can print version and validate configuration.
- [x] Ensure `go vet ./...`, `go test -race ./...`, standalone build and smoke commands pass in CI.

### Verification

GitHub Actions validated commit `b5f7b0868fbf9673f5512fe8ae41cd2a68a2c744` with Go 1.27.1 on Linux. The successful Wave 01 foundation job executed:

- `gofmt` cleanliness check;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- `go build -o ./bin/wos ./cmd/wos`;
- `./bin/wos version`;
- `./bin/wos config validate`.

The test suite includes the external-module Core import contract and static dependency-boundary verification, so the four Wave 01 completion gates are satisfied on the declared Go toolchain.

### Completion gate

Wave 01 is complete only when:

- `go test ./...` passes;
- external Core import example compiles;
- no prohibited dependency enters Core;
- the binary can print its version and validate configuration.

**Target integration commit:**  
`chore: bootstrap standalone WOS module and public core boundaries`

---

## Wave 02 — Transactions, Memory and Initial Domain

**Status:** ✅ Done

Implement Outcome, Objective, WorkItem and criteria with pure rules; application services; repository/transaction ports; transactional memory adapter; rollback; initial claim/complete semantics and Outcome coordination revision in memory.

### Implementation checklist

- [x] Implement Outcome lifecycle rules.
- [x] Implement Objective lifecycle rules.
- [x] Implement WorkItem lifecycle rules.
- [x] Keep archive state orthogonal to Outcome lifecycle.
- [x] Add versioned SuccessCriterion definitions.
- [x] Invalidate current assessment when a criterion revision changes.
- [x] Add minimum attestation assessments for Wave 02.
- [x] Enforce required-criterion gates before achievement.
- [x] Preserve explicit Conclusions and historical Conclusions on reopen.
- [x] Preserve immutable CriterionAssessment history while keeping a separate current-assessment projection.
- [x] Add basic WorkItem claim/complete semantics with Principal, TTL and fencing token.
- [x] Add declarative Outcome/Objective owners and WorkItem assignees, separate from lease ownership.
- [x] Keep WorkItem completion separate from Objective achievement.
- [x] Add aggregate optimistic versions.
- [x] Add repository ports for Outcome, Objective and WorkItem.
- [x] Add TransactionManager / UnitOfWork / CoordinationStore ports.
- [x] Add public transactional `packages/wos-core/storage/memory` adapter.
- [x] Add rollback behavior to memory transactions.
- [x] Add in-memory Outcome coordination guard/revision marker.
- [x] Add application services over ports instead of direct adapter access.
- [x] Expose the complete Wave 02 lifecycle command surface through Application services.
- [x] Enforce `required_for_outcome` before Outcome achievement.
- [x] Add human-only end-to-end scenario with no DB, network or Agent dependency.
- [x] Add failed-command rollback test for state and `outcome_revision`.
- [x] Validate aggregate lifecycle/lease/conclusion/criterion invariants at the domain boundary.
- [x] Reject structurally invalid aggregates before memory adapter saves.
- [x] Pass repository CI on the declared Go toolchain.
- [x] Review CI findings and close remaining Wave 02 contract gaps.

### Verification

GitHub Actions validated implementation commit `6590842a76b9fd5d6d4a370f1a8f8f4b0e5e41d2` with Go 1.27.1 on Linux. The successful run executed:

- `gofmt` cleanliness;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- standalone binary build;
- `wos version` smoke test;
- `wos config validate` smoke test.

The suite covers pure lifecycle rules, criterion revision/assessment semantics, ownership/assignment separation, optimistic versions, transactional memory rollback, invalid aggregate rejection, Outcome revision markers, required-objective achievement gates and a full human-only scenario without database, network or Agent dependencies.

### Scope boundary

Wave 02 does **not** add Domain Events, IdempotencyStore, command replay or event timeline behavior. Those remain Wave 03.

**Completion gate:** a human-only scenario can create/activate an Outcome and organize Objectives/WorkItems without database, network or Agent dependencies.

**Target commit:**  
`feat(core): implement outcome objective work item and criterion lifecycles`

---

## Wave 03 — Event Log and Idempotency

**Status:** ✅ Done

Add command envelope, immutable Domain Events, ordered Outcome revisions, IdempotencyStore, normalized fingerprints and transactional replay.

### Implementation checklist

- [x] Add immutable DomainEvent envelope with command, actor, execution and ordering metadata.
- [x] Add event validation and 256 KiB payload bound.
- [x] Add CausationID to transport-neutral CommandContext.
- [x] Keep administrative audit records structurally separate from Outcome Domain Events.
- [x] Add IdempotencyIdentity and safe key validation.
- [x] Add persisted StoredCommandResult contract.
- [x] Add DomainEventLog and IdempotencyStore ports to UnitOfWork.
- [x] Add transactional in-memory Event Log.
- [x] Add transactional in-memory idempotency reservations/results.
- [x] Ensure rollback discards events and idempotency state.
- [x] Add normalized SHA-256 command fingerprints excluding retry-observational context.
- [x] Normalize declarative actor sets before fingerprinting.
- [x] Wrap all Wave 02 application mutations in the transactional audit/idempotency pipeline.
- [x] Preserve original IDs, versions, revision and command_id on successful replay.
- [x] Reject same idempotency identity with a different fingerprint.
- [x] Emit multiple ordered Events under one outcome_revision for compound command effects.
- [x] Implement explicit no-op semantics without version/revision advancement or misleading Event.
- [x] Add memory timeline snapshot ordered by (outcome_revision,event_index).
- [x] Add replay/conflict/no-op/rollback/event-ordering tests.
- [x] Pass repository CI on the declared Go toolchain.
- [x] Review CI findings and close remaining Wave 03 contract gaps.

### Verification

GitHub Actions validated commit `d9d380712721b58c7484c4a59c9244894cac73da` with Go 1.27.1 on Linux. The successful `WOS verification` run executed:

- `gofmt` cleanliness;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- standalone binary build;
- `wos version` smoke test;
- `wos config validate` smoke test.

The suite verifies same-payload replay, different-payload conflict, transactional rollback of Event/idempotency state, multiple ordered Events under one Outcome revision, original result replay and explicit no-op behavior without misleading Events.

### Scope boundary

Wave 03 does **not** add SQLite persistence, HTTP/MCP transport requirements, Integration Events, trigger firings or outbox delivery. Those remain later waves.

**Completion gate:** repeated CreateWorkItem returns the original ID without duplicate Events, and timeline ordering is stable.

**Target commit:**  
`feat(core): add transactional audit events and command idempotency`

---

## Wave 04 — SQLite Persistence and Migrations

**Status:** ✅ Done

Implement SQLite adapter, schema migrations, SQL UnitOfWork, registry/coordination rows, repository contracts, idempotency/event persistence, restart-safe state and SQLite writer coordination.

### Implementation checklist

- [x] Pin the SQLite `database/sql` driver used by the adapter.
- [x] Record the SQLite driver/writer-acquisition decision in ADR 0006.
- [x] Enable `foreign_keys` for every adapter connection.
- [x] Enable WAL and synchronous NORMAL for the local durability profile.
- [x] Configure bounded `busy_timeout` integrated with command contexts.
- [x] Acquire the SQLite writer lock at transaction start with the immediate transaction profile.
- [x] Restrict the Store to one writer connection instead of pretending SQLite has parallel writers.
- [x] Add embedded, forward-only numbered migrations.
- [x] Add `schema_migrations` with SHA-256 checksum verification.
- [x] Add the canonical UTC microsecond timestamp codec.
- [x] Add Namespace, Principal, entity registry and Outcome coordination tables.
- [x] Add normalized relational persistence for Outcome, Objective and WorkItem.
- [x] Add normalized ownership/assignee links.
- [x] Persist revisioned SuccessCriteria and immutable CriterionAssessment history/current projection.
- [x] Persist current and historical Conclusions.
- [x] Persist Domain Events in the same SQL UnitOfWork as current state.
- [x] Persist idempotency reservations/results in the same SQL UnitOfWork.
- [x] Enforce optimistic writes with `UPDATE ... WHERE version = ?`.
- [x] Distinguish not-found from stale-version conflicts after zero-row updates.
- [x] Persist lease principals before FK-protected WorkItem lease writes.
- [x] Add restart-safe reconstruction of Wave 02 aggregates and nested state.
- [x] Add persisted idempotent replay across process restart.
- [x] Add compound rollback coverage for state + Outcome revision + Event + idempotency.
- [x] Add cross-scope foreign-key rejection coverage.
- [x] Add SQLite connection-profile and migration-idempotency tests.
- [x] Add consistent `VACUUM ... INTO` backup plus closed-destination restore.
- [x] Add writer-contention/busy-timeout context coverage.
- [x] Add transaction-cancellation coverage.
- [x] Add explicit claim-by-new-principal FK coverage.
- [x] Pin and verify the Go module graph with CI module hygiene.
- [x] Pass `go vet`, full unit/contract tests and race detector.

### Verification

GitHub Actions validated implementation commit `1895c642d2c7155c737d1d3adf9d95923d0889d0` with Go 1.27.1 on Linux in run `36963863527`.

The successful verification executed:

- module graph/tidy verification;
- `gofmt` cleanliness;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- standalone binary build;
- `wos version` smoke test;
- `wos config validate` smoke test.

The SQLite suite verifies the full Wave 02 scenario after close/reopen, Outcome revision preservation, nested criteria/assessment/conclusion reconstruction, persisted Event ordering, replay after restart, stale `expected_version` rejection, compound rollback, scope foreign keys, backup/restore, writer contention, transaction cancellation and lease-principal integrity.

### Scope boundary

Wave 04 does **not** add the HTTP transport, transport-level authentication/local principal resolution, ETags/preconditions, OpenAPI or the first durable state query. Those remain Wave 05.

**Completion gate:** satisfied — the Wave 02 scenario survives process restart and versioned writes reject stale `expected_version`.

**Target commit:**  
`feat(storage): add SQLite persistence migrations and outcome coordination guards`

---

## Wave 05 — HTTP Vertical Slice

**Status:** ✅ Done

Expose the M1 domain through HTTP with DTOs, auth-local mode, version preconditions, idempotency keys, ETags, error mapping, OpenAPI and first state query.

### Completed

- [x] Add standalone SQLite + local-auth HTTP runtime and `wos server` command.
- [x] Add `/livez` and storage-backed `/readyz` health endpoints.
- [x] Expose Outcome, Objective, WorkItem and SuccessCriterion operations through versioned HTTP routes.
- [x] Require `Idempotency-Key` on remote mutations.
- [x] Translate aggregate versions through strong ETags and `If-Match` / `expected_version`.
- [x] Preserve/request correlation IDs and return a stable JSON error envelope.
- [x] Add the first coherent Outcome state query.
- [x] Add OpenAPI 3.1 contract under `packages/wos-api/openapi.yaml`.
- [x] Add reproducible HTTP documentation under `docs/http.md`.
- [x] Add transport contract coverage for replay, stale versions, no-op writes, invalid payloads, scope isolation and payload bounds.
- [x] Add durable SQLite HTTP restart coverage.
- [x] Add standalone HTTP runtime smoke verification to CI.
- [x] Pass formatting, vet, unit/contract tests, race detector, standalone build and runtime smoke verification.

### Verification

GitHub Actions run `36967299973` validated head `9a6cc4af2526420b6a8a6b6bbff209bed480f24a` with the complete verification job, including the standalone HTTP runtime smoke test. The suite includes a durable create/update/restart/read scenario over SQLite and an M1 HTTP contract scenario for Outcome, Objective, WorkItem and criteria.

**Completion gate:** satisfied — the standalone binary can create, modify, persist, restart and query durable state through HTTP, with reproducible documentation and a machine-readable OpenAPI contract.

**Target commit:**  
`feat(http): expose versioned outcome coordination API`

---

## Wave 06 — Dependencies, Graph and Readiness

**Status:** ✅ Done

Implement canonical `depends_on` relations, hard/advisory semantics, DAG validation, Objective hierarchy constraints, deterministic readiness, `not_before`, readiness reasons and ready-work queries.

### Completed

- [x] Add typed Outcome-local `Relation/depends_on` domain model with hard/advisory strength and `target_completed` satisfaction semantics.
- [x] Reject self-dependencies, invalid endpoint kinds, cross-Outcome references, duplicate active semantic edges and direct/indirect dependency cycles.
- [x] Validate DAG mutations while holding the Outcome coordination guard so opposite concurrent edge insertion cannot commit a cycle.
- [x] Preserve explicit audit reasons for dependency changes and require a reason when adding a dependency to an in-progress source.
- [x] Prevent dependency changes on terminal sources until they are reopened.
- [x] Enforce hard dependencies on Objective achievement and WorkItem completion without allowing advisory dependencies to block transitions.
- [x] Add deterministic Objective/WorkItem readiness policies and explicit readiness reasons.
- [x] Persist and restore WorkItem `not_before` in memory/SQLite and honor the exact `not_before <= evaluated_at` boundary.
- [x] Add deterministic `ListReadyWork` ordering by priority, creation time and ID.
- [x] Persist Relations through memory and SQLite UnitOfWork repositories with migration `0002_relations.sql`.
- [x] Include Relations in Outcome state and expose relation read/list operations.
- [x] Expose HTTP `/relations`, relation removal and `/ready-work` through the same Application services.
- [x] Extend OpenAPI and HTTP documentation for dependency/readiness contracts.
- [x] Prove hard/advisory behavior, cancelled-target semantics, dependency removal, in-progress mutation rules and indirect cycle rejection.
- [x] Prove memory/SQLite readiness parity, SQLite restart durability and concurrent opposite-edge rejection.
- [x] Pass formatting, vet, unit/contract tests, race detector, standalone build and runtime smoke verification.

### Verification

CI run `37008538950` (#97) validated implementation head `8549346c96e6e2c7f940b3c681158ac0064e011c` with module hygiene, formatting, vet, the full unit/contract suite, race detector, standalone build, version/configuration smoke checks and standalone HTTP runtime smoke verification.

**Completion gate:** satisfied — memory and SQLite produce equivalent readiness results for the tested state/time boundaries, and concurrent opposite dependency insertion cannot commit a dependency cycle.

**Target commit:**  
`feat(core): add dependency validation and deterministic work readiness`

---

## Wave 07 — Issues and Blockers

**Status:** ✅ Done and merged into `master`

Implement independent Issue and Blocker aggregates, typed causes/targets, direct/subtree propagation, explicit resolution, inherited blocking projection and bounded compound commands.

### Completed

- [x] Add independent Issue and Blocker aggregates with explicit versions and lifecycles.
- [x] Add Issue severity, affected references, investigation, resolution, wont-fix, duplicate and reopen semantics.
- [x] Reject duplicate-Issue cycles with a bounded chain walk.
- [x] Add Blocker targets for Outcome, Objective and WorkItem with direct/subtree propagation rules.
- [x] Support persisted local causes and explicit external causes while keeping Issue and Blocker lifecycles independent.
- [x] Reject new active Blockers on terminal targets.
- [x] Derive deterministic direct/inherited blocking state with explicit origin information.
- [x] Apply active blocking to WorkItem readiness without hiding the underlying Blockers.
- [x] Ensure resolving an Issue never resolves any Blocker implicitly.
- [x] Ensure resolving one Blocker never releases another active impediment.
- [x] Add `ReportIssueWithBlocker` as one atomic multi-aggregate command with separate per-aggregate Domain Events.
- [x] Add bounded `ResolveIssueAndBlockers` with explicit Blocker IDs, versions and `release_confirmed` values.
- [x] Keep compound mutations under one `command_id` and one `outcome_revision` while emitting one Event per changed aggregate.
- [x] Add versioned Issue detail updates and active-Blocker description updates with explicit no-op semantics.
- [x] Validate Issue `affected_refs` against persisted entities before mutation so memory and SQLite fail consistently.
- [x] Keep Decision as a reserved Blocker-cause domain kind but reject it at the Application boundary until persisted Decision records exist in Wave 09.
- [x] Persist Issues, affected references and Blockers through memory and SQLite UnitOfWork repositories.
- [x] Add SQLite migration and restart-safe reconstruction of Issue/Blocker state and blocking projection.
- [x] Extend Outcome state snapshots with Issues, Blockers and `blocking_states`.
- [x] Expose Issue/Blocker create/read/list/update/lifecycle operations through HTTP.
- [x] Expose the atomic report and explicit multi-aggregate resolution commands through HTTP.
- [x] Preserve idempotent replay, ETags and optimistic version preconditions for the new transport surface.
- [x] Extend OpenAPI and HTTP documentation through Wave 07.
- [x] Prove inherited subtree blocking, independent resolution, terminal-target rejection and duplicate-cycle rejection.
- [x] Prove compound rollback when one expected version is stale and prove unlisted Blockers remain active.
- [x] Prove the end-to-end HTTP contract, including replay, state projection, explicit release and external causes.
- [x] Pass formatting, vet, full unit/contract tests, race detector, standalone build and runtime smoke verification.

### Verification

GitHub Actions run `37020264539` validated implementation head `c21e454071f2159538af57e5e927330c52d7c979` with module hygiene, formatting, `go vet ./...`, the complete unit/contract suite, `go test -race ./...`, standalone build and all smoke checks.

The verified suite includes direct/inherited blocking, Issue/Blocker lifecycle independence, SQLite restart durability, atomic compound commands, rollback on stale multi-aggregate versions, explicit release confirmation, idempotent HTTP replay, versioned HTTP mutations and readiness recovery only after every applicable Blocker is closed.

### Scope boundary

`Decision` is already a valid Blocker-cause kind in the domain model, but Wave 07 does not invent non-persisted Decisions. The Application layer therefore rejects a Decision cause until the documentary-record implementation in Wave 09 can resolve that reference identically in every storage adapter.

**Completion gate:** satisfied — snapshots explain direct/inherited target/cause state, and resolving an Issue or one Blocker never silently resolves any other active Blocker.

**Target integration commit:**  
`feat(core): separate issues from blockers and implement blocking projections`

---

## Wave 08 — Leases and Fencing

**Status:** ✅ Done and merged into `master`

Harden WorkItem execution coordination with Claim/Renew/Release/Reclaim, principal-bound leases, TTL, fencing tokens, expiration semantics, attention-needed projection and audited administrative override.

### Initial implementation

- [x] Preserve the existing Claim/Release contract and harden lease TTL validation.
- [x] Add owner-bound lease renewal without rotating claim identity or fencing token.
- [x] Add explicit reclaim of expired in-progress work with a new claim ID and monotonically increasing fencing token.
- [x] Reject stale claimant completion after reclaim.
- [x] Add Application commands for Renew/Reclaim through the existing transaction, idempotency and Domain Event pipeline.
- [x] Emit explicit `work_item.lease_renewed` and `work_item.reclaimed` events.
- [x] Add deterministic `lease_status` and WorkItem `display_state` projection, including `attention_needed` for expired leases.

### Remaining work

- [x] Integrate the operational projection into the broader Outcome continuity/state queries.
- [x] Add audited administrative override semantics and the required authorization boundary.
- [x] Expose Renew/Reclaim and operational lease state through HTTP, OpenAPI and HTTP documentation.
- [x] Add SQLite restart/parity and concentrated concurrent-claim/reclaim contract coverage for the hardened semantics.
- [x] Complete the Wave 08 end-to-end transport and concurrency verification before marking the wave done.

### Administrative override boundary

- [x] Normal cancellation cannot silently terminate an in-progress leased WorkItem.
- [x] Add explicit `work:admin_cancel` and `work:admin_complete` permissions behind a transport-neutral Authorizer port.
- [x] Default privileged authorization to deny unless a deployment composes an explicit authorizer.
- [x] Keep local standalone overrides disabled unless `WOS_LOCAL_ADMIN_OVERRIDES=true` is explicitly configured.
- [x] Audit privileged state changes through Outcome-scoped Domain Events with principal, actor, command and reason.
- [x] Administrative completion overrides lease ownership/expiry only; Blocker, hard-dependency and completion-evidence gates remain active.

**Completion gate:** satisfied on the implementation branch — only one current lease can survive claim/reclaim races, fencing tokens advance monotonically, stale claimants cannot complete after reclaim, and bypassing lease ownership requires an explicit privileged permission and auditable reason.

**Target commit:**  
`feat(coordination): add work item leases and fencing tokens`

---

## Wave 09 — Documentary Records and Decisions

**Status:** ✅ Done and merged into `master`

**Goal:** preserve durable facts, deliverables and explicit choices with enough provenance that another consumer can understand the current state without relying on previous chat/session history.

### Domain contract

- [x] Add `Artifact` aggregate with immutable documentary identity/content metadata and `registered|withdrawn` lifecycle.
- [x] Add `Evidence` aggregate with immutable observation content and `registered|retracted` lifecycle.
- [x] Add `EvidenceLink` aggregate with link-local stance (`supports|contradicts|context`), rationale and `active|retracted` lifecycle.
- [x] Add `Decision` aggregate with `proposed|accepted|rejected|superseded` lifecycle.
- [x] Permit editing only while a Decision is `proposed`; accepted/rejected/superseded decision content is immutable.
- [x] Require explicit reason for Evidence retraction, Artifact withdrawal and Decision rejection.
- [x] Preserve producer/source provenance without claiming external authenticity or truth.
- [x] Keep Artifact registration reference-only: WOS must not dereference URIs or store arbitrary artifact bytes.
- [x] Keep EvidenceLink separate from criterion assessment: linking Evidence never marks a SuccessCriterion as met.
- [x] Validate all local targets as same Namespace + Outcome and compatible entity kinds.
- [x] Allow Evidence to carry different stances toward different targets.
- [x] Model Decision supersession as a dedicated semantic relation, not as a generic `Relation`.
- [x] Enforce no supersession cycles and at most one accepted direct successor for an accepted Decision.
- [x] Supersede an accepted Decision atomically when the successor Decision is accepted.

### Application and transaction contract

- [x] Add create/read/list commands for Artifact, Evidence, EvidenceLink and Decision.
- [x] Add explicit retract/withdraw commands with optimistic `expected_version`.
- [x] Add Decision update/accept/reject/supersede commands.
- [x] Route all Wave 09 mutations through the existing transaction + idempotency + Domain Event pipeline.
- [x] Hold the Outcome coordination guard for cross-aggregate reference validation and Decision supersession.
- [x] Emit canonical events: `artifact.registered`, `artifact.withdrawn`, `evidence.registered`, `evidence.retracted`, `evidence.link_created`, `evidence.link_retracted`, `decision.proposed`, `decision.updated`, `decision.accepted`, `decision.rejected`, `decision.superseded`.
- [x] Preserve no-op semantics without advancing aggregate version/outcome revision or emitting misleading events.

### Storage contract

- [x] Extend UnitOfWork/ports with Artifact, Evidence, EvidenceLink and Decision repositories.
- [x] Implement transactional memory repositories with clone/rollback parity.
- [x] Add SQLite migration `0004_documentary_records.sql`.
- [x] Persist provenance, immutable content fields, lifecycle, versions and Decision supersession constraints.
- [x] Register Wave 09 aggregates in the local entity registry.
- [x] Preserve all records and supersession state across SQLite restart.
- [x] Reject cross-scope references at both application and persistence boundaries where possible.

### HTTP and contract surface

- [x] Expose versioned HTTP routes for Artifact, Evidence, EvidenceLink and Decision operations.
- [x] Require `Idempotency-Key` on remote mutations.
- [x] Preserve ETag / `If-Match` semantics for mutable lifecycle/version transitions.
- [x] Extend OpenAPI 3.1 with Wave 09 schemas, enums, requests, responses and error behavior.
- [x] Extend `docs/http.md` with reproducible documentary-record scenarios.

### Verification

- [x] Prove registering an Artifact cannot conclude an Objective, WorkItem or Outcome.
- [x] Prove the same Evidence can support one target and contradict another.
- [x] Prove Evidence retraction does not rewrite the original observation.
- [x] Prove Artifact withdrawal does not delete historical metadata.
- [x] Prove accepted Decision content cannot be edited.
- [x] Prove Decision rejection requires a reason.
- [x] Prove two concurrent attempts to supersede the same accepted Decision cannot both commit accepted successors.
- [x] Prove supersession is atomic: successor acceptance and predecessor superseded state commit/rollback together.
- [x] Prove supersession cycles are rejected.
- [x] Prove cross-Outcome and cross-Namespace documentary links are rejected.
- [x] Prove memory and SQLite behavior parity for the Wave 09 lifecycle.
- [x] Prove SQLite restart preserves documentary history and current Decision state.
- [x] Pass module hygiene, formatting, vet, full unit/contract tests, race detector, standalone build and HTTP runtime smoke verification.

**Completion gate:** another consumer can reconstruct the current factual/documentary context and active choices of an Outcome from WOS state alone, including provenance, contradictory Evidence, withdrawn/retracted records and Decision supersession history, without previous chat history.

**Target commit:**  
`feat(records): add evidence artifacts and explicit decision history`
---

## Wave 10 — Assessments and Verifiable Conclusions

**Status:** ✅ Done and merged into `master`

**Goal:** make every positive conclusion reproducible from immutable criterion definitions, immutable assessments and an explicit obligation snapshot, while preserving later contradictory evidence/assessments as contestations rather than silently rewriting lifecycle.

### Domain contract

- [ ] Preserve an immutable definition snapshot for every SuccessCriterion revision.
- [ ] Keep the current SuccessCriterion definition separate from its immutable revision history.
- [x] Extend CriterionAssessment with explicit `evidence_ids` and optional `evaluator_ref`.
- [x] Support `met|not_met|inconclusive|waived` without conflating EvidenceLink with assessment.
- [x] Require `evidence_review` assessments to reference at least one active, same-Outcome Evidence.
- [ ] Require `external_evaluation` assessments to include evaluator identity/version metadata and evidence required by the caller contract.
- [x] Require explicit authorization and reason for `waived`.
- [x] Keep CriterionAssessment immutable and retain supersession history.
- [x] Clear the current-assessment binding when a criterion is revised or retired; retain assessment history.
- [ ] Introduce first-class Conclusion identity/history for Outcome, Objective and WorkItem.
- [x] Store the exact owner version, lifecycle result, criterion revisions, assessment IDs and structural obligations used by a Conclusion.
- [x] Reopening clears only the current Conclusion binding; historical Conclusions remain immutable.
- [x] Project `conclusion_contested` and concrete causes when current Evidence/assessments contradict the recorded Conclusion.
- [x] Never auto-reopen or auto-change lifecycle because a Conclusion becomes contested.

### Application and authorization contract

- [x] Replace the Wave 02 attestation-only command path with a generic RecordCriterionAssessment path.
- [x] Preserve attestation as a compatibility/application specialization where useful, but not as the domain's only assessment mode.
- [x] Add `assessment:waive` permission to the Authorizer contract.
- [x] Validate assessment Evidence against Namespace/Outcome, lifecycle and criterion revision inside the Outcome transaction guard.
- [x] Require expected owner version when updating the current assessment binding.
- [ ] Detect concurrent assessments/current-binding replacements with optimistic version conflict.
- [ ] Make AchieveOutcome/AchieveObjective/CompleteWorkItem build explicit immutable Conclusion records.
- [x] Enforce required Objective obligations for Outcome achievement.
- [x] Capture structural obligation snapshots before the terminal transition commits.
- [ ] Emit canonical assessment/conclusion Domain Events through the existing transaction/idempotency pipeline.

### Storage contract

- [ ] Treat `criterion_revisions`, `criterion_assessments`, `conclusions` and conclusion-assessment links as append-only historical records.
- [x] Persist assessment Evidence references and evaluator metadata.
- [x] Persist current-assessment/current-conclusion bindings separately from historical rows.
- [x] Add a Wave 10 SQLite migration for fields/relations missing from the predeclared foundation schema.
- [x] Reject assessment references to nonexistent criterion revisions.
- [x] Preserve criterion revision history, assessment history, current bindings and Conclusions across restart.
- [x] Keep memory and SQLite behavior equivalent.

### HTTP and contract surface

- [x] Expose generic assessment recording for all verification modes.
- [x] Require Idempotency-Key for assessment/conclusion mutations and ETag/If-Match for owner-bound current state.
- [x] Expose immutable assessment history and current bindings.
- [x] Expose current/historical Conclusions and contestation state.
- [x] Extend OpenAPI 3.1 and `docs/http.md` with Wave 10 contracts.

### Verification

- [x] Prove an assessment for criterion revision N never satisfies revision N+1.
- [x] Prove revising a criterion clears only the current binding and preserves old assessments.
- [x] Prove `evidence_review` without Evidence is rejected.
- [x] Prove retracted Evidence cannot support a new positive assessment/conclusion.
- [x] Prove a waiver without `assessment:waive` authorization is rejected.
- [x] Prove two concurrent assessments cannot silently replace the same current binding.
- [x] Prove required Objectives block Outcome achievement when not achieved.
- [x] Prove a Conclusion stores exactly the criterion revisions and assessment IDs used at commit time.
- [x] Prove later `not_met` assessment or Evidence retraction contests but does not erase/reopen a terminal entity.
- [x] Prove memory/SQLite restart preserves immutable validation history.
- [x] Pass module hygiene, gofmt, vet, unit/contract tests, race detector, standalone build and HTTP runtime smoke.

**Completion gate:** achievement records exactly which criterion revisions, assessments and structural obligations justified the conclusion, and later contradictory facts are visible as contestations without historical rewriting.

**Target commit:**  
`feat(validation): enforce versioned criterion assessments and explicit achievement`

---

## Wave 11 — Roadmaps and Planning History

**Status:** 🚧 In progress

**Goal:** preserve planning and re-planning as durable history without creating a second operational graph or changing live execution state implicitly.

### Domain contract

- [x] Introduce Roadmap as an Outcome-scoped aggregate with `scope_kind=outcome|objective`, `scope_id`, title, lifecycle `open|archived` and optimistic `version`.
- [x] Keep Roadmap ownership separate from operational entities: Roadmap references Objectives/WorkItems and never owns or duplicates them.
- [x] Introduce editable Roadmap drafts with `draft_version` independent from aggregate `version` and published `revision_number`.
- [x] Introduce immutable RoadmapRevision records with content hash and publication metadata.
- [x] Introduce stable `node_key` identity across revisions.
- [x] Support RoadmapNode kinds `reference|phase|milestone`.
- [x] For reference nodes, allow only Objective or WorkItem targets from the same Outcome and permitted Roadmap scope.
- [x] Keep phase/milestone planning-only; they do not gain operational lifecycle, Blockers or leases.
- [x] Support optional parent grouping, presentation `position`, `after` ordering links, planned start/end and criterion references.
- [x] Reject duplicate reference targets within one revision.
- [x] Reject cycles in parent grouping and `after` plan ordering.
- [x] Snapshot published reference labels/scope so historical revisions remain understandable after live entities change.
- [x] Snapshot informative live dependency edges between referenced entities at publication time; do not duplicate operational dependencies as mutable Roadmap state.

### Publication and activation contract

- [x] Draft editing requires both `expected_roadmap_version` and `expected_draft_version`.
- [x] Published revision content is immutable.
- [x] Publishing computes/stores a deterministic content hash.
- [x] Publishing validates all references, scope restrictions, criterion references and plan DAG invariants.
- [x] `PublishRoadmapRevision` may include an explicit batch of operational dependency changes and applies plan publication + dependency changes atomically.
- [x] Absence of a plan edge or dependency never implies an execution command.
- [x] Introduce one active Roadmap revision slot per scope.
- [x] Activation only accepts a published revision belonging to the matching scope.
- [x] Activation history is append-only and distinguishes activation/supersession/deactivation without mutating published revision content.
- [x] Archiving a Roadmap clears its active slot in the same transaction.
- [x] Reopening a Roadmap never silently reactivates a prior revision.

### Storage contract

- [x] Add repository ports for Roadmap aggregate, drafts, published revisions and active-slot/history access.
- [x] Implement Memory adapter with deep-copy and append-only publication/history enforcement.
- [x] Add SQLite migrations/tables for roadmaps, drafts, revisions, nodes, plan edges, reference snapshots, active slots and activation history.
- [x] Enforce unique active slot per scope and unique published revision number per Roadmap.
- [x] Preserve all published revisions and activation history across restart.
- [x] Keep Memory/SQLite behavior equivalent.

### Application contract

- [x] Create Roadmap in Outcome or Objective scope.
- [x] Open a draft from empty content or an existing published revision.
- [x] Edit draft metadata/nodes/ordering with optimistic concurrency.
- [x] Discard draft explicitly; discarded draft cannot later be published.
- [x] Publish a draft atomically into an immutable RoadmapRevision.
- [x] Activate/deactivate a published revision for its scope.
- [x] Archive/reopen Roadmap with explicit lifecycle commands.
- [x] Ensure removing a node from a plan does not cancel/delete the referenced Objective/WorkItem.
- [x] Ensure cancelling operational work does not mutate historical Roadmap revisions.
- [x] Emit canonical Roadmap draft/publication/activation/archive Domain Events.

### HTTP and contract surface

- [x] Expose Roadmap create/read/list operations scoped to Outcome.
- [x] Expose draft open/edit/discard/publish operations with ETag/If-Match + Idempotency-Key.
- [x] Expose revision history and immutable revision reads.
- [x] Expose active Roadmap slots for Outcome and Objective scopes.
- [x] Extend OpenAPI 3.1 and `docs/http.md` with planning semantics and examples.

### Verification

- [x] Prove historical revisions remain byte/semantically stable after later draft edits.
- [x] Prove concurrent publication of the same draft has one winner.
- [x] Prove stale draft/roadmap versions fail explicitly.
- [x] Prove invalid cross-Outcome and invalid Objective-subtree references are rejected.
- [x] Prove parent and `after` cycles are rejected.
- [x] Prove one entity cannot appear twice as a reference in one revision.
- [x] Prove active slot uniqueness under concurrent activation.
- [x] Prove archived Roadmap clears the active slot and reopen does not reactivate it.
- [x] Prove node removal does not cancel live WorkItems/Objectives.
- [x] Prove cancelling live work does not rewrite prior revisions.
- [x] Prove title/criterion/reference snapshots survive live-entity mutation and SQLite restart.
- [ ] Pass module hygiene, gofmt, vet, tests, race detector, standalone build and HTTP runtime smoke.

**Completion gate:** historical published revisions remain recoverable and immutable; each scope has at most one active published revision; Roadmap state never duplicates or silently mutates operational work.

**Target commit:**  
`feat(planning): add immutable roadmap revisions and scoped activation`

---

## Wave 12 — Continuity Queries

**Status:** 🚧 Implemented in draft PR #14; acceptance in progress

Implement coherent Outcome state snapshots, graph queries, timeline, search/filter, pagination, readiness explanations, plan projection, current decisions, contestations, progress metrics and bounded output.

**Completion gate:** another consumer can resume the complete scenario from snapshot + cursors without the previous Session.

**Target commit:**  
`feat(queries): add coherent outcome state graph and timeline projections`

---

## Wave 13 — First-Class MCP

**Status:** 🚧 Implemented in draft PR #14; acceptance in progress

Expose the Application layer through the pinned official Go MCP SDK with tools/resources, schemas, stdio and Streamable HTTP, explicit scope/version/claim arguments and parity with HTTP semantics.

**Completion gate:** a real MCP client can inspect, claim, update and resume an Outcome without duplicated domain logic.

**Target commit:**  
`feat(mcp): expose outcome coordination tools resources and protocol profiles`

---

## Wave 14 — PostgreSQL Parity

**Status:** 🚧 Adapter implemented in draft PR #14; shared PostgreSQL contracts and race detector passed in CI #232; four runtime transport/session scenarios and clean integration restore passed in CI #233; proof/planning/lease restore passed in CI #234

Implement PostgreSQL migrations, repositories, query store, row guard, isolation rules, pool/timeouts and the full shared storage/concurrency contract.

**Completion gate:** SQLite and PostgreSQL satisfy the same functional contract and concurrency suite.

**Target commit:**  
`feat(storage): add PostgreSQL adapter with coordination contract parity`

---

## Wave 15 — Authentication, Grants and External Context

**Status:** 🚧 Implemented in draft PR #14; acceptance in progress

Implement token authentication, Namespace grants, Authorizer, Principal/ActorRef delegation, ExternalContext indexing, ExecutionContext separation, external refs and administrative audit.

**Completion gate:** namespace isolation and delegated authorship are enforced independently of consumer metadata.

**Target commit:**  
`feat(auth): add namespace grants delegated actors and external context indexing`

---

## Wave 16 — Triggers, Integration Events and Delivery

**Status:** 🚧 Implemented in draft PR #14; acceptance in progress

Implement declarative event Triggers, public Integration Event mapping, TriggerFiring deduplication, outbox delivery, webhook signing, endpoint policy, worker leases, retries/exhaustion and redelivery.

**Completion gate:** consumers receive durable at-least-once signals with stable identity, while WOS still never executes WorkItems or LLMs autonomously.

**Target commit:**  
`feat(integration): add declarative event triggers and durable webhook delivery`

---

## Wave 17 — Packaging, SDK and Examples

**Status:** 🚧 Implemented in draft PR #14; acceptance in progress

Add complete binary commands, Docker/Compose, health/readiness, graceful shutdown, Go client, embedded/standalone/Woobe/product-MCP examples, backup/restore and operator documentation.

**Completion gate:** a third party can run WOS without Woobe and resume an Outcome using published documentation.

**Target commit:**  
`feat(server): package standalone deployment embedded usage and Go client`

---

## Wave 18 — Release, Validation and Demonstration

**Status:** 🚧 Implemented in draft PR #14; acceptance in progress

Complete CI, contract suites, race detector, benchmark fixtures, failure injection, migration upgrade validation, schema review, transport/storage compatibility matrix, changelog and real multi-actor continuation demonstration.

**Completion gate:** every Release 0.1 criterion has verifiable evidence and no missing feature is represented as complete.

**Target commit:**  
`test: validate WOS release contracts concurrency recovery and performance`

---

# Release 0.1 acceptance checklist

- [ ] Standalone WOS starts without Woobe.
- [ ] External embedded Go application compiles and executes.
- [ ] Every remote mutation is protected by idempotency.
- [ ] Stale aggregate versions are rejected.
- [ ] Concurrent claim and dependency-cycle races preserve invariants.
- [ ] SQLite and PostgreSQL pass the same storage contract suite.
- [ ] HTTP and MCP use the same Application services.
- [ ] Snapshots are coherent, bounded and explicit about omissions.
- [ ] Published Roadmap revisions are immutable and recoverable.
- [ ] Assessments preserve criteria, revisions and Evidence used.
- [ ] Triggers produce durable signals without autonomous execution.
- [ ] Documentation covers license, auth, backup, limits and supported MCP profile.
- [ ] Restart/restore preserves state required to resume work.

# Current next actions

Waves 01–10 are merged into `master` at `2f9967689fa954b7c371781bc7a00c1ae760825d`. Wave 11 is implemented on unmerged PR #13. PR #14 builds on #13 and contains continuity, auth, MCP, PostgreSQL adapter, signals, official web, Go SDK and operations work. This is implementation status, not Release 0.1 acceptance.

1. Extend the exhaustive HTTP/MCP storage matrix and clean restore coverage. CI #232 passed real PostgreSQL 18.6 shared contracts and race detection after the nested-result reconstruction fix; four runtime transport/session scenarios and clean integration restore passed in CI #233. The proof/planning/lease restore extension passed in CI #234; full 18-step independent/Woobe client references passed locally, with the unmerged Woobe compatibility fix explicitly required.
2. Extend the passing official web journey (including replanning and restart) to evidence review, Objectives, Issues/Blockers, conflicts and full accessibility.
3. Expand active-plan live-projection tests and optimize aggregate continuity reads; publish SQL-query/guard/throughput measurements.
4. Prove full clean-install restore, in-flight shutdown, outbox crash exhaustion/redelivery, build reproducibility and container deployment.
5. Validate complete OpenAPI behavior and real Woobe/product consumers; associate all release obligations with passing evidence before integration or release.

## Reconciliation evidence — 2026-10-05

Wave 10 stale checkboxes were checked against `application/wave10_concurrency_contestation_test.go`, `wave10_contestation_test.go`, `storage/sqlite/wave10_assessment_test.go`, `wave10_conclusion_test.go`, `wave10_immutability_test.go`, HTTP criterion-history routes and `docs/http.md`/OpenAPI. Wave 11 stale checkboxes were checked against `wave11_completion_test.go`, `wave11_planning_test.go` and `storage/sqlite/wave11_roadmap_restart_test.go`. Local tests pass; the earlier PR #13 CI passed. PR #13 integration and PR #14 complete acceptance remain separate open gates.

The item/evidence matrix is `docs/auditoria-conclusao-2026-10-05.md`; execution results are `docs/verification-2026-10-05.md`. All unchecked release requirements remain obligations. No feature is counted as integrated in master merely because it exists in a draft PR.
