import contextlib
import hashlib
import importlib.util
import io
from pathlib import Path
import sqlite3
import tempfile
import unittest

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
        import json
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


if __name__ == "__main__":
    unittest.main()
