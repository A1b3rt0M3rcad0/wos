# WOS Roadmap

**Status document:** live and mandatory  
**Canonical design:** `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`  
**Last reviewed:** 2026-10-09
**Current target:** Release 0.1  
**Current wave:** Waves 01–18 accepted for validation; final integration through PR #14

**Distribution extension:** npm service/skills and automated releases requested by the owner; see D1–D4 below.

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
| M3 — Planning and continuity | ✅ | Roadmaps, snapshot/graph/timeline and MCP |
| M4 — Complete standalone | ✅ | PostgreSQL parity, triggers/outbox, auth and packaging |
| Release 0.1 validation candidate | ✅ | Thirteen canonical release gates have executable evidence; release tag publication is separate |

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

- [x] Preserve an immutable definition snapshot for every SuccessCriterion revision.
- [x] Keep the current SuccessCriterion definition separate from its immutable revision history.
- [x] Extend CriterionAssessment with explicit `evidence_ids` and optional `evaluator_ref`.
- [x] Support `met|not_met|inconclusive|waived` without conflating EvidenceLink with assessment.
- [x] Require `evidence_review` assessments to reference at least one active, same-Outcome Evidence.
- [x] Require `external_evaluation` assessments to include evaluator identity/version metadata and evidence required by the caller contract.
- [x] Require explicit authorization and reason for `waived`.
- [x] Keep CriterionAssessment immutable and retain supersession history.
- [x] Clear the current-assessment binding when a criterion is revised or retired; retain assessment history.
- [x] Introduce first-class Conclusion identity/history for Outcome, Objective and WorkItem.
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
- [x] Detect concurrent assessments/current-binding replacements with optimistic version conflict.
- [x] Make AchieveOutcome/AchieveObjective/CompleteWorkItem build explicit immutable Conclusion records.
- [x] Enforce required Objective obligations for Outcome achievement.
- [x] Capture structural obligation snapshots before the terminal transition commits.
- [x] Emit canonical assessment/conclusion Domain Events through the existing transaction/idempotency pipeline.

### Storage contract

- [x] Treat `criterion_revisions`, `criterion_assessments`, `conclusions` and conclusion-assessment links as append-only historical records.
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

**Status:** ✅ Done; integrated through PR #13

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
- [x] Pass module hygiene, gofmt, vet, tests, race detector, standalone build and HTTP runtime smoke.

**Completion gate:** historical published revisions remain recoverable and immutable; each scope has at most one active published revision; Roadmap state never duplicates or silently mutates operational work.

**Target commit:**  
`feat(planning): add immutable roadmap revisions and scoped activation`

---

## Wave 12 — Continuity Queries

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement coherent Outcome state snapshots, graph queries, timeline, search/filter, pagination, readiness explanations, plan projection, current decisions, contestations, progress metrics and bounded output.

**Completion gate:** another consumer can resume the complete scenario from snapshot + cursors without the previous Session.

**Target commit:**  
`feat(queries): add coherent outcome state graph and timeline projections`

---

## Wave 13 — First-Class MCP

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Expose the Application layer through the pinned official Go MCP SDK with tools/resources, schemas, stdio and Streamable HTTP, explicit scope/version/claim arguments and parity with HTTP semantics.

**Completion gate:** a real MCP client can inspect, claim, update and resume an Outcome without duplicated domain logic.

**Target commit:**  
`feat(mcp): expose outcome coordination tools resources and protocol profiles`

---

## Wave 14 — PostgreSQL Parity

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement PostgreSQL migrations, repositories, query store, row guard, isolation rules, pool/timeouts and the full shared storage/concurrency contract.

**Completion gate:** SQLite and PostgreSQL satisfy the same functional contract and concurrency suite.

**Target commit:**  
`feat(storage): add PostgreSQL adapter with coordination contract parity`

---

## Wave 15 — Authentication, Grants and External Context

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement token authentication, Namespace grants, Authorizer, Principal/ActorRef delegation, ExternalContext indexing, ExecutionContext separation, external refs and administrative audit.

**Completion gate:** namespace isolation and delegated authorship are enforced independently of consumer metadata.

**Target commit:**  
`feat(auth): add namespace grants delegated actors and external context indexing`

---

## Wave 16 — Triggers, Integration Events and Delivery

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement declarative event Triggers, public Integration Event mapping, TriggerFiring deduplication, outbox delivery, webhook signing, endpoint policy, worker leases, retries/exhaustion and redelivery.

**Completion gate:** consumers receive durable at-least-once signals with stable identity, while WOS still never executes WorkItems or LLMs autonomously.

**Target commit:**  
`feat(integration): add declarative event triggers and durable webhook delivery`

---

## Wave 17 — Packaging, SDK and Examples

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Add complete binary commands, Docker/Compose, health/readiness, graceful shutdown, Go client, embedded/standalone/Woobe/product-MCP examples, backup/restore and operator documentation.

**Completion gate:** a third party can run WOS without Woobe and resume an Outcome using published documentation.

**Target commit:**  
`feat(server): package standalone deployment embedded usage and Go client`

---

## Wave 18 — Release, Validation and Demonstration

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Complete CI, contract suites, race detector, benchmark fixtures, failure injection, migration upgrade validation, schema review, transport/storage compatibility matrix, changelog and real multi-actor continuation demonstration.

**Completion gate:** every Release 0.1 criterion has verifiable evidence and no missing feature is represented as complete.

**Target commit:**  
`test: validate WOS release contracts concurrency recovery and performance`

---

# Release 0.1 acceptance checklist

- [x] Standalone WOS starts without Woobe.
- [x] External embedded Go application compiles and executes.
- [x] Every remote mutation is protected by idempotency.
- [x] Stale aggregate versions are rejected.
- [x] Concurrent claim and dependency-cycle races preserve invariants.
- [x] SQLite and PostgreSQL pass the same storage contract suite.
- [x] HTTP and MCP use the same Application services.
- [x] Snapshots are coherent, bounded and explicit about omissions.
- [x] Published Roadmap revisions are immutable and recoverable.
- [x] Assessments preserve criteria, revisions and Evidence used.
- [x] Triggers produce durable signals without autonomous execution.
- [x] Documentation covers license, auth, backup, limits and supported MCP profile.
- [x] Restart/restore preserves state required to resume work.

# Current integration and validation

## Distribution extension — npm service, portable skills, automated releases

| Stage | Status | Acceptance |
| --- | --- | --- |
| D1 — Design and release policy | ✅ | [Complete plan](docs/distribution-plan.md) and ADR-017 define boundaries, tradeoffs, PR sequence, compatibility and external requirements |
| D2 — Portable skills installer | ✅ | Four portable skills, npm tarball, four project/user destinations and generic path; safe install/update/remove; four tests and offline installed-tarball consumer pass. Native runtime execution is not claimed |
| D3 — Service distribution | ✅ | Native metadata/integrity, versioned tarballs and Linux archive; real offline-installed service passes HTTP/MCP/UI, SIGTERM, exit status and corruption checks; Docker release metadata supported |
| D4 — Automated release | ✅ | Changes/version PR and reusable exact-source gates; verified immutable npm/GHCR/GitHub publication adapters and partial-failure replay; workflow lint, 15 Node contracts, reproducible assets and full local WOS gates pass. Hosted publication remains separately blocked |
| External publication | ⛔ | npm publishing identity not available; Actions billing prevents jobs; package ownership/trusted publisher requires titular configuration |

Do not conflate a portable skill with executed integration in every agent, nor implemented CI/CD with a successful hosted release. Original Waves 01–18 remain accepted; this is distribution work outside the original completion claim.

[Distribution verification](docs/verification-distribution-2026-10-07.md) records 437 Go cases with zero skipped tests, race, three browser journeys, the 18-step independent journey, installed npm service/skills, cross-path reproducibility and versioned read-only/authenticated container smoke. Nine Go packages without test files are not executed test cases. Publication requires working hosted infrastructure and titular credentials; a prepared 0.1.0 manifest is not a published npm version.

The first release is prepared as `0.1.0` in `release-manifest.json` with [version notes](docs/releases/0.1.0.md). D1–D4 integrate through PRs #16–#19; version preparation is [PR #20](https://github.com/A1b3rt0M3rcad0/wos/pull/20). The release job still requires the hosted gates before npm/GHCR/final GitHub publication. A GitHub draft with locally verified artifacts is preparation only, not a completed registry release.

Wave 11 was merged through [PR #13](https://github.com/A1b3rt0M3rcad0/wos/pull/13) at `f009b76dd4924490f0162839391218202abaeb79`. [PR #14](https://github.com/A1b3rt0M3rcad0/wos/pull/14), rebased in scope onto `master`, carries Waves 12–18 and the final hardening. Completion status above refers to the tested candidate; integration is confirmed by the PR's merged state, not by an unchecked draft or an earlier CI head.

[Verification dated 2026-10-07](docs/verification-2026-10-07.md) maps all thirteen canonical Release 0.1 gates to code and executed tests. The supported runtime matrix is Linux amd64, SQLite and PostgreSQL 18.6, HTTP and MCP; embedded and independent client examples are executable. The [agent guide](docs/agents.md) explains continuation, credentials, version conflicts, uncertain commits, leases and fencing.

1. Integration requires all final-head verification gates. Hosted Actions did not start on 2026-10-07 because GitHub reported failed account payments/spending limit. Equivalent gates were executed locally, including both Compose profiles, reproducible build and non-root/read-only container; this infrastructure failure is documented in the current verification and is not reported as a green hosted CI.
2. Validate the candidate in the consumer deployment using the documented commands, namespaces and credentials.
3. Publish a versioned release only when the maintainer elects to distribute it. No version tag or release asset is inferred from a merged PR.

## Acceptance evidence — 2026-10-07

- `internal/server/release_matrix_test.go`: 78 command schemas/idempotency guards, three proof modes and lost-response reconciliation across HTTP/MCP and both databases; real SDK clients.
- `storage/{sqlite,postgres}/release_concurrency_test.go` and `tests/boundary/dependency_concurrency_test.go`: separate connections/pools for claims, assessments, active slots, idempotency and opposite dependency edges.
- `storage/{sqlite,postgres}/bulk_history_test.go`: bulk/single read equivalence for owners, criteria, Evidence, current/historical Conclusions and reopen; immutable publication with live plan readiness and bounded graph references.
- `storage/postgres/clock_test.go`: authorized lease arbitration uses database time despite a replica clock skew of one year.
- Shared migration/restart/backup/restore contracts execute on real PostgreSQL with the test container; the independent 18-step standalone journey resumes the same Outcome after clean restore.
- Browser acceptance covers planning/replanning, human/agent continuation, proof review, Objectives, Issues/Blockers, conflicts, contestations, archive/resume, keyboard/focus and mobile rendering.
- In-flight HTTP shutdown and interrupted webhook retry preserve committed state and delivery identity. Query/guard/delivery metadata metrics exclude documentary payloads and credentials.
- Graph reachability, cursor scope and public command serialization fuzzing passed; measured query counts, pool wait and concurrent read throughput are in [benchmarks](docs/benchmarks.md).

The October 5 [audit](docs/auditoria-conclusao-2026-10-05.md) and [verification](docs/verification-2026-10-05.md) remain dated historical evidence. Its broader product checklist includes complete deployed Woobe acceptance and consumer distribution work. These remain explicit consumer obligations; they are not claims that WOS runs a deployed Woobe product. Current WOS release-gate acceptance is the October 7 matrix.

## Human workspace redesign — Jira UX / Guild.ai visual reference

**Status:** ✅ Implemented and locally verified (2026-10-07); integration through [PR #15](https://github.com/A1b3rt0M3rcad0/wos/pull/15).

Acceptance: persistent contextual navigation; Outcome summary, item list and operational board; readable entity/history/plan details; scoped human actions without exposing internal payloads by default; search/filter and pagination preserving snapshot semantics; keyboard/mobile behavior; existing human/agent journeys plus board/list regressions; real screenshots and PR integration. This is a presentation extension to Waves 12/17/18 and keeps WOS domain transitions, authorization and transport contracts intact.

Verified against the resulting repository: three Playwright journeys pass, including the real 32-item workspace, paginated board/list, proof selectors, cancellation, delayed Outcome responses, keyboard and mobile. API/server/boundary tests pass with real PostgreSQL enabled, plus vet/build/module syntax. The frontend is embedded with no new dependency or Domain/HTTP/MCP change. [UX guide and limits](docs/workspace-ux.md), [actual screenshots](docs/ui-workspace/README.md) and audit logs make the result reviewable. Hosted CI remains blocked before steps by GitHub account billing; it is not claimed green. PR integration is established by its merged state.

## Execution contracts and API-only client — C01–C12

Owner authorized execution on 2026-10-08 of [the supplied plan](docs/work-contract-implementation-plan.md), baseline bcd1714ad7cded666890d0b5c9f711772c4f8d7a (confirmed unchanged). ADR-018 explicitly supersedes legacy release/override semantics in migrated Namespaces; ADR-019 defines client/workspace boundaries. Original Waves 01–18 and distribution evidence remain historical.

| Wave | State | Required gate |
| --- | --- | --- |
| C01 | Accepted design and typed fixtures | Plan, ADRs, bypass inventory and typed preliminary fixtures |
| C02 | Domain implemented and tested | Pure domain validity/terminal/overflow tests |
| C03 | Implemented in Memory; tested | Transactional Memory acquire/renew/revoke/takeover, replay and exclusivity |
| C04 | SQL parity and restore tested | SQLite/PostgreSQL parity, indexes, exact fencing, restore |
| C05 | Implemented and transport-tested | Typed HTTP/MCP/SDK, bounded queries, capabilities, drift |
| C06 | Implementado e testado | Sync documental atômico com local keys, checkpoints, submissões imutáveis, avaliação vinculada ao material, finalize e proteções de bypass; testes de aplicação, HTTP/MCP e restauração SQL. |
| C07 | Implementado e testado em Linux | Cliente API-only, YAML restrito/schemas, checkout/dry-run, planejamento/revisão via DTOs tipados; percurso real HTTP/CLI. |
| C08 | Implementado, gate Windows pendente | Journal/recibos, recover de aquisição e mutações, locks, refresh preservando rascunhos, renew/takeover/keepalive foreground, sync/submit/finalize/finish; falhas de transporte/disco testadas em Linux. Windows precisa execução real em CI. |
| C09 | Interface/skills implementadas e verificadas | Human contract UI, skills and client planning |
| C10 | Implemented and locally verified | Explicit Namespace phases, current lease policy, old receipt replay, shared/exclusive writer guards and bounded expiry reconciliation. Actual SQLite/PostgreSQL transition/restart tests pass; operators must stop old binaries before activation. |
| C11 | Implemented and locally verified | Independent HTTP/MCP + local CLI on Memory/SQLite/PostgreSQL, focal reads and bounded progress expansion; 54 local load cases passed. Woobe runtime and actual Windows remain external gates. |
| C12 | Implemented and locally verified; external release gates pending | Coordinated 0.2.0 service/client/five skills, Linux npm and Linux/Windows native assets, exact-source platform receipts, checksums/notices and Docker client. Offline authenticated package journey passed. Actual Windows and registry publication remain blocked externally. |

T01–T88 in the plan are acceptance traceability, not passed tests. Actual Windows execution and deployed Woobe are not currently available; neither may be inferred from cross-compilation or HTTP parity. Existing Actions billing/npm identity blockers remain separate operational limitations; useful implementation continues independently.

C02: `domain/work_contract.go`, `work_submission.go` and `work_contract_test.go` cover exact expiry, all terminal causes, takeover, separate CAS, legacy direct release/override rejection, unsigned fencing including initial-claim overflow, JCS vectors and immutable-result digests. Existing domain/application/memory suites pass. Persistence and Namespace activation are not yet implemented.

C03: optional `WorkContractUnitOfWork` plus bounded repositories, transactional acquisition/snapshot, expiry+reacquisition, renewal, explicit takeover and privileged revocation. Memory deep-copies immutable material and CAS guards content/lease independently. Application race tests prove one concurrent winner, original replay, renew replay without extension, exact-deadline rejection and stale takeover fencing. WorkContract events are facts about their owning WorkItem; payload records contract identity and acquisition/result, avoiding a second entity_refs graph identity. Namespace cutover remains C10; no automatic migration occurs on acquire.

C04: additive migration 0016, one-open-contract uniqueness, holder/history/expiry/page indexes and scope FKs. Immutable spec is stored separately and never updated by renewal. New writers persist full unsigned task fencing as canonical decimal; original signed field is a legacy hint (zero above signed range), never the authority when decimal exists. PostgreSQL contract token uses NUMERIC(20,0); SQLite uses exact text. Both complete storage suites pass with race on real PostgreSQL 18.6 and clean SQLite/PostgreSQL restore. New cases cover independent connections, replay after restore, expired authority, >2^53 and uint64 maximum. PostgreSQL authorized contract acquire/renew/query also pass with replica clock skew of a year. Expiration is recorded during reacquisition, effective at expires_at, independently of maintenance workers.

C05: generated HTTP/MCP/SDK catalogue now includes 83 commands; contract state/spec/history, submissions, checkpoint pages, capabilities, own-principal receipt lookup and available/recoverable candidates are exposed. Authority DTO is a named `authority` object with decimal-string fencing; new wire contract was not released previously. Acquire-next scans bounded indexed headers in deterministic priority/creation/ID order, declares incomplete search and preserves empty-intent replay. Work context expands current contract only when revision matches, refusing a mixed snapshot. Contract spec cap is 128 KiB; command/query caps remain 256 KiB. Independent Go SDK + real Streamable HTTP MCP client proves HTTP acquisition → MCP takeover → stale HTTP renewal rejection, exact >2^53 fencing, own receipt, bounded history and search continuation; race passes. SQL contract suites remain verified; broader multi-storage transport proof follows C11.

C06: testes com race detector em aplicação, SQLite/PostgreSQL reais e cliente HTTP/MCP independente; restauração preserva submissões, avaliações, conclusões e recibos. O modo novo ainda depende do cutover explícito C10; CLI e validação Windows seguem pendentes.

C09: painel de contrato/último checkpoint/submissão/revisão/revogação, histórico sob demanda e exclusão dos comandos legados para WorkItems migradas. Cinco skills e quatro destinos npm testados; três jornadas Chromium existentes passaram. A jornada Chromium específica de cutover + checkpoint + revisão de submissão vinculada + revogação privilegiada passou em C10. Instalação de skill não certifica execução em cada runtime.

C10: ADR-020 documents anchor-Outcome auditing, no historical contract fabrication, operational old-writer retirement and no silent downgrade. Migrated work preserves IDs/versions/fencing; new work supports typed ExecutionSpec. Domain permits administrative cancellation only after authority detachment. CLI protocol get/set/reconcile uses authenticated public API and durable command journal. New optional command fields omit nil values to preserve legacy command fingerprints.

C11: context query reads selected work under one guard and preserves current proof, with historical proof expansion declared. Contract history reads metadata only; large latest checkpoint/submission has explicit expansion paths. Redirected mutations stay sent_unknown; acquisition recovery orders prepared intents. Frozen spec includes Outcome/Objective intent for clean sessions. Load fixture counts transaction conflicts and bounded identical-intent retries instead of hiding them; same-Outcome PostgreSQL contention is material. The 54-case report identifies tested implementation commit 6179387 and preserves the full measured matrix in docs/work-contract-load-evidence.json. Four Chromium journeys and all five skill installation tests passed. A full race run exposed the new transitive-module dependency in the external consumer test; the corrected real external consumer test passed. Final all-package integrated verification remains a C12 gate.

C12: WOS npm now contains wos + wosctl; separate client archives include schemas/notices/source metadata. Publisher requires native Linux/Windows receipts matching source and binary hashes, and hosted release depends on the CLI matrix. Version 0.2.0 prepared coherently; five skills install 20 managed copies across four discovery roots. Local authorizer grants ordinary acquisition to its single Principal, and revocation only with existing admin opt-in; Namespace administration still requires an authenticated Namespace administrator. Offline packaging uses an isolated API-token bootstrap instance and does not relax runtime authorization. Exact implementation commit 2ebe93fa18cbc2ec3d9570fd2725cec5ff9d3ddf passed all-package race/vet, four Chromium journeys, 16 Node tests, offline packaging, cross-path reproducibility, native Linux acceptance and non-root/read-only container verification. See docs/verification-work-contracts-2026-10-08.md; native Windows, Woobe and external publication are still pending.

## Signed contracts and independent minimal profiles — P00–P13

Owner authorized execution of [the new plan](docs/signed-contracts-implementation-plan.md), baseline b75aa3e confirmed against origin/master. C01–C12 above remain historical. ADRs 021–026 explicitly change behavior only for signed_contracts_v2. Product remains state/coordination, not agent execution.

| Wave | Actual state | Gate |
| --- | --- | --- |
| P00 | Design accepted; implementation baseline frozen | Full original plan, ADRs, bypass/tables inventory, migration reservations and four design fixtures; no proposed command advertised as existing. |
| P01 | Codec implemented and locally verified | Public pure signing package, six payload purposes, typed bindings/counters, exact verified DTO, local mapping/proof and independent Python vector. Race signing/domain, vet and two 15-second fuzz campaigns passed; no runtime authorization claim. |
| P02 | Identity foundation implemented and locally verified | Authorized one-use enrollment, possession-bound rotation, credential restrictions/floors, stable Principal groups, durable security receipts and all three adapters. Signed work/review operations remain P03–P06. |
| P03 | Implemented incrementally; exhaustive acceptance remains P12 | Signed issuance/runtime trust and storage foundations integrated with subsequent delivery/review/cutover work; final-source gates remain open. |
| P04 | Core implemented and locally verified; exhaustive acceptance remains P12 | Atomic signed return implemented: exact envelope/spec verification, documentary local keys, fresh explicit direct assessments, delivery/review handoff and durable signed acceptance. |
| P05 | Core implemented and locally verified; exhaustive acceptance remains P06/P12 | Independent signed review/leases, exact immutable targets, correction rounds, Task completion and explicit administrative interventions. Exhaustive cross-transport/fault acceptance remains P06/P12. |
| P06 | Implemented and locally verified; exhaustive matrix remains P12 | Exact signed schemas/counters, live Outcome scope filtering, public identity/enrollment, bounded focal proofs/material, unsigned guards and actual HTTP/MCP/SDK independent review journey. Full race/vet/restore and generator gates passed locally. |
| P07 | Implemented and locally verified; native secret-store gates remain P12 | Project/profile CRUD, approved destination and pins, protected stdin/generation/import, possession enrollment with durable replay, secret references and exact single-contract YAML. Operational execution/review workflows are implemented in P08/P09; native keyring execution remains unaccepted. |
| P08 | Implemented incrementally; exhaustive fault acceptance remains P12 | Stable locks, bounded execution/review batches, embedded intentions, verified atomic materialization, renew/resume/refresh and supervised keepalive integrated. Extended final-source process/file failure gates remain open. |
| P09 | In progress — signed CLI return and receipt cleanup | Protected local sign/send/finish, original-CID frozen recovery, issuer receipt verification and single-file cleanup implemented; final-source/full and exhaustive fault acceptance remain required. |
| P10 | In progress | Explicit cutover, preserving workspace migration, schema compatibility, host issuer recovery and client pin approval integrated; historical v0.2 dataset/old-binary and complete restore acceptance remain pending. |
| P11 | In progress | Five signed-protocol skills and client guide alignment locally verified; UI read models/guarded actions and final operational guide acceptance remain pending. |
| P12 | Planned; external platform/pilot gates unresolved | Real SQL/restore, independent processes, failures/load/context benchmarks; native Windows and actual consumer separately proven. |
| P13 | Planned; publication separately gated | Exact-source packages/notices/native receipts and upgrade documentation; no registry claim from build. |

R01–R24 / T01–T96 traceability is in plan section 20.13; acceptance is not inferred from fixture/file/commit counts. Native Windows, hosted Actions billing, registry identity and actual Woobe/model telemetry were unavailable in the previous phase and must be rechecked, not assumed fixed. No secret values are requested or persisted by this design wave.

P01: crypto/ed25519 + existing pinned JCS, no new product dependency. Independent Python cryptography 50.0.0 generated the published RFC8032-seed DSSE vector; Go matches exact bytes/signature. Tests reject duplicate keys, lone surrogates, noncanonical bytes/base64, unsafe numeric integers, malformed key sizes, multiple signatures, wrong purpose/inner signer, unknown verified DTO fields and unsigned companion commands. Exact uint64 counters remain strings through maximum. Semantic scalar cap is distinct from technical envelope base64. Linux race/domain and vet passed; fuzz canonical/envelope ran 15 seconds each (50,477 and 552,465 executions). Actual Windows remains unexecuted. No mutable-domain/transport behavior changes in this phase.

P02: additive migration 0020 stores only public signing identities, bounded one-use enrollments, credential/acceptance policies, Principal separation groups and immutable security receipts. Application requires authenticated credentials and a live transactional access snapshot, uses authoritative transaction time, verifies possession before deriving mutations and audits Namespace CAS changes. Rotation requires the previous authorized active key; it retires that key while preserving historical verification and the stable Principal/floor. Explicit administrator enrollment plus audited key revocation supports recovery without treating a bearer token as key-replacement authority. Missing legacy policy does not authorize registration. Memory now implements security ports; SQLite and PostgreSQL share generated repository contracts. Focused race contracts exercise forged key hints, another challenge, scoped views, replay versus second consumption, rotation/revocation, policy/grant intersection, concurrent enrollment and stale authenticated contexts after bearer revocation. v2 work execution, quotas, scoped read filtering and transport exposure are subsequent waves, not asserted by P02.

P02 verification: full repository `go test -race -p=1 ./...` passed with actual PostgreSQL and clean pg_dump/pg_restore fixtures enabled. Final identity/concurrency tests passed under race detection on Memory, SQLite and PostgreSQL after authoritative-clock and scoped-view changes; vet passed for changed Application/domain/ports/storage packages. No native Windows or hosted Actions execution is inferred.

P03 foundation (partial wave): delivered is a signed-only terminal execution status, while review cases/authorities preserve independent Principal/execution/fencing/lease state and immutable submission/spec/policy/round/history. Migration 0021 adds separate signed tables, immutable payload/proof facts and indexed scoped counts without rewriting v1 data. v2 execution/review SQL versions remain exact above 2^63/2^53 (SQLite decimal text, PostgreSQL NUMERIC(20,0)); the specification payload excludes a circular self-digest. Both generators are checked by CI. Open reviews block embedded unsigned Task edits, assessments and parent completion; active signed authorities block material edits. Full repository race suite passed against real PostgreSQL with clean restore enabled, plus vet and actionlint. A test-only pg_dump client option bounds parallel catalog workers after Docker shared-memory exhaustion; production settings are unchanged. Domain/storage fixtures are not signed acquisition/return endpoints, activation or workspace support. Remaining P03 gates are server identity/trust composition, signed issuance/acquisition/renewal and full protocol integration; P04–P13 remain pending.

P03 signed issuance implemented and locally verified: persistent immutable public server identity and deployment-local Ed25519 signer, secret seed environment reference, quoted signed work-spec/material counters and distinct v2 material digest, epoch-2 metadata constraints with row-preserving migration 0023, authenticated acquisition with frozen strongest acceptance floor, indexed Principal/credential/Namespace counts, exact signed specification/grant, renewable grant and fenced takeover without TTL extension. Targeted race tests passed on Memory/SQLite/PostgreSQL for acquisition/quota/replay/revocation and lease/spec separation; restart tests pin server public trust and reject silent seed replacement. Generated HTTP/MCP/Go SDK catalog now includes the three acquisition/lease commands. Security/trust/profile query exposure and full transport acceptance remain P06; public protocol activation and signed delivery/review remain gated to following waves. No Namespace is automatically activated.

P03 issuance verification: full `go test -race -p=1 ./...` and `go vet ./...` passed with actual PostgreSQL and clean restore enabled. Generated HTTP/MCP/Go SDK catalog and boundary tests passed after regeneration; integration event catalogue checks passed. Hosted Actions and native Windows remain external gates, not inferred from local results.

P04 return verification: shared transaction composition reuses documentary validation without nested commands; v1 finalization cannot finalize a signed Domain contract. Targeted race fixtures exercise actual credential/key policies, exact signed material with newly registered artifact/evidence, required assessments, review-floor rejection, full authority preservation on invalid references, signed receipt binding, revoked-bearer rejection and durable receipt replay after SQL idempotency-cache expiry. Memory, SQLite and actual PostgreSQL targeted tests passed; full repository `go test -race -p=1 ./...` and `go vet ./...` passed with clean PostgreSQL restore enabled. Final refinements (Tasks with and without required criteria, accepted-material digest and optional conclusion reason) passed targeted race tests on all three adapters, transport/boundary checks and vet. Broader fault injection, cross-transport flows, corrections, profiles and release acceptance remain P05–P13.


P05 verification: independent signed review/renewal/takeover, inconclusive release, accepted changes and a new correction executor, live blocker rollback, supplemental same-source evidence, administrative review revocation and case cancellation/supersession passed shared race fixtures in Memory/SQLite/actual PostgreSQL. The original executor credential is revoked before correction/review completion; original delivered contracts and signed submissions remain immutable. Replanning after intervention requires explicit case acknowledgement and a new fenced authority. Full repository race tests passed, including PostgreSQL clean restore and architectural boundaries; generated HTTP/MCP/SDK and 102-event catalog have no drift. Metadata counter transport and complete focal/bypass acceptance remain P06; exhaustive T01–T96, native Windows and real consumer telemetry remain P12 gates.


P06 first-part verification: exact new signed command and mutation counters (including Task/lease fences) round-trip above 2^53 and at uint64 maximum without changing v1 numeric encoding or immutable issued proof. Strict signed DTOs reject nested/case aliases even under an otherwise valid signature. Generated OpenAPI includes purpose-specific exact-field schemas. Shared signed acquisition/return/review/correction and legacy lease/idempotency race fixtures passed in Memory/SQLite/actual PostgreSQL; Core, command/HTTP/runtime, boundaries and vet passed. Focal queries, permitted-Outcome filtering, full remaining guards and cross-transport acceptance are still in progress; this is not P06 completion.


P06 focal verification in progress: bounded contract/case/correction/receipt/submission queries expose verified exact envelopes and public historical keys, with digest-bound expansions and explicit scan cursors. Public HTTP/MCP/SDK signing administration uses the same transactional enrollment/policy service; identity metadata includes Namespace CAS and key-list truncation. Outcome restrictions filter storage before search pagination and revalidate within legacy/focal read snapshots, including Namespace receipt policy changes. Active signed authority/open reviews freeze parent material and obligations; accepted signed Task assessments reject unsigned rewrites after completion. Legacy material-edit behavior remains preserved. Available-work projections exclude pending review/correction and resolve signed authorities separately from v1 repositories. Shared Memory/SQLite/actual PostgreSQL targeted race fixtures passed for focal queries and changes-requested acceptance. Real HTTP acquisition, MCP return and an independent SDK reviewer after executor bearer revocation passed on all three adapters, preserving exact receipt bytes/fencing and the original delivered execution. Final full repository race suite and final public bypass refinements are being completed; no hosted CI/native Windows/publication/pilot claim.

P06 verification discovered two environment/bootstrap limits: the dedicated PostgreSQL container's accumulated test catalog required more than 64 MiB shared memory (94 MiB observed), so its data volume was preserved while recreating the container with 512 MiB and retaining the old stopped container for reversal. Development-container parallel scan/maintenance settings are zero. No production data/configuration was changed. SQLite startup/migration cancellation previously reused a minimum five-second writer timeout and intermittently interrupted migration 24 under race detection. `Options.StartupTimeout` separates the bounded migration budget (30-second default) from unchanged writer contention. A targeted cancellation/contention/indexed-context race test passed; final full-suite rerun remains required before P06 integration.

The full frozen-source run subsequently passed SQLite (107.122s) but the historical cross-transport PostgreSQL fixture used a one-second HTTP request budget and timed out during its normal CLI submission under race instrumentation. That acceptance fixture now uses ten seconds, matching the signed transport fixture; dedicated deadline/cancellation assertions and production request-timeout configuration are unchanged. A complete rerun of the resulting source is required before merge.


P06 final verification: the frozen source 0eb0f66b040c239c4e8559af009f54b209613a51 passed full `go test -race -p=1 ./...` with explicit actual PostgreSQL DSN and clean restore container settings after the recorded environment/bootstrap fixes. PostgreSQL full contracts passed in 115.665s and SQLite in 107.122s; the final cross-transport acceptance suite passed in 14.409s. Earlier interrupted/mixed-source/SHM-exhausted runs are not counted as acceptance. Final scoped/focal/embedded/public unsigned guards and possession-based HTTP/MCP/SDK review after executor revocation passed on all three adapters; final vet/actionlint and generated artifacts/catalog had no drift. Hosted Actions still fail before steps because account payments/spending limit; no hosted/native Windows/publication/pilot claim. P07–P13 remain pending and are the next work, not declared complete by P06.


P07 foundation: schema-2 project/profile and one-contract documents, explicit flag/env/unambiguous profile selection, exact-case collisions, confined regular-file checks, bounded semantic versus technical YAML scalars, exact uint64 counters, proof verification, editable human material names, content comparison before replacement, and explicit env/mounted/OS-keyring secret backends. Profiles carry authenticated local destination/pending bindings without adding operational directories. The pinned MIT keyring dependency is CLI-only; notices include Windows dependencies. Targeted race tests verify changed destination/identity/references/pending rejection, signature round trips and tampering, editor conflict preservation, YAML limits, symlink/hardlink rejection and redacted backend errors. This is a foundation, not completed P07: CLI enrollment/profile commands, stable process locks, durable recovery, sign/send/finish, cutover, skills/UI and release gates remain pending. Native Windows and actual OS-keyring execution remain unproven.


P07 onboarding continuation: `init --workspace-schema 2` / `project init`, explicit `profile create/list/inspect/recover/remove`, protected `--token-stdin`, generated OS-keyring seeds or dedicated `--signing-key-stdin`, and preprovisioned env/mounted proposals. First onboarding requires independent exact origin/server ID/issuer fingerprint approval; existing keyring entries are reused after interruption and never silently overwritten. Credentials/keys stay out of arguments, YAML and diagnostic output. New profiles persist authenticated frozen enrollment intent before registration; recovery replays the exact durable security intention. Actual HTTP/Core acceptance on Memory/SQLite/PostgreSQL commits enrollment, repeatedly drops its response, preserves `sent_unknown`, replays the original intention and consumes it once. Live identity inspection is compact; profiles do not advertise server permissions as authority. Removal refuses pending intentions or remaining contract files, preserves shared secrets and does not impersonate server revocation. A signed project cannot silently dispatch legacy operations.

P08 lock foundation was intentionally implemented with onboarding to prevent concurrent reservations: canonical root plus case-collision-safe profile and contract identities, private stable Unix advisory lock files outside the workspace, and Windows named mutexes with thread ownership. Native Linux process fixtures verify exclusion after YAML replacement, independent profiles, case collisions and automatic release after owner death. These do not establish Windows-native behavior. Acquisition recovery after server cache expiry, batches, file cleanup and the exhaustive T49–T72 matrix remain unimplemented gates, not inferred from enrollment recovery.

P07 verification: full repository race suite passed with explicit actual PostgreSQL DSN and clean restore container, including the real HTTP enrollment response-loss/replay journey on all adapters. Vet, generated schemas/catalog, actionlint and Windows cross-compilation passed. The dedicated protected-seed import fixture subsequently passed race/vet and does not claim native keyring execution. Hosted Actions still cannot start because account billing/spending limit; native Windows, registry publication and actual consumer/model telemetry remain external acceptance gates.


P08 server recovery foundation: additive migration 0025 and a shared public Core port retain immutable signed-operation responses independently of short cache expiry. The common pipeline reauthorizes current credential/policy/scope and selected acquisition/lease key, then binds replay to exact Principal/CredentialID/Outcome/command/fingerprint and original command ID/revision. Replays return original grant/fencing/CAS/expiry after renewal/takeover and do not emit another event or acquire/renew work. A bounded credential-bound `operation` expansion supports harness reconciliation; initial agent context does not include its technical blob. Actual Memory/SQLite/PostgreSQL race contracts passed cache expiry plus an unavailable-cache adapter, changed payload/command/CID rejection, original grant preservation and clean SQLite/PostgreSQL restore. HTTP/SDK recovery expansion and existing independent review/return fixtures passed. The full frozen-source race suite passed with actual PostgreSQL (166.753s), SQLite (105.092s) and cross-transport acceptance (13.110s). The final SDK-only refinement also passed race/vet: operation expansion permits the full bounded 1 MiB registry response plus base64/metadata overhead while ordinary v1 responses retain their 256 KiB limit. Exact committed-source verification is recorded in the integration PR. Workspace pending/materialization/batch/sign/send/cleanup and T57–T72 remain subsequent work; durable enrollment alone did not satisfy work recovery.


P08 compatibility refinement: before reserving/deleting a short-cache entry, the new pipeline can recover a still-present pre-P08 accepted signed response even when expired. It resolves the original CID from the historical contract, rejects substitution, and promotes the exact original result once. Read-only focal recovery does not persist a promotion. Actual SQL race/restore fixtures cover expired legacy rows plus another CID sharing Principal/Actor/key permission. Ambiguous or unresolvable historical cache intentions require explicit reconciliation; no retroactive proof is fabricated.


P08 work acquisition continuation: public `AcquireNextSignedWorkContract` scans bounded candidates under the Outcome guard and persists empty/acquired results with independent intentions. Replay does not become a new search after a Task appears; incomplete pages preserve cursor/completeness. The signed Namespace creation path now enables contracts for new Tasks, including during signed drain, without granting acquisition during drain. Schema-2 work checkout/count/recover/list/show persists the bounded independent set before network calls, uses short profile locks and observed content snapshots, verifies pinned issuer documents, persists accepted-unmaterialized state and preserves existing edited drafts. Recovery first reconciles credential-bound accepted state and exact command fingerprint; accepted acquisition remains recoverable after key revocation, with no acquisition call. Five-item/second-response-loss actual HTTP fixtures passed on Memory/SQLite/PostgreSQL, materializing exactly two original eligible contracts and preserving a first draft edit. Shared tests passed empty replay, incomplete cursor, unavailable cache, original public response, technical expansion above 256 KiB and quota. Snapshot/independent-intention race fixtures passed. Review checkout, lease maintenance, crash-in-file-write/cleanup fault coverage, P09 returns, P10 cutover and P11–P13 remain pending; this is not completion of P08. Full frozen-source `go test -race -p=1 ./...` passed with explicit actual PostgreSQL and clean restore (PostgreSQL 152.920s, SQLite 119.659s, acceptance 19.707s). Vet, actionlint, generated contracts/catalog and Windows cross-compilation passed. The owner explicitly accepts hosted jobs being refused and authorized continued local validation/integration; refused jobs are not counted as executed CI or native Windows acceptance.

P08 independent review continuation: bounded public signed-next review acquisition shares the same transaction, historical/current independence checks and quota with specific-case checkout. Empty/acquired results use the immutable credential-bound operation registry. Schema-2 review checkout/recover/list/show reuses independent profile intentions, verifies issuer/target documents, materializes an inconclusive draft and retains exact acquired case CAS. Agent projection verifies nested delivery/work documents and keeps authentication proofs outside model output. Actual HTTP loss-after-commit recovery acquires no replacement and preserves the editable draft; the subsequent independent SDK review still completes after executor credential revocation. Targeted shared race journeys passed Memory/SQLite/actual PostgreSQL, including original review authority replay after takeover with unavailable short cache. Full frozen-source race suite passed with actual PostgreSQL/clean restore (165.153s), SQLite (127.266s), acceptance (20.951s) and architectural boundaries. Vet, actionlint, 102-event catalog and Windows cross-compilation passed; Windows-native and hosted jobs are not counted as executed. Lease maintenance, file-write/cleanup fault acceptance and P09–P13 remain pending.

P08 lease maintenance continuation: schema-2 work/review renew, resume and explicit refresh, profile-embedded immutable lease intentions, accepted-operation lookup before uncertain replay, and verified authority-only merge preserving observed editable drafts. Resume rejects TTL overrides and retains expiry; explicit refresh preserves semantic Task/case CAS. Foreground/one-pass keepalive handles items independently, retains revoked/expired files and pauses unresolved final/lease operations without a server scheduler. Actual HTTP response-loss fixtures passed Memory/SQLite/PostgreSQL, preserving a draft edited after the failed renewal and reconciling without another renewal call; takeover fencing/expiry, two-contract keepalive, one revoked contract alongside a renewed live contract, and review renewal/resume/refresh/keepalive followed by independent SDK approval passed. Unit race checks bind exact frozen authority/CAS and reject competing intentions/final markers. Signed SDK commands now allow bounded technical responses above 256 KiB without widening legacy limits. The profile-lock/network assertion passed in the actual HTTP journey. Full final-source race suite passed with explicit actual PostgreSQL/clean restore environment (unchanged Core/storage fixtures cached), final acceptance 38.017s and architectural boundaries. Vet, actionlint, catalog and Windows cross-compilation passed; no Windows-native or hosted execution is inferred. Foreground intervals are bounded to 300 seconds and must be shorter than the requested TTL. Atomic file publication/abandoned staging reconciliation, P09 receipt cleanup and P10–P13 remain pending.

P08 atomic materialization continuation: complete fsynced staging plus exclusive publication replaces direct writes to final paths during initial v2 creation. Accepted contracts use an original-intention-bound staging path and recover constructor-prefix partial writes, verified complete drafts, published two-alias pairs and post-unlink process death without reacquisition. Exact two-alias confinement preserves current destination contents; additional/outside links and differing staged data remain rejected/preserved. `.yml` destination renames retain draft contents rather than creating another `.yaml`. Technical directory scanning is bounded. Native process-death race fixtures and full CLI race suite passed on Linux; actual Memory/SQLite/PostgreSQL HTTP recovery repaired half-written materialization from the original accepted response while preserving the first edited draft. Generic pre-publication onboarding/replacement stages remain preserved for diagnosis, and noncooperative final-compare/rename races are not claimed as an OS isolation guarantee. Final alias/concurrent-reader refinements passed the full frozen-source race suite with explicit actual PostgreSQL/clean restore environment (unchanged Core/storage fixtures cached), acceptance 31.240s and architectural boundaries. Vet, actionlint, 102-event catalog and Windows cross-compilation passed; Windows-native/hosted execution is not inferred. P09 receipt cleanup and P10–P13 remain pending; this is not final product acceptance.

P09 credential-bound cleanup foundation: focal signed execution/review metadata now exposes the original public CredentialID, and historical work/review acceptance fallback reauthorizes and compares that original CID before disclosing a mutation replay when both cache layers have no row. Existing signed payloads/signatures are not rewritten. Shared Memory/SQLite/actual PostgreSQL port fixtures retain authoritative signed facts while modeling absent cache layers: original ownership replays exact acceptance after key retirement; another CID with the same Principal/Actor/key permissions is rejected for execution and review. Actual public HTTP projection retains original execution CID after independent completion. Targeted race tests passed (SQLite 19.480s, PostgreSQL 13.584s, acceptance 23.505s). CLI refresh/keepalive now checks original CID as well. Full frozen-source race suite passed with explicit actual PostgreSQL/restore (PostgreSQL 167.946s, SQLite unchanged-source cached, acceptance 29.440s, boundaries 6.281s); vet, actionlint and catalog passed. An earlier full run failed only after the environment disk filled and is not counted; disposable Docker build cache was reclaimed without dropping database data. Sign/send/finish and receipt-bound deletion remain subsequent P09 work.

P12 hosted platform recheck: PR #49 hosted browser, packages and Linux platform jobs executed successfully after prior account refusals. Windows executed and failed, but the distribution runner hid Go test stdout whenever dependency-download stderr existed. Diagnostics now retain both output streams so the native failure can be investigated; Windows remains unaccepted until the corrected native job passes.

P09 CLI return continuation: work/review sign freezes exact payload, signature, editable-draft digest and original-CID local provenance before network mutation. Fresh focal authority/Task/case CAS and own active key/policy are preflighted without implicit renewal/rebase. Send reconciles original signed acceptance before replaying identical bytes; prepared-only recovery never sends automatically. Confirmed acceptance is persisted before cleanup, and remote commit remains explicit when local cleanup fails. Cleanup verifies issuer/targets/request/disposition/closed obligation, preserves unconfirmed edits and removes only the corresponding regular contract file. Native child-process exit fixtures at receipt persistence and unlink passed on Linux. Actual HTTP Memory/SQLite/PostgreSQL execution delivery with lost response preserved edits and recovered without another return; independent CLI review signs/finishes after executor bearer revocation, deletes its file and SDK replays the original accepted decision. Unit fixtures reject copied-CID provenance, valid-local-MAC/invalid-signature and wrong/open receipts. Final frozen-source full race suite passed with explicit real PostgreSQL/restore environment (unchanged Core/storage cached), acceptance 41.013s and native Linux cleanup fixtures. Vet, actionlint, generated workspace schemas and the 102-event catalog passed. Extended P12 fault/direct/correction/platform acceptance remains required.

P12 native Windows diagnosis continuation: the hosted diagnostic job identified rejection of native relative backslash paths by a Linux-only confinement rule, Windows drive letters encoded as file-URI authorities for SQLite, and a mounted-secret fixture relying on Unix chmod/canonical spelling. Native relative paths now follow the platform local-path rules while absolute paths, volumes/ADS and symlink/reparse escapes remain rejected. SQLite emits a local-drive URI path. The mounted-secret fixture resolves its canonical spelling and explicitly establishes a restricted Windows DACL instead of weakening product secret validation. Linux targeted race and Windows cross-compilation passed; hosted native Windows execution is still required for acceptance.

The first hosted Windows fix passed native CLI filesystem/secret tests and SDK tests; SQLite acceptance still failed at open. Dependency inspection showed that the Go SQLite VFS receives the parsed filename directly and does not remove the RFC-style leading slash before a drive. The DSN now uses an escaped opaque file:C:/ URI to retain the native drive path, with all pragmas unchanged; the native rerun remains decisive.

Hosted Windows now passes native Go CLI/SDK/protocol checks for the opaque-drive fix. The next failure was skill frontmatter checked against LF after Git autocrlf produced CRLF. Repository attributes now keep text LF on all platforms, preserving canonical skill and embedded migration bytes; the integrated P09/native installer rerun is required.

Integrated native Windows Go checks including P09 returns/cleanup passed. The remaining installer test expected POSIX absolute strings on Windows even though installation/update/edit protection all passed; expected discovery roots now use the native absolute-path representation without changing their documented root mapping. The full platform receipt remains pending until every check and source-identity build passes.

P10 activation foundation: Namespace CAS supports contracts_v1 -> draining_to_signed_v2 -> signed_contracts_v2. Signed drain clears the earlier writer acknowledgement and refuses new unsigned acquisition. Activation rechecks live legacy claims, unsigned contracts, matching persistent issuer and an explicit Namespace acceptance policy under the existing transaction/coordination guards. Read-only administrative protocol_preflight is advisory and shared across transports. Authenticated signed-phase operator intentions use the immutable original-CID operation registry, without requiring an agent signing key or changing v1 fingerprints. Schema-2 protocol get/preflight/set/recover persists exact phase/version/reason/acknowledgement before sending and reconciles an uncertain response before any identical replay. Targeted Memory/SQLite/PostgreSQL cutover tests passed (SQLite 2.790s, PostgreSQL 1.899s); the actual HTTP/CLI journey also passed on all three adapters (33.601s), including activation response loss and recovery without another mutation. Full frozen-source race suite passed with actual PostgreSQL/clean restore (220.613s), SQLite (131.714s), then the final guard-order source passed full race again (PostgreSQL 183.647s, SQLite 125.448s, acceptance 54.303s), vet, actionlint and catalog. Namespace is locked before Outcome in protocol preflight, matching activation order. Final CLI-only precision/printable-error changes passed the CLI suite (7.804s), complete acceptance (40.291s), boundaries and CLI vet; quoted local CAS above the JavaScript limit and confirmed target/version validation have regression tests. Workspace v1 migration, historical upgrade/restore, issuer recovery and P11–P13 remain pending.

PR #51 integrated Windows portability: the full native Linux/Windows workflow run 37874587697 passed Go CLI/SDK/protocol, five-skill installer and executable identity checks. Receipts identify synthetic PR merge source 4080b609ae2583645fabb538395e8d0b0f28f4ad; this is executed native evidence for that source, not final-release proof. Earlier hosted billing refusals are historical; current runs execute.

Local Compose onboarding: the owner requested a copy-only .env.example setup. A committed development template provides a localhost bootstrap credential, valid Namespace ID and persistent SQLite defaults; Compose accepts a configurable localhost port and optional protected runtime signing references. PostgreSQL remains opt-in. Config interpolation and actionlint pass. Actual non-root image build/start, readiness, frontend, browser-session login, Namespace discovery and authenticated create passed; restart retained the original Outcome and idempotent receipt. The sandbox required a temporary build-only host-network/proxy/CA override, outside the repository; normal Compose settings remain portable. The immediate post-restart probe ran before readiness and was retried successfully; it is not counted as a product failure. Signing activation remains explicit and no shared issuer key is shipped.

P10 workspace migration continuation: read-only bounded inventory preserves exact draft/journal/receipt bytes and reports prepared/uncertain intentions and accepted acquisitions without materialization without network calls. Explicit separate schema-2 destination checks current approved server/Namespace/CID and remote original contracts, freezes a bounded profile migration intention and publishes explicitly legacy_unsigned single documents. Source files remain untouched; neither issuer nor agent signatures are fabricated. Recovery preserves source/target edits and reconciles exact publication before clearing the marker; repeating a verified migration recognizes existing matching files without a new intention. Draft interpretation uses the observed bytes, not a second read. Generated unsigned-local schema is packaged separately from signed contracts. Linux native child-process death after publication, original materialization, changed source/target recovery and decimal counters above JavaScript precision passed race tests (CLI 1.521s); actual HTTP/CLI conversion plus signed activation/execution/review passed Memory/SQLite/real PostgreSQL (35.679s), including refused signed mutation of legacy imports. Full frozen-source race suite passed with real PostgreSQL/clean restore (236.833s), SQLite (161.677s), CLI (17.842s), acceptance (87.457s) and boundaries; vet, actionlint, 102-event catalogue and Compose interpolation passed. Hosted Windows execution is required for this source. Historical database upgrade/restore, issuer recovery and P11–P13 remain pending; no P10 completion claim.

P10 database compatibility continuation: migration checks the highest persisted
schema under its transaction/ PostgreSQL advisory lock and rejects a database
newer than the compiled migrations without altering history. Runtime also checks
schema compatibility before bootstrap/serving when migrations are disabled.
Targeted SQLite and actual PostgreSQL future-schema/older-upgrade race checks
passed (6.957s/3.535s). Runtime restart/future-schema race tests passed (4.531s). Full race
regression passed with real PostgreSQL/restore (303.254s), SQLite (180.163s),
acceptance (79.802s) and CLI (18.613s); runtime passed (47.088s) on a sequential
package rerun with unchanged Core/storage/acceptance tests cached. The initial
parallel run timed out in the existing 30-second go-run MCP subprocess discovery
and is not counted as a full pass. Vet, actionlint and catalogue passed. PR #55
hosted browser, packages, full verification and native Linux/Windows passed. This adds forward protection, not retroactive protection to released
v0.2.0 executables; actual old-binary and historical dataset acceptance remain P10.


P11 agent guides started: all five portable skills now select live Namespace
protocol explicitly. Signed work uses protected schema-2 named profiles,
focal agent projections, distinct execution/review obligations and exact-CID
recovery; contracts_v1 instructions are explicitly retained only for legacy
Namespaces. Progressive bundled references explain immutable issued documents,
editable single-file drafts, quotas/partial batches, supervised leases, signed
acceptance versus quality, explicit trust rotation and unsigned migration.
No skill installs credentials, starts agents or certifies runtime integration.
All five installer tests passed: discovery roots, repeatable all-agent install/update,
protection of local edits, progressive references/frontmatter and actual main-skill
MCP tool names. UI/docs alignment remains required; this is not P11 completion
or final native/consumer acceptance.

P11 guide alignment removes stale claims that schema-2 review, lease maintenance
and sign/send/finish are unimplemented. Client/skill READMEs explicitly separate
legacy recipes from the protected signed flow; signed workspace docs explain
immutable original-CID returns and receipt-bound cleanup, distinguish executor
acceptance from review/quality, and retain native-keyring/final-source gates.
Registry publication is not inferred from source guide changes.
P10 issuer recovery design started: ADR 028 records the missing host-managed
replacement path required by T14/T87, separate from tenant administration and
without rewriting old signed facts. Implementation must retain persistent
ServerID/public history, freeze CAS and provenance, order global issuance against
replacement, and explicitly update client trust without breaking pending returns.
No recovery command is implemented by this design note; acceptance remains open.


Issuer recovery foundation work in progress: pure host composition proves the
approved replacement signer before its transaction and rejects tenant identity
contexts. Memory/SQLite/PostgreSQL persist immutable public issuer history and
frozen recovery receipts, preserve ServerID/instance creation time, CAS the
predecessor and replay a receipt without reinstalling a superseded issuer. SQL
shared issuer reads order issuance against the exclusive recovery lock. Targeted
race/clean restore tests passed (SQLite 4.106s, PostgreSQL 16.661s), verifying old
public proofs after clearing the old private key and rejecting stale configuration.
The source host executable command and paginated public trust projection are
implemented; actual runtime restart/HTTP tests passed SQLite and PostgreSQL
(4.215s), including refusal of silent replacement, wrong declared pin, exact
receipt replay and private-seed absence from the receipt. Client pin updates,
actual signed issuance/return after rotation and complete regression remain
unimplemented or unaccepted. This increment is not integrated and is not P10
completion.



PR #56 integrated workspace migration together with PR #55 compatibility checks.
The combined source d3e9d9e passed actual Memory/SQLite/PostgreSQL signed CLI
acceptance (48.186s), migration/precision race checks (1.959s), and runtime
schema/MCP subprocess checks (7.695s). All hosted browser, package, full
verification and native Linux/Windows checks passed on its synthetic PR merge;
these receipts do not certify a later final-release source.


P10 issuer recovery/client trust continuation: schema-2 `profile trust` verifies
explicit independently approved current fingerprint and the unchanged origin,
ServerID, Namespace, Principal and original CID. One atomic profile append retains
old pins and authenticated bounded prior binding MACs; contract bytes, signatures,
request IDs and pending intentions are unchanged. No trust RPC holds the profile
publication guard. Native Linux child-process exit after publication, `.yml`
mutation, repeated approval without rewrites, tampered lineage and moved-origin
rejection passed targeted race tests. The actual HTTP/CLI journey now rotates
after preparing a return, rejects stale issuance and unapproved trust, clears the
old private issuer, approves both local profiles, sends the original return under
the new issuer and recovers a lost response before independent review. This
passed Memory/SQLite/real PostgreSQL (42.384s). Shared recovery/restore tests also
verify reused-fingerprint parity and that replaying an old host receipt after a
second rotation never reinstalls its retired key. The host command checks schema
compatibility even with migrations disabled, before mutation; future-history
sentinels remain untouched. Combined targeted race passed SQLite 3.023s,
PostgreSQL 20.374s, runtime 5.592s and CLI 2.207s. Full regression, hosted native
checks and integration remain pending; this does not complete P10 or P12.


P10/P12 T15 readiness continuation: host readiness observes the current instance
issuer and whether any Namespace requires signed issuance in one read snapshot.
A signed runtime without a configured signer and a stale replica after explicit
issuer recovery report HTTP 503; ordinary unsigned startup remains ready without
a signing key. Authenticated historical trust remains readable and liveness is
independent. No key is generated silently. The actual SDK fixture provisions a
valid credential policy/agent key, explicitly activates signed protocol, restarts
without the issuer and verifies readiness/liveness, unchanged public trust,
negative issuer preflight and explicit new-issuance refusal on SQLite/PostgreSQL.
Targeted race passed (10.827s); initial fixture runs lacked an enrollment grant,
used a too-short idempotency key and expected readiness in the trust resource
rather than protocol_preflight. Those failed fixtures are not counted as passes;
the product assertions remain intact. Broader runtime regression and hosted
gates remain required before integration; historical dataset and P11–P13 remain
open.
PR #57 integrated host issuer recovery and interruption-safe client trust on
master f06d63c. Full frozen-source race suite passed with real PostgreSQL/clean
restore (214.673s), SQLite (126.291s), runtime (47.389s), CLI (8.862s), actual
signed acceptance (41.944s) and architectural boundaries (6.441s). Vet,
actionlint and catalogue passed. All hosted browser/package/full/native checks
passed. Native Linux/Windows receipts from run 37885413406 identify synthetic PR
merge source 78bf2e2d005ce64d02032e75608c823bd9730450 and version 0.2.0;
these are incremental source-specific proofs, not final-release certification.
P10 historical dataset/old binary and P11 UI/P12/P13 remain required. P11 guide
work proceeds against stable signed interfaces while those independent gates
continue; no wave-completion claim is inferred from this merge.


P11 signed UI projection work started: the workspace observes live Namespace
protocol, shows a public issuer/credential/key/onboarding panel, and reads focal
signed execution/review cases instead of routing signed Tasks through legacy
contract queries. Pending review, acceptance versus quality and protected-profile
next steps are explicit. Legacy execution/raw Task assessment actions and signed
proof-bearing commands are excluded from browser forms; a fresh protocol read
before mutation blocks a stale-form substitution. Parent planning/assessment
remains separate. Browser fixtures use the actual protected CLI with separate executor/reviewer
Principals, exact accepted material and both approved/changes_requested decisions.
Corrections expose accepted findings on existing obligations; no unsigned browser
mutation replaces review. All six browser journeys passed (30.1s), including
legacy cutover and planning/assessment regression. Native onboarding remains
explicit through profiles; runtime pilot and final source acceptance remain open.

PR #58 integrated the five signed-protocol skills and client-guide alignment on
master c91c694. All hosted browser/package/full/native Linux/Windows checks
passed on combined source 129797f. Installer acceptance is not certification of
execution in every consumer agent; P11 UI and final operational acceptance remain
open. T15 readiness source 2941bba passed the complete runtime race suite
(51.364s), actual Memory/SQLite/PostgreSQL signed acceptance (43.777s), vet,
actionlint and catalogue. Merging the guide baseline changes documentation only;
new hosted source checks remain required for the readiness PR.

P10 T81/T82/T83/T88 actual historical fixture: the pinned v0.2.0 executable
produces schema-19 unsigned contracts/material/criteria/seven receipts. Closed
SQLite backup API and PostgreSQL custom dump restore into clean targets; schema
26 upgrade preserves original rows, WorkItem versions and exact replay results
without historical signatures. Premature activation with an active v1 contract
is refused, followed by explicit revocation/cutover. The exact old process may
bind but returns readiness 503 and rejects unsigned acquisition with unknown
work protocol (422), committing no history; full startup refusal cannot be
retrofitted and deployment must retire old writers/revoke database credentials.
Both local drivers passed on service source 5461815. Initial fixture copied
SQLite without closing a reader/checkpointing WAL; using the backup API fixed
the fixture, not product code. Its initial guessed error status 400 was corrected
to the observed precise compatibility rejection 422/invalid_argument. CI now
builds both exact sources and uploads provenance; hosted validation pending.
Signed pending-state restore and final P12/P13 acceptance remain separate.
P11/P12 signed transport consistency: Namespace protocol CAS counters in signed
trust are quoted separately from the literal protocol-2 marker; signed schema
counter types match exact decimal metadata while unsigned encoders/schemas stay
unchanged. Max uint64/typed-client and protocol-marker tests passed. Application, command
and SDK race suites passed (2.322s/1.080s/2.218s); actual signed Memory/SQLite/
PostgreSQL acceptance passed (35.559s). OpenAPI regenerated without altering
v1 command encoders. Hosted/native checks remain required on this source.
PR #59 readiness integrated after all five hosted checks passed on 95e3921;
master merge 5461815. Historical upgrade/restore and final release gates remain open.

P12 native keyring gate started: platform CI now requires an actual OS credential
service. An isolated Linux DBus/Secret Service fixture and native Windows
Credential Manager run protected reference read/write, exact profile MAC and
signing-key fingerprint checks, signing/verification and child-process reads. The actual executor/reviewer
HTTP/CLI lost-response fixture selects those native references in platform CI,
without passing credentials/seeds through child process environments.
Randomly named synthetic entries are deleted; no plaintext fallback, private
process arguments or repository secrets are introduced. Native receipt verification
rejects proofs without the executed keyring gate. Local Linux lacks the daemon;
actual native execution remains pending until hosted jobs finish. Native receipts
now list actual passing Go test count and individual skips, including unavailable
PostgreSQL on client-only hosts, instead of implying every suite ran. Local
Memory/SQLite fallback transport passed (24.272s); PostgreSQL was explicitly
skipped there. Distribution gate unit tests and negative secret redaction pass;
those are not counted as real local keyring execution.

The first native keyring jobs reached actual protected reads and the real
HTTP/CLI journey on Linux/Windows, but the standalone signing fixture omitted
the required signer_key_id from its payload. Both failed and produced no accepted
platform proof. The fixture now supplies the exact selected identity; signing
validation is unchanged and the native gates must rerun.
P12 T86 pending-state restore started: the existing authenticated signed
execution/review/correction journey now takes clean SQL backups at each pending
authority stage, restores into clean targets and resumes the same workflow.
Version/fencing/lease metadata and exact authority envelope bytes are compared
across restore, then delivery, review takeover, correction and approval continue
against restored storage. Memory is explicitly excluded from database restore
claims. Both clean restores passed with the complete authenticated correction/approval
journey: SQLite race 13.231s, PostgreSQL race 30.151s with three empty target
databases. The initial correction-stage fixture used the already revoked
original executor to read metadata; it now uses the current authorized
correction executor, retaining denial assertions. Hosted checks remain pending.

P13 separates coordinated verification from registry publication. Manifest
integration runs all four same-source workflows and uploads hash-checked assets,
actual native receipts and package acceptance. Real registry publication requires
explicit workflow_dispatch publish=true, so finishing the plan does not silently
publish a version. Registry credentials and final source acceptance remain separate.
Existing hosted Linux/Windows execution replaces stale billing-only claims in
release instructions; no registry publication is claimed.
Native keyring gate rerun passed all five hosted checks on PR64 head 6a26bcc.
The downloaded Linux/Windows receipts explicitly record native_keyring_executed,
actual passing counts and individual skips. PR62 historical upgrade and PR63
pending execution/review/correction restores are integrated after their hosted
gates passed. This documentation merge requires a fresh source check before
PR64 integration; final release receipts still require the final combined source.

P12 T32 signed replica-clock experiment passed three PostgreSQL race runs
(3.058s) on Go 1.27.2. Two service instances use clocks +/-365 days; acquisition
and issuer authority deadlines use database time, renewal remains current,
takeover preserves expiry and increases fencing, and old authority is denied.
This is an actual shared-database test, not a claim of separately deployed hosts.
Existing unsigned clock tests remain intact; final source/platform gates apply.
