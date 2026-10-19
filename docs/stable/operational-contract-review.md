# Operational candidate compatibility review

Baseline is the actually published [v0.4.2](https://github.com/A1b3rt0M3rcad0/wos/releases/tag/v0.4.2), source b572269085494ddd4d6370716ff1cb360b46065f, published 2026-10-13. The operational changes are unreleased and queued as a minor addition; VERSION is advanced only by the existing release preparer. This PR neither republishes 0.4.2 nor certifies/publishes 1.0.0.

## Reviewed public changes

- New command resolve_signed_return_intent, matching HTTP/MCP/Go SDK/CLI paths and optional capability. Its immutable negative receipt closes only the original rejected return; it cannot discard an accepted fact or bypass current grants, credential/key revocation, issuer pin, protocol/contract binding, CAS or fencing. New signed_intent_closed is a typed conflict for that terminal intent. Original accepted receipt/replay semantics remain intact. ADR 034 defines the authority boundary.
- New authenticated Namespace-administrative delivery-health read with optional Outcome scope, plus process-only worker diagnostics. It exposes aggregate queue/age/attempt metadata without bodies, URLs or secrets. Degraded/disabled delivery is separate from API readiness and external receiver availability. ADR 031 retains at-least-once receiver dedupe.
- Optional Namespace policy max_active_work_per_principal/max_active_review_per_principal, 1..100, retain absent defaults 3/1. CID and Namespace limits remain independent ceilings. Signing-identity response retains existing JSON fields and adds own-Principal quota diagnostics. Its former generic OpenAPI response is now accurately typed; route/query/status contracts remain. ADR 033 forbids tenant-supplied scope escalation.
- Workspace schema 2 retains its existing required fields and adds optional nested pending return_resolution. Its frozen envelope/receipt are separate from the original intention. New DSSE return-resolution and return-resolution-receipt purposes join the generic payload-type schema, matching the already verified codec. Issued fields/signatures, decimal counters, old pending MAC inputs, workspace schema 1 and existing contract/profile files are retained. An old strict CLI may refuse a new-feature pending resolution; it must not clear/modify it. Use the updated CLI to reconcile it. Reverse execution of a new feature by an old CLI is not promised.
- Scoped continuity/focal/graph read ports are optional Go adapter extensions; old third-party adapters retain explicit fallback behavior and are not performance-qualified by SQL measurements. Complete DTO/golden/legacy cursor, coherent authorization/revision/evaluation-time and current-proof semantics remain required. ADR 032 records query choices.
- Schema migrations 0027–0029 are additive and previous 0001–0026 remain unchanged. Older binaries must refuse the newer schema. Rollback uses a pre-upgrade backup in a separate database; it does not run old writers against new facts.

## Frozen references and verification

`contracts/v0.4.2/` contains exact baseline OpenAPI/profile/contract schema blobs and the historical fingerprint manifest. Its 0.4.0 origin label is kept as historical metadata. The current public-contract.json freezes the reviewed candidate; changing any listed schema still fails the strict exact-hash gate. A separate compatibility test binds the original manifest hash, verifies released schema hashes, rejects removed old properties/enum values/new required inputs/changed old constraints, preserves every old command route and separately reviews the signing-identity response annotation. Current signing/YAML/transport/recovery tests remain necessary; schema equality alone cannot certify authority.

The review found and fixed the generic DSSE enum omitting the new resolution purposes. A codec/schema property regression now compares validity for every existing/new purpose and unknown input. No previous public field, command, package identity or required input was renamed. npm remains wosctl; all public operations remain wos <command>, including skill installation. Apache-2.0 is unchanged. Native Linux/Windows and complete source-bound gates are still required before technical acceptance.

## Central agent workflow review — 2026-10-16

Compared with source `80074f470d44a55c33c2ea297e509f9e14871d96`, generated OpenAPI adds three schemas (`Catalog_append_work_activity`, `WorkActivityPage`, `WorkAttention`) and three routes (append, scoped history, attention). Existing signed-state query/result schemas gain only optional execution-header discovery fields. The existing signed-state route adds exactly five optional query parameters: holder, WorkItem, persisted status, cursor, and bounded limit. All original path parameters, responses, annotations, required fields and other routes are retained. The compatibility test checks that exact reviewed parameter addition while continuing to compare every original route property.

The local project schema makes only `default_outcome_id` optional. Namespace, server pins and every supplied Outcome ID retain their validation; a missing default requires explicit or uniquely discovered current scope. Profile/issued-contract schemas, signed purposes, old canonical bytes and the released reference manifest remain unchanged. The central installation index is separate, versioned local metadata. Explicit legacy workspace selection retains the earlier binding path; the default initialization workflow intentionally changes to installation scope and requires the current binary.

Migration `0030` adds immutable activity storage and scoped indexed chronology without rewriting earlier migrations. Activity is documentary and authorized under the live signed holder; it cannot renew authority or evaluate a criterion. Attention checks exact current submission bindings and distinguishes terminal state. The five new query parameters grant no additional read authority. See ADRs 035/036, authority/replay/restore and HTTP/MCP/SDK tests, and the [source-bound central workflow validation record](../central-agent-workflow/validation.md).

Only the current OpenAPI and project-schema fingerprints were refreshed after this semantic review. All seven other frozen schema digests and the byte-preserved released baseline remain unchanged. This records reviewed compatibility, not release publication or native-platform certification.

## Operational workspace review — 2026-10-19

Compared with the current branch base, this candidate adds nine mutation routes
and four bounded read routes. It removes or changes no existing route. The new
commands cover immutable Entity notes, atomic planning and documentary creation,
and typed Task/Objective setters. The new reads expose an authorized coherent
workspace snapshot, compact cards, a documentary timeline and full Entity notes.
All transports call the same Core methods; new capabilities grant no implicit
execution or acceptance authority. The [workspace API guide](../operational-workspace-api.md)
records the actual HTTP, MCP, SDK and CLI contracts.

Existing create commands gain only optional `initial_criteria`; the existing
Evidence-link command gains only optional observed target-version and
criterion-revision fields. Its previously unconstrained target-kind wire schema
is preserved for released clients. Canonical eligibility remains enforced by the
Core and exposed as `workspace_rules`; the new compound Evidence-link input uses
the canonical enum. The existing Work attention projection gains only optional
criterion diagnostic reason codes. No old required field, constraint, enum value,
command path, numeric counter format or signed canonical representation changes.

New workspace counters are quoted decimal strings, including optimistic
preconditions. Explicitly required nullable setter fields distinguish an omitted
field from an intentional clear. Typed errors add optional structured details.
New bounded mutation receipts preserve every generated identifier when a full
result exceeds the response budget, so uncertain-response recovery never needs
another mutation. CLI planning journals reuse frozen durable intentions and
receipt lookup; they cannot invoke signed execution through an unsigned fallback.

New workspace projections include revision-bound criteria, relationship and
required-Objective cursors, honest partial coverage, and actual observation
provenance. An unavailable internal note stays unavailable; external URLs are
never fetched. These fields and commands are additive, rather than a claim that
an older client can operate the new features.

The current OpenAPI fingerprint alone was refreshed after inspecting all added
routes/schemas and rerunning the recursive released-v0.4.2 compatibility guard.
The eight CLI schema fingerprints and byte-preserved released reference remain
unchanged. Two consecutive MCP/catalog/SDK/OpenAPI generations produced identical
output hashes. This review records wire compatibility; the separate PostgreSQL,
SQLite, transport, recovery, browser and upgrade qualification evidence still
controls acceptance and does not imply release publication.


## Reviewed impact previews — 2026-10-19

The conclusion and relationship review adds no route and changes no existing
property constraint. The existing `achieve_outcome`, `achieve_objective` and
`report_issue_with_blocker` commands gain only optional, quoted-decimal
`observed_outcome_revision`. Omitting it preserves the old Core intention bytes
and the existing numeric `expected_version` encoding. A supplied revision is
checked under the Outcome guard before mutation; changed scope returns typed
`conclusion_impact_changed` or `blocker_impact_changed` and requires an explicit
review. It grants no authority or automatic child conclusion.

The workspace response adds bounded `conclusion_impact`, `placement_preview` and
`blocker_preview` facets. The existing query adds only optional opt-in preview
objects. Placement requires an explicit nullable `parent_id`; an omitted key is
invalid, while `null` deliberately selects the Outcome. Both previews use the
same authorized revision and evaluation time as the inspector, expose honest
coverage and remain advisory reads. Structural setters retain their existing
`expected_outcome_revision` precondition. Existing read routes, signed protocols
and execution permissions are unchanged.

Review against the preceding candidate found additions to five existing schema
definitions and no changed old route, required input, constraint or field representation.
The current OpenAPI digest alone was refreshed; the eight CLI schema digests
and byte-preserved released reference remain unchanged. Tests exercise old
omission, exact revisions above JavaScript's integer precision, numeric rejection
for the new counters, and explicit nullable query binding. Two consecutive
MCP/catalog/SDK/OpenAPI generations are deterministic; recursive released-wire
compatibility and the release metadata tests remain required independently.
