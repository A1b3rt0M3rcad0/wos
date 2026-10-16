import hashlib
import importlib.util
from pathlib import Path
import tempfile
import unittest

spec=importlib.util.spec_from_file_location('v042_upgrade',Path(__file__).with_name('v042-operational-upgrade.py'))
upgrade=importlib.util.module_from_spec(spec);spec.loader.exec_module(upgrade)


class CandidateMigrationContract(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root=Path(self.temp.name)
        for backend in ('sqlite','postgres'):
            directory=self.directory(backend)
            directory.mkdir(parents=True)
            for version in range(1,4):
                (directory/f'{version:04d}_migration.sql').write_text(f'-- {backend} migration {version}\n')

    def directory(self,backend):
        return self.root/'packages/wos-core/storage'/backend/'migrations'

    def test_new_additive_migration_advances_expected_schema_without_a_frozen_number(self):
        original=upgrade.source_migrations(self.root)
        self.assertEqual(upgrade.verify_current_migrations(original['sqlite'],original['sqlite']),3)
        for backend in ('sqlite','postgres'):
            (self.directory(backend)/'0004_activity.sql').write_text(f'-- {backend} additive activity\n')
        current=upgrade.source_migrations(self.root)
        self.assertEqual(upgrade.verify_current_migrations(current['postgres'],current['postgres']),4)
        self.assertEqual(current['sqlite'][-1]['checksum'],hashlib.sha256((self.directory('sqlite')/'0004_activity.sql').read_bytes()).hexdigest())
        self.assertNotEqual(current['sqlite'][-1]['checksum'],current['postgres'][-1]['checksum'])
        with self.assertRaisesRegex(AssertionError,'complete candidate source'):
            upgrade.verify_current_migrations(original['sqlite'],current['sqlite'])

    def test_missing_duplicate_malformed_or_different_adapter_chains_fail(self):
        for name in ('missing','duplicate','malformed','different'):
            with self.subTest(name=name):
                directory=self.directory('postgres')
                original=directory/'0002_migration.sql'
                if name=='missing':
                    original.unlink()
                    undo=lambda: original.write_text('-- postgres migration 2\n')
                else:
                    changed=directory/({'duplicate':'0002_duplicate.sql','malformed':'bad.sql','different':'0004_only_postgres.sql'}[name])
                    changed.write_text('-- invalid chain\n')
                    undo=changed.unlink
                try:
                    with self.assertRaises(AssertionError):upgrade.source_migrations(self.root)
                finally:undo()

    def test_applied_migrations_must_match_names_checksums_and_complete_sequence(self):
        expected=upgrade.source_migrations(self.root)['sqlite']
        for field,value in (('version',9),('name','0002_wrong.sql'),('checksum','0'*64)):
            with self.subTest(field=field):
                changed=[dict(item) for item in expected]
                changed[1][field]=value
                with self.assertRaisesRegex(AssertionError,'complete candidate source'):
                    upgrade.verify_current_migrations(changed,expected)
        with self.assertRaisesRegex(AssertionError,'complete candidate source'):
            upgrade.verify_current_migrations(expected[:-1],expected)


if __name__=='__main__':unittest.main()
