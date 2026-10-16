import argparse
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('wos_backup_restore', Path(__file__).with_name('backup_restore.py'))
ops = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ops)

class BackupProcedure(unittest.TestCase):
    # Minimal tool-format fixtures test file/snapshot behavior only. Real WOS
    # signed authority and upgrade/restore journeys are qualified separately.
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.source = self.root/'source.sqlite3'
        self.backup = self.root/'snapshot'
        self.destination = self.root/'restored.sqlite3'
        self.old_umask = os.umask(0o077)
        with sqlite3.connect(self.source) as db:
            db.execute('PRAGMA journal_mode=WAL')
            db.executescript('CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY); INSERT INTO schema_migrations VALUES(29); CREATE TABLE outcome_coordination(state_revision INTEGER); INSERT INTO outcome_coordination VALUES(1); CREATE TABLE notes(id INTEGER PRIMARY KEY,value TEXT);')
            db.executemany('INSERT INTO notes VALUES(?,?)', [(i, 'note '+str(i)) for i in range(10)])
        self.args = argparse.Namespace(backend='sqlite', sqlite=str(self.source), schema='public',
                                      output=str(self.backup), source_commit='b'*40, product_version='fixture',
                                      backup=str(self.backup), manifest_sha256=None, writers_stopped=True,
                                      expected_server_id=None, issuer_fingerprint=None)
    def tearDown(self):
        os.umask(self.old_umask)
        self.temp.cleanup()
    def restored_args(self):
        return argparse.Namespace(**{**vars(self.args), 'sqlite':str(self.destination)})
    def test_online_snapshot_remains_consistent_across_a_concurrent_commit(self):
        original = ops.sqlite_inspect
        def inspect_and_write(source):
            result = original(source)
            with sqlite3.connect(self.source) as writer:
                writer.executemany('INSERT INTO notes VALUES(?,?)', [(i, 'later '+str(i)) for i in range(10,20)])
            return result
        ops.sqlite_inspect = inspect_and_write
        try:
            created = ops.backup(self.args)
        finally:
            ops.sqlite_inspect = original
        restored = ops.restore(self.restored_args())
        self.assertTrue(restored['verified'])
        self.assertFalse(restored['read_only'])
        self.assertTrue(restored['validation_read_only'])
        self.assertGreaterEqual(restored['rto_restore_and_verify_seconds'],0)
        with sqlite3.connect(self.destination) as db:
            self.assertEqual(db.execute('SELECT COUNT(*) FROM notes').fetchone()[0],10)
        with sqlite3.connect(self.source) as db:
            self.assertEqual(db.execute('SELECT COUNT(*) FROM notes').fetchone()[0],20)
        self.assertEqual(created['schema_version'],29)
        self.assertEqual((self.backup.stat().st_mode & 0o777),0o700)
        self.assertEqual(((self.backup/'database.sqlite3').stat().st_mode & 0o777),0o600)
    def test_checksum_failure_happens_before_a_destination_is_created(self):
        ops.backup(self.args)
        with (self.backup/'database.sqlite3').open('ab') as data:
            data.write(b'corrupt')
        with self.assertRaisesRegex(ops.OperationError,'checksum'):
            ops.restore(self.restored_args())
        self.assertFalse(self.destination.exists())
    def test_existing_destination_and_missing_cutover_confirmation_are_refused(self):
        ops.backup(self.args)
        self.destination.write_bytes(b'live-file-must-stay')
        with self.assertRaises(FileExistsError):
            ops.restore(self.restored_args())
        self.assertEqual(self.destination.read_bytes(),b'live-file-must-stay')
        args=self.restored_args();args.writers_stopped=False
        with self.assertRaisesRegex(ops.OperationError,'writers-stopped'):
            ops.restore(args)
    def test_verify_is_read_only_and_detects_changed_rows(self):
        ops.backup(self.args)
        ops.restore(self.restored_args())
        with sqlite3.connect(self.destination) as db:
            db.execute("UPDATE notes SET value='different' WHERE id=1")
        before=ops.file_hash(self.destination)
        with self.assertRaisesRegex(ops.OperationError,'logical state differs'):
            ops.verify(self.restored_args())
        self.assertEqual(before,ops.file_hash(self.destination))
    def test_independent_manifest_pin_rejects_rewritten_metadata(self):
        result=ops.backup(self.args)
        path=self.backup/'manifest.json';manifest=json.loads(path.read_text());manifest['product_version']='changed';path.write_text(json.dumps(manifest))
        sums=json.loads((self.backup/'checksums.json').read_text());sums['manifest.json']=ops.file_hash(path);(self.backup/'checksums.json').write_text(json.dumps(sums))
        with self.assertRaisesRegex(ops.OperationError,'approved manifest SHA256'):
            ops.check_backup(self.backup,result['manifest_sha256'])
    def test_signed_snapshot_requires_an_independently_approved_identity(self):
        identity={'server_id':'0199f016-0000-7000-8000-000000000001','fingerprint':'sha256:'+'a'*64}
        with sqlite3.connect(self.source) as db:
            db.execute('CREATE TABLE server_protocol_identity(singleton INTEGER PRIMARY KEY,state_json TEXT)')
            db.execute('INSERT INTO server_protocol_identity VALUES(1,?)',(json.dumps(identity),))
        ops.backup(self.args)
        args=self.restored_args()
        with self.assertRaisesRegex(ops.OperationError,'independently approved'):
            ops.restore(args)
        self.assertFalse(self.destination.exists())
        args.expected_server_id=identity['server_id'];args.issuer_fingerprint='sha256:'+'f'*64
        with self.assertRaisesRegex(ops.OperationError,'pin differs'):
            ops.restore(args)
        self.assertFalse(self.destination.exists())
        args.issuer_fingerprint=identity['fingerprint']
        self.assertTrue(ops.restore(args)['approved_pin_checked'])

if __name__=='__main__':
    unittest.main()
