# Human workspace

The bundled workspace is served at `/app/`. Sign in with an access credential, choose an authorized Workspace and open or create an Outcome. The credential is exchanged for a secure session; shareable links contain scope and navigation, never credentials. Official product copy is English and dates label UTC. Authored titles, descriptions and external context retain their original language and values.

For the local Docker example:

```sh
cp .env.example .env
docker compose up --build -d
```

Open `http://localhost:8080/app/` and use the example bootstrap credential in `.env` for this local demonstration. Replace it before exposing an instance. The SQLite volume preserves state across restarts. [Standalone configuration](operations.md) describes production credentials and PostgreSQL.

## Start with a verifiable result

Choose **New outcome**, give it a name and describe the desired outcome. Saving persists a Draft; it never activates the Outcome implicitly. Use **Review outcome** to inspect criteria and choose permitted actions. The Overview area retains the preparation checklist for success criteria, Objectives and Tasks. Roadmaps are optional. Choose activation explicitly when preparation is ready.

An Outcome describes the desired result. Objectives describe intermediate conditions, including parent/child organization. Tasks describe bounded operational work. Completing a Task does not achieve an Objective or Outcome. Define success criteria and record their assessments before an explicit conclusion.

Choose **Add objective** or **New task** to open dedicated forms. Common creation requires no UUID, command name, JSON or fencing token. Task instructions, constraints, deliverables and scope hints use editable lines. Objective and context references use typed, authorized searches. Optional priority, scheduling and specialized settings appear under Advanced.

## Work in context

Opening an Outcome starts in its operational canvas. The Outcome anchors real Objective hierarchy and Task ownership. Select a note or connection to open current detail and permitted actions alongside the map; contextual Add and Connect controls use the existing forms and authorized reference pickers. Moving a note changes only its local visual position. It does not reparent work, create a dependency or change lifecycle.

Map view provides pan, zoom, fit, auto-layout and progressive expansion. List view presents the same loaded context and is the default on compact screens. Both expose selection and contextual actions without requiring dragging. Map note arrow keys navigate visible hierarchy; background arrow keys pan, `+`/`-` zoom and `0` fits. Search supports keyboard selection and authorized server results beyond the loaded map. Focus returns to an originating control when a contextual surface closes and that control remains available.

The map explicitly reports partial data and loaded counts. Load more retrieves bounded graph/continuity pages, including disconnected Objectives and Tasks. A cursor or selected detail must match the same Outcome revision; a concurrent change requires a refresh rather than combining different revisions. Unloaded context is not an absence of work or proof of readiness. Signed execution authority is resolved from current Task attention on selection rather than inferred from a legacy lease summary.

See [the canvas design and data contract](design/operational-canvas/README.md) for relationship semantics, API boundaries, controls and source-art provenance.

## Navigate by purpose

The existing areas remain available as alternatives and focused collections:

| Area     | Purpose                                                                          |
| -------- | -------------------------------------------------------------------------------- |
| Overview | Desired outcome, progress with denominators, preparation and suggested next step |
| Plan     | Objectives and parent relationships, Roadmaps and published history              |
| Work     | Task list/board, status filters, readiness and execution detail                  |
| Issues   | Problems and their separate blocking impacts                                     |
| Evidence | Observations, deliverables, links and assessment material                        |
| Activity | Events, decisions and participation history                                      |

Deep links restore Workspace, Outcome, area, collection and detail after signing in, reload and browser Back/Forward. Canvas links retain their selected entity; camera movement and local note placement are not domain state. Copy link does not grant access. Server authorization is checked when resolving shared context.

**All Tasks** list search covers the selected Outcome on the server, including title and priority filtering with pagination. Outcome discovery also searches the server. The Board, canvas alternative list and other loaded-collection filters explicitly identify their loaded scope; they do not promise a complete Outcome collection. Searchable reference fields retrieve the authorized candidate set rather than relying on loaded collections.

## Choose references reliably

Type into a reference field to search eligible types in the current scope. Results include type, title, state, context and short ID to distinguish duplicate titles. Arrow keys and Enter select; Escape closes suggestions. Load more retrieves the next bounded page. Selected IDs are resolved independently of the visible search page. If the Outcome changes and a cursor expires, Reload results starts a fresh search while preserving selected references. Revocation invalidates access and cached material.

## Plan without changing execution

Create a Roadmap, edit its Draft, add phases, milestones and Objective/Task references, arrange their hierarchy and order, then review the diff and publication reason. **Publish revision** freezes a historical revision. **Activate revision** is a separate action. Removing a reference or moving a phase does not cancel or complete operational work. Dependencies are explicit actions on operational entities, never an implicit consequence of Roadmap order.

View a Draft or a particular published revision on the canvas without activating it. Planning nodes and explicit plan order remain an optional overlay; they do not replace Objective hierarchy, Task ownership or hard/advisory dependency relations. Viewing a revision does not make it the active plan. Replacing an overlay leaves operational relationships intact.

## Verify and resolve explicitly

Register Evidence with provenance, then assess a specific success criterion and its current definition revision. Evidence alone does not certify success; Waived is explicitly not verified and requires permission and rationale. Evidence links express a stance toward a target. Artifacts record deliverables. Outcome, Objective and Task conclusions remain separate.

Report an Issue to record a problem. Mark an item blocked to record its blocking impact; a combined reporting flow makes both effects explicit. Resolve issue and Release blocker are independent decisions. The explicit combined resolution requires choosing the blocking impacts to release; resolving an Issue by itself never releases them.

## Execute with the right authority

Task detail shows readiness, current execution state, accepted/submitted material, expiry, review state and permitted next actions. Suggestions are presentation hints from persisted state; WOS does not plan or execute work autonomously.

Legacy reservations are available only under the matching protocol and current lease rules. Unsigned contract actions require their actual permission and accepted material. Under `signed_contracts_v2`, contextual copy controls hand execution or independent review commands to an authorized `wosctl` host profile. A pending review offers reviewer handoff. The browser stores no signing keys and cannot sign, accept or decide a review on the host's behalf. Use [signed host profiles](signed-workspace-profiles.md) for provisioning and operational recovery.

Workspace settings expose only authorized operations, including typed permissions. Advanced participant assignment and external key/value context grant no lease, authentication or execution authority. The technical schema/command console is confined to permission-gated Developer tools.

## Save and recover safely

Changes preserve local inputs on errors. If the item version changed, compare local edits with the current item and explicitly confirm a new intent or discard the edit. Execution/assessment authority is never silently rebased.

An uncertain response freezes the original payload and idempotency key. Retry sends the same bytes and key in the same authenticated scope; changing identity, scope or losing authorization prevents dispatch. A successful write followed by a failed refresh is not presented as a new write intent. Late responses and failures from an obsolete context cannot replace the current view.

## Verification and limits

The [canvas guide](design/operational-canvas/README.md#acceptance-boundary) records the new implementation and [automated verification](design/operational-canvas/validation.md). Earlier workspace behavior and captures are recorded separately in [the historical acceptance record](ux/final-acceptance.md), [experience contracts](ux/experience-contract.md), [architecture decision](adr/0027-human-workspace-client.md) and [editorial gallery](design/editorial/README.md#main-pages).

The earlier real-server and automated accessibility results cover their recorded baseline. They do not qualify the new canvas, its inspector or its keyboard/mobile behavior. Equivalent 200%/400% viewport reflow is distinct from native browser zoom.

Representative-user usability, human comprehension/timing, manual screen-reader checks and native browser zoom remain pending. The owner confirmed that no representative users or screen-reader testers are available. Automated acceptance is not a usability measurement or WCAG certification.
