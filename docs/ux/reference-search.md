# Authorized reference search

`GET /api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/references`

Required `kind` is a comma-separated list drawn from: outcome, objective,
work_item, issue, blocker, evidence, artifact, decision, roadmap, relation,
evidence_link. Optional `query` (512 UTF-8 bytes), `lifecycle`, `limit` (1–100,
default 25), and `cursor` filter bounded metadata. `id` resolves a previously
selected reference by exact ID; it cannot combine with query or cursor. Missing
selected IDs return an empty authorized page. Integration triggers and signed
contract resources keep their existing specialized APIs.

HTTP and MCP `wos_search_references` call Application `SearchReferences`. MCP uses
`kinds` as an array, `text` as the query, and `entity_id` for exact resolution,
consistent with the existing MCP query convention. The generated OpenAPI includes
the HTTP route and typed page schema. Existing commands and payloads are unchanged.

Results contain typed `ref`, bounded `title`, `lifecycle`, current `version`, and
optional parent/objective `display_context`. Match against the full title (Evidence
description, Blocker description, relation type, or evidence-link rationale).
ASCII letters match without case; other Unicode remains case-sensitive. Display
text is capped at 128 characters. Client labels and short IDs disambiguate duplicate
titles; neither title nor eligibility grants write permission.

Order is kind then ID. Cursors bind scope, kinds, query, lifecycle and Outcome
revision. A changed filter/scope rejects the cursor with HTTP 422; a changed revision returns
`precondition_failed`/HTTP 409. The picker must explicitly reload results while
retaining selected IDs. Authorization is rechecked inside the UnitOfWork, including
credential and grant revocation, before any metadata query.

SQL searches scoped aggregate metadata in a fixed allowlist of UNION branches;
all branches use existing UNIQUE(namespace_id,outcome_id,id) indexes for scope and
keyset scans. Substring matching scans metadata within the authorized Outcome;
it is not a full-text index or a promise of constant-time search. SQL returns at
most limit+1 metadata rows and never hydrates full aggregates/evidence bodies.
Memory retains at most limit+1 sorted candidates. No migration or data rewrite is
required. Future language-aware/full-text matching requires a separate contract.

Query observations record latency, returned page counts and empty-page counts;
they do not include search text, tokens, or evidence material. Test acceptance
covers 1,001 Tasks and Evidence records per store, all eleven types, duplicate
titles, off-page exact resolution, revision/filter cursors, Unicode parity and
cached-identity revocation. Human usability remains a separate measurement.
