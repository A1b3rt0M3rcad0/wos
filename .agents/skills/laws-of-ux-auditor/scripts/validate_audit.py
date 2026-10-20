#!/usr/bin/env python3
"""Validate a report JSON and recalculate ordinal impact/effort/index (stdlib only)."""
from __future__ import annotations

import argparse
import json
import math
import re
import sys
from pathlib import Path

AREAS = set("ABCDEFGHIJ")
STATES = {"ok_verificado", "problema_verificado", "hipotese", "nao_testado", "nao_aplicavel"}
CONFIDENCES = {"alto", "medio", "baixo"}
CLASSES = {"observed", "verified", "hypothesis"}
TIERS = {"P0", "P1", "P2", "P3"}
VALID_MODES = {"visual", "interativo", "codigo", "hibrido"}


def is_nonempty_string(v) -> bool:
    return isinstance(v, str) and bool(v.strip())


def rating(obj: dict, labels: tuple[str, ...], weights=None) -> tuple[float | None, str | None]:
    values = []
    for label in labels:
        v = obj.get(label)
        if not isinstance(v, (int, float)) or isinstance(v, bool) or not math.isfinite(v) or not 1 <= v <= 5:
            return None, f"{label} must be numeric in [1, 5]"
        values.append(v)
    if weights is None:
        return sum(values) / len(values), None
    return sum(x * w for x, w in zip(values, weights)), None


def approx(a, b) -> bool:
    return isinstance(a, (int, float)) and not isinstance(a, bool) and math.isfinite(a) and abs(a - b) <= 0.015


def validate(data: dict) -> list[str]:
    issues: list[str] = []
    if not isinstance(data, dict):
        return ["JSON root must be an object"]
    if data.get("schema_version") != "1.0.0" or data.get("skill") != "laws-of-ux-auditor":
        issues.append("schema_version must be 1.0.0 and skill must be laws-of-ux-auditor")
    audit = data.get("audit")
    if not isinstance(audit, dict):
        return issues + ["audit must be an object"]
    for field in ("product", "scope", "date"):
        if not is_nonempty_string(audit.get(field)):
            issues.append(f"audit.{field} is required")
    if audit.get("mode") not in VALID_MODES:
        issues.append("audit.mode must be visual/interativo/codigo/hibrido")
    coverage = data.get("coverage")
    if not isinstance(coverage, list):
        issues.append("coverage must be a list")
    else:
        seen = set()
        for i, row in enumerate(coverage):
            if not isinstance(row, dict):
                issues.append(f"coverage[{i}] must be an object")
                continue
            area = row.get("area")
            if area not in AREAS or area in seen:
                issues.append(f"coverage[{i}].area must be unique and A-J")
            seen.add(area)
            if row.get("status") not in STATES:
                issues.append(f"coverage[{i}].status is invalid")
            if row.get("status") in ("ok_verificado", "problema_verificado") and not row.get("evidence"):
                issues.append(f"coverage[{i}] verified status requires evidence")
        if seen != AREAS:
            issues.append("coverage must include exactly areas A-J")
    findings = data.get("findings")
    if not isinstance(findings, list):
        return issues + ["findings must be a list"]
    seen_ids = set()
    for i, f in enumerate(findings):
        at = f"findings[{i}]"
        if not isinstance(f, dict):
            issues.append(f"{at} must be an object")
            continue
        fid = f.get("id")
        if not isinstance(fid, str) or not re.fullmatch(r"[A-Z]{2,8}-[A-Z0-9]{1,10}", fid):
            issues.append(f"{at}.id must look like UX-001")
        elif fid in seen_ids:
            issues.append(f"{at}.id duplicated")
        seen_ids.add(fid)
        if f.get("classification") not in CLASSES:
            issues.append(f"{at}.classification is invalid")
        if f.get("area") not in AREAS:
            issues.append(f"{at}.area must be A-J")
        if f.get("confidence") not in CONFIDENCES:
            issues.append(f"{at}.confidence is invalid")
        for field in ("title", "screen", "user_goal", "observed", "expected", "limitations"):
            if not is_nonempty_string(f.get(field)):
                issues.append(f"{at}.{field} must be nonempty")
        if not isinstance(f.get("repro_steps"), list) or not f.get("repro_steps") or not all(is_nonempty_string(v) for v in f["repro_steps"]):
            issues.append(f"{at}.repro_steps must contain steps")
        evidence = f.get("evidence")
        if not isinstance(evidence, list) or not evidence:
            issues.append(f"{at}.evidence must be a nonempty list of observable references or explicit hypothesis source")
        else:
            for j, e in enumerate(evidence):
                if not isinstance(e, dict) or not is_nonempty_string(e.get("observation")) or not is_nonempty_string(e.get("type")):
                    issues.append(f"{at}.evidence[{j}] requires type and observation")
                elif f.get("classification") == "verified" and e.get("type") == "hypothesis":
                    issues.append(f"{at}.verified finding cannot rely only on a hypothesis type")
        principles = f.get("principles")
        if not isinstance(principles, list) or not principles or not all(is_nonempty_string(x) for x in principles):
            issues.append(f"{at}.principles must list relevant laws, WCAG or heuristics")
        fix = f.get("proposal")
        if not isinstance(fix, dict):
            issues.append(f"{at}.proposal must be an object")
        else:
            if not is_nonempty_string(fix.get("change")):
                issues.append(f"{at}.proposal.change is required")
            for key in ("acceptance", "verification"):
                if not isinstance(fix.get(key), list) or not fix[key] or not all(is_nonempty_string(x) for x in fix[key]):
                    issues.append(f"{at}.proposal.{key} needs testable entries")
        impact, effort, priority = f.get("impact"), f.get("effort"), f.get("priority")
        impact_n, effort_n = None, None
        if isinstance(impact, dict):
            impact_n, err = rating(impact, ("severity", "criticality", "reach"), (0.4, 0.35, 0.25))
            if err:
                issues.append(f"{at}.impact.{err}")
            elif not approx(impact.get("score"), impact_n):
                issues.append(f"{at}.impact.score must be {impact_n:.2f}")
            if not is_nonempty_string(impact.get("rationale")):
                issues.append(f"{at}.impact.rationale is required")
        else:
            issues.append(f"{at}.impact must be an object")
        if isinstance(effort, dict):
            effort_n, err = rating(effort, ("complexity", "dependencies", "verification"))
            if err:
                issues.append(f"{at}.effort.{err}")
            elif not approx(effort.get("score"), effort_n):
                issues.append(f"{at}.effort.score must be {effort_n:.2f}")
            if not is_nonempty_string(effort.get("rationale")):
                issues.append(f"{at}.effort.rationale is required")
        else:
            issues.append(f"{at}.effort must be an object")
        if isinstance(priority, dict):
            if priority.get("tier") not in TIERS:
                issues.append(f"{at}.priority.tier must be P0/P1/P2/P3")
            if priority.get("tier") == "P0" and not is_nonempty_string(priority.get("blocking_reason")):
                issues.append(f"{at}.priority.blocking_reason required for P0")
            if not is_nonempty_string(priority.get("rationale")):
                issues.append(f"{at}.priority.rationale is required")
            if impact_n is not None and effort_n is not None and not approx(priority.get("index"), impact_n / effort_n):
                issues.append(f"{at}.priority.index must be {impact_n / effort_n:.2f}")
        else:
            issues.append(f"{at}.priority must be an object")
    return issues


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("json_file", type=Path)
    args = ap.parse_args()
    try:
        data = json.loads(args.json_file.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        print(f"FAIL: cannot read JSON: {exc}", file=sys.stderr)
        return 1
    issues = validate(data)
    if issues:
        for issue in issues:
            print(f"FAIL: {issue}", file=sys.stderr)
        return 1
    print(f"OK: audit JSON is structurally valid, {len(data['findings'])} findings checked, all priority scores consistent.")
    print("Nota: não verifica se evidências são verdadeiras nem conformidade WCAG.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
