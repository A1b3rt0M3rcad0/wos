# Human workspace

The bundled workspace is served at `/app/`. Sign in with an access credential, choose an authorized Workspace and open or create an Outcome. The credential is exchanged for a secure session; shareable links contain scope and navigation, never credentials. Official product copy is English and dates label UTC. Authored titles, descriptions and external context retain their original language and values.

For the local Docker example:

```sh
cp .env.example .env
docker compose up --build -d
```

Open `http://localhost:8080/app/` and use the example bootstrap credential in `.env` for this local demonstration. Replace it before exposing an instance. The SQLite volume preserves state across restarts. [Standalone configuration](operations.md) describes production credentials and PostgreSQL.

## Start with a verifiable result

Choose **New outcome**, give it a name and describe the desired outcome. Saving persists a Draft; it never activates the Outcome implicitly. The preparation checklist links to success criteria, Objectives and Tasks. Roadmaps are optional. Choose activation explicitly when preparation is ready.

An Outcome describes the desired result. Objectives describe intermediate conditions, including parent/child organization. Tasks describe bounded operational work. Completing a Task does not achieve an Objective or Outcome. Define success criteria and record their assessments before an explicit conclusion.

Choose **Add objective** or **New task** to open dedicated forms. Common creation requires no UUID, command name, JSON or fencing token. Task instructions, constraints, deliverables and scope hints use editable lines. Objective and context references use typed, authorized searches. Optional priority, scheduling and specialized settings appear under Advanced.

## Navigate by purpose

| Area | Purpose |
| --- | --- |
| Overview | Desired outcome, progress with denominators, preparation and suggested next step |
| Plan | Objectives and parent relationships, Roadmaps and published history |
| Work | Task list/board, status filters, readiness and execution detail |
| Issues | Problems and their separate blocking impacts |
| Evidence | Observations, deliverables, links and assessment material |
| Activity | Events, decisions and participation history |

Deep links restore Workspace, Outcome, area, collection and detail after signing in, reload and browser Back/Forward. Copy link does not grant access. Server authorization is checked when resolving shared context.

**All Tasks** list search covers the selected Outcome on the server, including title and priority filtering with pagination. Outcome discovery also searches the server. The Board and other loaded-collection filters explicitly identify their loaded scope; they do not promise a global search. Searchable reference fields retrieve the authorized candidate set rather than relying on loaded collections.

## Choose references reliably

Type into a reference field to search eligible types in the current scope. Results include type, title, state, context and short ID to distinguish duplicate titles. Arrow keys and Enter select; Escape closes suggestions. Load more retrieves the next bounded page. Selected IDs are resolved independently of the visible search page. If the Outcome changes and a cursor expires, Reload results starts a fresh search while preserving selected references. Revocation invalidates access and cached material.

## Plan without changing execution

Create a Roadmap, edit its Draft, add phases, milestones and Objective/Task references, arrange their hierarchy and order, then review the diff and publication reason. **Publish revision** freezes a historical revision. **Activate revision** is a separate action. Removing a reference or moving a phase does not cancel or complete operational work. Dependencies are explicit actions on operational entities, never an implicit consequence of Roadmap order.

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

See [the final acceptance record](ux/final-acceptance.md), [experience contracts](ux/experience-contract.md), [architecture decision](adr/0027-human-workspace-client.md) and [current screenshots](ui-workspace/README.md).

The real-server browser suite covers 31 journeys, including keyboard controls, 390px layouts, conflict/replay, large reference searches and legacy/unsigned/signed regressions. Automated axe scans cover login, Overview, Task creation and the reference combobox. Equivalent 200%/400% viewport reflow is distinct from native browser zoom.

Representative-user usability, human comprehension/timing, manual screen-reader checks and native browser zoom remain pending. The owner confirmed that no representative users or screen-reader testers are available. Automated acceptance is not a usability measurement or WCAG certification.
