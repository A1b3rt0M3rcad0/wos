# Human workspace

The bundled workspace is served at `/app/`. Exchange an access credential for a secure session, choose a workspace and open an Outcome. Official interface copy is English; stored user content and wire identifiers retain their original values. Dates explicitly use UTC.

An Outcome describes the intended result. Objectives describe intermediate conditions; Tasks describe work. Completing a Task does not achieve an Objective or Outcome. Evidence supports an assessment; registering evidence does not certify success. A waived criterion is not verified. Issues and their blocking impacts have separate lifecycles. Roadmaps reference work and retain immutable publication history.

## Current acceptance and remaining implementation

The UX/DX program is tracked in [ROADMAP](../ROADMAP.md#human-workspace-uxdx-program--2026-10-09) and [the experience contract](ux/experience-contract.md). The authorized reference query supports eleven types, bounded metadata, exact-ID resolution and revision-bound pagination in Memory, SQLite and PostgreSQL. A stale cursor requires a fresh search; access is checked again on every request.

The English migration precedes the dedicated forms, searchable reference picker, six-area navigation and visual roadmap editor. Existing local collection filters operate on loaded pages until the new server-search view is introduced. Translation alone does not resolve those interaction problems. Historical October 7 screenshots document the earlier interface, not acceptance of this program.

## Verification

Browser tests run the actual embedded service and preserve Portuguese fixture titles as user content. Pure presentation checks verify English fallbacks, official terminology and UTC formatting. Reference tests exercise 1,001 Tasks and Evidence per store, duplicate names, scoped pagination and credential revocation. Hosted CI is available and verified for the foundation and reference-query pull requests.

Representative-user usability and manual screen-reader validation remain pending by owner decision. Automated checks do not establish a comprehension rate or WCAG certification.
