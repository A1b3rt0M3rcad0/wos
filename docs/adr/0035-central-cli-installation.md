# ADR-035: One local CLI installation with pinned profiles

Status: accepted for implementation in the centralized-agent-workflow branch; integration and release remain owner actions.

## Context

The owner requests CLI operations from any working directory, without per-project `.wos` state or agent editing of YAML. Different profiles may address different servers and Namespaces. Existing signed specifications, protected secret references, frozen intentions, durable receipts, stable OS locks, and local content comparisons must remain valid.

## Decision

The default `wos init` initializes one private installation data directory, using the OS data location or an explicit absolute `WOS_DATA_DIR`. There is no `home` command or second user-facing storage abstraction. Project source directories do not receive operational state.

Each named profile has an isolated private store. Existing schema-2 document encoding can remain an internal implementation detail; it is not an agent interface. A profile retains its exact server/Namespace/Principal/credential/key binding. Separate private stores allow profiles to address different destinations without weakening that binding.

Selection is explicit `--profile`, then process `WOS_PROFILE`, then the installation's persistent active profile. A sole-profile fallback is permitted when no active choice exists. The process captures that choice once before performing work; another process changing the persistent preference cannot redirect existing intentions or a long-running lease operation. Ambiguous identities and ambiguous contract references fail with an explicit choice rather than guessing.

Profile and material mutations use the established protected writes, stable locks, content comparison, binding validation, and exact-intent journal. Selection metadata cannot grant remote permissions. Removing a profile does not revoke server authority or delete shared external secret entries; active material and unresolved intentions prevent destructive removal.

Operational changes use CLI commands that validate, lock, and update only editable material. Issued specification and authority remain immutable. A frozen signed return prevents conflicting edits until its original receipt or terminal-resolution proof is reconciled. Recorded local progress is not an Assessment.

## Migration and compatibility

New default commands do not discover state from the current directory. Explicit legacy `--workspace` / `WOS_WORKSPACE` remains a bounded compatibility and migration route; it is not the recommended agent workflow.

An explicit import requires old writers to be drained, checks the source's protected identity and content, refuses ambiguous paths/collisions/unsafe references, and publishes validated destination material without rewriting issued facts or pending requests. It preserves the source and records its retirement for supported new clients. A marker cannot stop an old binary or an arbitrary same-user process; stopping those writers remains an operator obligation. Unknown, pending, corrupt, or interrupted states are retained for reconciliation rather than silently discarded.

The source-to-central transition must preserve credential/key references and historical signatures. Unsupported legacy combinations fail with a concrete conversion/recovery path. No automatic Namespace protocol cutover, enrollment, key regeneration, lease renewal, expected-version change, or authorization bypass accompanies initialization or migration.

## Acceptance

Validate CWD independence, profile-selection precedence and process pinning, separate destinations, case/path safety, active/pending removal guards, migration interruption and replay, stable lock contention, original protected bytes, and signed lifecycle recovery. Native Windows, OS secret storage, and actual SQL evidence retain their separate execution boundaries.
