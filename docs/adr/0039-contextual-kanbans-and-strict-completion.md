# ADR-039: Contextual Kanbans and opt-in strict completion

Status: accepted for implementation by the owner's 2026-10-19 request; qualification is separate.

## Context and decision

The spatial Outcome Board remains the primary workspace. A contextual Kanban is a
read projection over direct children, never a new persisted Board, Card or Column.
One native modal keeps a navigation stack for Objectives, direct Tasks, Tasks of
an Objective, and subobjectives. Stage classification belongs to Application:
terminal accepted work takes precedence over stale review indicators; cancelled
work is excluded from accepted counts. A verification column requires actual
review facts. Read pages retain exact decimal revisions and authenticated cursors.
No drag gesture bypasses commands or changes ownership.

The shared inspector initially shows identity, meaningful content, attention and
a primary action, followed by exactly two principal disclosures: Details and
management, and Activity history. History can be requested lazily through the
additive `include_activity` workspace read option; omitted preserves existing API
behavior. Criteria retain current-assessment applicability. Material registration
reuses atomic Artifact/Evidence commands; neither implies approval. Signed work
continues through the existing authorized-host handoff.

## Completion policy

Outcome gains `completion_policy`: `compatible` preserves historical acceptance;
`strict_work` additionally gates completion on applicable operational work. API,
MCP, CLI and SDK omission remains compatible. Human creation visibly selects strict
work by default. Policy is chosen at creation; changing it on an existing Outcome
is not exposed by this increment. SQLite/PostgreSQL migration 0033 adds the column
with a compatible default; old rows and conclusions remain unchanged.

For an Objective, strict completion checks its complete descendant Objective tree
and Tasks, independent of client paging. A child must be achieved/done or have an
explicit cancellation conclusion with a reason. For Outcome, existing globally
required Objective gates remain; direct Tasks are additionally checked. Optional
Objectives are not made required by their position. A required cancelled Objective
still fails the existing achieved requirement.

Cancellation never counts as acceptance. Each excluded cancelled item must appear
in an exact, duplicate-free acknowledgement set, with an observed Outcome revision.
The existing Outcome guard and optimistic versions revalidate both the work and
acknowledgements. The immutable Conclusion obligations record the acknowledged
references alongside the author's reason and identity. Parent completion never
changes child lifecycle or proof. The repository metadata scans are complete and
exclude historical payload hydration; there is no truncated client-side gate.

## Boundaries

No agent runtime, review queue for parents, storage upload, Cloud subsystem or
implicit Issue/Blocker creation is introduced. Browser controls are capability
hints; command authorization, signatures, separation of duties, fencing and exact
intent recovery remain authoritative. Human comprehension and reference-environment
latency require their own evidence and cannot be inferred from automated tests.
