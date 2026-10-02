# ADR-012 — Documentary records and explicit decision history

- Status: Accepted
- Date: 2026-10-02

## Context

WOS must preserve facts, deliverables and choices independently of the session, human or agent that produced them. Documentary state must remain auditable without turning references into truth claims or allowing later edits to rewrite historical observations.

Evidence, deliverables and decisions also have different semantics: a file is not proof by itself, a piece of Evidence can support or contradict different targets, and a Decision is an explicit choice whose accepted content must remain historically stable.

## Decision

Wave 09 introduces four Outcome-local aggregates:

- `Artifact` identifies a material deliverable or external object by metadata/reference. WOS does not dereference the URI or store arbitrary bytes.
- `Evidence` records an immutable observation with provenance. Retraction changes lifecycle but never rewrites the observation.
- `EvidenceLink` assigns `supports`, `contradicts` or `context` stance to Evidence relative to a local target. Stance belongs to the link, not to Evidence itself.
- `Decision` records proposal, alternatives, rationale and explicit lifecycle. Proposed content may be edited optimistically; accepted/rejected/superseded content is immutable.

EvidenceLink is not a CriterionAssessment and never marks a criterion as met.

Decision supersession is a dedicated semantic relation. Accepting a successor that supersedes an accepted Decision and marking the predecessor `superseded` occur in the same Outcome-coordinated transaction. An accepted Decision may have at most one accepted direct successor and supersession cycles are forbidden.

Public deletion is not part of the MVP. Artifact withdrawal and Evidence retraction preserve documentary history.

## Consequences

Documentary records remain understandable across sessions and consumers while preserving original content and provenance.

External URIs and producer/source metadata are recorded as claims/references; WOS does not certify their authenticity.

Cross-Outcome/cross-Namespace documentary links remain invalid in the MVP.

Storage adapters must preserve the same lifecycle, optimistic concurrency, idempotency, restart behavior and supersession invariants.
