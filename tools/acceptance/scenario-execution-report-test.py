import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('execution', Path(__file__).with_name('scenario-execution-report.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ExecutionTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root/'go.mod').write_text('module test-only\n')
        (self.root/'fixture').mkdir()
        (self.root/'fixture/case_test.go').write_text('func TestCase(t *testing.T) {}\n')
        self.matrix = {'scenarios': [{'id': f'T{i:02}', 'required_assertion': 'synthetic unit-test assertion',
                                     'coverage_status': 'mapped_requires_assertion_audit',
                                     'candidate_go_tests': [{'path': 'fixture/case_test.go', 'test': 'TestCase'}],
                                     'other_resources': [], 'limitation': 'synthetic data is not execution evidence'} for i in range(1, 97)]}
        self.log = self.root/'test.jsonl'

    def run_report(self, events, matrix=None):
        self.log.write_text('\n'.join(json.dumps(e) for e in events))
        return module.execution_report(matrix or self.matrix, [self.log], 'a'*40, self.root)

    def test_actual_pass_never_automatically_accepts_a_scenario(self):
        result = self.run_report([{'Action': 'pass', 'Package': 'test-only/fixture', 'Test': 'TestCase'},
                                  {'Action': 'skip', 'Package': 'test-only/fixture', 'Test': 'TestCase/optional'},
                                  {'Action': 'pass', 'Package': 'test-only/fixture'}])
        row = result['scenarios'][0]
        self.assertTrue(row['candidate_test_results'][0]['executed_pass'])
        self.assertEqual(row['candidate_test_results'][0]['skipped_subtests'], ['TestCase/optional'])
        self.assertFalse(row['assertion_audit_complete'])
        self.assertFalse(row['scenario_accepted_automatically'])

    def test_zero_tests_and_failures_rejected(self):
        for events in [[{'Action': 'pass', 'Package': 'test-only/fixture'}],
                       [{'Action': 'fail', 'Package': 'test-only/fixture', 'Test': 'TestCase'}],
                       [{'Action': 'build-fail', 'Package': 'test-only/fixture'}]]:
            with self.subTest(events=events), self.assertRaises(ValueError):
                self.run_report(events)

    def test_missing_mapping_rejected(self):
        matrix = copy.deepcopy(self.matrix)
        matrix['scenarios'][0]['candidate_go_tests'][0]['test'] = 'TestMissing'
        with self.assertRaisesRegex(ValueError, 'missing candidate'):
            self.run_report([{'Action': 'pass', 'Package': 'test-only/fixture', 'Test': 'TestCase'},
                             {'Action': 'pass', 'Package': 'test-only/fixture'}], matrix)

    def test_incomplete_scenario_declaration_rejected(self):
        matrix = copy.deepcopy(self.matrix)
        matrix['scenarios'].pop()
        with self.assertRaisesRegex(ValueError, 'complete ordered'):
            self.run_report([], matrix)


if __name__ == '__main__':
    unittest.main()
