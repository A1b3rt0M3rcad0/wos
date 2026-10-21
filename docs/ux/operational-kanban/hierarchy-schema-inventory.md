# Hierarchy and Issue schema inventory

**Status:** pre-implementation inventory for Stage 03 of `WOS_PLANO_IMPLEMENTACAO_HIERARQUIA_ISSUES_DEPENDENCIAS.md`. This file records the current branch state and implementation considerations; it does not implement the proposed hierarchy or Issue changes.

**Baseline inspected:** `fix/workspace-operational-density`, commit `d80d8329e097b7252b65b9c75f49e00d6cdcfa39`, 2026-10-20. No code, migration, or test files were edited for this inventory.

## Confirmed facts: migrations and DDL

Migration files are embedded independently by the SQLite and PostgreSQL adapters. Both directories currently end at `0033_completion_policy.sql`; **0034 is the next unused ordinal in this checkout**, subject to rechecking the branch immediately before creating a migration. The plan's `0034_issue_objective_assignment.sql` is a provisional name, not an existing migration or a committed choice.

The latest migration is byte-for-byte identical in both adapters and contains exactly:

```sql
ALTER TABLE outcomes
ADD COLUMN completion_policy TEXT NOT NULL DEFAULT 'compatible'
CHECK (completion_policy IN ('compatible','strict_work'));
```

Source: `storage/{sqlite,postgres}/migrations/0033_completion_policy.sql:1`. It adds one non-null Outcome policy with a compatibility-preserving default and a constrained value set; it does not add or migrate Issue or Task ownership.

For the requested base migrations, `packages/wos-core/storage/sqlite/migrations/0001_foundation.sql` and `0003_issues_blockers.sql` are byte-for-byte identical to their PostgreSQL counterparts in this checkout. SQLite is the authored SQL source for the PostgreSQL mirror; see `tools/postgresgen/generate.py:10-13,77-94`. The generator makes dialect changes elsewhere, so parity must be checked after regeneration rather than assumed for every future migration.

### `0001_foundation.sql`: scope, hierarchy, actor links and criteria

The relevant DDL is the same in `storage/sqlite/migrations/0001_foundation.sql` and `storage/postgres/migrations/0001_foundation.sql`:

| Table | Confirmed columns and constraints | Relevance |
|---|---|---|
| `outcome_coordination` (`:18-23`) | `(namespace_id, outcome_id)` primary key; `state_revision`; FK to `namespaces(id)` | Shared scope/revision row used by commands and revision-bound reads. |
| `entity_refs` (`:25-37`) | `id`, `namespace_id`, `outcome_id`, `kind`; unique `(namespace_id,outcome_id,id)` and `(namespace_id,outcome_id,id,kind)`; composite FK to `outcome_coordination` | Typed, same-Outcome reference target used by the hierarchy, Issues, Blockers and relations. |
| `outcomes` (`:39-61`) | Identity/scope, `kind='outcome'`, title/description/priority/lifecycle; FK to its typed `entity_refs`; `id = outcome_id` | Root entity for the operational records. |
| `objectives` (`:63-88`) | `parent_objective_id` nullable; lifecycle `planned/in_progress/achieved/cancelled`; `required_for_outcome`; unique `(namespace_id,outcome_id,id)`; FKs to its typed `entity_refs`, Outcome, and parent Objective in the same scope; self-parent check | Existing Objective/subobjective hierarchy. A root Objective has a null parent. |
| `work_items` (`:90-129`) | `objective_id TEXT` is nullable (`:103`); lifecycle `backlog/todo/in_progress/done/cancelled`; unique `(namespace_id,outcome_id,id)`; composite FKs to its typed `entity_refs`, Outcome, and `(namespace_id,outcome_id,objective_id)` in `objectives` (`:114-120`) | A Task can currently be unowned (`objective_id IS NULL`) while remaining scoped to an Outcome. The composite FK is skipped when its nullable Objective ID is null. |
| `entity_actor_links` (`:131-146`) | scoped entity ID, role constrained to `owner/assignee`, actor kind/provider/id/JSON; compound primary key; same-scope FK to `entity_refs` | A normalized actor-link table already permits owner/assignee links for any typed entity at the DDL level. Issue repository integration is absent today; do not add a second assignment source without choosing which representation is canonical. |
| `success_criteria` (`:148-163`) | `owner_id`, `owner_kind` constrained to `outcome/objective/work_item`, revisions/status; FK to typed `entity_refs` and Outcome | Criteria are owned by existing Outcome, Objective or WorkItem; Issue is not a criterion owner. |

Existing indexes in this migration include `idx_objectives_scope(namespace_id,outcome_id,lifecycle,created_at,id)` and `idx_objectives_parent(namespace_id,outcome_id,parent_objective_id)` (`:310-314`), plus `idx_work_scope(namespace_id,outcome_id,lifecycle,priority,id)` and `idx_work_objective(namespace_id,outcome_id,objective_id)` (`:316-320`). The latter supports looking up Tasks by Objective, but there is no Issue owner index because the owner column does not exist.

### `0002_relations.sql`: dependency source of truth

`storage/{sqlite,postgres}/migrations/0002_relations.sql` defines `relations` with `source_id/source_kind`, `relation_type`, `target_id/target_kind`, optional `strength` and `satisfaction`, lifecycle and reason (`:1-36`). Composite FKs bind the relation and both endpoints to typed `entity_refs` within the same Outcome (`:18-25`). A `depends_on` check restricts both endpoint kinds to `objective/work_item` and requires `strength` plus `satisfaction='target_completed'` (`:27-35`).

The source/target of a dependency is therefore the `relations` row; there is no dependency column on `work_items` or `objectives`. The active semantic uniqueness index is `uq_relations_active_semantic` over scope, source, relation type and target (`:38-44`); `idx_relations_source` and `idx_relations_target` support directional lookup (`:46-50`). Application validation adds rules not expressible in this DDL, including dependency DAG validation and rejecting a hard Task-to-own-Objective edge (`application/dependencies.go:49-80`, `:128-155`; `domain/relation.go:96-141`).

### `0003_issues_blockers.sql`: Issue, affected references and blockers

The relevant DDL is the same in `storage/sqlite/migrations/0003_issues_blockers.sql` and `storage/postgres/migrations/0003_issues_blockers.sql`:

```sql
CREATE TABLE issues (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'issue' CHECK (kind = 'issue'),
    version BIGINT NOT NULL CHECK (version >= 1),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('critical','major','minor','informational')),
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('open','investigating','resolved','wont_fix','duplicate')),
    reported_by_json TEXT NOT NULL,
    resolution_summary TEXT NOT NULL DEFAULT '',
    duplicate_of_issue_id TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, duplicate_of_issue_id)
        REFERENCES issues(namespace_id, outcome_id, id),
    CHECK (duplicate_of_issue_id IS NULL OR duplicate_of_issue_id <> id)
);
```

Source: `storage/sqlite/migrations/0003_issues_blockers.sql:1-24` (the PostgreSQL file matches). The row has **no `objective_id`, `assignee_ref`, or owner relation**. It only stores its reporter (`reported_by_json`), state and optional duplicate target. `issue_affected_refs` (`:26-37`) is a many-to-many-style list keyed by `(namespace_id,outcome_id,issue_id,affected_id,affected_kind)`, with an FK to the Issue (cascade delete) and a same-scope typed FK to `entity_refs`. This is impact/context metadata, not Objective ownership.

Issue indexes are `idx_issues_scope(namespace_id,outcome_id,lifecycle,severity,id)` (`:70-71`) and `idx_issue_affected_ref(namespace_id,outcome_id,affected_id,affected_kind,issue_id)` (`:73-74`). Neither supports Issue-owner filtering today. The same migration defines `blockers` independently (`:39-68`), including `blocked_id/blocked_kind`, optional `cause_id/cause_kind` or external cause, `direct/subtree` propagation, same-scope typed-reference FKs, and indexes `idx_blockers_scope` and `idx_blockers_cause` (`:76-80`). A Blocker is its own record; resolving an Issue is not represented by a Blocker lifecycle change.

## Confirmed facts: migration execution and SQLite rebuild constraints

`storage/sqlite/migrations.go:21-22,31-54,127-173` embeds `migrations/*.sql`, creates `schema_migrations(version,name,checksum,applied_at)`, reads and sorts filenames, rejects duplicate ordinals, computes SHA-256 over exact file bytes, applies each migration transactionally, and rejects a DB schema newer than the binary. Applied migration checksums must match (`:86-105`); the migration SQL and checksum record commit together (`:112-124`). Version 29 also has a version-specific application-side backfill hook (`:107-110`). Applied migration files must therefore remain immutable.

The generated PostgreSQL runner is `storage/postgres/migrations.go`. It has the same migration behavior and adds transaction-scoped advisory lock `pg_advisory_xact_lock(846239010)` (`:67-75`); placeholders and error labels are PostgreSQL-specific. Both adapters calculate their latest supported schema ordinal from embedded files. The generator transforms SQL migrations as well as repository Go code (`tools/postgresgen/generate.py:77-94`); PostgreSQL migration files are generated/committed output.

SQLite opens with foreign keys enabled (`storage/sqlite/adapter.go:109,137,278-279`). The runner begins a transaction before executing each migration SQL file. SQLite supports adding a nullable column, but `ALTER TABLE ... ADD COLUMN` cannot add a table-level composite FK after the fact. A same-scope `(namespace_id,outcome_id,objective_id)` FK on `issues` therefore needs a deliberate SQLite-compatible schema strategy. Existing rebuild precedent `0023_signed_protocol_epoch.sql:1-14` renames, recreates, copies and drops a table **whose comments explicitly state it has no inbound FKs**. It is not a safe template for `issues` without adapting for the existing inbound self-FK (`duplicate_of_issue_id`) and `issue_affected_refs.issue_id ... ON DELETE CASCADE` FK.

**Implementation recommendation:** before choosing the Issue-owner DDL, document the SQLite rebuild/constraint plan and prove preservation of duplicate references, affected refs, indexes and foreign-key validity from a populated pre-change DB. Do not toggle foreign-key enforcement inside the existing migration transaction or perform a naïve rename/drop that rewrites child FK targets or cascades affected refs. Run a real SQLite migration compatibility test and `PRAGMA foreign_key_check` as part of that slice. The likely later `work_items.objective_id NOT NULL` contract is also not a simple additive column change; keep it behind audited backfill and a separate contract migration, as the plan proposes.

## Confirmed facts: repository and query surfaces

| Layer / adapter | Current source of truth | Confirmed behavior and gap |
|---|---|---|
| Domain | `packages/wos-core/domain/issue.go:45-59,61-87,93-157` | `domain.Issue` owns lifecycle, reporter, affected refs, resolution and duplicate link. It has no Objective owner or assignee field. `NewIssue` creates `open`; validation does not require an owner. |
| Application port | `packages/wos-core/ports/repositories.go:36-41` | `IssueRepository` has only `Get`, `ListByOutcome`, `Insert`, `Save`. No owner-specific lookup or count contract exists. |
| Memory | `packages/wos-core/storage/memory/adapter.go:803-850` | The transaction's `issues` map is authoritative; scoped lookup/list clones domain Issues; list scans all entries and sorts by ID. It has no independent SQL-like index. |
| SQLite | `packages/wos-core/storage/sqlite/issues_blockers.go:12-68,71-100,102-138,140-229` | The SQLite Issue row plus `issue_affected_refs` rows are loaded inside the UoW. `ListByOutcome` first reads all IDs, then calls `Get` per Issue, and each `Get` separately loads affected refs (N+1 query pattern). Inserts/saves write no owner or assignee. |
| PostgreSQL | `packages/wos-core/storage/postgres/issues_blockers.go:1-2,13-69,72-100,103-139,141-230` | Generated mirror of the SQLite repository using `$n` placeholders. `tools/postgresgen/generate.py:12-13,27-32` identifies SQLite as source and applies dialect transformation. Semantics match SQLite in the inspected methods. |

Application reads expose whole-Outcome `ListIssues` with `OutcomeRevision` (`application/issues_blockers.go:299-325`); direct HTTP `GET /.../issues` returns that unbounded collection (`http/handler.go:244-254`, `http/issues_blockers.go:58-71`). There is no bounded `ListPendingIssuesByObjective` repository/read contract. Existing generic continuity pages can list/count all records in the `issues` section by Outcome (`storage/sqlite/continuity_read.go:25-65,95-104`), but cannot filter pending Issues by Objective because ownership is absent. Generic graph node loading can summarize selected Issue IDs (`storage/sqlite/graph_read.go:125-180`); graph edges are backed by hierarchy, blockers and `relations`, not an Issue-owner edge (`:57-86`). `ports/workspace_read.go:40-53` has workspace structure/card/children reads but no Issue page or per-Objective Issue aggregate.

Objective acceptance currently checks independent review, blockers, hard dependencies, its criteria/proof, and completion policy in `application/workspace_preconditions.go:52-75`; it does not query Issues. The objective capability path calls that existing shared predicate (`application/workspace_capabilities.go:335` in this baseline), so a future gate should be one UoW-coherent application rule reused by the capability and mutation paths rather than a UI-only check.

There is no issue-specific `COUNT`/pending-by-owner query in the inspected repository/read-model SQL. The generic continuity `COUNT(*)` is a total for the selected section within a whole Outcome, not an Objective-owned open count. Existing Issue handling in compounds reads the whole repository list (`application/issues_blockers.go:594`); this is not a scalable owner aggregate.

## Confirmed facts: command, OpenAPI, MCP, SDK and CLI generation

| Contract surface | Source / generated output | Current Issue facts |
|---|---|---|
| Domain command | `application/issues_blockers.go:13-19,80-108` | `CreateIssueCommand` takes Scope, title, description, severity and `AffectedRefs`; it has no `ObjectiveID`. Exported `Service` methods and typed `...Command` signatures are the input to the command generator. |
| Legacy REST | `http/handler.go:244-254`; handlers in `http/issues_blockers.go` and DTOs in `http/issues_blockers_dto.go` | Issue create/list/get/update and lifecycle action routes are registered directly. `createIssue` maps the REST request to `application.CreateIssueCommand` (`:10-42`). These routes need their REST DTO/handler/spec updated separately from generated command catalog entries. |
| Command catalog / MCP / Go SDK | `tools/mcpgen/main.go:28-84,85-115` | Generator scans public Application `Service` methods and writes `packages/wos-api/mcp/commands_generated.go`, `packages/wos-api/commands/catalog_generated.go`, and `packages/wos-sdk-go/commands_generated.go`. Existing Issue commands are represented there (for example generated `CreateIssue` at line 103 in API catalog, MCP registry and SDK method). Regenerate; do not edit generated files by hand. |
| Generated command OpenAPI | `tools/openapigen/main.go:1,61-69,87-95` | It reads catalog descriptors, emits `/api/v1/commands/{command}` schemas/paths into `packages/wos-api/commands.openapi.json`, and embeds `VERSION`. `packages/wos-api/openapi.yaml` separately references the generated document; it is not generated by this tool. |
| CLI | `packages/wos-cli/client.go:122-140,282-300`; `packages/wos-cli/planning.go:14-53,95-143`; `packages/wos-cli/help.go` | CLI is hand-routed, not an output of `mcpgen`. Its top-level usage and planning verbs expose work/objective/outcome/review/etc., but no Issue create/list/assign/investigate command. `mcpgen` provides SDK methods; CLI parity requires explicit CLI dispatch, input validation, permission/error presentation and tests. |

CI regenerates storage/API artifacts with `python3 tools/postgresgen/generate.py`, `go run ./tools/mcpgen` and `go run ./tools/openapigen` (`.github/workflows/ci.yml:88-90`). The SQLite migration SQL and Issue implementation are the authored sources; PostgreSQL files are generated mirrors. Application/domain contracts are the shared business source of truth. CLI remains its own explicit user-facing interface.

## Recommendations for later implementation stages

These are recommendations, not current behavior or approved DDL:

1. Keep **structural ownership**, **impact references**, **dependencies**, **Blockers**, and **assignment** as distinct relations. The approved plan names `work_items.objective_id` and `issues.objective_id` as owner fields; retain `objectives.parent_objective_id`, `relations(depends_on)`, and `blockers` for their separate meanings. `issue_affected_refs` must not be reinterpreted as ownership.
2. Before implementing Issue assignment, choose between the existing normalized `entity_actor_links` owner/assignee vocabulary and a dedicated serialized Issue assignee field. Ensure domain serialization, read, write, CAS/version behavior, events and authorization all use one canonical form.
3. Add a repository/read-model query that can return Issues for one owner Objective, only the pending lifecycle states needed by the Objective gate, plus bounded IDs/count/coverage if the UI is paged. Read it in the same UoW/snapshot as the acceptance decision; avoid projecting a gate from cards currently loaded in the browser.
4. Implement the Objective Issue gate in the shared application precondition and capability path. Keep Issue lifecycle, duplicate-chain closure, Blocker state, Criterion assessment and Conclusion separate; do not add Issue as a Task readiness reason or make Issue closure auto-resolve Blockers.
5. Run SQLite/PostgreSQL parity from the authored SQLite sources and generated outputs. Add real migration tests for old databases with populated Issue duplicate chains and affected refs, plus repository tests for same-scope owner queries, concurrent version conflicts, counts/page bounds and adapter parity.
6. Update REST routes/DTOs and `openapi.yaml` where applicable, regenerate catalog/MCP/SDK/OpenAPI artifacts, and implement CLI behavior explicitly. Add parity coverage so the new owner/assignee and list/read behavior does not silently exist only in browser or one transport.

## Existing files and tests to review when implementation starts

- DDL: `packages/wos-core/storage/{sqlite,postgres}/migrations/0001_foundation.sql`, `0002_relations.sql`, `0003_issues_blockers.sql`, `0023_signed_protocol_epoch.sql`, `0033_completion_policy.sql`.
- Storage generators/runners: `tools/postgresgen/generate.py`, `packages/wos-core/storage/{sqlite,postgres}/migrations.go`, `packages/wos-core/storage/{sqlite,postgres}/issues_blockers.go`, `packages/wos-core/storage/memory/adapter.go`.
- Domain/application/read contracts: `packages/wos-core/domain/{issue,work_item,objective,relation,blocker}.go`, `packages/wos-core/ports/{repositories,workspace_read}.go`, `packages/wos-core/application/{issues_blockers,dependencies,workspace_preconditions,workspace_capabilities,workspace_read_types,workspace_cards,workspace_frame}.go`.
- Transport/generation: `packages/wos-api/http/{handler,issues_blockers,issues_blockers_dto}.go`, `packages/wos-api/{openapi.yaml,commands.openapi.json}`, `packages/wos-api/commands/catalog_generated.go`, `packages/wos-api/mcp/commands_generated.go`, `packages/wos-sdk-go/commands_generated.go`, `packages/wos-cli/{client,planning,help}.go`, `tools/{mcpgen,openapigen}`.
- Existing tests to extend: `packages/wos-core/application/issues_blockers_test.go`, `packages/wos-core/storage/sqlite/issues_blockers_test.go`, generated `packages/wos-core/storage/postgres/issues_blockers_test.go`, `packages/wos-api/http/wave07_test.go`, both adapters' `migration_compatibility_test.go`, and CI regeneration checks.

This inventory does not approve a backfill, a schema contract migration, new command names, or a change to ADR-041. Those choices remain subject to the later staged review and migration-data audit.
