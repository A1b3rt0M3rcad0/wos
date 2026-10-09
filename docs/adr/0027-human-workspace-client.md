# ADR-027 — Human workspace client and reference discovery

Status: accepted for implementation, 2026-10-09.

The owner requested the UX/DX program in `docs/ux/original-plan-2026-10-09.md`.
The existing schema-derived global command picker does not describe human intent.

Use explicit ES modules for the first complete correction: API/mutation
coordination, human action registry, form models, reference components, navigation,
and feature views. This is option A of the plan. It preserves stdlib Go static
embedding, self-only CSP, deterministic archives and zero production JS package
dependencies. The tradeoff is explicit DOM/state work; component contracts and
unit/browser tests are required. React/TypeScript/Vite is a later architectural
choice rather than a prerequisite or a second concurrent UI. Do not retain the
schema renderer as the human creation surface.

The catalog is technical compatibility metadata, not navigation. Every operation
has an explicitly checked exposure classification. Presentation guards are
conservative; server authorization and domain guards always decide. Signed v2
execution and review use external authorized profiles; the Web never stores
private keys or fabricates signatures.

Add an authorized Application reference-search query backed by a bounded query
port in Memory, SQLite and PostgreSQL. HTTP and MCP adapt the same query. Fixed
typed SQL branches prevent user-controlled table names. Scope/filter/revision
bound deterministic keyset cursors follow ADR-014. Resolve selected references by
authorized ID; result eligibility is not mutation authority. No domain lifecycle
or aggregate ownership changes are introduced.

Deep links carry scope/view/item but no credentials. Browser reload/back/forward
restore context only after authenticated reads. User content is never translated;
official English copy and human labels do not rename wire identifiers.
