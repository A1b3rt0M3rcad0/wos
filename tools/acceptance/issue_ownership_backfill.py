#!/usr/bin/env python3
"""Audit and explicitly backfill legacy Tasks and Issues without an Objective.

The default operation is read-only. An operator-reviewed JSON mapping can be
dry-run, or applied atomically with --apply after writers have been stopped.
The tool never infers or creates an Objective. PostgreSQL mappings use a
serializable transaction; SQLite mappings use BEGIN IMMEDIATE.
"""

import argparse
from contextlib import closing
import contextlib
import importlib.util
import json
from pathlib import Path
import re
import sqlite3
import sys
import os
import queue
import secrets
import subprocess
import tempfile
import threading
import time
import uuid
from urllib.parse import unquote, urlparse


MAPPING_VERSION = 1
UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
)

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
        for task in task_orphans.get(scope, []):
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
        "mapping_would_apply": 0,
        "outcome_count_with_orphans": len(report_outcomes),
        "orphan_task_count": sum(item["orphan_task_count"] for item in report_outcomes),
        "orphan_issue_count": sum(item["orphan_issue_count"] for item in report_outcomes),
        "outcomes": report_outcomes,
    }


def _reject_duplicate_json_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise OPS.OperationError("mapping contains a duplicate JSON key")
        result[key] = value
    return result


def parse_mapping(path):
    mapping_path = Path(path)
    if mapping_path.is_symlink() or not mapping_path.is_file():
        raise OPS.OperationError("mapping must be a regular file")
    if mapping_path.stat().st_size > 5 * 1024 * 1024:
        raise OPS.OperationError("mapping file exceeds the 5 MiB safety limit")
    try:
        mapping = json.loads(
            mapping_path.read_text(encoding="utf-8"),
            object_pairs_hook=_reject_duplicate_json_keys,
        )
    except (UnicodeError, json.JSONDecodeError) as error:
        raise OPS.OperationError("mapping must be valid UTF-8 JSON") from error
    if not isinstance(mapping, dict) or set(mapping) != {"mapping_version", "operator", "assignments"}:
        raise OPS.OperationError("mapping must contain exactly mapping_version, operator and assignments")
    if type(mapping["mapping_version"]) is not int or mapping["mapping_version"] != MAPPING_VERSION:
        raise OPS.OperationError("unsupported mapping_version")
    operator = mapping["operator"]
    if not isinstance(operator, dict) or set(operator) != {"principal_id", "actor"}:
        raise OPS.OperationError("mapping operator must identify an active principal and ActorRef")
    if not isinstance(operator["principal_id"], str) or not UUID_PATTERN.fullmatch(operator["principal_id"]):
        raise OPS.OperationError("mapping operator principal_id is invalid")
    actor = operator["actor"]
    if not isinstance(actor, dict) or not {"kind", "provider", "id"} <= set(actor) or set(actor) - {"kind", "provider", "id", "display_name"}:
        raise OPS.OperationError("mapping operator actor must be a valid ActorRef")
    if not isinstance(actor["kind"], str) or actor["kind"] not in {"human", "agent", "service", "automation", "external_system"}:
        raise OPS.OperationError("mapping operator ActorRef kind is invalid")
    if any(not isinstance(actor.get(key), str) or not actor[key].strip() for key in ("provider", "id")):
        raise OPS.OperationError("mapping operator ActorRef provider and id are required")
    if "display_name" in actor and not isinstance(actor["display_name"], str):
        raise OPS.OperationError("mapping operator display_name must be text")
    operator = {"principal_id": operator["principal_id"], "actor": actor}
    if not isinstance(mapping["assignments"], list) or not mapping["assignments"]:
        raise OPS.OperationError("mapping assignments must be a non-empty array")

    assignments = []
    seen = set()
    for number, item in enumerate(mapping["assignments"], start=1):
        if not isinstance(item, dict):
            raise OPS.OperationError("mapping assignment " + str(number) + " must be an object")
        kind = item.get("kind")
        expected_keys = {
            "task": {"kind", "namespace_id", "outcome_id", "work_item_id", "objective_id", "expected_version", "reason"},
            "issue": {"kind", "namespace_id", "outcome_id", "issue_id", "objective_id", "expected_version", "reason"},
        }.get(kind)
        if expected_keys is None or set(item) != expected_keys:
            raise OPS.OperationError("mapping assignment " + str(number) + " has unsupported kind or fields")
        normalized = dict(item)
        for key in ("namespace_id", "outcome_id", "objective_id"):
            if not isinstance(item[key], str) or not UUID_PATTERN.fullmatch(item[key]):
                raise OPS.OperationError("mapping assignment " + str(number) + " has an invalid " + key)
        record_key = "work_item_id" if kind == "task" else "issue_id"
        if not isinstance(item[record_key], str) or not UUID_PATTERN.fullmatch(item[record_key]):
            raise OPS.OperationError("mapping assignment " + str(number) + " has an invalid " + record_key)
        if type(item["expected_version"]) is not int or item["expected_version"] < 1:
            raise OPS.OperationError("mapping expected_version must be a positive integer")
        if not isinstance(item["reason"], str) or not item["reason"].strip() or len(item["reason"]) > 2000:
            raise OPS.OperationError("mapping reason must be non-empty and at most 2000 characters")
        key = (kind, item["namespace_id"], item["outcome_id"], item[record_key])
        if key in seen:
            raise OPS.OperationError("mapping contains more than one assignment for the same record")
        seen.add(key)
        normalized["reason"] = item["reason"].strip()
        assignments.append(normalized)
    canonical_mapping = {"mapping_version": MAPPING_VERSION, "operator": operator, "assignments": assignments}
    digest = __import__("hashlib").sha256(OPS.canonical(canonical_mapping)).hexdigest()
    return canonical_mapping, digest


def _sql_text(value):
    """Quote a validated UUID/string for psql; free-form reasons never enter SQL."""
    if not isinstance(value, str) or not UUID_PATTERN.fullmatch(value):
        raise OPS.OperationError("unsafe identifier in reviewed mapping")
    return "'" + value + "'"


def _quoted_text(value):
    if not isinstance(value, str) or "\x00" in value:
        raise OPS.OperationError("text field contains an unsupported value")
    return "'" + value.replace("'", "''") + "'"


def _uuid7():
    timestamp_ms = time.time_ns() // 1_000_000
    random_a = secrets.randbits(12)
    random_b = secrets.randbits(62)
    value = (timestamp_ms << 80) | (0x7 << 76) | (random_a << 64) | (0b10 << 62) | random_b
    return str(uuid.UUID(int=value))


class PostgresWriter:
    """Small serializable psql session used only by explicit --apply."""
    def __init__(self, environment, schema):
        self.schema = schema
        OPS.identifier(schema)
        self.errors = tempfile.TemporaryFile()
        try:
            self.proc = subprocess.Popen(
                ["psql", "-X", "-qAt", "-v", "ON_ERROR_STOP=1"],
                env=environment, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                stderr=self.errors, text=True, bufsize=1,
            )
        except OSError as error:
            self.errors.close()
            raise OPS.OperationError("psql is required for PostgreSQL backfill") from error
        self.lines = queue.Queue()

        def drain():
            try:
                for line in self.proc.stdout:
                    self.lines.put(line.rstrip("\n"))
            finally:
                self.lines.put(None)

        self.reader = threading.Thread(target=drain, daemon=True)
        self.reader.start()
        self.committed = False
        try:
            self.run("BEGIN ISOLATION LEVEL SERIALIZABLE")
            self.run("SET LOCAL search_path TO " + OPS.identifier(schema) + ",pg_catalog")
        except Exception:
            self.close()
            raise

    def run(self, sql):
        marker = "_wos_backfill_" + os.urandom(16).hex()
        try:
            self.proc.stdin.write(sql.rstrip(";") + ";\nSELECT '" + marker + "';\n")
            self.proc.stdin.flush()
        except (OSError, BrokenPipeError) as error:
            raise OPS.OperationError("PostgreSQL backfill transaction failed") from error
        result = []
        while True:
            try:
                line = self.lines.get(timeout=60)
            except queue.Empty as error:
                raise OPS.OperationError("PostgreSQL backfill transaction timed out") from error
            if line is None:
                raise OPS.OperationError("PostgreSQL backfill transaction failed; verify the database and retry")
            if line == marker:
                return result
            result.append(line)

    def commit(self):
        self.run("COMMIT")
        self.committed = True

    def close(self):
        with contextlib.suppress(Exception):
            self.proc.stdin.write(("\\q\n" if self.committed else "ROLLBACK;\n\\q\n"))
            self.proc.stdin.close()
        try:
            self.proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.proc.kill()
            self.proc.wait(timeout=5)
        self.reader.join(timeout=1)
        self.proc.stdout.close()
        self.errors.close()

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.close()


def _schema_version(connection, backend):
    return int(rows(connection, backend,
        "SELECT COALESCE(MAX(version), 0) AS schema_version FROM schema_migrations")[0]["schema_version"])


def _require_backfill_schema(connection, backend):
    version = _schema_version(connection, backend)
    if version < 34:
        raise OPS.OperationError("explicit ownership backfill requires schema 0034 or newer")
    required = {
        "work_items": {"namespace_id", "outcome_id", "id", "objective_id", "version", "updated_at", "lifecycle", "lease_claim_id"},
        "issues": {"namespace_id", "outcome_id", "id", "objective_id", "version", "updated_at", "lifecycle"},
        "objectives": {"namespace_id", "outcome_id", "id", "lifecycle"},
        "outcome_coordination": {"namespace_id", "outcome_id", "state_revision"},
        "work_contracts": {"namespace_id", "outcome_id", "work_item_id", "status"},
        "relations": {"namespace_id", "outcome_id", "source_id", "source_kind", "target_id", "target_kind", "relation_type", "strength", "lifecycle"},
        "principals": {"id", "lifecycle"},
        "domain_events": {"id", "namespace_id", "outcome_id", "event_type", "schema_version", "outcome_revision", "event_index", "aggregate_id", "aggregate_kind", "aggregate_version_before", "aggregate_version_after", "command_id", "principal_id", "actor_json", "recorded_at", "correlation_id", "causation_id", "execution_context_json", "payload_json"},
    }
    for table, expected in required.items():
        if not expected <= columns(connection, backend, table):
            raise OPS.OperationError("database schema is missing required backfill columns in " + table)
    return version


def _assignment_state(connection, backend, assignment):
    namespace = assignment["namespace_id"]
    outcome = assignment["outcome_id"]
    objective = assignment["objective_id"]
    kind = assignment["kind"]
    record_key = "work_item_id" if kind == "task" else "issue_id"
    record = assignment[record_key]
    objective_rows = rows(connection, backend,
        "SELECT lifecycle FROM objectives WHERE namespace_id=" + _sql_value(backend, namespace) +
        " AND outcome_id=" + _sql_value(backend, outcome) + " AND id=" + _sql_value(backend, objective))
    if not objective_rows:
        raise OPS.OperationError("mapped Objective does not exist in the specified scope")
    table = "work_items" if kind == "task" else "issues"
    record_rows = rows(connection, backend,
        "SELECT objective_id, version, lifecycle, updated_at" + (", lease_claim_id" if kind == "task" else "") +
        " FROM " + table + " WHERE namespace_id=" + _sql_value(backend, namespace) +
        " AND outcome_id=" + _sql_value(backend, outcome) + " AND id=" + _sql_value(backend, record))
    if not record_rows:
        raise OPS.OperationError("mapped " + kind + " does not exist in the specified scope")
    state = record_rows[0]
    coordination = rows(connection, backend,
        "SELECT state_revision FROM outcome_coordination WHERE namespace_id=" + _sql_value(backend, namespace) +
        " AND outcome_id=" + _sql_value(backend, outcome))
    if not coordination:
        raise OPS.OperationError("mapped Outcome coordination record does not exist")
    state["objective_lifecycle"] = objective_rows[0]["lifecycle"]
    state["state_revision"] = int(coordination[0]["state_revision"])
    if kind == "task":
        active_contract = rows(connection, backend,
            "SELECT EXISTS(SELECT 1 FROM work_contracts WHERE namespace_id=" + _sql_value(backend, namespace) +
            " AND outcome_id=" + _sql_value(backend, outcome) + " AND work_item_id=" +
            _sql_value(backend, record) + " AND status='active') AS active_contract")[0]["active_contract"]
        hard_own_objective = rows(connection, backend,
            "SELECT EXISTS(SELECT 1 FROM relations WHERE namespace_id=" + _sql_value(backend, namespace) +
            " AND outcome_id=" + _sql_value(backend, outcome) + " AND source_id=" + _sql_value(backend, record) +
            " AND source_kind='work_item' AND target_id=" + _sql_value(backend, objective) +
            " AND target_kind='objective' AND relation_type='depends_on' AND strength='hard' AND lifecycle='active') AS invalid_edge")[0]["invalid_edge"]
        state["active_contract"] = bool(active_contract)
        state["hard_own_objective_dependency"] = bool(hard_own_objective)
    return state


def _sql_value(backend, value):
    if backend == "postgres":
        return _sql_text(value)
    # SQLite queries are generated only from validated UUIDs as well.
    if not isinstance(value, str) or not UUID_PATTERN.fullmatch(value):
        raise OPS.OperationError("unsafe identifier in reviewed mapping")
    return "'" + value + "'"


def validate_operator(connection, backend, operator):
    principal = rows(connection, backend,
        "SELECT lifecycle FROM principals WHERE id=" + _sql_value(backend, operator["principal_id"]))
    if not principal or principal[0]["lifecycle"] != "active":
        raise OPS.OperationError("mapping operator must be an existing active principal")


def validate_assignments(connection, backend, mapping):
    validate_operator(connection, backend, mapping["operator"])
    rows_to_apply = []
    scopes = {}
    for assignment in mapping["assignments"]:
        state = _assignment_state(connection, backend, assignment)
        owner = state["objective_id"]
        if owner == assignment["objective_id"]:
            state["action"] = "noop"
            rows_to_apply.append((assignment, state))
            continue
        if owner is not None:
            raise OPS.OperationError("mapped record already belongs to a different Objective; no owner is overwritten")
        if int(state["version"]) != assignment["expected_version"]:
            raise OPS.OperationError("mapped record version changed; refresh the audit and review the mapping")
        if state["objective_lifecycle"] in {"achieved", "cancelled"}:
            if assignment["kind"] == "task" or state["lifecycle"] in {"open", "investigating", "duplicate"}:
                raise OPS.OperationError("unresolved work cannot be mapped to a terminal Objective")
        if assignment["kind"] == "task":
            if state["lifecycle"] != "backlog":
                raise OPS.OperationError("only a Backlog Task without an Objective can be mapped safely")
            if state["lease_claim_id"] is not None or state["active_contract"]:
                raise OPS.OperationError("Task has execution authority; drain it through the domain protocol first")
            if state["hard_own_objective_dependency"]:
                raise OPS.OperationError("mapping would make the Task hard-depend on its own Objective")
        state["action"] = "apply"
        rows_to_apply.append((assignment, state))
        scopes[(assignment["namespace_id"], assignment["outcome_id"])] = state["state_revision"]
    return rows_to_apply, scopes


def apply_sqlite(path, mapping, mapping_sha256):
    raw_path = Path(path)
    if raw_path.is_symlink():
        raise OPS.OperationError("SQLite source must be a regular file")
    db_path = raw_path.resolve(strict=True)
    if not db_path.is_file():
        raise OPS.OperationError("SQLite source must be a regular file")
    connection = sqlite3.connect(db_path, timeout=30, isolation_level=None)
    connection.row_factory = sqlite3.Row
    try:
        connection.execute("PRAGMA foreign_keys=ON")
        connection.execute("BEGIN IMMEDIATE")
        _require_backfill_schema(connection, "sqlite")
        rows_to_apply, scopes = validate_assignments(connection, "sqlite", mapping)
        now = time.time_ns() // 1000
        changed = []
        changed_times = {}
        for assignment, state in rows_to_apply:
            if state["action"] == "noop":
                continue
            table = "work_items" if assignment["kind"] == "task" else "issues"
            record_key = "work_item_id" if assignment["kind"] == "task" else "issue_id"
            updated_at = max(now, int(state["updated_at"]) + 1)
            cursor = connection.execute(
                "UPDATE " + table + " SET objective_id=?, version=version+1, updated_at=? "
                "WHERE namespace_id=? AND outcome_id=? AND id=? AND objective_id IS NULL AND version=?",
                (assignment["objective_id"], updated_at, assignment["namespace_id"], assignment["outcome_id"],
                 assignment[record_key], assignment["expected_version"]),
            )
            if cursor.rowcount != 1:
                raise OPS.OperationError("record changed during backfill; transaction rolled back")
            changed.append(assignment)
            changed_times[(assignment["kind"], assignment["namespace_id"], assignment["outcome_id"], assignment[record_key])] = updated_at
        changed_scopes = sorted({(item["namespace_id"], item["outcome_id"]) for item in changed})
        after_revisions = {}
        command_ids = {}
        revision_indexes = {}
        for namespace, outcome in changed_scopes:
            cursor = connection.execute(
                "UPDATE outcome_coordination SET state_revision=state_revision+1 WHERE namespace_id=? AND outcome_id=?",
                (namespace, outcome),
            )
            if cursor.rowcount != 1:
                raise OPS.OperationError("Outcome coordination changed during backfill; transaction rolled back")
            revision = connection.execute(
                "SELECT state_revision FROM outcome_coordination WHERE namespace_id=? AND outcome_id=?",
                (namespace, outcome),
            ).fetchone()[0]
            after_revisions[namespace + "/" + outcome] = int(revision)
            command_ids[(namespace, outcome)] = _uuid7()
            existing = connection.execute(
                "SELECT COALESCE(MAX(event_index),-1) FROM domain_events WHERE namespace_id=? AND outcome_id=? AND outcome_revision=?",
                (namespace, outcome, int(revision)),
            ).fetchone()[0]
            if int(existing) >= 0:
                raise OPS.OperationError("unexpected domain events already exist at the new Outcome revision")
            revision_indexes[(namespace, outcome)] = 0
        operator = mapping["operator"]
        actor_json = json.dumps(operator["actor"], ensure_ascii=False, separators=(",", ":"))
        for assignment in changed:
            scope = (assignment["namespace_id"], assignment["outcome_id"])
            kind = assignment["kind"]
            record_id = assignment.get("work_item_id", assignment.get("issue_id"))
            recorded_at = changed_times[(kind, scope[0], scope[1], record_id)]
            event_id = _uuid7()
            event_index = revision_indexes[scope]
            revision_indexes[scope] += 1
            event_type = "work_item.objective_changed" if kind == "task" else "issue.objective_backfilled"
            payload = json.dumps({"source": "explicit_ownership_backfill", "mapping_sha256": mapping_sha256,
                                  "objective_id": assignment["objective_id"], "reason": assignment["reason"]},
                                 ensure_ascii=False, separators=(",", ":"))
            connection.execute(
                "INSERT INTO domain_events (id,namespace_id,outcome_id,event_type,schema_version,outcome_revision,event_index,"
                "aggregate_id,aggregate_kind,aggregate_version_before,aggregate_version_after,command_id,principal_id,actor_json,"
                "recorded_at,correlation_id,causation_id,execution_context_json,payload_json) "
                "VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                (event_id, scope[0], scope[1], event_type, 1, after_revisions[scope[0] + "/" + scope[1]], event_index,
                 record_id, "work_item" if kind == "task" else "issue", assignment["expected_version"],
                 assignment["expected_version"] + 1, command_ids[scope], operator["principal_id"], actor_json,
                 recorded_at, None, None, "{}", payload),
            )
        connection.commit()
        return {
            "changed_count": len(changed),
            "noop_count": len(rows_to_apply) - len(changed),
            "changed_outcome_revisions": after_revisions,
        }
    except Exception:
        connection.rollback()
        raise
    finally:
        connection.close()


def apply_postgres(environment, schema, mapping, mapping_sha256):
    with PostgresWriter(environment, schema) as connection:
        _require_backfill_schema(connection, "postgres")
        assignments = mapping["assignments"]
        scopes_to_lock = sorted({(item["namespace_id"], item["outcome_id"]) for item in assignments})
        for namespace, outcome in scopes_to_lock:
            locked = connection.run(
                "SELECT state_revision FROM outcome_coordination WHERE namespace_id=" + _sql_text(namespace) +
                " AND outcome_id=" + _sql_text(outcome) + " FOR UPDATE"
            )
            if not locked:
                raise OPS.OperationError("mapped Outcome coordination record does not exist")
        rows_to_apply, _ = validate_assignments(connection, "postgres", mapping)
        now = time.time_ns() // 1000
        changed = []
        changed_times = {}
        for assignment, state in rows_to_apply:
            if state["action"] == "noop":
                continue
            table = "work_items" if assignment["kind"] == "task" else "issues"
            record_key = "work_item_id" if assignment["kind"] == "task" else "issue_id"
            updated_at = max(now, int(state["updated_at"]) + 1)
            updated = connection.run(
                "UPDATE " + table + " SET objective_id=" + _sql_text(assignment["objective_id"]) +
                ", version=version+1, updated_at=" + str(updated_at) +
                " WHERE namespace_id=" + _sql_text(assignment["namespace_id"]) +
                " AND outcome_id=" + _sql_text(assignment["outcome_id"]) +
                " AND id=" + _sql_text(assignment[record_key]) +
                " AND objective_id IS NULL AND version=" + str(assignment["expected_version"]) +
                " RETURNING id"
            )
            if len(updated) != 1:
                raise OPS.OperationError("record changed during backfill; transaction rolled back")
            changed.append(assignment)
            changed_times[(assignment["kind"], assignment["namespace_id"], assignment["outcome_id"], assignment[record_key])] = updated_at
        changed_scopes = sorted({(item["namespace_id"], item["outcome_id"]) for item in changed})
        after_revisions = {}
        command_ids = {}
        revision_indexes = {}
        for namespace, outcome in changed_scopes:
            values = connection.run(
                "UPDATE outcome_coordination SET state_revision=state_revision+1 WHERE namespace_id=" +
                _sql_text(namespace) + " AND outcome_id=" + _sql_text(outcome) + " RETURNING state_revision"
            )
            if len(values) != 1:
                raise OPS.OperationError("Outcome coordination changed during backfill; transaction rolled back")
            after_revisions[namespace + "/" + outcome] = int(values[0])
            command_ids[(namespace, outcome)] = _uuid7()
            existing = connection.run(
                "SELECT COALESCE(MAX(event_index),-1) FROM domain_events WHERE namespace_id=" +
                _sql_text(namespace) + " AND outcome_id=" + _sql_text(outcome) +
                " AND outcome_revision=" + str(int(values[0]))
            )
            if int(existing[0]) >= 0:
                raise OPS.OperationError("unexpected domain events already exist at the new Outcome revision")
            revision_indexes[(namespace, outcome)] = 0
        operator = mapping["operator"]
        actor_json = json.dumps(operator["actor"], ensure_ascii=False, separators=(",", ":"))
        for assignment in changed:
            scope = (assignment["namespace_id"], assignment["outcome_id"])
            kind = assignment["kind"]
            record_id = assignment.get("work_item_id", assignment.get("issue_id"))
            recorded_at = changed_times[(kind, scope[0], scope[1], record_id)]
            event_index = revision_indexes[scope]
            revision_indexes[scope] += 1
            event_type = "work_item.objective_changed" if kind == "task" else "issue.objective_backfilled"
            payload = json.dumps({"source": "explicit_ownership_backfill", "mapping_sha256": mapping_sha256,
                                  "objective_id": assignment["objective_id"], "reason": assignment["reason"]},
                                 ensure_ascii=False, separators=(",", ":"))
            literals = [
                _sql_text(_uuid7()), _sql_text(scope[0]), _sql_text(scope[1]), _quoted_text(event_type), "1",
                str(after_revisions[scope[0] + "/" + scope[1]]), str(event_index), _sql_text(record_id),
                _quoted_text("work_item" if kind == "task" else "issue"), str(assignment["expected_version"]),
                str(assignment["expected_version"] + 1), _sql_text(command_ids[scope]),
                _sql_text(operator["principal_id"]), _quoted_text(actor_json), str(recorded_at), "NULL", "NULL",
                _quoted_text("{}"), _quoted_text(payload),
            ]
            connection.run(
                "INSERT INTO domain_events (id,namespace_id,outcome_id,event_type,schema_version,outcome_revision,event_index,"
                "aggregate_id,aggregate_kind,aggregate_version_before,aggregate_version_after,command_id,principal_id,actor_json,"
                "recorded_at,correlation_id,causation_id,execution_context_json,payload_json) VALUES (" + ",".join(literals) + ")"
            )
        connection.commit()
        return {
            "changed_count": len(changed),
            "noop_count": len(rows_to_apply) - len(changed),
            "changed_outcome_revisions": after_revisions,
        }


def write_private_receipt(path, value, create=False):
    target = Path(path).absolute()
    if target.exists() or target.is_symlink():
        if create:
            raise OPS.OperationError("receipt path already exists")
        temp = target.with_name("." + target.name + ".tmp-" + str(os.getpid()) + "-" + os.urandom(4).hex())
        OPS.private_json(temp, value)
        os.replace(temp, target)
    else:
        OPS.private_json(target, value)
    OPS.sync_directory(target.parent)
    return target


def _audit_after_apply(args):
    if args.backend == "sqlite":
        with closing(OPS.sqlite_read(args.sqlite)) as database:
            return build_report(load_snapshot(database, "sqlite"))
    with OPS.PostgresSnapshot(OPS.postgres_env(args.dsn_env), args.schema) as database:
        return build_report(load_snapshot(database, "postgres"))


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--backend", choices=("sqlite", "postgres"), required=True)
    parser.add_argument("--sqlite", help="SQLite database path (read-only unless --apply is supplied)")
    parser.add_argument("--dsn-env", default="WOS_POSTGRES_DSN",
                        help="environment variable name containing the PostgreSQL DSN")
    parser.add_argument("--schema", default="public", help="PostgreSQL schema to inspect")
    parser.add_argument("--mapping", help="operator-reviewed versioned JSON ownership mapping")
    parser.add_argument("--apply", action="store_true", help="apply the explicit mapping in one transaction")
    parser.add_argument("--writers-stopped", action="store_true",
                        help="confirm all WOS writers for this database/schema are stopped")
    parser.add_argument("--receipt", help="new private file for the mapping hash and transaction result (required with --apply)")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", args.dsn_env):
        raise OPS.OperationError("invalid PostgreSQL DSN environment variable name")
    if args.apply and not args.mapping:
        raise OPS.OperationError("--apply requires --mapping")
    if args.apply and not args.writers_stopped:
        raise OPS.OperationError("--apply requires --writers-stopped after all writers are drained")
    if args.apply and not args.receipt:
        raise OPS.OperationError("--apply requires --receipt for an auditable maintenance record")
    if args.writers_stopped and not args.apply:
        raise OPS.OperationError("--writers-stopped is only valid with --apply")
    mapping, digest = parse_mapping(args.mapping) if args.mapping else (None, None)

    if args.backend == "sqlite":
        if not args.sqlite:
            raise OPS.OperationError("--sqlite is required for the SQLite backend")
        if args.apply:
            # An exclusive lock plus CAS protects against stale records. The caller
            # must still drain application writers before entering maintenance mode.
            receipt = {
                "format": "wos-issue-ownership-backfill-receipt-v1",
                "status": "prepared",
                "backend": "sqlite",
                "database": str(Path(args.sqlite).resolve(strict=True)),
                "mapping_sha256": digest,
                "mapping": mapping,
                "prepared_at": OPS.utc(),
            }
            receipt_path = write_private_receipt(args.receipt, receipt, create=True)
            try:
                result = apply_sqlite(args.sqlite, mapping, digest)
                report = _audit_after_apply(args)
                receipt.update({"status": "committed", "completed_at": OPS.utc(), "result": result,
                                "post_apply_orphan_task_count": report["orphan_task_count"],
                                "post_apply_orphan_issue_count": report["orphan_issue_count"]})
                write_private_receipt(receipt_path, receipt)
            except Exception as error:
                receipt.update({"status": "failed_or_unknown", "completed_at": OPS.utc(),
                                "error": str(error)[:500]})
                write_private_receipt(receipt_path, receipt)
                raise
            report.update({"mapping_sha256": digest, "mapping_applied": True,
                           "read_only": False, "mapping_result": result, "receipt": str(receipt_path),
                           "ownership_orphan_checkpoint_passed": report["orphan_task_count"] == 0 and report["orphan_issue_count"] == 0})
        else:
            with closing(OPS.sqlite_read(args.sqlite)) as database:
                report = build_report(load_snapshot(database, "sqlite"))
                if mapping:
                    _require_backfill_schema(database, "sqlite")
                    rows_to_apply, scopes = validate_assignments(database, "sqlite", mapping)
                    report.update({"mapping_sha256": digest,
                                   "mapping_would_apply": sum(1 for _, state in rows_to_apply if state["action"] == "apply"),
                                   "mapping_noop_count": sum(1 for _, state in rows_to_apply if state["action"] == "noop"),
                                   "mapping_outcome_scopes": len(scopes),
                                   "mapping_assignments": [
                                       {"kind": item["kind"], "namespace_id": item["namespace_id"],
                                        "outcome_id": item["outcome_id"],
                                        "record_id": item.get("work_item_id", item.get("issue_id")),
                                        "objective_id": item["objective_id"], "action": state["action"]}
                                       for item, state in rows_to_apply]})
    else:
        if args.sqlite:
            raise OPS.OperationError("--sqlite cannot be used with the PostgreSQL backend")
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]{0,62}", args.schema):
            raise OPS.OperationError("invalid PostgreSQL schema name")
        environment = OPS.postgres_env(args.dsn_env)
        if args.apply:
            receipt = {
                "format": "wos-issue-ownership-backfill-receipt-v1",
                "status": "prepared",
                "backend": "postgres",
                "schema": args.schema,
                "database": unquote(urlparse(os.environ[args.dsn_env]).path.lstrip("/")),
                "mapping_sha256": digest,
                "mapping": mapping,
                "prepared_at": OPS.utc(),
            }
            receipt_path = write_private_receipt(args.receipt, receipt, create=True)
            try:
                result = apply_postgres(environment, args.schema, mapping, digest)
                report = _audit_after_apply(args)
                receipt.update({"status": "committed", "completed_at": OPS.utc(), "result": result,
                                "post_apply_orphan_task_count": report["orphan_task_count"],
                                "post_apply_orphan_issue_count": report["orphan_issue_count"]})
                write_private_receipt(receipt_path, receipt)
            except Exception as error:
                receipt.update({"status": "failed_or_unknown", "completed_at": OPS.utc(),
                                "error": str(error)[:500]})
                write_private_receipt(receipt_path, receipt)
                raise
            report.update({"mapping_sha256": digest, "mapping_applied": True,
                           "read_only": False, "mapping_result": result, "receipt": str(receipt_path),
                           "ownership_orphan_checkpoint_passed": report["orphan_task_count"] == 0 and report["orphan_issue_count"] == 0})
        else:
            with OPS.PostgresSnapshot(environment, args.schema) as database:
                report = build_report(load_snapshot(database, "postgres"))
                if mapping:
                    _require_backfill_schema(database, "postgres")
                    rows_to_apply, scopes = validate_assignments(database, "postgres", mapping)
                    report.update({"mapping_sha256": digest,
                                   "mapping_would_apply": sum(1 for _, state in rows_to_apply if state["action"] == "apply"),
                                   "mapping_noop_count": sum(1 for _, state in rows_to_apply if state["action"] == "noop"),
                                   "mapping_outcome_scopes": len(scopes),
                                   "mapping_assignments": [
                                       {"kind": item["kind"], "namespace_id": item["namespace_id"],
                                        "outcome_id": item["outcome_id"],
                                        "record_id": item.get("work_item_id", item.get("issue_id")),
                                        "objective_id": item["objective_id"], "action": state["action"]}
                                       for item, state in rows_to_apply]})
    print(json.dumps(report, ensure_ascii=False, sort_keys=True, separators=(",", ":")))


if __name__ == "__main__":
    try:
        main()
    except (OPS.OperationError, sqlite3.Error, OSError, ValueError) as error:
        print("issue ownership audit failed: " + str(error), file=sys.stderr)
        raise SystemExit(2)
