#!/usr/bin/env python3
"""Read-only audit of Outcomes before an explicit operational-profile migration.

SQLite is opened with mode=ro and query_only. PostgreSQL uses a bounded
repeatable-read, read-only snapshot. The script never changes profile, policy,
ownership, lifecycle, criteria, evidence, or authority, and does not infer owners.
"""

import argparse
from contextlib import closing
import importlib.util
import json
import os
from pathlib import Path
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


def load_snapshot(connection, backend):
    version = int(rows(connection, backend,
        "SELECT COALESCE(MAX(version), 0) AS schema_version FROM schema_migrations"
    )[0]["schema_version"])
    if version < 37:
        raise OPS.OperationError("database must be migrated through schema version 37")

    outcomes = rows(connection, backend,
        "SELECT namespace_id, id AS outcome_id, version, lifecycle, archived_at, "
        "operational_profile, completion_policy FROM outcomes ORDER BY namespace_id, id")
    revisions = rows(connection, backend,
        "SELECT o.namespace_id, o.id AS outcome_id, COALESCE(c.state_revision, 0) AS outcome_revision "
        "FROM outcomes o LEFT JOIN outcome_coordination c "
        "ON c.namespace_id=o.namespace_id AND c.outcome_id=o.id ORDER BY o.namespace_id, o.id")
    objectives = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS objective_id, version, lifecycle, "
        "completion_mode, completion_cycle FROM objectives ORDER BY namespace_id, outcome_id, id")
    work_items = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS work_item_id, version, objective_id, lifecycle, "
        "lease_claim_id, lease_principal_id, lease_expires_at, current_contract_id "
        "FROM work_items ORDER BY namespace_id, outcome_id, id")
    owners = rows(connection, backend,
        "SELECT namespace_id, outcome_id, entity_id, role, COUNT(*) AS actor_count "
        "FROM entity_actor_links WHERE role IN ('owner','assignee') "
        "GROUP BY namespace_id, outcome_id, entity_id, role ORDER BY namespace_id, outcome_id, entity_id, role")
    required_criteria = rows(connection, backend,
        "SELECT c.namespace_id, c.outcome_id, c.owner_id, COUNT(*) AS required_count "
        "FROM success_criteria c JOIN criterion_revisions r "
        "ON r.namespace_id=c.namespace_id AND r.outcome_id=c.outcome_id "
        "AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision "
        "WHERE c.status='active' AND r.required=1 "
        "GROUP BY c.namespace_id, c.outcome_id, c.owner_id "
        "ORDER BY c.namespace_id, c.outcome_id, c.owner_id")
    legacy_contracts = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS contract_id, work_item_id, status, state_json "
        "FROM work_contracts ORDER BY namespace_id, outcome_id, id")
    signed_contracts = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS contract_id, work_item_id, credential_id, status "
        "FROM signed_work_contracts ORDER BY namespace_id, outcome_id, id")
    review_cases = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS review_case_id, work_item_id, status "
        "FROM work_review_cases ORDER BY namespace_id, outcome_id, id")
    review_contracts = rows(connection, backend,
        "SELECT namespace_id, outcome_id, id AS review_contract_id, work_item_id, credential_id, status "
        "FROM work_review_contracts ORDER BY namespace_id, outcome_id, id")
    return (version, outcomes, revisions, objectives, work_items, owners, required_criteria,
            legacy_contracts, signed_contracts, review_cases, review_contracts)


def build_report(snapshot, only_outcome=None):
    (schema_version, outcomes, revisions, objectives, work_items, owners, criteria,
     legacy_contracts, signed_contracts, review_cases, review_contracts) = snapshot
    revision_by_scope = {(r["namespace_id"], r["outcome_id"]): str(r["outcome_revision"]) for r in revisions}
    actor_count = {(r["namespace_id"], r["outcome_id"], r["entity_id"], r["role"]): int(r["actor_count"]) for r in owners}
    required_count = {(r["namespace_id"], r["outcome_id"], r["owner_id"]): int(r["required_count"]) for r in criteria}
    objectives_by_scope = {}
    tasks_by_scope = {}
    for row in objectives:
        objectives_by_scope.setdefault((row["namespace_id"], row["outcome_id"]), []).append(row)
    for row in work_items:
        tasks_by_scope.setdefault((row["namespace_id"], row["outcome_id"]), []).append(row)
    legacy_by_scope = {}
    signed_by_scope = {}
    cases_by_scope = {}
    reviews_by_scope = {}
    for row in legacy_contracts:
        legacy_by_scope.setdefault((row["namespace_id"], row["outcome_id"]), []).append(row)
    for row in signed_contracts:
        signed_by_scope.setdefault((row["namespace_id"], row["outcome_id"]), []).append(row)
    for row in review_cases:
        cases_by_scope.setdefault((row["namespace_id"], row["outcome_id"]), []).append(row)
    for row in review_contracts:
        reviews_by_scope.setdefault((row["namespace_id"], row["outcome_id"]), []).append(row)

    report = []
    for outcome in outcomes:
        namespace, oid = outcome["namespace_id"], outcome["outcome_id"]
        if only_outcome and oid != only_outcome:
            continue
        scope = (namespace, oid)
        outcome_ref = {"kind": "outcome", "id": oid}
        findings = []

        def finding(code, severity, ref, message, remedy):
            findings.append({"code": code, "severity": severity, "ref": ref,
                             "message": message, "remedy": remedy})

        profile = outcome["operational_profile"]
        if profile != "compatibility_v1":
            finding("profile_not_compatibility_v1", "blocked", outcome_ref,
                    "Only compatibility_v1 Outcomes use this migration.",
                    "Use the documented upgrade or recovery procedure for the current profile.")
        if outcome["archived_at"] is not None or outcome["lifecycle"] in ("achieved", "failed", "abandoned"):
            finding("outcome_not_open", "blocked", outcome_ref,
                    "Archived or terminal Outcomes cannot change operational profile.",
                    "Use the supported unarchive or reopen operation, then rerun the audit.")
        if actor_count.get((namespace, oid, oid, "owner"), 0) == 0:
            finding("outcome_owner_missing", "info", outcome_ref, "No Outcome owner is recorded.",
                    "Assign the intended owner through the normal operation; this migration never invents ownership.")
        if required_count.get((namespace, oid, oid), 0) == 0:
            finding("outcome_required_criteria_missing", "info", outcome_ref,
                    "The Outcome has no active required success criteria.",
                    "Define criteria before work begins; this migration will not create them.")

        for objective in objectives_by_scope.get(scope, []):
            ref = {"kind": "objective", "id": objective["objective_id"]}
            if objective["completion_mode"] == "automatic":
                finding("objective_automatic_completion_requires_separate_delegation", "blocked", ref,
                        "An automatic Objective completion mode cannot be reinterpreted by this migration.",
                        "Review the Objective completion policy separately; migration never changes its mode.")
            if actor_count.get((namespace, oid, objective["objective_id"], "owner"), 0) == 0:
                finding("objective_owner_missing", "info", ref, "This Objective has no recorded owner.",
                        "Assign an owner if required; the audit and migration do not assign one.")
            if required_count.get((namespace, oid, objective["objective_id"]), 0) == 0:
                finding("objective_required_criteria_missing", "info", ref,
                        "This Objective has no active required criteria.",
                        "Define its criteria separately; migration does not create them.")

        work_by_id = {task["work_item_id"]: task for task in tasks_by_scope.get(scope, [])}
        for task in work_by_id.values():
            ref = {"kind": "work_item", "id": task["work_item_id"]}
            if task["lease_claim_id"] is not None:
                finding("legacy_lease_unreconciled", "blocked", ref,
                        "This Task has a legacy lease without credential binding.",
                        "Release or explicitly reconcile it through the existing authority protocol; do not infer a credential.")
            if actor_count.get((namespace, oid, task["work_item_id"], "assignee"), 0) == 0:
                finding("task_assignee_missing", "info", ref,
                        "This Task has no recorded assignee.",
                        "Review assignment using the existing operation; migration does not assign Tasks.")
            if required_count.get((namespace, oid, task["work_item_id"]), 0) == 0:
                finding("task_required_criteria_missing", "info", ref,
                        "This Task has no active required criteria.",
                        "Define acceptance criteria separately; migration does not create them.")

        known_contract_ids = set()
        for contract in legacy_by_scope.get(scope, []):
            known_contract_ids.add(contract["contract_id"])
            task = work_by_id.get(contract["work_item_id"])
            current = task is not None and task["current_contract_id"] == contract["contract_id"]
            if contract["status"] != "active" and not current:
                continue
            try:
                state = json.loads(contract["state_json"])
            except (TypeError, json.JSONDecodeError):
                state = {}
            binding = state.get("signed_binding") or {}
            bound = bool(binding.get("credential_id"))
            ref = {"kind": "work_item", "id": contract["work_item_id"]}
            finding("unsigned_work_authority_unbound" if not bound else "work_contract_in_flight", "blocked", ref,
                    "A work contract is active or remains attached to a Task.",
                    "Complete, revoke, or explicitly reconcile the contract, then rerun the audit.")
        for contract in signed_by_scope.get(scope, []):
            known_contract_ids.add(contract["contract_id"])
            task = work_by_id.get(contract["work_item_id"])
            current = task is not None and task["current_contract_id"] == contract["contract_id"]
            if contract["status"] != "active" and not current:
                continue
            ref = {"kind": "work_item", "id": contract["work_item_id"]}
            finding("signed_work_contract_in_flight", "blocked", ref,
                    "Signed execution authority is still active or attached to a Task.",
                    "Finish, revoke, or reconcile it under the signed protocol, then rerun the audit.")
        for task in work_by_id.values():
            if task["current_contract_id"] is not None and task["current_contract_id"] not in known_contract_ids:
                finding("current_work_contract_unresolved", "blocked",
                        {"kind": "work_item", "id": task["work_item_id"]},
                        "This Task references a current contract that cannot be resolved in the audit snapshot.",
                        "Reconcile the contract reference through its owning protocol before migration.")
        for case in cases_by_scope.get(scope, []):
            if case["status"] in ("pending", "in_review"):
                finding("review_case_in_flight", "blocked", {"kind": "work_item", "id": case["work_item_id"]},
                        "A delivery review is pending or in progress.",
                        "Complete or explicitly intervene in the review before migrating the Outcome.")
        for review in reviews_by_scope.get(scope, []):
            if review["status"] == "active":
                finding("review_contract_in_flight", "blocked", {"kind": "work_item", "id": review["work_item_id"]},
                        "Review authority is still active.",
                        "Complete, revoke, or reconcile the review authority under its protocol, then rerun the audit.")

        blockers = [item for item in findings if item["severity"] == "blocked"]
        report.append({
            "scope": {"namespace_id": namespace, "outcome_id": oid},
            "outcome": {"version": str(outcome["version"]), "lifecycle": outcome["lifecycle"],
                        "operational_profile": profile, "completion_policy": outcome["completion_policy"]},
            "target": {"operational_profile": "operational_v2", "completion_policy": "strict_work"},
            "expected_outcome_revision": revision_by_scope.get(scope, "0"),
            "can_migrate": profile == "compatibility_v1" and not blockers,
            "findings": findings,
            "preservation": "No Objective or Task lifecycle, owner, evidence, assessment, conclusion, or protocol is changed; Objective completion modes are not selected by this migration.",
        })
    if only_outcome and not report:
        raise OPS.OperationError("requested Outcome was not found in the audited snapshot")
    return {"schema_version": schema_version, "read_only": True, "target_policy": "strict_work",
            "outcomes": report, "count": len(report)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--backend", choices=("sqlite", "postgres"), required=True)
    parser.add_argument("--sqlite", help="explicit SQLite file (read-only)")
    parser.add_argument("--dsn-env", default="WOS_POSTGRES_DSN", help="environment variable holding the PostgreSQL URI")
    parser.add_argument("--schema", default="public", help="PostgreSQL schema to inspect")
    parser.add_argument("--outcome", help="optional Outcome ID to audit")
    parser.add_argument("--output", help="write the JSON report to this path; default stdout")
    args = parser.parse_args()
    try:
        if args.backend == "sqlite":
            if not args.sqlite:
                parser.error("--sqlite is required with --backend sqlite")
            with closing(OPS.sqlite_read(args.sqlite)) as connection:
                report = build_report(load_snapshot(connection, "sqlite"), args.outcome)
        else:
            if not args.schema.isidentifier():
                parser.error("--schema must be a plain SQL identifier")
            with OPS.PostgresSnapshot(OPS.postgres_env(args.dsn_env), args.schema) as connection:
                report = build_report(load_snapshot(connection, "postgres"), args.outcome)
        rendered = json.dumps(report, indent=2, ensure_ascii=False) + "\n"
        if args.output:
            path = Path(args.output)
            path.write_text(rendered, encoding="utf-8")
            os.chmod(path, 0o600)
        else:
            sys.stdout.write(rendered)
        return 0
    except (OPS.OperationError, OSError, sqlite3.Error) as error:
        print("operational profile audit failed: " + str(error), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
