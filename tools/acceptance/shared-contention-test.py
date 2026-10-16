#!/usr/bin/env python3
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
spec=importlib.util.spec_from_file_location('shared_contention',Path(__file__).with_name('shared-contention.py'))
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
class EvidenceContract(unittest.TestCase):
    def fixture(self,extra=(),valid=True):
        row={'delivery_p95_ms':10,'conflicts':2,'retries':2,'accepted_per_second':4,'pool_wait_ms':3,'approval_replay_deduplicated':True}
        values=[{'Action':'output','Output':'SIGNED_LOAD '+json.dumps(row)}]
        if valid:values.append({'Action':'pass','Test':'TestSignedWorkReviewLoadMatrix/outcomes=1/history=0/consumers=2'})
        return values+list(extra)
    def parse(self,values):
        with tempfile.TemporaryDirectory() as root:
            path=Path(root)/'result.jsonl';path.write_text(''.join(json.dumps(v)+'\n' for v in values))
            return module.parse_load(path)
    def test_complete_executed_result(self):
        self.assertEqual(self.parse(self.fixture())['retries'],2)
    def test_fail_skip_partial_and_duplicate_rejected(self):
        for values in [self.fixture([{'Action':'fail'}]),self.fixture([{'Action':'skip'}]),self.fixture(valid=False),self.fixture()+self.fixture()]:
            with self.assertRaises(ValueError):self.parse(values)
    def test_incomplete_metrics_rejected(self):
        with self.assertRaises(ValueError):self.parse([{'Action':'output','Output':'SIGNED_LOAD {}'},{'Action':'pass','Test':'Test/case/consumers=2'}])
if __name__=='__main__':unittest.main()
