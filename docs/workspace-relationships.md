# Understand the work on an Outcome Board

An Outcome is the result you want. Objectives describe conditions that make
that result possible; Tasks record the work people and agents perform.
Evidence records what was observed. Issues explain problems, and Blockers
identify the work an impediment actually prevents.

For example, an Outcome could be **Customers can receive partial refunds**.
An Objective could be **Partial refunds are implemented and tested**, with a
criterion such as **A partial refund preserves the remaining refundable balance**.
A Task implements the endpoint. A test report is Evidence. An incorrect balance
is an Issue; it blocks the Objective only when an explicit Blocker records that
impact.

```mermaid
flowchart TD
    O[Outcome: desired result] -->|contains| J[Objective: verifiable condition]
    J -->|associated work| T[Task: bounded execution]
    E[Evidence: observation or source] -->|EvidenceLink: stance and rationale| C[Success criterion on an Outcome, Objective or Task]
    C -->|explicit current assessment| A[Assessment: met, not met, inconclusive or waived]
    I[Issue: identified problem] -->|recorded cause| B[Blocker: explicit impediment]
    B -->|blocks a named target| T
```

The arrows explain different relationships. They are not an automatic
completion pipeline. Completing a Task does not achieve its Objective;
achieving an Objective does not achieve its Outcome. Linking Evidence does
not assess a criterion. `waived` is explicitly not verified.

## Choose the relationship you mean

| You want to express | Use | What it means |
| --- | --- | --- |
| This condition is part of another condition | **Add objective** from an Objective; review its fixed placement (or explicitly change it) | Decomposition in the same Outcome, without an implied execution dependency. |
| This Task contributes work to an Objective | **New task** from that Objective; review its fixed Objective placement | Explicit Task association. A Task may also belong directly to the Outcome. |
| This work needs something else first | **Add dependency** on a Task or Objective | A hard prerequisite affects readiness and completion. An advisory dependency records context without blocking transitions. |
| This observation supports or challenges a statement | **Link evidence** from Evidence | Select the target, stance (`supports`, `contradicts` or `context`) and rationale. Stance belongs to this individual EvidenceLink. |
| This criterion was evaluated | Select a success criterion and **Assess criterion** where permitted | Record a result for the current criterion revision, with rationale and the appropriate Evidence. |
| There is a problem with these items | **Report issue**; select **Affected items** | The Issue records problem scope. Association alone does not block work. |
| This problem actually prevents work | **Report issue and block work**, or **Mark work as blocked** | An explicit Blocker records one target, its cause and propagation. |
| The problem is fixed and certain blocked targets can continue | **Resolve issue and release selected blockers** | Resolve the Issue and explicitly select the blocking impacts to release. |
| There is a material deliverable or a recorded choice | **Register artifact** or **Propose decision** | Preserve a reference to a deliverable or an explicit choice and rationale. Neither implies verification or execution authority. |

The available actions depend on your permissions, the current lifecycle and
the instance's execution protocol. The server validates every submitted
operation. A missing action is not a reason to bypass that validation.

## Read a selected item

The Board and specialist views use the same inspector. Its context path identifies
the Outcome and actual parent; its relationship rows identify real endpoints.
A Task can belong directly to the Outcome. An Objective's **Required for Outcome**
obligation remains global, including when that Objective is nested.

Task inspection groups instructions, people and recorded authority, attention,
success conditions, Evidence, delivery and recent activity. Other entity types
show the facets relevant to them. Capabilities explain available actions and
preparation; every mutation still checks current permissions and domain rules.
A contract records authority, not proof that a worker process is currently running.

- **Depends on** points from the selected work to its prerequisite.
  **Required by** points to work that references the selected item.
  An unknown dependency strength is reported as unknown.
- Evidence labels keep the stance of each individual EvidenceLink visible.
  The same Evidence may support one target and contradict another. A criterion
  reference identifies the statement involved; it does not say it passed.
- **Affected item** describes an Issue's recorded scope. **Direct Blocker**
  identifies an actual impediment link. The Task's authoritative blocking and
  attention sections explain inherited blocking, current readiness and review.
- A resolved Issue can remain the cause of an active Blocker. Confirm that
  the impediment has been removed before releasing it. Resolving the Issue
  alone does not release its Blockers.

Only persisted ownership and Objective hierarchy position work in the tree.
Moving notes or changing the map's local arrangement does not change that
ownership. Roadmap grouping and order describe a plan, not operational
dependencies.

## Follow one result from planning to acceptance

1. Create the Outcome and describe the desired result. Add its required success
   criteria before activation.
2. Add an Objective for a verifiable intermediate condition. Mark
   **Required for Outcome** only when achieving that Objective must be a gate
   for the Outcome. Define the Objective's own criteria.
3. From the Objective, choose **New task** and review its preselected Objective.
   Add concrete instructions and deliverables. Add an explicit dependency only
   when prerequisite work is actually required.
4. Activate the Outcome when preparation is ready. Use the Task's current
   execution controls or CLI handoff to acquire bounded authority, perform
   the work externally and submit the delivery. Execution and independent
   review follow the instance's protocol; the browser does not sign on behalf
   of an agent.
5. Register the test observation as Evidence. Link it to the relevant target
   with the appropriate stance and rationale. Review the criterion separately;
   attaching a report does not approve it.
6. Investigate Issues and confirm removal of any blocking impacts. Review
   current required criteria, dependencies and obligations. Explicitly
   **Certify objective** and, when its own gates are satisfied,
   **Certify outcome**. Task progress percentages are not acceptance decisions.

A qualifying Objective can be certified directly from its planned state when
its actual gates are satisfied. Starting it first is not a fabricated requirement.
The action explains unmet gates rather than automatically adding criteria,
accepting Evidence or releasing blockers.

## Understand what is loaded

The Board is a bounded view of persistent state, not the entire database.
**Load more** retrieves additional available pages. Search and specialist views
help investigate records beyond the currently visible notes. Expanding a
branch or inspecting a Task can add context, while retaining a consistent
Outcome revision.

The inspector queries its own authorized, coherent workspace projection. It does
not infer absence from the currently loaded Board edges. Facet coverage
distinguishes complete, partial, unavailable and not-loaded data; explicit
pagination retains the observed revision. If another actor changes the Outcome,
refresh rather than combining incompatible pages. Timeline pagination pins a
history frontier; descendant history describes current structural placement.

See the [operational workspace guide](ux/operational-workspace/README.md) for
progressive creation, documentary updates, typed preparation and the shared inspector.

For navigation, execution authority and uncertain-response recovery, see
[the human workspace guide](workspace-ux.md). For the domain's precise source
of truth, see [the canonical architecture](WOS_Design_Arquitetura_Planejamento_Atualizado.md).
