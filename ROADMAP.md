# WOS Roadmap

**Status document:** live and mandatory  
**Canonical design:** `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`  
**Last reviewed:** 2026-10-01  
**Current target:** Release 0.1  
**Current wave:** Wave 03 — Event Log and Idempotency (complete in PR #3; not yet merged)

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
| M1 — Local vertical slice | ⬜ | Core + SQLite + HTTP with Outcome, Objective, WorkItem and criteria |
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

**Status:** ⬜ Planned

Implement SQLite adapter, schema migrations, SQL UnitOfWork, registry/coordination rows, repository contracts, idempotency/event persistence, restart-safe state and SQLite writer coordination.

**Completion gate:** Wave 02 scenario survives process restart and all writes enforce expected version.

**Target commit:**  
`feat(storage): add SQLite persistence migrations and outcome coordination guards`

---

## Wave 05 — HTTP Vertical Slice

**Status:** ⬜ Planned

Expose the M1 domain through HTTP with DTOs, auth-local mode, version preconditions, idempotency keys, ETags, error mapping, OpenAPI and first state query.

**Completion gate:** standalone binary can create, modify and query durable state through HTTP with reproducible documentation.

**Target commit:**  
`feat(http): expose versioned outcome coordination API`

---

## Wave 06 — Dependencies, Graph and Readiness

**Status:** ⬜ Planned

Implement canonical `depends_on` relations, hard/advisory semantics, DAG validation, Objective hierarchy constraints, deterministic readiness, `not_before`, readiness reasons and ready-work queries.

**Completion gate:** memory and SQLite readiness are equivalent and concurrent edge insertion cannot create dependency cycles.

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

Wave 01 and Wave 02 are merged into `master`. Wave 03 is technically complete on PR #3 and remains unmerged pending review.

1. Review PR #3 and its final CI result.
2. Merge Wave 03 only when approved by the repository owner.
3. Keep Wave 04 — SQLite Persistence and Migrations as **Planned** until that merge/authorization.
4. Do not move SQLite persistence concerns into Wave 03 follow-up work.

Wave 04 implementation has not started.
