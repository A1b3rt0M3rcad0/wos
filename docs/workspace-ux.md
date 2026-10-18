# Human workspace

The bundled workspace is served at `/app/`. Sign in with an access credential, choose an authorized Workspace and open or create an Outcome. The credential is exchanged for a secure session; shareable links contain scope and navigation, never credentials. Official product copy is English and dates label UTC. Authored titles, descriptions and external context retain their original language and values.

The default login fits compact desktop and mobile screens. Its introduction becomes smaller while the credential field, guidance and Sign in remain complete. Expanding local access or increasing text size can scroll naturally; controls are not clipped to force the page into one screen.

For the local Docker example:

```sh
cp .env.example .env
docker compose up --build -d
```

Open `http://localhost:8080/app/` and use the example bootstrap credential in `.env` for this local demonstration. Replace it before exposing an instance. The SQLite volume preserves state across restarts. [Standalone configuration](operations.md) describes production credentials and PostgreSQL.

## Start with a verifiable result

Choose **New outcome**, give it a name and describe the desired outcome. Saving persists a Draft; it never activates the Outcome implicitly. In Overview, **Review outcome** opens the existing Outcome detail with its criteria and permitted entity actions. The Overview area retains the preparation checklist for success criteria, Objectives and Tasks. Roadmaps are optional. Choose activation explicitly when preparation is ready.

An Outcome describes the desired result. Objectives describe intermediate conditions, including parent/child organization. Tasks describe bounded operational work. Completing a Task does not achieve an Objective or Outcome. Define success criteria and record their assessments before an explicit conclusion.

Choose **Add objective** or **New task** to open dedicated forms. Common creation requires no UUID, command name, JSON or fencing token. Task instructions, constraints, deliverables and scope hints use editable lines. Objective and context references use typed, authorized searches. Optional priority, scheduling and specialized settings appear under Advanced.

## Work in context

Opening an Outcome starts in its **Outcome Board**. The Outcome anchors real Objective hierarchy and Task ownership. Select a note or connection to open current detail and permitted actions alongside the map; contextual Add and Connect controls use the existing forms and authorized reference pickers. Moving a note changes only its local visual position. It does not reparent work, create a dependency or change lifecycle.

The Outcome Board keeps the working region within the available viewport height and uses the full workspace width, with no left sidebar in this view. The Outcome header retains its name and lifecycle, with a direct **Outcome Board** control and a **Views** disclosure for specialist areas. **Outcome context** exposes the retained summary and planning information. Outcome Board alone shows the global **Outcome tools** disclosure and **Choose an action** menu: sharing, refresh, criteria review and permitted Outcome-wide operations remain available there. See the [layout decisions and verification status](design/operational-canvas/canvas-first/README.md).

Choose **Views → Overview** to restore the sidebar and switch Outcomes or Workspaces, create an Outcome, or access authorized settings and tools. The sidebar remains available in Overview, Plan, Work, Issues, Evidence and Activity. Hiding it in the Outcome Board does not remove its operations or change its compact-screen disclosure behavior in those other views.

Specialist views retain their contextual collection and entity actions. Their quiet **Refresh view** control refreshes the current view without navigating to Outcome Board. Overview also retains its local Review outcome action; opening a detail still exposes that entity's permitted actions and criteria. The separation of global header controls does not remove those workflows.

Map view keeps **Zoom in**, **Zoom out**, **Center view** and **Fit view** in a floating group. Center view preserves the current zoom; Fit view shows the complete loaded map in the available area beside inspection. **Auto-layout** in the toolbar explicitly resets local note placement. List view presents the same loaded context and is the default on compact screens. Both expose selection and contextual actions without requiring dragging. Map note arrow keys navigate visible hierarchy; background arrow keys pan, `+`/`-` zoom and `0` fits. Search supports keyboard selection and authorized server results beyond the loaded map. Focus returns to an originating control when a contextual surface closes and that control remains available.

At widths of 1400 CSS px or less, or heights of 800 CSS px or less, the inspector overlays only the map/list stage, preserving the map's size and camera. Wider and taller screens dock it beside the map. Opening inspection leaves the Outcome header, Board toolbar and footer in place. Its body and long detail-action footer normally scroll independently; on phones at most 700 CSS px high, the whole inspector surface scrolls to preserve reading space. Contextual forms retain reachable submit controls. It remains nonmodal: the map is not hidden from accessibility tools and opening the inspector does not introduce a focus trap. Escape first belongs to nested controls, then closes the inspector when appropriate. Native dialogs and forms keep their existing focus, close and recovery behavior.

The map explicitly reports partial data and loaded counts. Load more retrieves bounded graph/continuity pages, including disconnected Objectives and Tasks. A cursor or selected detail must match the same Outcome revision; a concurrent change requires a refresh rather than combining different revisions. Unloaded context is not an absence of work or proof of readiness. Signed execution authority is resolved from current Task attention on selection rather than inferred from a legacy lease summary.

See [the Outcome Board design and data contract](design/operational-canvas/README.md) for relationship semantics, API boundaries, controls and source-art provenance.

## Navigate by purpose

Choose **Views** to open the existing areas as alternatives and focused collections. **Outcome Board** returns to the spatial workspace; the current specialist area is identified in the navigation. Work's **Task Board** is a Task-specific view, distinct from the Outcome Board:

| Area     | Purpose                                                                          |
| -------- | -------------------------------------------------------------------------------- |
| Overview | Desired outcome, progress with denominators, preparation and suggested next step |
| Plan     | Objectives and parent relationships, Roadmaps and published history              |
| Work     | Task list and Task Board, status filters, readiness and execution detail          |
| Issues   | Problems and their separate blocking impacts                                     |
| Evidence | Observations, deliverables, links and assessment material                        |
| Activity | Events, decisions and participation history                                      |

### Find another Outcome

In the sidebar, type into **Search outcomes** to query the authorized Outcome collection. Results update as you type; Enter starts the current search immediately. Open **Filters** to change lifecycle or archive scope; those changes apply immediately. On compact screens, **Switch outcome** expands this navigation.

**More outcomes** loads the next bounded page for the current search and filters. A new query resets pagination, and later responses from an older query cannot replace the current results. This is server-backed discovery, independent of which Outcomes were already loaded. Losing authorization clears private Outcome results rather than retaining an accessible stale list.

Deep links restore Workspace, Outcome, area, collection and detail after signing in, reload and browser Back/Forward. Outcome Board links retain their selected entity; camera movement and local note placement are not domain state. Existing `canvas` route values remain compatible. Copy link does not grant access. Server authorization is checked when resolving shared context.

**All Tasks** list search covers the selected Outcome on the server, including title and priority filtering with pagination. Outcome discovery also searches the server. The Task Board, Outcome Board alternative list and other loaded-collection filters explicitly identify their loaded scope; they do not promise a complete Outcome collection. Searchable reference fields retrieve the authorized candidate set rather than relying on loaded collections.

## Choose references reliably

Type into a reference field to search eligible types in the current scope. Results include type, title, state, context and short ID to distinguish duplicate titles. Arrow keys and Enter select; Escape closes suggestions. Load more retrieves the next bounded page. Selected IDs are resolved independently of the visible search page. If the Outcome changes and a cursor expires, Reload results starts a fresh search while preserving selected references. Revocation invalidates access and cached material.

## Plan without changing execution

Create a Roadmap, edit its Draft, add phases, milestones and Objective/Task references, arrange their hierarchy and order, then review the diff and publication reason. **Publish revision** freezes a historical revision. **Activate revision** is a separate action. Removing a reference or moving a phase does not cancel or complete operational work. Dependencies are explicit actions on operational entities, never an implicit consequence of Roadmap order.

View a Draft or a particular published revision on the Outcome Board without activating it. Planning nodes and explicit plan order remain an optional overlay; they do not replace Objective hierarchy, Task ownership or hard/advisory dependency relations. Viewing a revision does not make it the active plan. Replacing an overlay leaves operational relationships intact.

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

The [Outcome Board guide](design/operational-canvas/README.md#acceptance-boundary) records the map and action contracts. [Viewport-workspace verification](design/operational-canvas/canvas-first/README.md#verification-and-capture-provenance) tracks the current shell and compact inspector separately from the [earlier automated verification](design/operational-canvas/validation.md). Earlier workspace behavior and captures are recorded in [the historical acceptance record](ux/final-acceptance.md), [experience contracts](ux/experience-contract.md), [architecture decision](adr/0027-human-workspace-client.md) and [editorial gallery](design/editorial/README.md#main-pages).

The earlier real-server and automated accessibility results cover their recorded baseline. They do not qualify the sidebar-free Outcome Board, its inspector or its keyboard/mobile behavior. Equivalent viewport reflow and device scale factors are distinct from native browser zoom; the follow-up records those checks independently.

The [current density gallery](design/editorial/density/README.md) records stable
operational sizes, compact navigation and readable long-record inspection.
The earlier [35-image gallery](design/operational-canvas/canvas-first/README.md#screenshots)
and its 44 unit/99 browser checks remain historical qualification at `0d84588`.
Each gallery identifies its own source, capture method and verification scope.

Representative-user usability, human comprehension/timing, manual screen-reader checks and other-browser acceptance remain pending. Native Chromium page zoom at 125% and 150% is tracked in the viewport-workspace qualification rather than inferred from smaller screenshot dimensions. The owner confirmed that no representative users or screen-reader testers are available. Automated acceptance is not a usability measurement or WCAG certification.
