import contextlib
import hashlib
import importlib.util
import io
from pathlib import Path
import sqlite3
import tempfile
import unittest
import json
import os

SCRIPT = Path(__file__).with_name("issue_ownership_backfill.py")
SPEC = importlib.util.spec_from_file_location("issue_ownership_backfill", SCRIPT)
AUDIT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(AUDIT)


class ReadOnlyIssueOwnershipAudit(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.db_path = Path(self.temp.name) / "workspace.sqlite3"
        with sqlite3.connect(self.db_path) as db:
            db.executescript("""
                CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);
                INSERT INTO schema_migrations VALUES (33);
                CREATE TABLE outcomes(id TEXT PRIMARY KEY, namespace_id TEXT NOT NULL);
                CREATE TABLE objectives(namespace_id TEXT, outcome_id TEXT, id TEXT, title TEXT);
                CREATE TABLE work_items(namespace_id TEXT, outcome_id TEXT, id TEXT, title TEXT,
                    lifecycle TEXT, objective_id TEXT);
                CREATE TABLE issues(namespace_id TEXT, outcome_id TEXT, id TEXT, title TEXT,
                    lifecycle TEXT);
                CREATE TABLE issue_affected_refs(namespace_id TEXT, outcome_id TEXT, issue_id TEXT,
                    affected_id TEXT, affected_kind TEXT);
                CREATE TABLE relations(namespace_id TEXT, outcome_id TEXT, id TEXT, source_id TEXT,
                    source_kind TEXT, target_id TEXT, target_kind TEXT, relation_type TEXT,
                    strength TEXT, lifecycle TEXT);
                INSERT INTO outcomes VALUES ('outcome-1','namespace-1'), ('outcome-2','namespace-1');
                INSERT INTO objectives VALUES
                    ('namespace-1','outcome-1','objective-a','Improve login'),
                    ('namespace-1','outcome-1','objective-b','Document recovery');
                INSERT INTO work_items VALUES
                    ('namespace-1','outcome-1','orphan-task','Unplaced task','backlog',NULL),
                    ('namespace-1','outcome-1','owned-task','Owned task','todo','objective-a');
                INSERT INTO issues VALUES
                    ('namespace-1','outcome-1','orphan-issue-task','Problem with owned work','open'),
                    ('namespace-1','outcome-1','orphan-issue-unmapped','Unmapped problem','investigating');
                INSERT INTO issue_affected_refs VALUES
                    ('namespace-1','outcome-1','orphan-issue-task','owned-task','work_item'),
                    ('namespace-1','outcome-1','orphan-issue-unmapped','orphan-task','work_item');
                INSERT INTO relations VALUES
                    ('namespace-1','outcome-1','dependency-1','owned-task','work_item',
                     'orphan-task','work_item','depends_on','hard','active');
            """)

    def snapshot_hash(self):
        return hashlib.sha256(self.db_path.read_bytes()).hexdigest()

    def test_report_is_deterministic_scoped_and_does_not_assign_or_write(self):
        before = self.snapshot_hash()
        output_a, output_b = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(output_a):
            AUDIT.main(["--backend", "sqlite", "--sqlite", str(self.db_path)])
        with contextlib.redirect_stdout(output_b):
            AUDIT.main(["--backend", "sqlite", "--sqlite", str(self.db_path)])
        self.assertEqual(output_a.getvalue(), output_b.getvalue())
        self.assertEqual(before, self.snapshot_hash())
        report = json.loads(output_a.getvalue())
        self.assertEqual(report["schema_version"], 33)
        self.assertTrue(report["read_only"])
        self.assertFalse(report["mapping_applied"])
        self.assertEqual(report["orphan_task_count"], 1)
        self.assertEqual(report["orphan_issue_count"], 2)
        self.assertEqual(report["outcome_count_with_orphans"], 1)
        scope = report["outcomes"][0]
        self.assertEqual(scope["outcome_id"], "outcome-1")
        self.assertEqual(scope["orphan_tasks"][0]["objective_candidates"], [
            {"objective_id": "objective-a", "title": "Improve login", "basis": "same_outcome_only"},
            {"objective_id": "objective-b", "title": "Document recovery", "basis": "same_outcome_only"},
        ])
        self.assertEqual(scope["orphan_tasks"][0]["dependency_edges"][0]["relation_id"], "dependency-1")
        issue_with_owner_hint = next(item for item in scope["orphan_issues"] if item["issue_id"] == "orphan-issue-task")
        self.assertEqual(issue_with_owner_hint["objective_candidates_for_human_review"][0]["objective_id"], "objective-a")
        self.assertEqual(issue_with_owner_hint["objective_candidates_for_human_review"][0]["basis"], "affected_task_current_objective")
        no_hint = next(item for item in scope["orphan_issues"] if item["issue_id"] == "orphan-issue-unmapped")
        self.assertEqual(no_hint["objective_candidates_for_human_review"], [])
        with self.assertRaises(sqlite3.OperationalError):
            with contextlib.closing(AUDIT.OPS.sqlite_read(self.db_path)) as readonly:
                readonly.execute("UPDATE work_items SET objective_id='objective-a'")

    def test_rejects_unmigrated_or_wrong_schema_without_changes(self):
        with sqlite3.connect(self.db_path) as db:
            db.execute("DELETE FROM schema_migrations")
        before = self.snapshot_hash()
        with self.assertRaises(AUDIT.OPS.OperationError):
            with contextlib.closing(AUDIT.OPS.sqlite_read(self.db_path)) as readonly:
                AUDIT.load_snapshot(readonly, "sqlite")
        self.assertEqual(before, self.snapshot_hash())


class ExplicitOwnershipBackfill(unittest.TestCase):
    namespace = "0199edc4-0000-7000-8000-000000000001"
    outcome = "0199edc4-0000-7000-8000-000000000002"
    objective = "0199edc4-0000-7000-8000-000000000003"
    task = "0199edc4-0000-7000-8000-000000000004"
    issue = "0199edc4-0000-7000-8000-000000000005"
    principal = "0199edc4-0000-7000-8000-000000000006"
    other_objective = "0199edc4-0000-7000-8000-000000000007"

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.db_path = root / "workspace.sqlite3"
        self.mapping_path = root / "mapping.json"
        self.receipt_path = root / "receipt.json"
        with sqlite3.connect(self.db_path) as db:
            db.executescript("""
                CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);
                INSERT INTO schema_migrations VALUES (34);
                CREATE TABLE outcomes(id TEXT PRIMARY KEY, namespace_id TEXT NOT NULL);
                CREATE TABLE principals(id TEXT PRIMARY KEY, lifecycle TEXT NOT NULL);
                CREATE TABLE outcome_coordination(namespace_id TEXT, outcome_id TEXT, state_revision INTEGER,
                    PRIMARY KEY(namespace_id,outcome_id));
                CREATE TABLE objectives(namespace_id TEXT, outcome_id TEXT, id TEXT, title TEXT, lifecycle TEXT);
                CREATE TABLE work_items(namespace_id TEXT, outcome_id TEXT, id TEXT, title TEXT,
                    lifecycle TEXT, objective_id TEXT, version INTEGER, updated_at INTEGER, lease_claim_id TEXT);
                CREATE TABLE issues(namespace_id TEXT, outcome_id TEXT, id TEXT, title TEXT,
                    lifecycle TEXT, objective_id TEXT, version INTEGER, updated_at INTEGER);
                CREATE TABLE work_contracts(namespace_id TEXT, outcome_id TEXT, work_item_id TEXT, status TEXT);
                CREATE TABLE issue_affected_refs(namespace_id TEXT, outcome_id TEXT, issue_id TEXT,
                    affected_id TEXT, affected_kind TEXT);
                CREATE TABLE relations(namespace_id TEXT, outcome_id TEXT, id TEXT, source_id TEXT,
                    source_kind TEXT, target_id TEXT, target_kind TEXT, relation_type TEXT,
                    strength TEXT, lifecycle TEXT);
                CREATE TABLE domain_events(id TEXT PRIMARY KEY, namespace_id TEXT, outcome_id TEXT,
                    event_type TEXT, schema_version INTEGER, outcome_revision INTEGER, event_index INTEGER,
                    aggregate_id TEXT, aggregate_kind TEXT, aggregate_version_before INTEGER,
                    aggregate_version_after INTEGER, command_id TEXT, principal_id TEXT, actor_json TEXT,
                    recorded_at INTEGER, correlation_id TEXT, causation_id TEXT, execution_context_json TEXT,
                    payload_json TEXT, UNIQUE(namespace_id,outcome_id,outcome_revision,event_index));
                INSERT INTO principals VALUES ('""" + self.principal + """','active');
                INSERT INTO outcomes VALUES ('""" + self.outcome + """','""" + self.namespace + """');
                INSERT INTO outcome_coordination VALUES ('""" + self.namespace + """','""" + self.outcome + """',7);
                INSERT INTO objectives VALUES ('""" + self.namespace + """','""" + self.outcome + """',
                    '""" + self.objective + """','Improve login','planned');
                INSERT INTO objectives VALUES ('""" + self.namespace + """','""" + self.outcome + """',
                    '""" + self.other_objective + """','Improve recovery','planned');
                INSERT INTO work_items VALUES ('""" + self.namespace + """','""" + self.outcome + """',
                    '""" + self.task + """','Add login test','backlog',NULL,3,100,NULL);
                INSERT INTO issues VALUES ('""" + self.namespace + """','""" + self.outcome + """',
                    '""" + self.issue + """','Missing login feedback','open',NULL,4,100);
            """)
        self.mapping = {
            "mapping_version": 1,
            "operator": {"principal_id": self.principal,
                         "actor": {"kind": "human", "provider": "local", "id": "operator", "display_name": "Database operator"}},
            "assignments": [
                {"kind": "task", "namespace_id": self.namespace, "outcome_id": self.outcome,
                 "work_item_id": self.task, "objective_id": self.objective, "expected_version": 3,
                 "reason": "The recorded acceptance test belongs to the login Objective."},
                {"kind": "issue", "namespace_id": self.namespace, "outcome_id": self.outcome,
                 "issue_id": self.issue, "objective_id": self.objective, "expected_version": 4,
                 "reason": "The issue report concerns this Objective's login flow."},
            ],
        }
        self.write_mapping(self.mapping)

    def write_mapping(self, mapping):
        self.mapping_path.write_text(json.dumps(mapping), encoding="utf-8")

    def snapshot_hash(self):
        return hashlib.sha256(self.db_path.read_bytes()).hexdigest()

    def apply(self, receipt=None):
        target = receipt or self.receipt_path
        with contextlib.redirect_stdout(io.StringIO()) as output:
            AUDIT.main(["--backend", "sqlite", "--sqlite", str(self.db_path), "--mapping",
                        str(self.mapping_path), "--apply", "--writers-stopped", "--receipt", str(target)])
        return json.loads(output.getvalue())

    def test_dry_run_and_apply_are_explicit_auditable_and_replayable(self):
        before = self.snapshot_hash()
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            AUDIT.main(["--backend", "sqlite", "--sqlite", str(self.db_path), "--mapping", str(self.mapping_path)])
        preview = json.loads(output.getvalue())
        self.assertEqual(preview["mapping_would_apply"], 2)
        self.assertFalse(preview["mapping_applied"])
        self.assertEqual(before, self.snapshot_hash())

        result = self.apply()
        self.assertTrue(result["mapping_applied"])
        self.assertEqual(result["mapping_result"]["changed_count"], 2)
        self.assertEqual(result["mapping_result"]["noop_count"], 0)
        self.assertEqual(result["orphan_task_count"], 0)
        self.assertEqual(result["orphan_issue_count"], 0)
        receipt = json.loads(self.receipt_path.read_text(encoding="utf-8"))
        self.assertEqual(receipt["status"], "committed")
        self.assertEqual(receipt["mapping_sha256"], result["mapping_sha256"])
        self.assertEqual(len(receipt["mapping"]["assignments"][0]["reason"]), 60)
        with sqlite3.connect(self.db_path) as db:
            task = db.execute("SELECT objective_id,version FROM work_items WHERE id=?", (self.task,)).fetchone()
            issue = db.execute("SELECT objective_id,version FROM issues WHERE id=?", (self.issue,)).fetchone()
            revision = db.execute("SELECT state_revision FROM outcome_coordination WHERE outcome_id=?",
                                  (self.outcome,)).fetchone()[0]
        self.assertEqual(task, (self.objective, 4))
        self.assertEqual(issue, (self.objective, 5))
        self.assertEqual(revision, 8, "two ownership changes in one Outcome advance its revision once")
        with sqlite3.connect(self.db_path) as db:
            events = db.execute("SELECT event_type,outcome_revision,event_index,principal_id,payload_json "
                                "FROM domain_events ORDER BY event_index").fetchall()
        self.assertEqual([row[:3] for row in events], [
            ("work_item.objective_changed", 8, 0), ("issue.objective_backfilled", 8, 1),
        ])
        self.assertTrue(all(row[3] == self.principal for row in events))
        self.assertIn(result["mapping_sha256"], events[0][4])

        replay = self.apply(self.db_path.parent / "replay-receipt.json")
        self.assertEqual(replay["mapping_result"]["changed_count"], 0)
        self.assertEqual(replay["mapping_result"]["noop_count"], 2)
        with sqlite3.connect(self.db_path) as db:
            self.assertEqual(db.execute("SELECT state_revision FROM outcome_coordination WHERE outcome_id=?",
                                        (self.outcome,)).fetchone()[0], 8)

    def test_invalid_second_assignment_rolls_back_the_entire_mapping(self):
        mapping = json.loads(json.dumps(self.mapping))
        mapping["assignments"][1]["expected_version"] = 99
        self.write_mapping(mapping)
        before = self.snapshot_hash()
        with self.assertRaises(AUDIT.OPS.OperationError):
            self.apply()
        self.assertEqual(before, self.snapshot_hash())
        self.assertEqual(json.loads(self.receipt_path.read_text())["status"], "failed_or_unknown")

    def test_conflicting_owner_active_task_and_invalid_scope_fail_closed(self):
        with sqlite3.connect(self.db_path) as db:
            db.execute("UPDATE work_items SET objective_id=? WHERE id=?", (self.other_objective, self.task))
        before = self.snapshot_hash()
        with self.assertRaises(AUDIT.OPS.OperationError):
            self.apply(self.db_path.parent / "conflict-receipt.json")
        self.assertEqual(before, self.snapshot_hash())

        with sqlite3.connect(self.db_path) as db:
            db.execute("UPDATE work_items SET objective_id=NULL, lifecycle='todo' WHERE id=?", (self.task,))
        before = self.snapshot_hash()
        with self.assertRaises(AUDIT.OPS.OperationError):
            self.apply(self.db_path.parent / "todo-receipt.json")
        self.assertEqual(before, self.snapshot_hash())

        with sqlite3.connect(self.db_path) as db:
            db.execute("UPDATE work_items SET lifecycle='backlog' WHERE id=?", (self.task,))
            db.execute("INSERT INTO work_contracts VALUES (?,?,?,'active')",
                       (self.namespace, self.outcome, self.task))
        with self.assertRaises(AUDIT.OPS.OperationError):
            self.apply(self.db_path.parent / "active-contract-receipt.json")

    def test_terminal_issue_owner_and_self_objective_dependency_fail_closed(self):
        with sqlite3.connect(self.db_path) as db:
            db.execute("UPDATE objectives SET lifecycle='achieved' WHERE id=?", (self.objective,))
        before = self.snapshot_hash()
        with self.assertRaises(AUDIT.OPS.OperationError):
            self.apply(self.db_path.parent / "terminal-objective-receipt.json")
        self.assertEqual(before, self.snapshot_hash())

        with sqlite3.connect(self.db_path) as db:
            db.execute("UPDATE objectives SET lifecycle='planned' WHERE id=?", (self.objective,))
            db.execute("INSERT INTO relations VALUES (?,?,?,?,?,?,?,?,?,?)",
                       (self.namespace, self.outcome, "0199edc4-0000-7000-8000-000000000008",
                        self.task, "work_item", self.objective, "objective", "depends_on", "hard", "active"))
        before = self.snapshot_hash()
        with self.assertRaises(AUDIT.OPS.OperationError):
            self.apply(self.db_path.parent / "self-dependency-receipt.json")
        self.assertEqual(before, self.snapshot_hash())

    def test_apply_requires_mapping_writers_stopped_and_private_receipt(self):
        with self.assertRaises(AUDIT.OPS.OperationError):
            AUDIT.main(["--backend", "sqlite", "--sqlite", str(self.db_path), "--mapping", str(self.mapping_path), "--apply"])


class PostgresExplicitOwnershipBackfill(unittest.TestCase):
    @unittest.skipUnless(os.environ.get("WOS_TEST_POSTGRES_DSN"), "set WOS_TEST_POSTGRES_DSN for PostgreSQL backfill integration")
    def test_apply_replay_event_and_rollback(self):
        env = AUDIT.OPS.postgres_env("WOS_TEST_POSTGRES_DSN")
        schema = "wos_backfill_" + AUDIT._uuid7().replace("-", "")[:24]
        namespace, outcome, objective, task, issue, principal = [AUDIT._uuid7() for _ in range(6)]
        q = AUDIT._sql_text
        try:
            with AUDIT.PostgresWriter(env, schema) as db:
                db.run("CREATE SCHEMA " + AUDIT.OPS.identifier(schema))
                db.run("CREATE TABLE schema_migrations(version integer primary key); INSERT INTO schema_migrations VALUES (34)")
                db.run("CREATE TABLE outcomes(id text primary key, namespace_id text not null)")
                db.run("CREATE TABLE principals(id text primary key, lifecycle text not null)")
                db.run("CREATE TABLE outcome_coordination(namespace_id text, outcome_id text, state_revision bigint, primary key(namespace_id,outcome_id))")
                db.run("CREATE TABLE objectives(namespace_id text, outcome_id text, id text, title text, lifecycle text)")
                db.run("CREATE TABLE work_items(namespace_id text, outcome_id text, id text, title text, lifecycle text, objective_id text, version bigint, updated_at bigint, lease_claim_id text)")
                db.run("CREATE TABLE issues(namespace_id text, outcome_id text, id text, title text, lifecycle text, objective_id text, version bigint, updated_at bigint)")
                db.run("CREATE TABLE work_contracts(namespace_id text, outcome_id text, work_item_id text, status text)")
                db.run("CREATE TABLE issue_affected_refs(namespace_id text, outcome_id text, issue_id text, affected_id text, affected_kind text)")
                db.run("CREATE TABLE relations(namespace_id text, outcome_id text, id text, source_id text, source_kind text, target_id text, target_kind text, relation_type text, strength text, lifecycle text)")
                db.run("CREATE TABLE domain_events(id text primary key, namespace_id text, outcome_id text, event_type text, schema_version int, outcome_revision bigint, event_index int, aggregate_id text, aggregate_kind text, aggregate_version_before bigint, aggregate_version_after bigint, command_id text, principal_id text, actor_json text, recorded_at bigint, correlation_id text, causation_id text, execution_context_json text, payload_json text, unique(namespace_id,outcome_id,outcome_revision,event_index))")
                db.run("INSERT INTO outcomes VALUES (" + q(outcome) + "," + q(namespace) + ")")
                db.run("INSERT INTO principals VALUES (" + q(principal) + ",'active')")
                db.run("INSERT INTO outcome_coordination VALUES (" + q(namespace) + "," + q(outcome) + ",7)")
                db.run("INSERT INTO objectives VALUES (" + q(namespace) + "," + q(outcome) + "," + q(objective) + ",'Improve login','planned')")
                db.run("INSERT INTO work_items VALUES (" + q(namespace) + "," + q(outcome) + "," + q(task) + ",'Add test','backlog',NULL,3,100,NULL)")
                db.run("INSERT INTO issues VALUES (" + q(namespace) + "," + q(outcome) + "," + q(issue) + ",'Missing feedback','open',NULL,4,100)")
                db.commit()
            mapping = {"mapping_version": 1,
                       "operator": {"principal_id": principal, "actor": {"kind": "human", "provider": "local", "id": "operator"}},
                       "assignments": [
                           {"kind": "task", "namespace_id": namespace, "outcome_id": outcome, "work_item_id": task,
                            "objective_id": objective, "expected_version": 3, "reason": "Reviewed Task ownership."},
                           {"kind": "issue", "namespace_id": namespace, "outcome_id": outcome, "issue_id": issue,
                            "objective_id": objective, "expected_version": 4, "reason": "Reviewed Issue ownership."},
                       ]}
            with AUDIT.OPS.PostgresSnapshot(env, schema) as db:
                AUDIT._require_backfill_schema(db, "postgres")
                self.assertEqual(len(AUDIT.validate_assignments(db, "postgres", mapping)[0]), 2)
            digest = "a" * 64
            result = AUDIT.apply_postgres(env, schema, mapping, digest)
            self.assertEqual((result["changed_count"], result["noop_count"]), (2, 0))
            with AUDIT.OPS.PostgresSnapshot(env, schema) as db:
                report = AUDIT.build_report(AUDIT.load_snapshot(db, "postgres"))
                events = db.run("SELECT row_to_json(a)::text FROM (SELECT event_type,outcome_revision,event_index,payload_json FROM domain_events ORDER BY event_index) a")
                self.assertEqual((report["orphan_task_count"], report["orphan_issue_count"]), (0, 0))
                self.assertEqual(len(events), 2)
                self.assertIn(digest, events[0])
            replay = AUDIT.apply_postgres(env, schema, mapping, digest)
            self.assertEqual((replay["changed_count"], replay["noop_count"]), (0, 2))

            # Simulate a legacy record modified by another writer after the
            # reviewed map was generated; the stale CAS must abort the batch.
            with AUDIT.PostgresWriter(env, schema) as db:
                db.run("UPDATE work_items SET objective_id=NULL,version=4 WHERE id=" + q(task))
                db.commit()
            stale = json.loads(json.dumps(mapping))
            with self.assertRaises(AUDIT.OPS.OperationError):
                AUDIT.apply_postgres(env, schema, stale, digest)
            with AUDIT.OPS.PostgresSnapshot(env, schema) as db:
                self.assertEqual(db.run("SELECT objective_id FROM work_items WHERE id=" + q(task)), [""])
        finally:
            with AUDIT.PostgresWriter(env, schema) as db:
                db.run("DROP SCHEMA IF EXISTS " + AUDIT.OPS.identifier(schema) + " CASCADE")
                db.commit()


if __name__ == "__main__":
    unittest.main()
