# AGENTS.md — WOS Development Contract

This file defines the operating rules for any human or AI agent that changes this repository.

The canonical architecture specification is:

`docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`

The live implementation state is:

`ROADMAP.md`

The README explains the project publicly. The architecture document defines the intended design. The ROADMAP records what is actually implemented.

---

## 1. Mandatory startup sequence

Before changing code, configuration, contracts, schemas, tests, documentation, or dependencies:

1. Read this `AGENTS.md`.
2. Read `ROADMAP.md`.
3. Read the relevant sections of the canonical architecture specification.
4. Inspect the current repository state; do not assume a planned component already exists.
5. Identify which implementation wave and acceptance criteria the work belongs to.
6. Keep the change inside the established WOS boundaries unless an explicit architectural decision changes them.

Do not infer implementation status from the design document. The design is a plan; `ROADMAP.md` is the live status.

---

## 2. ROADMAP.md is mandatory and must stay current

`ROADMAP.md` is a living project artifact.

Every agent that makes a meaningful repository change MUST review the ROADMAP before finishing.

Update `ROADMAP.md` in the same work whenever the change:

- starts a planned wave;
- completes or partially completes a roadmap item;
- adds a new required task;
- changes implementation order or dependencies;
- discovers a blocker;
- resolves a blocker;
- changes an acceptance criterion;
- changes a milestone or release status;
- introduces an architectural decision that affects future work.

Never mark an item complete because code was merely created. Mark it complete only when the corresponding acceptance condition is satisfied and required tests pass.

A work session is not finished until the ROADMAP has been checked against the resulting repository state.

When no roadmap status changes, do not manufacture progress. Keep the roadmap truthful.

---

## 3. Source-of-truth hierarchy

When information conflicts, use this order:

1. Explicit current instruction from the repository owner.
2. Accepted ADRs in `docs/adr/`.
3. Canonical architecture specification.
4. `ROADMAP.md` for implementation status and current execution order.
5. `README.md` for public explanation.
6. Existing code behavior, when not contradicted by an accepted design change.

If the code and specification diverge, do not silently choose one. Record the discrepancy in the ROADMAP and, when architectural, create/update an ADR.

---

## 4. Product boundary

WOS is an independent, agent-agnostic state and coordination system oriented around Outcomes.

WOS persists and exposes shared state. Humans, agents, Agent Networks, and applications decide and execute.

WOS MUST NOT silently become:

- an autonomous planner;
- an agent runtime;
- an LLM inference layer;
- a Tool executor;
- an agent scheduler;
- a product-specific backend;
- a universal knowledge/RAG service;
- a scriptable automation engine.

Triggers emit durable integration signals. They do not decide or execute work.

Woobe may consume WOS through MCP or another supported contract, but Woobe is not a runtime dependency of the WOS Core.

---

## 5. Core domain invariants

Preserve these semantics unless an explicit ADR changes them.

### Outcome

- Outcome is the semantic root: a desired, verifiable state.
- Outcome completion is explicit.
- WorkItem completion never automatically achieves an Objective.
- Objective achievement never automatically achieves an Outcome.
- Archival is separate from lifecycle.

### Objective

- Objective is a first-class aggregate, not a string.
- An Objective represents a verifiable condition within an Outcome.
- Parent/child hierarchy organizes decomposition; it does not imply execution dependency or automatic completion.

### WorkItem

- WorkItem represents operational work.
- Persist lifecycle, not derived presentation states.
- `ready`, `blocked`, `scheduled`, and `attention_needed` are projections.
- Work acquisition is explicit and lease-based.

### SuccessCriterion / Assessment

- Criteria are structured, identifiable, and revisioned.
- Evidence does not mean a criterion is met.
- CriterionAssessment is the explicit evaluation.
- A new criterion revision does not inherit proof from an older revision.
- Achievement commands validate current required criteria.

### Roadmap

- Roadmap is a versioned plan.
- Roadmap does not own Objectives or WorkItems.
- Roadmap revisions reference live operational entities.
- Published revisions are immutable.
- Plan ordering does not automatically create operational dependencies.

### Issue / Blocker

- Issue describes a problem.
- Blocker represents an actual impediment on an explicit target.
- An Issue may exist without a Blocker.
- Resolving an Issue does not automatically resolve associated Blockers.

### Evidence / Artifact / Decision

- Evidence represents an observation or source.
- Artifact represents a material deliverable/reference.
- EvidenceLink carries stance; Evidence itself does not globally support or contradict everything.
- Decision records an explicit choice and rationale.
- Historical documentary content is not silently rewritten.

### Relations

- `depends_on` is the canonical operational dependency relation.
- Dependency graphs must remain acyclic.
- Avoid duplicating semantics already owned by Blocker, EvidenceLink, Objective hierarchy, Decision supersession, or Roadmap structures.

---

## 6. Architecture boundaries

Product package dependency direction:

```text
wos-api ──> wos-core
```

Inside Core:

```text
application -> domain
      |
      v
    ports <- storage adapters
```

Rules:

- `packages/wos-core/domain` must not import HTTP, MCP, SQL adapters, server bootstrap, Woobe, or agent-runtime packages.
- `packages/wos-core/application` orchestrates commands/queries and uses ports.
- `packages/wos-core/ports` defines infrastructure contracts.
- `packages/wos-core/storage/*` implements ports.
- `packages/wos-api/http` and `packages/wos-api/mcp` map protocol DTOs to the same Application services.
- Transports never write storage directly.
- `packages/wos-api` owns transports and the standalone composition root; both remain outside Domain.
- Public embedded functionality cannot live only in Go `internal/`.
- Do not create microservices for components that are only logical boundaries in the current design.

Prefer standard library and explicit code over unnecessary framework abstraction.

---

## 7. Persistence and transaction rules

The current design uses relational current state plus immutable Domain Events and an outbox. It is not full Event Sourcing.

For domain mutations:

1. validate authenticated scope;
2. begin the UnitOfWork;
3. reserve/check idempotency when applicable;
4. acquire the Outcome coordination guard;
5. load the state required for validation;
6. verify authorization and expected versions;
7. execute domain logic;
8. persist aggregate changes;
9. advance `outcome_revision` exactly as defined;
10. append Domain Events;
11. map Integration Events and evaluate Triggers;
12. persist outbox/delivery state;
13. persist idempotent result;
14. commit.

No external HTTP call, LLM call, agent execution, or artifact URL fetch belongs inside a domain transaction.

SQLite and PostgreSQL must obey the same functional contract even when their locking mechanisms differ.

---

## 8. Concurrency rules

Do not weaken concurrency guarantees for convenience.

- Mutable aggregates use `expected_version`.
- Cross-aggregate invariants use the Outcome coordination guard.
- WorkItem claims use leases.
- Reclaims increment fencing tokens.
- A stale fencing token cannot complete work.
- Dependency cycle validation must be protected against concurrent opposite-edge insertion.
- Read snapshots that claim consistency must use a coherent database snapshot.

Never implement an automatic retry that changes the caller's expected version or semantic intent.

---

## 9. Namespace, Principal and ActorRef

Keep these separate:

- Namespace = internal isolation/authorization boundary.
- ExternalContext = consumer-provided addressing context.
- ExecutionContext = request/run/session correlation context.
- Principal = authenticated identity.
- ActorRef = authorized declared actor/authorship.

Never derive authorization merely from `tenant_id`, `user_id`, metadata, ExternalContext, or ActorRef supplied by a payload.

---

## 10. HTTP and MCP parity

HTTP and MCP are adapters over the same use cases.

For equivalent operations:

- call the same Application service;
- enforce the same authorization;
- enforce the same idempotency rules;
- produce equivalent Domain Events;
- return equivalent domain error semantics.

Do not implement domain logic twice.

MCP connection/session state must never become hidden business state.

---

## 11. Testing requirements

Tests are part of the feature, not follow-up work.

Use the test levels defined in the architecture plan:

- Domain unit tests;
- Application tests;
- storage contract tests;
- integration tests;
- concurrency tests;
- HTTP/MCP parity tests;
- migration/restore tests;
- fuzz/property tests when applicable;
- `go test -race` for concurrency-sensitive code.

A feature cannot be marked complete in `ROADMAP.md` if its planned verification is missing or failing.

SQLite passing does not prove PostgreSQL parity.

An in-memory adapter does not replace storage contract tests.

---

## 12. Documentation discipline

Keep these artifacts aligned:

- `README.md`: public product/architecture overview.
- `ROADMAP.md`: live implementation state.
- canonical architecture document: detailed design and original implementation plan.
- `docs/adr/`: accepted architecture decisions.
- `docs/dependencies.md`: current toolchain/dependency inventory and dependency admission rules.
- OpenAPI / JSON Schemas: protocol contracts once implemented.

When a public contract changes, update the related documentation and schemas in the same change.

Do not describe a planned feature as already implemented.

---

## 13. ADR policy

Create an ADR when a change alters a significant architectural choice, including:

- aggregate boundaries;
- persistence model;
- concurrency model;
- HTTP/MCP contract strategy;
- public Go Core API;
- authentication/authorization model;
- event/outbox semantics;
- roadmap ownership/versioning;
- trigger semantics;
- supported storage behavior.

Do not bury architectural decisions in commit messages or code comments.

The canonical specification proposes ADR-001 through ADR-015. Materialize them as repository ADRs as the corresponding areas are implemented or explicitly accepted.

---

## 14. Implementation waves

The canonical plan contains 18 ordered implementation waves.

Do not mark a later wave complete while required earlier foundations are absent unless the work is explicitly parallelizable in the plan.

Parallel work is allowed only when interfaces and dependencies required by that work are stable.

The ROADMAP must state when implementation order intentionally differs from the original wave order.

---

## 15. Change workflow

For each meaningful task:

### Before implementation

- inspect `ROADMAP.md`;
- identify affected wave(s);
- inspect relevant design sections and ADRs;
- inspect existing tests and contracts.

### During implementation

- preserve architectural boundaries;
- write/update tests together with behavior;
- keep changes reviewable;
- avoid unrelated refactors;
- avoid speculative abstractions not required by the active wave.

### Before finishing

- run the relevant test suite;
- run formatting/static checks appropriate to the current repository stage;
- compare result against wave acceptance criteria;
- update `ROADMAP.md`;
- update documentation/contracts when behavior changed;
- verify that no planned capability is falsely presented as implemented.

---

## 16. Commit discipline

Prefer small, semantic commits.

Use conventional commit prefixes when practical:

- `chore:`
- `feat(core):`
- `feat(storage):`
- `feat(http):`
- `feat(mcp):`
- `feat(auth):`
- `feat(integration):`
- `feat(queries):`
- `feat(planning):`
- `feat(records):`
- `feat(validation):`
- `feat(coordination):`
- `test:`
- `docs:`
- `fix:`
- `refactor:`

The wave commit messages in the canonical plan are target integration boundaries, not permission to combine unrelated work into one oversized commit.

---

## 17. Definition of done

A planned item is done only when all applicable conditions hold:

- implementation exists;
- domain invariants are preserved;
- tests required by the plan pass;
- public contracts are documented;
- repository state matches the ROADMAP;
- no false implementation claim remains in README/docs;
- required ADRs/contracts/migrations are present;
- the implementation can be resumed by another agent from repository state alone.

The repository must remain understandable without relying on the previous agent's chat history.
