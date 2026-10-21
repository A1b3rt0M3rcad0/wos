# ADR-001 — Outcome-oriented domain model

- Status: Accepted
- Date: 2026-10-01

## Context

A task-centric model does not preserve the verifiable state that WOS is meant to coordinate across humans, agents, sessions and applications.

## Decision

Outcome is the semantic root. Objective represents a verifiable intermediate condition. WorkItem records operational execution. Outcome, Objective and WorkItem use distinct lifecycle and conclusion semantics.

A completed WorkItem never automatically achieves an Objective. An achieved Objective never automatically achieves an Outcome.

## Consequences

Consumers must model completion criteria explicitly. The additional structure preserves result semantics across sessions instead of reducing progress to a checklist.

## Supersession scope

ADR-043 and ADR-045 supersede the blanket Task-to-Objective prohibition only
for an `operational_v2` Objective whose owner explicitly delegated the
versioned `automatic` completion policy. Task acceptance alone still does not
prove Objective criteria or create an Assessment; the derived Objective
Conclusion requires the real current assessments and all other completion
gates. Objective achievement never automatically achieves an Outcome. Manual
completion remains explicit under the applicable authorization and gates.
