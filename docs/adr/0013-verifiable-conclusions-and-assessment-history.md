# ADR-013 — Verifiable conclusions and immutable assessment history

- Status: Accepted
- Date: 2026-10-02

## Context

WOS already has SuccessCriteria, an attestation-oriented CriterionAssessment path and terminal lifecycle transitions. That foundation is insufficient for a verifiable conclusion because it does not yet model every verification mode, explicit Evidence use, waiver authorization, immutable criterion-definition history as a domain concept, or the exact structural obligations that justified a terminal transition.

A later contradictory assessment or retracted Evidence must not rewrite historical facts or silently reopen an Outcome, Objective or WorkItem.

## Decision

Wave 10 makes validation history explicit:

- each semantic SuccessCriterion change creates a new immutable criterion-definition revision;
- CriterionAssessment is immutable and targets exactly one criterion revision;
- the current assessment is a separate owner-bound pointer, replaced with optimistic concurrency;
- `evidence_review` requires active same-Outcome Evidence;
- `external_evaluation` records evaluator metadata;
- `waived` is distinct from `met` and requires explicit `assessment:waive` authorization plus a reason;
- a Conclusion is an immutable record of a terminal transition, not merely fields embedded in mutable owner state;
- a Conclusion snapshots the owner version, lifecycle result, criterion revision/assessment references and structural obligations used by the gate;
- reopening removes the current-Conclusion binding but preserves the historical Conclusion;
- later contradictions create a contestation projection and never change lifecycle autonomously.

The WOS still does not judge external truth. It records the assessment, Evidence references, evaluator/provenance and authorization that a consumer supplied.

## Consequences

Achievement becomes reproducible and auditable rather than inferred from current mutable state.

Criterion revisions and assessments can outlive the current binding without becoming applicable to newer revisions.

The existing foundation tables for criterion revisions, assessments and Conclusions remain useful, but Wave 10 may extend them where the domain contract now requires Evidence references, evaluator metadata and richer obligation snapshots.
