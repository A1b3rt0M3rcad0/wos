# Issue and Task ownership backfill

The ownership audit is read-only unless an operator explicitly supplies a reviewed mapping and requests an apply. It never guesses an Objective from a title, dependency, affected reference, or the number of available Objectives.

Run a database backup and verify it before making a repair. Stop every WOS writer for the database/schema for the whole apply operation. Apply this to one installation at a time.

## 1. Audit legacy records

For SQLite:

```sh
python3 tools/acceptance/issue_ownership_backfill.py \
  --backend sqlite --sqlite /path/to/wos.sqlite3 > ownership-audit.json
```

For PostgreSQL, provide the DSN through an environment variable and choose the WOS schema:

```sh
export WOS_POSTGRES_DSN='postgresql://…'
python3 tools/acceptance/issue_ownership_backfill.py \
  --backend postgres --schema public > ownership-audit.json
```

Review each orphan with its actual owner and context. The suggested Objectives in the report are leads for human investigation only; they are never applied automatically.

## 2. Write an explicit mapping

Use schema version 1. `operator` identifies the active WOS principal responsible for the maintenance operation and the actor recorded on its domain events. Every assignment names a scope, existing Objective, expected entity version, and human-readable reason.

```json
{
  "mapping_version": 1,
  "operator": {
    "principal_id": "0199edc4-0000-7000-8000-000000000006",
    "actor": {
      "kind": "human",
      "provider": "company-sso",
      "id": "operator-account",
      "display_name": "Workspace operator"
    }
  },
  "assignments": [
    {
      "kind": "task",
      "namespace_id": "0199edc4-0000-7000-8000-000000000001",
      "outcome_id": "0199edc4-0000-7000-8000-000000000002",
      "work_item_id": "0199edc4-0000-7000-8000-000000000004",
      "objective_id": "0199edc4-0000-7000-8000-000000000003",
      "expected_version": 3,
      "reason": "The recorded acceptance test belongs to this Objective."
    },
    {
      "kind": "issue",
      "namespace_id": "0199edc4-0000-7000-8000-000000000001",
      "outcome_id": "0199edc4-0000-7000-8000-000000000002",
      "issue_id": "0199edc4-0000-7000-8000-000000000005",
      "objective_id": "0199edc4-0000-7000-8000-000000000003",
      "expected_version": 1,
      "reason": "The report concerns this Objective's login flow."
    }
  ]
}
```

Replace all example IDs and versions with values from the installed database. The mapping rejects repeated records, unknown fields, invalid IDs, missing reasons, and stale versions.

## 3. Preview, then apply

Preview a mapping first. This remains read-only and reports whether each mapping would apply or replay as a no-op:

```sh
python3 tools/acceptance/issue_ownership_backfill.py \
  --backend sqlite --sqlite /path/to/wos.sqlite3 --mapping ownership-map.json
```

When the preview matches the review and all writers are stopped, apply it with a **new** receipt path:

```sh
python3 tools/acceptance/issue_ownership_backfill.py \
  --backend sqlite --sqlite /path/to/wos.sqlite3 --mapping ownership-map.json \
  --apply --writers-stopped --receipt /secure/path/ownership-receipt.json
```

PostgreSQL uses the same flags with `--backend postgres --schema public`; `WOS_POSTGRES_DSN` must be set. The tool creates the receipt with owner-only permissions, applies all mappings in one database transaction, increments each changed record version, advances each affected Outcome revision once, and records an event for each changed record. The receipt includes the mapping hash, operator, reasons, result, and post-apply orphan counts.

Replaying a successful mapping against the same Objective is a no-op and does not add another event or advance the Outcome revision. A changed owner, stale unassigned record, missing/terminal Objective for unresolved work, active Task lease/contract, invalid dependency, or any other conflict aborts the full transaction. Review the audit again after apply. The ownership checkpoint passes only when no orphan Task or Issue remains in any Outcome; later adapter-parity and validation stages must also pass before a contract migration is approved.

The maintenance repair does not infer domain facts or change Issue affected references, Task lifecycle, criteria, evidence, Blockers, or dependency relations.
