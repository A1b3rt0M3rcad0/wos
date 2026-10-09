# Human workspace experience contract

Scope: the owner's October 9 plan. The command catalog remains the wire contract;
the UI registry describes human intent and never grants authority. Wire enums,
user-authored content, CAS, fencing, and idempotency remain unchanged.

## Information architecture and glossary

Access → Workspace → Outcomes → selected Outcome → Overview / Plan / Work /
Issues / Evidence / Activity. Workspace settings is separate. Work projections
are filters, not top-level entity collections. Task lifecycle, readiness, and
execution authority have separate labels. A board presents state; it does not
authorize a transition by dragging a card.

Workspace is the human name for Namespace. Outcome is a desired verifiable
result; Objective is an intermediate verifiable condition; Task is operational
work. Roadmap references those entities and does not own them. Evidence records
observations; an assessment evaluates a particular criterion revision. Waived
means not verified. An Issue describes a problem; a Blocker independently
impedes a target. A signed work contract needs an external authorized profile.

## Journey contracts

| Flow | Entry and essential controls | Wire mapping and explicit follow-up |
| --- | --- | --- |
| F01 | New outcome: name, desired outcome, optional description | `create_outcome`; persisted Draft → criteria/objectives/tasks checklist → explicit `activate_outcome` |
| F02 | Plan: Add objective, title, optional parent, required for outcome | `create_objective`; typed parent ID, optional priority → criteria / tasks / explicit start |
| F03 | Work: New task, title, optional Objective, initial Backlog/To do | `create_work_item`; optional `execution_spec` typed list controls, schedule → separate dependency/acceptance actions |
| F04 | Plan: Create roadmap → Phase/Milestone/Reference rows → Review changes | `create_roadmap`, `open_roadmap_draft`, `replace_roadmap_draft`; stable internal node keys → separate publish and activate |
| F05 | Owner detail: Define success criteria → Register/link evidence → Assess | `add_criterion`, documentary commands, `record_criterion_assessment`/`attest_criterion`; exact current revision, no inferred Met |
| F06 | Issues: Report issue / Block work; distinct targets and causes | Individual commands or explicitly labelled compound report/resolve; resolving Issue alone retains Blockers |
| F07 | Task detail: execution, expiry, submission, review, findings | Legacy commands only in compatible protocol; signed v2 is read-only status and copyable profile/CLI guidance |
| F08 | Editing: server conflict comparison / uncertain response | Retain local fields; conflict requires explicit renewed intent and current version; uncertainty retries identical frozen payload/key |

Each mutation uses an immutable intent: operation, payload, namespace/outcome,
expected version, idempotency key. Recoverable errors retain fields. Changing
scope never changes the scope of an in-flight intent. Query generations discard
late responses. Reconfirmation after a conflict creates a new intent; retrying
an uncertain response does not.

## Reference contract

The additive bounded reference query returns typed EntityRefs, title, lifecycle,
short ID, and optional context. It searches storage, not a loaded UI page. The
allowlist covers all reference-bearing public collections. Deterministic keyset
cursors bind namespace, outcome, query, kinds, ordering and Outcome revision;
stale revisions are explicit errors. Authorized resolution by ID preserves
selected values outside a result page. Typed multi-selection is independent of
pagination. Search query/evidence contents are excluded from default telemetry.

## Baseline and acceptance evidence

Human creation times, error rates, help requests and comprehension are **not
measured**. The source audit and prior test logs are historical evidence only.
Automated journey durations will be recorded separately from human usability.
Seed sizes: empty, small, 32 tasks and 1,000+ duplicate-titled entities; mixed
lifecycles, criteria, independent Blockers, and actual scoped grants. Test setup
must use public commands, not database writes that bypass domain constraints.

Acceptance requires fresh F01–F08 browser tests, off-page duplicate reference
selection, revocation, conflict/retry and navigation races, real supported store
parity, keyboard/reflow/axe checks, and embedded build/distribution verification.
Manual representative-user and screen-reader results remain separate evidence;
passing automation is not a claim of 90% comprehension or full WCAG certification.
