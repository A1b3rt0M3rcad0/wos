# Operational workspace contracts

The Outcome Board and specialist views use the same Core commands and coherent
read models. HTTP, MCP, the Go SDK and the CLI do not have a privileged dashboard
write path. These additive contracts describe this source branch; check the
installed binary's `version`, `catalog` and server `capabilities` before using
them with a published package.

Execution remains external. A documentary note, Evidence, declared assignee or
received delivery does not grant execution authority, assess a Criterion or
achieve an Outcome. Signed execution and independent review retain their
authorized profile and recovery workflows.

## Read the workspace

Let `O` mean `/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}`.
All reads enforce current scoped authorization and return `Cache-Control:
no-store`. The two POST operations are read-only, accept no mutation idempotency
key and perform no domain mutation. Browser session requests still pass the
existing same-origin protection; read-only POST is not a CSRF exemption. Bearer
clients and MCP use their existing authentication contracts.

| HTTP | MCP tool | Go SDK method | Result |
| --- | --- | --- | --- |
| `POST O/workspace-view` | `wos_get_workspace_view` | `GetEntityWorkspaceView` | One selected subject, placement, pertinent facets, current capabilities and coverage |
| `POST O/workspace-cards` | `wos_get_workspace_cards` | `GetWorkspaceCards` | Compact summaries at one Outcome revision |
| `GET O/workspace-timeline` | `wos_get_workspace_timeline` | `GetWorkspaceTimeline` | Recent facts with a frozen historical frontier |
| `GET O/entity-notes/{note_id}` | `wos_get_entity_note` | `GetEntityNote` | Complete immutable note and recorded authorship |

MCP view/card arguments are `{scope, query}`. Timeline arguments are
`{scope, subject_ref?, criterion_id?, include_descendants?, limit?, cursor?}`;
note arguments are `{scope, note_id}`. Scope always contains the actual
`namespace_id` and `outcome_id` and is checked against every selected reference.

For `workspace-view`, the body is a `WorkspaceViewQuery`:

```json
{
  "subject": {
    "kind": "entity",
    "ref": {
      "namespace_id": "0199ec00-0000-7000-8000-000000000001",
      "outcome_id": "0199ec00-0000-7000-8000-000000000002",
      "kind": "work_item",
      "id": "0199ec00-0000-7000-8000-000000000003"
    }
  },
  "limit": 20
}
```

Replace example IDs with actual authorized records. The subject discriminator
also accepts `criterion` with its real owner reference and `criterion_id`,
`roadmap_revision` with a Roadmap reference and quoted `revision_number`, or
`execution_record` with a Task reference, `record_kind` and `record_id`.
Execution record kinds are `contract`, `submission` and `review_case`.
Criterion is not a new globally writable entity kind.

Follow the relevant facet's `next_cursor` using `criteria_cursor`,
`relation_cursor` or `required_objectives_cursor`. A facet reports `complete`,
`partial`, `not_loaded`, `unavailable`, `not_applicable` or `unknown`; counts are present
only when known. A missing count is not zero. A capability explains availability,
related references and the legitimate channel; the command always revalidates
current state.

The selected record, applicable Criteria, people, delivery, current conditions
and recent activity are assembled inside one authorized database snapshot with
one `outcome_revision` and `evaluated_at`. Reading several older resources and
combining equal-looking revisions is not this coherence contract. Facet cursors
bind the subject and snapshot revision; a changed revision requires an explicit
reload rather than mixing pages from different current states. Immutable
historical material retains its identity without becoming current authority.

Outcome views expose `required_objectives` across the whole Outcome, including
subobjectives and branches outside the currently visible Board. The
`required_objectives` coverage entry reports the known total, returned page and
continuation. Send `required_objectives_cursor` to read the next page. Optional
Objectives do not become acceptance gates because of their position or because
their Tasks are complete.

Outcome and Objective views include `conclusion_impact`: current open-Objective
and open-Task counts, a sample of at most 20 named items, and the consequences
of closing that scope. `coverage.conclusion_impact` distinguishes the sample
from the complete counts. Reviewing this information does not change a child
lifecycle or satisfy a Criterion.

Two optional read queries provide prospective impact in the same authorized
frame. For a Task or Objective, add
`"placement_preview": {"parent_id": "OBJECTIVE_UUID"}`; explicit `null`
selects the Outcome as parent. An empty preview object is invalid. The response
names both parents, reports affected counts, and explains blocked moves with
typed reasons and remedies. It samples at most 20 affected entities and five
Blockers per before/after state, with coverage for each sample. Task previews
also expose current and prospective readiness. An invalid or cyclic destination
returns its real typed error with unknown prospective coverage; it does not
invent an empty Blocker set or ready state. The existing structural graph
bounds remain enforced.

For a permitted Outcome, Objective or Task target, add
`"blocker_preview": {"propagation": "subtree"}` or `"direct"` according
to `workspace_rules`. The response identifies the target, counts affected open
work and samples at most 20 named items with explicit coverage. Reading either
preview performs no mutation. Omitting these query fields keeps them unloaded.

When confirming the reviewed conclusion, send the view's quoted
`outcome_revision` as optional `observed_outcome_revision` on
`achieve_outcome` or `achieve_objective`. The same optional field binds
`report_issue_with_blocker` to its reviewed scope. Existing `expected_version`
fields on the conclusion commands keep their legacy numeric encoding. Changed
scope returns a typed `conclusion_impact_changed` or `blocker_impact_changed`
precondition and requires another review. Existing clients may omit the new
field and retain their previous contract. Structural setters instead use their
existing quoted `expected_outcome_revision` to bind a placement preview.

For `workspace-cards`, supply either `refs` (at most 100 references) or one
`parent_ref` plus optional `limit` and `cursor`. Parent mode reads direct
structural children; reaching the end of that page does not imply the entire
Outcome graph was loaded. Default page size is 50, maximum 100. Summaries omit
full inspectors, history and signed payloads. Unknown execution authority is
reported as unknown, never filled from an assignee.

Timeline parameters are `kind` and `entity_id` together, optional `criterion_id`,
`include_descendants`, `limit` and `cursor`. With no subject, the Outcome is
selected. Default and maximum page size is 20. A cursor freezes the historical
frontier; it also binds the authorized subject and filters. New facts do not
enter older pages. Structural changes affecting descendant membership require
an explicit restart. Entries retain their actual origin and subject, distinguish
recording time from observation time, and use stable revision/event-index
ordering. Previews are at most 512 bytes; fetch a note by its ID for its full
body. The existing `/timeline` endpoint keeps its previous contract.

HTTP request and workspace response bounds are 256 KiB. Larger facets must use
their declared pages and coverage. No read fetches an Artifact or Evidence URL.

## Create a plan atomically

All mutations use `POST /api/v1/commands/{command_name}` with
`{"command": ...}` and an `Idempotency-Key`. MCP uses
`wos_{command_name}` with `{idempotency_key, command}`. Both call the same
Application handler. The Go SDK exposes the corresponding typed methods.

This `CreatePlanItems` command creates a new Objective and its planned Task in
one transaction:

```json
{
  "scope": {
    "namespace_id": "0199ec00-0000-7000-8000-000000000001",
    "outcome_id": "0199ec00-0000-7000-8000-000000000002"
  },
  "items": [
    {
      "local_key": "installation",
      "kind": "objective",
      "title": "Installation instructions can be followed",
      "required_for_outcome": true,
      "initial_criteria": [
        {
          "title": "A reviewer follows the documented installation steps",
          "required": true,
          "verification_mode": "evidence_review"
        }
      ]
    },
    {
      "local_key": "write-instructions",
      "kind": "work_item",
      "parent_local_key": "installation",
      "title": "Write the installation instructions"
    }
  ]
}
```

Send this object as `command` to `create_plan_items`. The response preserves the
`local_key` to canonical reference mapping and created versions on replay.
If a complete result exceeds the response bound, `result_omitted=true` retains
`created_refs` and compact `created_items` with versions and Criterion IDs.
Compound receipts retain the created record/link IDs. Replay and receipt lookup
preserve those identities without copying authored descriptions into the receipt;
the typed Go SDK exposes the same compact fields.
Tasks are created in `backlog`; no lease or contract is acquired. The batch
contains only new Objectives and Tasks in one Outcome, at most 50 items and 100
initial Criteria, subject to existing byte and domain limits. Invalid parents,
duplicate keys, cycles or an invalid final entry roll back the whole operation.
Permission is `planning:write` plus `outcome:write` and/or `work:write` for the
kinds actually created.

Existing `create_outcome`, `create_objective` and `create_work_item` commands
also accept optional `initial_criteria`. Older calls that omit this field retain
their previous defaults. The human quick-create flow explicitly selects draft
Outcome, required Objective and backlog Task defaults; these choices do not
rewrite existing records or silently change older clients.

## Edit deferred details and structure

| Command | Intent |
| --- | --- |
| `set_work_item_execution_spec` | Replace or explicitly clear instructions, constraints, deliverables and execution context |
| `set_work_item_not_before` | Set or explicitly clear scheduling time; this never starts execution |
| `set_work_item_objective` | Reassociate a Task with an Objective, or directly with its Outcome |
| `set_objective_parent` | Reparent a planned eligible Objective without changing identity or historical records |
| Existing `set_outcome_owners`, `set_objective_owners`, `set_work_item_assignees` | Replace declared participants; this grants no lease or contract |

The four new setters use quoted decimal `expected_version`. Their edited field
must be present; `null` explicitly clears it. Omission is rejected. For example:

```json
{
  "scope": {
    "namespace_id": "0199ec00-0000-7000-8000-000000000001",
    "outcome_id": "0199ec00-0000-7000-8000-000000000002"
  },
  "work_item_id": "0199ec00-0000-7000-8000-000000000003",
  "expected_version": "4",
  "expected_outcome_revision": "12",
  "objective_id": null
}
```

The optional quoted `expected_outcome_revision` on structural setters detects a
stale impact preview. Versions, lifecycle, execution/review freezes, scope and
dependencies are checked under the Outcome guard. Reparenting does not change
Criteria, Evidence, required-for-Outcome status or completion. Free card dragging
continues to change local visual position only.

## Record contextual information

`append_entity_note` accepts `scope`, `subject_ref`, `type`, `body` and optional
`criterion_id`, `occurred_at`, Artifact/Evidence references and
`supersedes_note_id`. Documentary types are `update`, `discovery` and
`observation`; body limit is 16 KiB, with at most 50 references of each kind.
The server records the authenticated Principal, authorized Actor, recording time
and commit revision. A correction appends a new note for the same subject and
Principal; it never edits the earlier body. Notes may document terminal,
non-archived work. They do not reopen it, create an Issue, complete a Task or
assess a Criterion. Permission is `records:write`.

`register_evidence_with_links` atomically registers new Evidence, optional
Artifact or verified internal observation, and explicit links. It can instead
reuse `existing_evidence_id` without copying documentary content. Each link
contains `target_ref`, visible `stance` and the user's `rationale`; a
Criterion-specific link also supplies observed `criterion_id`, quoted
`criterion_revision` and quoted `target_expected_version`. A failed link rolls
back all records from the intention. A human observation remains a participant's
declaration, with actual provenance; the server does not claim to have run tests.

When an Evidence source identifies an internal observation, its coherent
workspace view includes an `observation` facet. `status=resolved` exposes the
actual note ID, subject and optional Criterion, recording Principal, authorized
Actor, recording/observation times, correction reference and a bounded 512-byte
preview. It establishes documentary provenance, not truth or acceptance. Fetch
the note resource for its complete text. An older unresolved source reports
`status=unavailable` and `reason_code`; no author or date is invented. External
sources have no internal-observation facet, and their URLs are never fetched by
this read.

`register_artifact_in_context` registers material and its typed Relation in one
transaction. `propose_decision_in_context` records a proposal and contextual
Relation. These operations do not submit a signed delivery or release a Blocker.
Existing `report_issue_with_blocker` and `resolve_issue_and_blockers` remain the
explicit problem/impact operations; resolving an Issue alone does not release
all associated Blockers.

`GET /api/v1/capabilities` publishes `workspace_rules`, derived from the same
canonical rules used by mutations. `evidence_link_target_kinds` permits Outcome,
Objective, Task, Issue and Decision, excluding Blocker. `blocker_propagation`
permits Outcome subtree, Objective direct/subtree and Task direct. These are
type eligibility rules, not permission or current-state authority.

## Use the CLI and recover safely

The selected, verified Profile pins server, Namespace, Principal and credential.
The new `planning` surface permits an explicit allowlist of planning/documentary
commands, including the operations above and declared participant setters.
Signed acquisition, execution, review and administrative override cannot be
substituted through this surface.

Save the inner command object from the atomic example as `plan-command.json`:

```sh
wos --profile planner --outcome OUTCOME_ID planning apply --command create_plan_items --file plan-command.json --output json
wos --profile planner --outcome OUTCOME_ID planning recover --output json
```

Input files are explicit, bounded JSON or YAML data. They do not choose another
server, Namespace, authenticated Principal or idempotency key. The CLI writes its
durable intention before sending. If the response is lost, editing the file does
not alter that intention: run `planning recover` with the same Profile and
Outcome. Recovery repeats the original bytes and key, or returns the retained
local receipt. New planning intentions in that Outcome and Profile removal remain
blocked while recovery is pending. A definitive rejection allows a separately reviewed new intention;
no expected version is rebased automatically. Profile credential identity scopes
the local journal, so another identity cannot recover its intentions as their own.

Read files contain the corresponding query body, without HTTP/MCP wrappers:

```sh
wos --profile planner --outcome OUTCOME_ID inspect get --file view-query.json --output json
wos --profile planner --outcome OUTCOME_ID inspect cards --file card-query.json --output json
wos --profile planner --outcome OUTCOME_ID inspect timeline --file timeline-query.json --output json
wos --profile planner --outcome OUTCOME_ID inspect note NOTE_ID --output json
```

A timeline query file accepts `subject_ref`, `criterion_id`,
`include_descendants`, `limit` and `cursor`. A view/card file matches its HTTP POST
body. Explicit schema-1 workspaces can use the same `inspect` and `planning`
groups; existing legacy commands retain their contracts.

## Precision, errors and compatibility

New workspace projections preserve every version, revision and fencing integer
as an exact decimal string, including nested records. Never convert them through
JavaScript `Number`. The typed Go SDK decodes these projections without losing
precision. Existing resources, commands and signed protocols retain their
established numeric or quoted formats; the new read format does not change an
older endpoint's wire representation.

`precondition_failed` describes a domain condition, not an editing conflict.
Typed `details` can contain `reason_code`, `subject_ref`, `field_paths`,
`related_refs` and `remedies` with `action_id`, `subject_ref` and an optional
`criterion_id`. For example, `missing_required_criterion` can point to
`add_criterion`. HTTP, MCP and Go SDK preserve those details; CLI JSON adds
`error_code` and `error_details` while retaining its existing `error` text.
`version_conflict` continues to require reading current state and making an
explicit reconciliation decision. A capability or preview is never authorization
to force a later commit.

The executable command catalog, [OpenAPI](../packages/wos-api/openapi.yaml),
[generated schemas](../packages/wos-api/commands.openapi.json) and
[exposure inventory](ux/command-exposure-inventory.md) describe transport details.
See [workspace relationships](workspace-relationships.md) for operational meaning
and [signed profiles](signed-workspace-profiles.md) for execution authority.

## Contextual Kanbans and strict completion (unreleased)

`workspace-cards` now includes an additive `stage` projection: `todo`,
`in_progress`, `verification`, `accepted`, or `cancelled`. It never replaces
`lifecycle`. Actual pending review is required for the verification stage;
terminal completion/cancellation wins over stale review indicators. Child pages
retain their authenticated, revision-bound cursor and explicit coverage. Counts
in the browser's contextual Kanban describe loaded cards, not global totals.

`workspace-view` accepts optional `include_activity: false` to defer the activity
facet. Omission preserves the existing recent-history read. Deferred coverage is
`not_loaded`, never an empty complete history.

`create_outcome` accepts optional `completion_policy`: omitted/`compatible`
preserves existing completion behavior; `strict_work` additionally checks the
complete applicable work set. Human creation visibly defaults to strict work.
Existing Outcomes are not migrated to strict behavior, and policy editing is not
exposed. SQLite and PostgreSQL require migration 0033.

For strict completion, `achieve_outcome` and `achieve_objective` accept
`acknowledged_cancelled_work`, an exact set of references from the current
`conclusion_impact.cancelled_items` preview. A nonempty acknowledgement requires
`observed_outcome_revision`; duplicate, unrelated or stale acknowledgements fail.
The immutable Conclusion stores the set in
`obligations.acknowledged_cancelled_work`, with the existing reason/author/time.
A capability with `confirmation_kind: cancelled_scope_exclusion` allows opening
the confirmation UI; the command still revalidates the acknowledgement and all
acceptance rules. See [ADR-039](adr/0039-contextual-kanbans-and-strict-completion.md).

HTTP command dispatch, MCP, CLI and the Go SDK share these Application command
fields. No new writable Kanban entity, agent execution or file upload is added.
