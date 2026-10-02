# ADR-003 — Roadmap ownership and versioning

- Status: Accepted
- Date: 2026-10-02

## Context

WOS needs persistent planning that survives re-planning and allows another consumer to understand earlier plans. Treating a Roadmap as the owner of Objectives or WorkItems would duplicate operational identity and make re-planning capable of accidentally rewriting execution history.

A mutable plan also cannot provide reliable historical inspection. At the same time, activation is current coordination state and therefore must not be encoded by mutating old published content.

## Decision

Roadmap is an Outcome-scoped aggregate that organizes references to operational entities.

- Roadmap scope is either the Outcome itself or one Objective inside that Outcome.
- Objectives and WorkItems remain owned by the operational model; Roadmap only references them.
- Draft editing uses an independent `draft_version`.
- Publishing creates an immutable RoadmapRevision with a monotonically increasing `revision_number` and deterministic content hash.
- `node_key` is stable logical identity across revisions, while physical node records belong to exactly one revision.
- RoadmapNode kinds are `reference`, `phase` and `milestone`.
- Reference nodes may target Objectives or WorkItems; phase and milestone nodes are planning-only.
- Presentation ordering and `after` links are plan semantics, not operational `depends_on` Relations.
- Published revisions snapshot enough target/criterion/dependency information to remain intelligible after live state changes.
- At most one published revision is active per Roadmap scope.
- Activation/deactivation is append-only history plus a current slot, not mutation of published revision content.
- Archiving a Roadmap removes its active slot atomically; reopening does not reactivate a revision.
- If publication also changes operational dependencies, those explicit dependency changes are validated and committed atomically with publication.

## Consequences

Re-planning never changes the identity or lifecycle of live Objectives/WorkItems. Removing a node from a draft does not cancel work, and cancelling work does not rewrite historical plans.

Historical revisions can be audited independently of current execution state.

Roadmap requires its own aggregate/repositories and publication/activation storage, while the operational dependency graph remains canonical in Relations.
