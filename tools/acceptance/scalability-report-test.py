#!/usr/bin/env python3
"""Evidence parser rejects incomplete or mislabelled scalability measurements."""
import importlib.util
import json
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('scalability_report', Path(__file__).with_name('scalability-report.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class EvidenceContract(unittest.TestCase):
    def setUp(self):
        self.expected = {'tenants': 1, 'outcomes': 4, 'active_outcomes': 4,
                         'service_replicas': 2, 'consumers': 4,
                         'accepted_historical_tasks_per_outcome': 16,
                         'accepted_deliveries': 8, 'verified_outbox_deliveries': 72}
        self.row = dict(self.expected, accepted_per_second=2, delivery_p50_ms=3,
                        delivery_p95_ms=4, delivery_p99_ms=5, read_p50_ms=1,
                        read_p95_ms=2, read_p99_ms=3, transaction_wait_p95_ms=1,
                        guard_wait_p95_ms=1, pool_wait_ms=1, pool_wait_count=1,
                        conflicts=2, retries=2, read_transaction_conflicts=3, concurrent_reads=20, read_max_bytes=2048,
                        approval_replay_deduplicated=True, version_collision_winners=1,
                        version_collision_rejections=3, transport_measured=False)

    def log(self, row=None, extra=()):
        events = [{'Action': 'output', 'Output': 'SIGNED_LOAD '+json.dumps(row or self.row)},
                  {'Action': 'pass', 'Test': 'TestScalableSignedWorkMatrix/tenants=1/replicas=2'}]
        return '\n'.join(json.dumps(item) for item in events+list(extra))

    def test_actual_complete_result(self):
        self.assertEqual(module.parse_result(self.log(), self.expected)['accepted_deliveries'], 8)

    def test_skip_failure_missing_and_duplicate_evidence(self):
        for log in (self.log(extra=[{'Action': 'skip'}]), self.log(extra=[{'Action': 'fail'}]),
                    self.log().splitlines()[0], self.log()+'\n'+self.log()):
            with self.subTest(log=log), self.assertRaises(ValueError):
                module.parse_result(log, self.expected)

    def test_dimensions_and_correctness_are_not_inferred(self):
        for key, value in (('service_replicas', 1), ('active_outcomes', 1),
                           ('accepted_deliveries', 7), ('version_collision_winners', 2),
                           ('version_collision_rejections', 2), ('approval_replay_deduplicated', False),
                           ('transport_measured', True), ('concurrent_reads', 0),
                           ('read_max_bytes', 262145), ('delivery_p95_ms', float('nan')),
                           ('read_p95_ms', 0)):
            with self.subTest(key=key), self.assertRaises(ValueError):
                module.parse_result(self.log(dict(self.row, **{key: value})), self.expected)


if __name__ == '__main__':
    unittest.main()
