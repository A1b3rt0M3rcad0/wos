# ADR-038: Operational workspace projections and contextual writes

Status: accepted for implementation in draft PR #116. Qualification is tracked
separately in `docs/ux/operational-workspace/acceptance.json`.

## Context

Planning on the Outcome Board must expose the same current obligations as the
specialist views without turning the browser into another coordination engine.
The owner plan extends presentation with actual documentary notes, guarded
definition edits, structural moves, and atomic contextual registration.

## Decision

Keep the existing domain, Application pipeline, authority protocols and storage
adapters. New mutations use the same authorization, exact-intent replay, Outcome
guard, optimistic versions, events and outbox transaction. Compound creation and
documentary operations compose private primitives inside one UnitOfWork; they
never call public command handlers recursively.

An EntityNote is immutable documentary text with authenticated principal/actor,
server registration time, optional occurrence time and scoped references. A
correction preserves its original record. Notes do not acquire authority, execute
work, assess criteria or accept a delivery. Contract activity retains its signed
authority checks. An internal observation used as Evidence resolves to a real,
authorized note; a provider string alone is not provenance.

Workspace views pin one authorized read transaction, Outcome revision and
evaluation time. They reuse current work-context, attention and definition guards.
Cards use bounded batch reads. Coverage distinguishes unknown, partial and empty
facts. Timeline cursors pin an event frontier and deterministic event index;
structural changes invalidate descendant-history placement rather than inventing
historical ancestry. Versions in new workspace projections use exact decimal
strings; legacy public wire formats remain unchanged.

The browser shares one inspector renderer across Board and native dialogs.
Capabilities explain current availability, never grant authority. Every mutation
revalidates authorization and domain conditions. Structured error details carry
reason codes, affected fields/references and safe remedy action IDs; the browser
does not parse English errors into business rules. Only a version conflict opens
edit comparison. An uncertain dispatch retains its exact request and receipt key.

Human quick creation visibly defaults Tasks to backlog and Objectives to required
for the Outcome. Existing API defaults and stored records do not change. Initial
criteria are explicit records, never invented from a description. Reparenting is
an explicit guarded operation separate from local card movement. Hard dependency
on a Task's own Objective is rejected because its completion conditions cannot be
satisfied; existing bad configurations remain visible for deliberate correction.

## Boundaries and verification

No scheduler, execution runtime, remote test runner, upload service or presence
claim is added. Evidence registration, assessment and acceptance remain separate.
Memory, SQLite and PostgreSQL must agree, including rollback, replay, migrations
and concurrent guards. HTTP, MCP, CLI and SDK changes are generated and checked
together. Read-only POST queries retain authentication and CSRF protections.

Automated tests, measured performance, screenshots and independent human
comprehension tests are separate evidence. Pending human sessions do not become
passed through screenshots or author walkthroughs. No merge or release is implied.
