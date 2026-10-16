import importlib.util
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location('read_report',Path(__file__).with_name('operational-read-report.py'))
report=importlib.util.module_from_spec(spec);spec.loader.exec_module(report)

class StrictReadReport(unittest.TestCase):
    def fixture(self):
        return '\n'.join('BenchmarkOperationalReads/'+name+'/10000-3 40 100 ns/op 80 p50_ns 120 p95_ns 130 p99_ns 0 pool_wait_ns/op 0 pool_waits/op 200 response_bytes 8 sql_statements/op 300 B/op 10 allocs/op' for name in report.CASES)+'\nPASS\n'
    def test_accepts_complete_benchmark_only(self):
        self.assertEqual(set(report.parse(self.fixture(),10000,40,1)),set(report.CASES))
        for changed in (self.fixture().replace('PASS','FAIL'),self.fixture().replace('PASS',''),self.fixture()+self.fixture(),self.fixture().replace('/focal_with_proof/','/wrong_case/')):
            with self.assertRaises(AssertionError):report.parse(changed,10000,40,1)
    def test_rejects_wrong_samples_invalid_percentiles_and_unbounded_response(self):
        for changed in (self.fixture().replace(' 40 ',' 39 '),self.fixture().replace('120 p95_ns','50 p95_ns'),self.fixture().replace('200 response_bytes','262145 response_bytes'),self.fixture().replace('300 B/op','NaN B/op')):
            with self.assertRaises(AssertionError):report.parse(changed,10000,40,1)

if __name__=='__main__':unittest.main()
