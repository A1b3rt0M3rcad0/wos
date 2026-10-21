# Drag-only card movement — 2026-10-20

Owner-requested follow-up to [ADR-041](../../adr/0041-focused-kanban-modal-and-backlog-planning.md).
Cards no longer expose stage selects or destination menus. Drag/drop prepares the
existing operation; capability checks, readiness, proof, confirmation and recovery
are unchanged. Inspector operations remain available to keyboard and touch users.

## Executed checks

- Initial four-file browser run: **46 passed, one failed** of 47 distinct cases.
  The narrow-viewport workspace test scrolled its source away before pointerdown.
  Its helper now starts a real native mouse drag before revealing an offscreen lane;
  no synthetic drag events, force clicks, retries or longer deadlines were added.
- Final complete `kanban-workspace.spec.mjs`: **11 passed**, including all three
  viewport variants, activation/acquisition, direct Objective achievement, blocked
  acceptance, recovery and touch inspector operations. Together with the 36 other
  passing cases from the initial run this covers all 47 distinct affected scenarios;
  it is not a single all-green 47-case run.
- All **78 JavaScript units passed**. The web API executable rebuilt successfully.
- The previous head's hosted browser job passed, while native central/package/
  platform gates failed. Reproducing the central fixture gave HTTP 409 when it
  added a Task criterion after activation. Both central and signed-package fixtures
  now create Backlog Tasks, define criteria, then explicitly activate them.
- The corrected native central workflow and `verifySignedPackage` with terminal
  resolution required both pass against the actual local unified executable.
  The signed smoke uses a writable XDG cache; its first local attempt without that
  setting failed on the environment's read-only home directory.

No backend rules were weakened. Real PostgreSQL, native Windows keyring, actual
npm archives and all hosted gates remain separate CI checks for the new head.
The older 229-case receipt and screenshots retain their recorded runtime source.
No new full-suite, physical touch-drag or manual screen-reader claim is made.
No merge, release or version advancement.
