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

Identity, paper surfaces, robots, typography, spatial Board and post-it assets stay
unchanged. [ADR-041](../../adr/0041-focused-kanban-modal-and-backlog-planning.md)
records the presentation override and changed definition policy. Existing Defer is
an explicit planning transition, not an implicit mutation or a new lifecycle.

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

Execution results and screenshot source bindings will be recorded after qualification.
Earlier PR results remain historical and cannot qualify this changed source.
