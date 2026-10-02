# WOS Roadmap

**Status document:** live and mandatory  
**Canonical design:** `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`  
**Last reviewed:** 2026-10-02  
**Current target:** Release 0.1  
**Current wave:** Wave 07 — Issues and Blockers (planned)

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
| Public README | ✅ | Architecture/product overview, explicitly marked as planned state |
| Go module | ✅ | `github.com/A1b3rt0M3rcad0/wos` |
| Repository ignore rules | ✅ | Go, local DB/data, environment/secrets, IDE files |
| AGENTS development contract | ✅ | Root `AGENTS.md` |
| Live implementation roadmap | ✅ | Root `ROADMAP.md` |
| Concrete Open Source license | 🧭 | Must be chosen before first public release; do not inherit Woobe license implicitly |

## Milestones

| Milestone | Status | Definition |
| --- | --- | --- |
| M1 — Local vertical slice | ✅ | Core + SQLite + HTTP with Outcome, Objective, WorkItem and criteria |
| M2 — Full coordination | ⬜ | Dependencies, Blockers, Issues, leases and documentary records |
| M3 — Planning and continuity | ⬜ | Roadmaps, snapshot/graph/timeline and MCP |
| M4 — Complete standalone | ⬜ | PostgreSQL parity, triggers/outbox, auth and packaging |
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

- [x] Establish `core/domain`.
- [x] Establish `core/ports`.
- [x] Add foundational ID / UUIDv7 value object.
- [x] Add `Scope`.
- [x] Add `EntityRef`.
- [x] Add `ActorRef`.
- [x] Add aggregate `Version` and Outcome revision types.
- [x] Add `Clock` port.
- [x] Add `IDGenerator` port.
- [x] Add stable domain/application error model.
- [x] Add transport-neutral `CommandContext` separating authenticated Principal, declared `ActorRef`, and optional external execution correlation.
- [x] Add `cmd/wos/main.go`.
- [x] Add `internal/server` bootstrap/configuration skeleton.
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
- [x] Add public transactional `storage/memory` adapter.
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
- [x] Add OpenAPI 3.1 contract under `api/openapi.yaml`.
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

**Status:** ⬜ Planned

Implement independent Issue and Blocker aggregates, typed causes/targets, direct/subtree propagation, explicit resolution, inherited blocking projection and bounded compound commands.

**Completion gate:** snapshots explain target/cause/inheritance and resolving one impediment never silently resolves another.

**Target commit:**  
`feat(core): separate issues from blockers and implement blocking projections`

---

## Wave 08 — Leases and Fencing

**Status:** ⬜ Planned

Harden WorkItem execution coordination with Claim/Renew/Release/Reclaim, principal-bound leases, TTL, fencing tokens, expiration semantics, attention-needed projection and audited administrative override.

**Completion gate:** at most one current lease exists and a stale claimant cannot complete after reclaim.

**Target commit:**  
`feat(coordination): add work item leases and fencing tokens`

---

## Wave 09 — Documentary Records and Decisions

**Status:** ⬜ Planned

Implement Artifact, Evidence, EvidenceLink and Decision; immutable documentary content; retraction/withdrawal; stance; provenance; explicit acceptance and atomic supersession.

**Completion gate:** another consumer can understand current facts/choices without previous chat history.

**Target commit:**  
`feat(records): add evidence artifacts and explicit decision history`

---

## Wave 10 — Assessments and Verifiable Conclusions

**Status:** ⬜ Planned

Complete revisioned SuccessCriteria, immutable CriterionAssessments, current assessment bindings, waiver authorization, Conclusions, obligation snapshots and contested-conclusion projection.

**Completion gate:** achievement records exactly which criterion revisions and assessments justified the conclusion.

**Target commit:**  
`feat(validation): enforce versioned criterion assessments and explicit achievement`

---

## Wave 11 — Roadmaps and Planning History

**Status:** ⬜ Planned

Implement Roadmap/Revision/Node, editable versioned drafts, immutable publication, active slots, activation history, scopes, phase/reference/milestone nodes and atomic publication with explicit dependency changes.

**Completion gate:** historical revisions remain recoverable; active slot is unique; operational work is never duplicated into plan state.

**Target commit:**  
`feat(planning): add immutable roadmap revisions and scoped activation`

---

## Wave 12 — Continuity Queries

**Status:** ⬜ Planned

Implement coherent Outcome state snapshots, graph queries, timeline, search/filter, pagination, readiness explanations, plan projection, current decisions, contestations, progress metrics and bounded output.

**Completion gate:** another consumer can resume the complete scenario from snapshot + cursors without the previous Session.

**Target commit:**  
`feat(queries): add coherent outcome state graph and timeline projections`

---

## Wave 13 — First-Class MCP

**Status:** ⬜ Planned

Expose the Application layer through the pinned official Go MCP SDK with tools/resources, schemas, stdio and Streamable HTTP, explicit scope/version/claim arguments and parity with HTTP semantics.

**Completion gate:** a real MCP client can inspect, claim, update and resume an Outcome without duplicated domain logic.

**Target commit:**  
`feat(mcp): expose outcome coordination tools resources and protocol profiles`

---

## Wave 14 — PostgreSQL Parity

**Status:** ⬜ Planned

Implement PostgreSQL migrations, repositories, query store, row guard, isolation rules, pool/timeouts and the full shared storage/concurrency contract.

**Completion gate:** SQLite and PostgreSQL satisfy the same functional contract and concurrency suite.

**Target commit:**  
`feat(storage): add PostgreSQL adapter with coordination contract parity`

---

## Wave 15 — Authentication, Grants and External Context

**Status:** ⬜ Planned

Implement token authentication, Namespace grants, Authorizer, Principal/ActorRef delegation, ExternalContext indexing, ExecutionContext separation, external refs and administrative audit.

**Completion gate:** namespace isolation and delegated authorship are enforced independently of consumer metadata.

**Target commit:**  
`feat(auth): add namespace grants delegated actors and external context indexing`

---

## Wave 16 — Triggers, Integration Events and Delivery

**Status:** ⬜ Planned

Implement declarative event Triggers, public Integration Event mapping, TriggerFiring deduplication, outbox delivery, webhook signing, endpoint policy, worker leases, retries/exhaustion and redelivery.

**Completion gate:** consumers receive durable at-least-once signals with stable identity, while WOS still never executes WorkItems or LLMs autonomously.

**Target commit:**  
`feat(integration): add declarative event triggers and durable webhook delivery`

---

## Wave 17 — Packaging, SDK and Examples

**Status:** ⬜ Planned

Add complete binary commands, Docker/Compose, health/readiness, graceful shutdown, Go client, embedded/standalone/Woobe/product-MCP examples, backup/restore and operator documentation.

**Completion gate:** a third party can run WOS without Woobe and resume an Outcome using published documentation.

**Target commit:**  
`feat(server): package standalone deployment embedded usage and Go client`

---

## Wave 18 — Release, Validation and Demonstration

**Status:** ⬜ Planned

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

Waves 01–06 are merged into `master`. Wave 07 — Issues and Blockers is the next planned implementation wave.

1. Start Wave 07 from the current `master`.
2. Implement independent Issue and Blocker aggregates with explicit lifecycle and typed targets/causes.
3. Add direct/subtree blocking propagation and deterministic inherited blocking projections.
4. Integrate blocking into readiness without persisting `ready` or `blocked` as competing lifecycle states.
5. Prove that resolving one impediment never silently resolves another.
