#!/usr/bin/env python3
"""Validate controlled A/B/C measurements; unknown usage stays unknown."""
import argparse
import json
import re
from decimal import Decimal, InvalidOperation
from pathlib import Path

PHASES = {"planning", "execution", "review", "correction", "coordination", "integration"}
METRICS = ("input_tokens", "output_tokens", "cached_tokens", "tool_calls", "human_interventions")


def report(data):
    assert data.get("schema") == 1, "unknown measurement schema"
    assert re.fullmatch(r"[0-9a-f]{40}", data["source_commit"]), "full immutable source required"
    tasks = data["tasks"]
    assert tasks and len({t["id"] for t in tasks}) == len(tasks), "unique task set required"
    assert all(t.get("quality_contract_digest", "").startswith("sha256:") and
               re.fullmatch(r"sha256:[0-9a-f]{64}", t["quality_contract_digest"]) for t in tasks), "same quality contract required"
    task_ids = {t["id"] for t in tasks}
    outcomes = {t["outcome_id"] for t in tasks}
    strategies = data["strategies"]
    assert set(strategies) <= {"A", "B", "C"} and strategies, "strategies are A, B and C"
    for strategy in strategies.values():
        assert strategy["model_harness_configuration_digest"].startswith("sha256:"), "configuration identity required"
    if "B" in strategies and "C" in strategies:
        assert strategies["B"]["model_harness_configuration_digest"] == strategies["C"]["model_harness_configuration_digest"], "B/C must use identical models and harness configuration"
    result = {"schema": 1, "source_commit": data["source_commit"], "strategies": {},
              "comparison_available": False, "savings_percentage": None,
              "limitations": ["Bytes are not tokens. Self-reported values are not provider billing evidence.",
                              "Accepted tasks and Outcomes require the same external quality contract and final integration checks."]}
    for name, strategy in strategies.items():
        attempts = strategy["attempts"]
        assert attempts and len({a["id"] for a in attempts}) == len(attempts), "all attempts need unique identity"
        for attempt in attempts:
            assert attempt["phase"] in PHASES and attempt["task_id"] in task_ids, "attempt outside controlled scope"
            for metric in METRICS:
                value = attempt.get(metric)
                assert value is None or (type(value) is int and value >= 0), "invalid usage count"
            value = attempt.get("cost")
            if value is not None:
                assert isinstance(value, str), "decimal cost must be a string"
                try:
                    amount = Decimal(value)
                except InvalidOperation:
                    raise AssertionError("invalid decimal cost")
                assert amount.is_finite() and amount >= 0, "invalid cost"
                assert attempt.get("cost_evidence") in {"provider_invoice", "provider_usage", "documented_rate_estimate", "self_reported"}, "cost provenance required"
                assert attempt.get("cost_evidence_ref"), "reference to retained billing/rate evidence required"
                assert attempt.get("currency"), "currency required"
        accepted = strategy["accepted_tasks"]
        assert len(set(accepted)) == len(accepted) and set(accepted) <= task_ids, "invalid accepted tasks"
        accepted_outcomes = strategy["accepted_outcomes"]
        assert len(set(accepted_outcomes)) == len(accepted_outcomes) and set(accepted_outcomes) <= outcomes, "invalid accepted Outcomes"
        for outcome in accepted_outcomes:
            assert {t["id"] for t in tasks if t["outcome_id"] == outcome} <= set(accepted), "Outcome contains unaccepted tasks"
        assert strategy.get("quality_evidence_ref") and strategy.get("integration_evidence_ref"), "quality and integration evidence required"
        assert task_ids <= {a["task_id"] for a in attempts}, "missing tasks, including unsuccessful attempts"
        costs_complete = all(a.get("cost") is not None for a in attempts)
        currencies = {a.get("currency") for a in attempts if a.get("cost") is not None}
        assert len(currencies) <= 1, "mixed currencies cannot be summed"
        total = sum((Decimal(a["cost"]) for a in attempts), Decimal(0)) if costs_complete else None
        summary = {"attempts": len(attempts), "accepted_tasks": len(accepted), "accepted_outcomes": len(accepted_outcomes),
                   "acceptance_rate": len(accepted) / len(tasks), "total_cost": str(total) if total is not None else None,
                   "currency": next(iter(currencies), None), "cost_evidence": sorted({a["cost_evidence"] for a in attempts if a.get("cost") is not None}),
                   "cost_per_accepted_task": str(total / len(accepted)) if total is not None and accepted else None,
                   "cost_per_accepted_outcome": str(total / len(accepted_outcomes)) if total is not None and accepted_outcomes else None,
                   "cost_includes_failed_attempts_review_corrections": True}
        for metric in METRICS:
            summary[metric] = sum(a[metric] for a in attempts) if all(a.get(metric) is not None for a in attempts) else None
        result["strategies"][name] = summary
    if set(strategies) == {"A", "B", "C"}:
        result["comparison_available"] = True
    else:
        result["limitations"].append("A/B/C controls were not all executed; no comparative savings claim is available.")
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    args.output.write_text(json.dumps(report(json.loads(args.input.read_text())), indent=2) + "\n")
