# Focused Kanban follow-up — 2026-10-20

Owner-authorized scope: the seven simplifications following PR #116's workspace-mode
qualification. Mandatory local skills: `laws-of-ux-auditor` and `product-design-and-ux`.
This review uses the owner's screenshots, inspected code and executable browser/
backend scenarios; it is not a participant study or a global accessibility audit.

## Findings and decisions

| Finding | Observable problem | Decision | Priority / effort |
|---|---|---|---|
| S1 | Selected cards expose Verify criteria, Add material, technical audit and a large action directory | Remove duplicate entry points; retain contextual work, responsibility, proof, impediments and history | High / low |
| S2 | Review proof mixes planning actions with evaluation | Instructions + criterion checklist; existing canonical evidence/assessment forms | High / medium |
| S3 | Definition/dependencies can change after work becomes available | Shared backend Backlog gate for all transports and capability reads; keep proof/execution writable | High / medium |
| S4 | Kanban heading repeats navigation breadcrumbs | Plain parent caption and one Back to parent control where applicable | Medium / low |
| S5 | Views requires a menu step to reach Overview | Direct Overview; specialist navigation remains available outside Board | Medium / low |
| S6 | Owner now requests a modal over Board | One native modal at a time; preserve mounted camera and contextual return | High / medium |
| S7 | Create Task and Create Objective appear together regardless of collection | Context selector controls both collection and its single relevant create action | Medium / low |
| S8 | Lane menu paints above the mobile item inspector and can intercept its pointer input | Inspector stacking context plus actual hit-testing regression (commit `3d0d4ba`) | High / low |
| S9 | Executing Task capabilities could report the older lease reason while commands enforce the planning freeze | Task capability reads use the same planning predicate; verify both To do and In progress | High / low |
| S10 | Another Outcome could expose old header controls while its permissions were pending, losing a single Overview click | Reveal the Outcome after permissions and header rendering agree; delayed-permission regression | High / low |

Identity, paper surfaces, robots, typography, spatial Board and post-it assets stay
unchanged. [ADR-041](../../adr/0041-focused-kanban-modal-and-backlog-planning.md)
records the presentation override and changed definition policy. Existing Defer is
an explicit planning transition, not an implicit mutation or a new lifecycle.

S3 and S6 are owner-approved policy/presentation changes, not defects in the
previous approved contract. S1/S2/S4/S5/S7 are owner-reported friction confirmed in
screenshots/code. S8/S9/S10 were observed through visual inspection or controlled
tests. Confidence is high for these bounded findings; no participant study or
measured productivity improvement is claimed.

## Interaction contract

| Context | Action and effect | Guard and recovery |
|---|---|---|
| Collection | Context changes the collection and its single create action; no write | Preserve frame filters/scroll; parent return is navigation, not reassignment |
| Card movement | Prepare an existing command or inspect its requirements; no optimistic move | Fresh capability/version/authority check and explicit confirmation; invalid moves explain requirements |
| Task definition | Edit only in Backlog | Same backend gate across transports; no automatic defer or authority revocation |
| Proof checklist | Add/reuse Evidence, optionally reference an Artifact, then assess a criterion explicitly | Criterion-scoped canonical forms; Evidence never checks a criterion; signed review stays external |
| Long instructions | Read the complete Task definition | Require the same Outcome revision; a stale read asks for refresh without a write |
| Modal return | One native dialog open at a time; restore selection, focus and Board camera | Escape respects local selection and guards submitting/uncertain operations; exact-intent recovery stays intact |
| Overview | Direct navigation with specialist views retained | New Outcome controls remain hidden until current permissions and navigation agree; do not interrupt an unresolved operation |

## Verification contract

- HD 1280×720, QHD 2560×1440, 4K 3840×2160 and mobile 390×844.
- Exactly one open dialog; Escape closes selection before collection; return focus,
  camera, filters and scroll remain recoverable.
- Correct create action per collection; no removed inspector controls or breadcrumbs.
- Long instruction previews can load the complete canonical Task definition at the
  same Outcome revision without restoring an audit interface.
- Register/reuse Evidence against a specific criterion, assess explicitly, then
  complete real work without automatically accepting its Objective/Outcome.
- Post-activation definition rejection preserves versions/events; proof and execution
  stay operational. Existing concurrency, signed-protocol, storage and recovery
  regressions remain enabled. Fixtures prepare definitions before activation rather
  than bypassing the rule or weakening command assertions.

Execution results and screenshot source bindings are tracked in the [current qualification receipt](simplification-validation.md).
Earlier PR results remain historical and cannot qualify this changed source.
