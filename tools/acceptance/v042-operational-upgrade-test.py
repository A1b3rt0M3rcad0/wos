import hashlib
import importlib.util
import json
from unittest.mock import patch
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


class TenantCutoverPreservation(unittest.TestCase):
    def test_only_explicit_current_configuration_may_change(self):
        original={
            'namespace_grants':[{'namespace_id':'tenant','principal_id':'admin','permissions_json':'["state:read","namespace:admin"]'}],
            'namespace_work_protocol':[{'namespace_id':'tenant','version':2,'phase':'signed_contracts_v2','writer_epoch':2,'state_json':json.dumps({'namespace_id':'tenant','protocol_version':2,'phase':'signed_contracts_v2','writer_epoch':2,'reason':'namespace_id is authored text'})}],
            'credential_policies':[{'credential_id':'credential','version':1,'state_json':json.dumps({'namespace_id':'tenant','revision':'1','permitted_operations':['namespace:admin'],'metadata':{'namespace_id':'authored'}})}],
            'signed_protocol_facts':[{'namespace_id':'tenant','state_json':'{ "namespace_id": "tenant", "signature":"exact bytes" }'}],
        }
        actual={table:[upgrade.expected_configuration(table,row) for row in rows] for table,rows in original.items()}
        for rows in actual.values():
            for row in rows:
                if isinstance(row.get('state_json'),dict):row['state_json']=json.dumps(row['state_json'])
        upgrade.verify_preserved(original,actual)
        for table,field in [('signed_protocol_facts','state_json'),('namespace_work_protocol','phase')]:
            changed=json.loads(json.dumps(actual))
            changed[table][0][field]+='changed'
            with self.subTest(table=table),self.assertRaisesRegex(AssertionError,'historical row bytes'):
                upgrade.verify_preserved(original,changed)
        self.assertEqual(json.loads(actual['credential_policies'][0]['state_json'])['metadata'],{'namespace_id':'authored'})

    def test_fresh_schema_ledger_checks_checksums_but_preserves_source_applied_time(self):
        original={'schema_migrations':[{'version':1,'name':'0001.sql','checksum':'abc','applied_at':1}]}
        target={'schema_migrations':[{'version':1,'name':'0001.sql','checksum':'abc','applied_at':9}]}
        upgrade.verify_preserved(original,target,fresh_ledger=True)
        with self.assertRaisesRegex(AssertionError,'migration ledger'):
            upgrade.verify_preserved(original,target)
        target['schema_migrations'][0]['checksum']='changed'
        with self.assertRaisesRegex(AssertionError,'migration ledger'):
            upgrade.verify_preserved(original,target,fresh_ledger=True)

    def test_old_runtime_keeps_old_configuration_names(self):
        with tempfile.TemporaryDirectory() as name:
            directory=Path(name)
            (directory/'issuer.seed').write_text('private-fixture-seed')
            with patch.dict('os.environ',{'WOS_TENANT_HOSTS':'must-not-leak','WOS_BOOTSTRAP_TENANT_ID':'unrelated'}):
                old=upgrade.runtime_environment('sqlite','fixture.db',directory,current=False)
                new=upgrade.runtime_environment('sqlite','fixture.db',directory,current=True)
        self.assertEqual(old['WOS_BOOTSTRAP_NAMESPACE_ID'],upgrade.TENANT)
        self.assertNotIn('WOS_BOOTSTRAP_TENANT_ID',old)
        self.assertNotIn('WOS_TENANT_HOSTS',old)
        self.assertEqual(new['WOS_TENANT_ID'],upgrade.TENANT)
        self.assertNotIn('WOS_BOOTSTRAP_NAMESPACE_ID',new)


if __name__=='__main__':unittest.main()
