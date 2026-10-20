#!/usr/bin/env python3
"""Read-only audit of Tasks and Issues without an Objective owner.

This tool reports possible Objective choices for human review. It never assigns
an owner or writes to the inspected database. SQLite is opened read-only;
PostgreSQL is inspected in a repeatable-read, read-only transaction.
"""

import argparse
from contextlib import closing
import importlib.util
import json
from pathlib import Path
import re
import sqlite3
import sys

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "wos_backup_restore", ROOT / "tools/operations/backup_restore.py"
)
OPS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(OPS)


def rows(connection, backend, query):
    if backend == "sqlite":
        cursor = connection.execute(query)
        names = [column[0] for column in cursor.description]
        return [dict(zip(names, row)) for row in cursor.fetchall()]
    return [json.loads(line) for line in connection.run(
        "SELECT row_to_json(a)::text FROM (" + query + ") AS a"
    )]


def columns(connection, backend, table):
    if backend == "sqlite":
        return {row["name"] for row in rows(
            connection, backend, "PRAGMA table_info(" + OPS.identifier(table) + ")"
        )}
    return {row["column_name"] for row in rows(connection, backend,
        "SELECT column_name FROM information_schema.columns "
        "WHERE table_schema = current_schema() AND table_name = '" + table + "'")}


def load_snapshot(connection, backend):
    version = rows(connection, backend,
        "SELECT COALESCE(MAX(version), 0) AS schema_version FROM schema_migrations")[0]["schema_version"]
    if int(version) < 1:
        raise OPS.OperationError("source is not a migrated WOS database")

    required = {
        "outcomes": {"id", "namespace_id"},
        "objectives": {"id", "namespace_id", "outcome_id", "title"},
        "work_items": {"id", "namespace_id", "outcome_id", "objective_id", "title", "lifecycle"},
        "issues": {"id", "namespace_id", "outcome_id", "title", "lifecycle"},
        "issue_affected_refs": {"namespace_id", "outcome_id", "issue_id", "affected_id", "affected_kind"},
        "relations": {"id", "namespace_id", "outcome_id", "source_id", "source_kind", "target_id", "target_kind", "relation_type", "strength", "lifecycle"},
    }
    available = {table: columns(connection, backend, table) for table in required}
    for table, expected in required.items():
        if not expected <= available[table]:
            raise OPS.OperationError("database schema is missing required WOS columns in " + table)

    issue_has_owner = "objective_id" in available["issues"]
    owner_projection = "i.objective_id" if issue_has_owner else "NULL AS objective_id"
    outcomes = rows(connection, backend,
        "SELECT namespace_id, id AS outcome_id FROM outcomes ORDER BY namespace_id, id")
    objectives = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS objective_id, title "
        "FROM objectives ORDER BY namespace_id, outcome_id, id")
    tasks = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS work_item_id, title, lifecycle, objective_id "
        "FROM work_items WHERE objective_id IS NULL "
        "ORDER BY namespace_id, outcome_id, id")
    issues = rows(connection, backend,
        "SELECT i.namespace_id, i.outcome_id, i.id AS issue_id, i.title, i.lifecycle, " + owner_projection +
        " FROM issues i WHERE " + ("i.objective_id IS NULL" if issue_has_owner else "TRUE") +
        " ORDER BY i.namespace_id, i.outcome_id, i.id")
    affected = rows(connection, backend,
        "SELECT namespace_id, outcome_id, issue_id, affected_id, affected_kind "
        "FROM issue_affected_refs ORDER BY namespace_id, outcome_id, issue_id, affected_kind, affected_id")
    relations = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS relation_id, source_id, source_kind, "
        "target_id, target_kind, strength, lifecycle FROM relations "
        "WHERE relation_type = 'depends_on' "
        "ORDER BY namespace_id, outcome_id, source_kind, source_id, target_kind, target_id, id")
    work_owners = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS work_item_id, objective_id "
        "FROM work_items WHERE objective_id IS NOT NULL "
        "ORDER BY namespace_id, outcome_id, id")
    return int(version), outcomes, objectives, tasks, issues, affected, relations, work_owners


def build_report(snapshot):
    version, outcomes, objectives, tasks, issues, affected, relations, work_owners = snapshot
    scoped_objectives = {}
    objective_titles = {}
    for value in objectives:
        key = (value["namespace_id"], value["outcome_id"])
        scoped_objectives.setdefault(key, []).append({
            "objective_id": value["objective_id"], "title": value["title"]
        })
        objective_titles[(key[0], key[1], value["objective_id"])] = value["title"]

    task_orphans = {}
    for task in tasks:
        key = (task["namespace_id"], task["outcome_id"])
        task_orphans.setdefault(key, []).append(task)
    issue_orphans = {}
    for issue in issues:
        key = (issue["namespace_id"], issue["outcome_id"])
        issue_orphans.setdefault(key, []).append(issue)

    affected_by_issue = {}
    for ref in affected:
        key = (ref["namespace_id"], ref["outcome_id"], ref["issue_id"])
        affected_by_issue.setdefault(key, []).append({
            "kind": ref["affected_kind"], "id": ref["affected_id"]
        })
    work_objectives = {
        (value["namespace_id"], value["outcome_id"], value["work_item_id"]): value["objective_id"]
        for value in work_owners
    }

    report_outcomes = []
    for outcome in outcomes:
        scope = (outcome["namespace_id"], outcome["outcome_id"])
        orphan_tasks = []
        task_ids = set()
        for task in task_orphans.get(scope, []):
            task_ids.add(task["work_item_id"])
            impacted = [edge for edge in relations
                if (edge["namespace_id"], edge["outcome_id"]) == scope
                and ((edge["source_kind"] == "work_item" and edge["source_id"] == task["work_item_id"])
                     or (edge["target_kind"] == "work_item" and edge["target_id"] == task["work_item_id"]))]
            orphan_tasks.append({
                "work_item_id": task["work_item_id"],
                "title": task["title"],
                "lifecycle": task["lifecycle"],
                "objective_candidates": [dict(item, basis="same_outcome_only")
                    for item in scoped_objectives.get(scope, [])],
                "dependency_edges": [{
                    "relation_id": edge["relation_id"],
                    "source": {"kind": edge["source_kind"], "id": edge["source_id"]},
                    "target": {"kind": edge["target_kind"], "id": edge["target_id"]},
                    "strength": edge["strength"],
                    "lifecycle": edge["lifecycle"],
                } for edge in impacted],
            })

        orphan_issues = []
        for issue in issue_orphans.get(scope, []):
            refs = affected_by_issue.get((scope[0], scope[1], issue["issue_id"]), [])
            candidates = {}
            for ref in refs:
                objective_id = None
                basis = None
                if ref["kind"] == "objective":
                    objective_id, basis = ref["id"], "affected_objective"
                elif ref["kind"] == "work_item":
                    objective_id = work_objectives.get((scope[0], scope[1], ref["id"]))
                    if objective_id:
                        basis = "affected_task_current_objective"
                if objective_id:
                    candidate_key = (scope[0], scope[1], objective_id)
                    candidates[objective_id] = {
                        "objective_id": objective_id,
                        "title": objective_titles.get(candidate_key, ""),
                        "basis": basis,
                    }
            orphan_issues.append({
                "issue_id": issue["issue_id"],
                "title": issue["title"],
                "lifecycle": issue["lifecycle"],
                "affected_refs": refs,
                "objective_candidates_for_human_review": [candidates[key] for key in sorted(candidates)],
                "available_objective_ids": [item["objective_id"] for item in scoped_objectives.get(scope, [])],
            })

        if orphan_tasks or orphan_issues:
            report_outcomes.append({
                "namespace_id": scope[0],
                "outcome_id": scope[1],
                "orphan_task_count": len(orphan_tasks),
                "orphan_issue_count": len(orphan_issues),
                "available_objectives": scoped_objectives.get(scope, []),
                "orphan_tasks": orphan_tasks,
                "orphan_issues": orphan_issues,
            })
    return {
        "audit": "wos-hierarchy-issue-orphans",
        "read_only": True,
        "schema_version": version,
        "mapping_applied": False,
        "outcome_count_with_orphans": len(report_outcomes),
        "orphan_task_count": sum(item["orphan_task_count"] for item in report_outcomes),
        "orphan_issue_count": sum(item["orphan_issue_count"] for item in report_outcomes),
        "outcomes": report_outcomes,
    }


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--backend", choices=("sqlite", "postgres"), required=True)
    parser.add_argument("--sqlite", help="existing SQLite database path; opened read-only")
    parser.add_argument("--dsn-env", default="WOS_POSTGRES_DSN",
                        help="environment variable name containing the PostgreSQL DSN")
    parser.add_argument("--schema", default="public", help="PostgreSQL schema to inspect")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", args.dsn_env):
        raise OPS.OperationError("invalid PostgreSQL DSN environment variable name")
    if args.backend == "sqlite":
        if not args.sqlite:
            raise OPS.OperationError("--sqlite is required for the SQLite backend")
        with closing(OPS.sqlite_read(args.sqlite)) as database:
            report = build_report(load_snapshot(database, "sqlite"))
    else:
        if args.sqlite:
            raise OPS.OperationError("--sqlite cannot be used with the PostgreSQL backend")
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]{0,62}", args.schema):
            raise OPS.OperationError("invalid PostgreSQL schema name")
        environment = OPS.postgres_env(args.dsn_env)
        with OPS.PostgresSnapshot(environment, args.schema) as database:
            report = build_report(load_snapshot(database, "postgres"))
    print(json.dumps(report, ensure_ascii=False, sort_keys=True, separators=(",", ":")))


if __name__ == "__main__":
    try:
        main()
    except (OPS.OperationError, sqlite3.Error, OSError, ValueError) as error:
        print("issue ownership audit failed: " + str(error), file=sys.stderr)
        raise SystemExit(2)
