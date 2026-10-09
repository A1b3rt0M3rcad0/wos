import copy
import importlib.util
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location("measurement", Path(__file__).with_name("context-cost-report.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def fixture():
    # Synthetic unit-test data only; never an execution or billing receipt.
    attempt = {"id": "execute-1", "task_id": "t1", "phase": "execution", "cost": "2.50",
               "cost_evidence": "provider_usage", "cost_evidence_ref": "test-only:usage", "currency": "USD"}
    return {"schema": 1, "source_commit": "a" * 40,
            "tasks": [{"id": "t1", "outcome_id": "o1", "quality_contract_digest": "sha256:" + "b" * 64}],
            "strategies": {"C": {"model_harness_configuration_digest": "sha256:" + "c" * 64,
                                  "attempts": [attempt], "accepted_tasks": ["t1"], "accepted_outcomes": ["o1"],
                                  "quality_evidence_ref": "test-only:quality", "integration_evidence_ref": "test-only:integration"}}}


class MeasurementTests(unittest.TestCase):
    def test_failed_attempt_and_review_are_in_cost_denominator(self):
        data = fixture()
        for identifier, phase, cost in [("retry", "execution", "1.25"), ("review", "review", "0.50")]:
            attempt = copy.deepcopy(data["strategies"]["C"]["attempts"][0])
            attempt.update(id=identifier, phase=phase, cost=cost)
            data["strategies"]["C"]["attempts"].append(attempt)
        result = module.report(data)
        self.assertEqual(result["strategies"]["C"]["cost_per_accepted_task"], "4.25")
        self.assertFalse(result["comparison_available"])
        self.assertIsNone(result["savings_percentage"])

    def test_partial_usage_and_unknown_cost_are_not_invented(self):
        data = fixture()
        data["strategies"]["C"]["attempts"][0].update(cost=None, payload_bytes=5000)
        result = module.report(data)["strategies"]["C"]
        self.assertIsNone(result["total_cost"])
        self.assertIsNone(result["input_tokens"])

    def test_no_accepted_task_has_no_cost_per_acceptance(self):
        data = fixture()
        data["strategies"]["C"].update(accepted_tasks=[], accepted_outcomes=[])
        self.assertIsNone(module.report(data)["strategies"]["C"]["cost_per_accepted_task"])

    def test_model_change_is_not_a_wos_control(self):
        data = fixture()
        data["strategies"]["B"] = copy.deepcopy(data["strategies"]["C"])
        data["strategies"]["B"]["model_harness_configuration_digest"] = "sha256:" + "d" * 64
        with self.assertRaisesRegex(AssertionError, "identical models"):
            module.report(data)

    def test_missing_tasks_mixed_currency_and_missing_provenance_rejected(self):
        for change in ["missing_task", "mixed_currency", "no_provenance", "nan", "duplicate"]:
            with self.subTest(change=change):
                data = fixture()
                attempt = data["strategies"]["C"]["attempts"][0]
                if change == "missing_task":
                    data["tasks"].append(dict(data["tasks"][0], id="t2"))
                elif change == "mixed_currency":
                    data["strategies"]["C"]["attempts"].append(dict(attempt, id="review", phase="review", currency="BRL"))
                elif change == "no_provenance":
                    attempt.pop("cost_evidence_ref")
                elif change == "nan":
                    attempt["cost"] = "NaN"
                else:
                    data["strategies"]["C"]["attempts"].append(copy.deepcopy(attempt))
                with self.assertRaises(AssertionError):
                    module.report(data)


if __name__ == "__main__":
    unittest.main()
