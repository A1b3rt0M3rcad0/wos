# wos — Woobe Outcome Server

> Persistent, agent-agnostic infrastructure for outcome-oriented state and coordination.

**WOS (Woobe Outcome Server)** is an independent server and embeddable Go core for preserving the shared state required to pursue outcomes over time.

Humans, agents, agent networks, and conventional applications decide and execute. WOS validates, persists, relates, queries, and coordinates the state around that work.

> **Project status:** Waves 01–04 are merged and verified. Wave 05 is implemented and verified on PR #5, pending merge: WOS now has a standalone SQLite-backed HTTP runtime, local principal mode, durable M1 Outcome/Objective/WorkItem/criteria APIs, idempotency keys, ETags/preconditions, health endpoints, OpenAPI and restart-safe HTTP state recovery. MCP, dependency/readiness graphs, PostgreSQL and later-wave capabilities remain planned.

## Why WOS

Agent runtimes are good at executing. Product backends are good at owning product-specific business rules. Neither necessarily provides a durable, neutral model for answering questions such as:

- What result are we trying to achieve?
- Which verifiable objectives define that result?
- What work exists, what is ready, what is in progress, and what is blocked?
- Which plan is currently active?
- What did previous plan revisions contain?
- Which issues, blockers, decisions, artifacts, and evidence explain the current state?
- Which criteria were assessed before an Objective or Outcome was concluded?
- Can another consumer safely resume the same work without relying on chat history?

WOS exists to preserve that coordination state independently of any particular runtime, model, session, or product.

## Core principle

The main semantic unit is an **Outcome**: a desired, verifiable state.

A completed WorkItem is not the same thing as an achieved Objective. An achieved Objective is not automatically the same thing as an achieved Outcome.

```text
Outcome
├── SuccessCriteria
├── Objectives
│   ├── SuccessCriteria
│   └── WorkItems
├── WorkItems
├── Roadmaps
├── Issues
├── Blockers
├── Decisions
├── Evidence
├── Artifacts
├── Relations
├── Events
└── Triggers
```

A Roadmap organizes the plan but does not own the operational entities. The Outcome owns the operational scope. Objectives and WorkItems stay live while Roadmap revisions preserve historical plans.

## Design thesis

State belongs to the **work context**, not to the agent or session that started it.

A new session, another agent, a human, or another application must be able to inspect the same Outcome and continue from persisted state.

WOS is therefore a persistent coordination substrate, not an autonomous decision-maker.

## Responsibilities

WOS is designed to:

- create and maintain verifiable Outcomes and Objectives;
- coordinate WorkItems and explicit execution claims;
- preserve versioned Roadmaps and their historical revisions;
- validate transitions, references, criteria, dependencies, and graph invariants;
- represent Issues and Blockers as separate concepts;
- register Evidence, Artifacts, Decisions, and provenance;
- expose deterministic state, graph, readiness, and timeline queries;
- enforce optimistic concurrency and idempotent remote commands;
- coordinate WorkItem leases using fencing tokens;
- persist Domain Events and durable Integration Events;
- evaluate declarative event-based Triggers and persist signals;
- isolate state through Namespaces and authorization boundaries;
- support humans, agents, Agent Networks, and traditional software.

## Non-goals

WOS is intentionally **not**:

- an agent runtime;
- an LLM inference service;
- an autonomous planner;
- a tool executor;
- an agent scheduler;
- a replacement for a product backend;
- a universal knowledge base;
- a graph database requirement;
- a general-purpose workflow engine with arbitrary scripts.

WOS may report that work is ready or blocked. It does not decide which agent must execute it.

A Trigger may emit a durable signal. It does not start a Run or execute a WorkItem by itself.

## Core domain

### Outcome

Represents the final desired state.

Initial lifecycle:

```text
draft -> active -> achieved | failed | abandoned
                  ^                |
                  └----- reopen ---┘
```

Archiving is orthogonal to lifecycle. An achieved Outcome can also be archived.

### Objective

Represents a verifiable condition inside an Outcome.

Initial lifecycle:

```text
planned -> in_progress -> achieved
   |                         |
   └-------> cancelled <-----┘
```

Reopening returns an Objective to `planned`.

Objectives may form a hierarchy, but hierarchy does not imply execution dependency or automatic completion.

### WorkItem

Represents an operational unit of work.

Initial lifecycle:

```text
backlog -> todo -> in_progress -> done
   |        |          |
   └--------┴----------┴----> cancelled
```

Operational presentation such as `ready`, `blocked`, `scheduled`, or `attention_needed` is derived from lifecycle, dependencies, Blockers, time constraints, and leases.

### SuccessCriterion

A stable, addressable verification criterion owned by an Outcome, Objective, or WorkItem.

A criterion has its own semantic revision so that an assessment against revision 1 never silently proves revision 2.

Verification modes initially include:

- `attestation`;
- `evidence_review`;
- `external_evaluation`.

### CriterionAssessment

An immutable assessment of a criterion revision.

Results:

- `met`;
- `not_met`;
- `inconclusive`;
- `waived`.

Attaching Evidence does not automatically mark a criterion as met. Assessment is explicit.

### Roadmap

A persistent, versioned plan over an Outcome or Objective scope.

Roadmap revisions are immutable after publication. They reference live Objectives and WorkItems instead of duplicating them.

Roadmap ordering is not automatically an operational dependency graph.

### Issue

Represents a problem, anomaly, question, incompatibility, or unexpected result.

An Issue may exist without blocking execution.

### Blocker

Represents an actual impediment over exactly one operational target.

An Issue explains a problem. A Blocker states what is currently prevented.

Resolving an Issue does not automatically resolve its Blockers.

### Evidence

Represents an observation, measurement, test result, inspection, attestation, source, or external evaluation.

Evidence content is immutable; later invalidation is represented by lifecycle/retraction rather than rewriting history.

### Artifact

Represents a material deliverable or external object such as a PR, commit, document, report, dataset, release, deployment, or file.

The MVP stores references and metadata, not arbitrary artifact bytes.

### Decision

Represents an explicit choice, alternatives, rationale, authorship, and supersession history.

Accepted Decisions are not silently rewritten.

### Relation

Represents a typed local relationship.

Initial relation types:

- `depends_on`;
- `relates_to`;
- `produces`;
- `derived_from`.

Operational dependency uses `depends_on` and must remain acyclic.

### Event

Every confirmed domain mutation emits immutable Domain Events in the same transaction.

The current-state tables remain the source of truth. WOS does **not** require full Event Sourcing.

### Trigger

A declarative integration rule:

> when a compatible event occurs, persist a signal for configured destinations.

Triggers create durable integration signals. External consumers decide what to do with them.

## Readiness

Readiness is deterministic and derived.

Conceptually, a WorkItem is ready when:

```text
lifecycle == todo
AND outcome is active
AND outcome is not archived
AND associated objective permits execution
AND no applicable Blocker is active
AND all hard dependencies are satisfied
AND not_before <= evaluated_at
AND no active lease prevents acquisition
```

The result carries explicit reasons instead of a hidden model score.

## Execution coordination

WorkItems use explicit leases.

A claim contains:

- claim ID;
- authenticated principal;
- declared ActorRef;
- acquisition time;
- expiration time;
- fencing token.

The fencing token prevents a stale claimant from completing work after another consumer reclaimed an expired lease.

The lease coordinates WOS consumers. External systems still need their own idempotency or fencing semantics for side effects outside WOS.

## Concurrency model

WOS uses two complementary mechanisms:

1. **Outcome coordination guard** for invariants spanning several aggregates.
2. **`expected_version`** on mutable aggregates to protect client intent.

Each successful command advances an `outcome_revision`, which identifies the transactional order of changes inside one Outcome.

These concepts are deliberately distinct:

- aggregate `version`;
- Roadmap `revision_number`;
- Roadmap draft version;
- Outcome-wide `outcome_revision`.

## Idempotency

Remote mutations require an idempotency key.

Identity is based on:

```text
(namespace_id, principal_id, command_name, idempotency_key)
```

Replaying the same confirmed command returns the original stored result rather than creating duplicate work or duplicate Events.

## Namespace and identity

A **Namespace** is the internal isolation and authorization boundary.

`ExternalContext` is consumer-provided addressing context. It is not itself an authorization mechanism.

An authenticated **Principal** is distinct from an **ActorRef**:

- Principal = who is authenticated and authorized;
- ActorRef = who is declared as the author/actor of a domain action.

Delegated ActorRefs require explicit authorization.

## Architecture

WOS starts as a modular monolith with a public Go Core and multiple adapters.

```text
                  ┌────────────────────┐
                  │   WOS standalone   │
                  │      server        │
                  └─────────┬──────────┘
                            │
             ┌──────────────┴──────────────┐
             │                             │
             ▼                             ▼
        HTTP transport                MCP transport
             │                             │
             └──────────────┬──────────────┘
                            ▼
                    Application services
                            │
               ┌────────────┴────────────┐
               ▼                         ▼
             Domain                     Ports
                                          ▲
                                          │
                                Storage / integrations
```

Code dependency direction:

```text
transport -> application -> domain
                    |
                    v
                  ports <- adapters
```

The Domain layer does not import HTTP, MCP, SQL, server bootstrap, Woobe, or agent-runtime packages.

## Deployment modes

WOS is designed for several deployment modes:

| Mode | Description |
| --- | --- |
| Local | SQLite, loopback, local principal |
| Standalone | WOS server over HTTP/MCP |
| Embedded | Go application imports Core and a storage adapter |
| Multi-user | PostgreSQL plus service authentication |
| Composed | Another product integrates WOS without changing WOS semantics |

WOS must start and operate without Woobe.

## Storage

Initial storage contract:

- SQLite for local/single-node deployments;
- PostgreSQL for remote and multi-replica deployments;
- equivalent functional behavior validated by shared contract tests.

Current state is relational.

Domain Events are immutable.

Integration Events and outbox deliveries are persisted transactionally with domain mutations.

Redis, a vector database, or a distributed event bus are not MVP requirements.

## HTTP API

The HTTP API is versioned under:

```text
/api/v1/namespaces/{namespace_id}
```

Outcome-scoped operations live under:

```text
/outcomes/{outcome_id}
```

The transport handles DTOs, authentication context, preconditions, and error translation. It does not implement domain rules directly.

## MCP

MCP is a first-class transport over the same Application services as HTTP.

The intended catalog includes tools for:

- finding Outcomes;
- getting Outcome state;
- graph and timeline queries;
- listing ready work;
- creating/updating/transiting Outcomes;
- managing Objectives and WorkItems;
- claiming, renewing, releasing, reclaiming, and completing work;
- managing criteria and assessments;
- Issues and Blockers;
- Evidence and Artifacts;
- Decisions and Relations;
- Roadmaps;
- Triggers.

MCP protocol state never becomes the hidden source of business state.

## Integration with Woobe

WOS is independent from Woobe, but the two systems are complementary.

```text
Product backend
      │
      ├──────────────────────────────┐
      ▼                              ▼
    Woobe                           WOS
Agent execution             Outcome / Work state
      │                              ▲
      ├── Tools / MCP                │
      └──────────────────────────────┘
                    via MCP
```

A product backend may also consume WOS directly.

Woobe owns agent execution concerns such as Agent Networks, Sessions, Runs, and runtime traces.

WOS owns persistent state around Outcomes, Objectives, WorkItems, plans, problems, evidence, and coordination.

The product backend remains the source of truth for its own business domain.

## Planned repository structure

```text
cmd/
  wos/
    main.go

core/
  domain/
  application/
  ports/

storage/
  memory/
  sqlite/
  postgres/

internal/
  server/
  transport/
    http/
    mcp/
    contracts/
  authentication/
  integration/
    outbox/
    webhook/
  observability/

client/
  go/

api/
  openapi.yaml
  jsonschema/

docs/
  architecture.md
  domain.md
  deployment.md
  mcp.md
  adr/

examples/
  embedded/
  standalone/
  woobe-integration/
  product-mcp/

tests/
  contract/
  integration/
  concurrency/
  fixtures/

deploy/
  Dockerfile
  compose.sqlite.yaml
  compose.postgres.yaml
```

The public Core must stay importable by external Go applications, so it cannot live entirely under `internal/`.

## Initial implementation stack

- Go;
- `net/http`;
- `database/sql`;
- explicit SQLite and PostgreSQL adapters;
- official Go MCP SDK matching the pinned protocol profile;
- `log/slog`;
- UUIDv7 identifiers;
- versioned SQL migrations;
- optional OpenTelemetry instrumentation at the Server boundary.

The current foundation dependency inventory is documented in [`docs/dependencies.md`](./docs/dependencies.md). Wave 01 intentionally has no third-party Go module dependency; future dependencies are introduced and pinned only in the wave that needs them.

## Development milestones

| Milestone | Target |
| --- | --- |
| M1 | Core + SQLite + HTTP; Outcome, Objective, WorkItem and criteria |
| M2 | Dependencies, Blockers, Issues, leases and documentary records |
| M3 | Roadmaps, snapshot/graph/timeline and MCP |
| M4 | PostgreSQL parity, triggers/outbox, auth and packaging |
| 0.1 | Contracts, CI, examples and documentation validated |

## Release 0.1 acceptance criteria

The initial release is expected to prove that:

1. standalone WOS starts without Woobe;
2. an external Go program can use embedded mode;
3. all remote mutations are idempotency-protected;
4. stale aggregate versions are detected;
5. concurrent claims and dependency-cycle races preserve invariants;
6. SQLite and PostgreSQL pass the same storage contract suite;
7. HTTP and MCP execute the same Application services;
8. snapshots are coherent, bounded, and explicit about omissions;
9. published Roadmap revisions are immutable and recoverable;
10. assessments preserve the criterion revisions and Evidence used;
11. Triggers produce durable signals without autonomous execution;
12. restart/restore preserves state required for continuation.

## Testing strategy

The planned test suite includes:

- Domain unit tests for transitions and invariants;
- Application tests with deterministic Clock/ID generators;
- shared storage contract tests for SQLite and PostgreSQL;
- deterministic concurrency tests;
- HTTP/MCP contract parity tests;
- migration and restore tests;
- property tests and fuzzing for graphs, cursors, JSON, and Unicode;
- `go test -race`;
- independence tests proving operation without agents, Sessions, Runs, or Woobe.

## Documentation and development workflow

The repository maintains three complementary project documents:

- [`AGENTS.md`](./AGENTS.md) — mandatory development contract for humans and AI agents, including architecture boundaries, testing discipline, and the rule that the live roadmap must remain synchronized with implementation.
- [`ROADMAP.md`](./ROADMAP.md) — live implementation status. This is the source of truth for what is actually complete, in progress, blocked, or still planned.
- [`docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`](./docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md) — canonical architecture and detailed implementation specification.
- [`docs/dependencies.md`](./docs/dependencies.md) — current toolchain/dependency inventory and dependency admission rules.

The architecture document defines the target design. The ROADMAP records real implementation state. Planned capabilities must never be presented as implemented until their acceptance criteria are satisfied.

## License

The WOS design specifies that the project is intended to be Open Source and licensed independently from Woobe.

**No concrete license is being assumed by this bootstrap.** A specific license must be selected and recorded before the first public release.

## Current state

Wave 01 — Public Core Foundation is merged and verified.

Wave 02 — Transactions, Memory and Initial Domain is merged and verified. It provides pure Outcome/Objective/WorkItem lifecycle rules, revisioned criteria, immutable assessment history plus current projections, attestation-based assessments, explicit conclusions, declarative owners/assignees, basic WorkItem claims, repository/UnitOfWork ports, aggregate validation, a transactional in-memory adapter with rollback and Outcome revision markers, complete Wave 02 application commands, and a human-only end-to-end scenario.

Wave 03 — Event Log and Idempotency is merged and verified. It provides immutable Domain Events, ordered event indices under one Outcome revision, normalized command fingerprints, transactional idempotency reservations/results, replay of original command results, conflict detection and explicit no-op audit semantics.

Wave 04 — SQLite Persistence and Migrations is merged and verified. It adds durable normalized SQLite state, versioned/checksummed migrations, immediate writer acquisition, Outcome coordination rows, optimistic `expected_version` writes, persisted Domain Events/idempotency, restart-safe reconstruction, backup/restore and SQLite concurrency/cancellation tests.

Wave 05 — HTTP Vertical Slice is implemented and verified on PR #5, pending merge. It adds the standalone SQLite-backed HTTP runtime, local principal resolution, health endpoints, Outcome/Objective/WorkItem/criteria routes, idempotency-key enforcement, ETags and version preconditions, stable errors, the first durable Outcome state query, OpenAPI and restart-safe HTTP integration coverage.

Implementation follows the ordered waves and ADR decisions in the canonical design document, with each milestone remaining testable and independently reviewable.
