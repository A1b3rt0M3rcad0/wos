#!/usr/bin/env python3
"""Check basic Agent Skills structure and internal Markdown links (stdlib only).

This is NOT the upstream 'skills-ref' reference validator. To run the official
reference implementation, install it separately and run: skills-ref validate .
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

NAME_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
LINK_RE = re.compile(r"(?<!!)\[[^\]]*\]\(([^)]+)\)")
REQUIRED = [
    "SKILL.md", "README.md", "LICENSE", "VALIDATION.md", "references/sources.md",
    "references/research-overview.md", "references/audit-protocol.md",
    "references/audit-coverage.md", "references/accessibility-wcag22.md",
    "references/decision-prioritization.md", "references/implementation-patterns.md",
    "references/experiment-validation.md",
    *[f"references/{name}.md" for name in
      ("fitts", "hick", "jakob", "miller", "zeigarnik", "goal-gradient", "von-restorff")],
    "assets/templates/audit-report.md", "assets/templates/finding.md",
    "assets/templates/flow-inventory.md", "assets/templates/implementation-plan.md",
    "assets/templates/verification-checklist.md", "assets/templates/audit-record.json",
    "assets/examples/board-example.md", "assets/examples/form-example.md",
    "assets/examples/sample-audit.json",
    "scripts/scaffold_audit.py", "scripts/validate_audit.py", "scripts/test_skill.py",
]


def load_frontmatter(text: str):
    if not text.startswith("---\n"):
        raise ValueError("SKILL.md must begin with YAML frontmatter '---' on line one")
    chunks = text.split("\n---\n", 1)
    if len(chunks) != 2:
        raise ValueError("SKILL.md requires a closing frontmatter delimiter")
    raw = chunks[0][4:]
    if not chunks[1].strip():
        raise ValueError("SKILL.md body is empty")
    try:
        import yaml  # Optional: parse strictly if PyYAML is installed.
    except ImportError:
        # A deliberately limited fallback for simple scalar frontmatter.
        # Validate source against the upstream reference tool for full YAML.
        props = {}
        for line in raw.splitlines():
            if not line.strip() or line.lstrip().startswith("#") or line[0].isspace():
                continue
            pair = line.split(":", 1)
            if len(pair) != 2:
                raise ValueError(f"Malformed YAML scalar line: {line!r}")
            props[pair[0].strip()] = pair[1].strip().strip('"\'')
        return props
    else:
        props = yaml.safe_load(raw)
        if not isinstance(props, dict):
            raise ValueError("Frontmatter must be a YAML mapping")
        return props


def validate(root: Path) -> list[str]:
    errors = []
    if not root.is_dir():
        return [f"not a directory: {root}"]
    for rel in REQUIRED:
        if not (root / rel).is_file():
            errors.append(f"missing required bundle resource: {rel}")
    skillfile = root / "SKILL.md"
    if not skillfile.is_file():
        return errors
    body = skillfile.read_text(encoding="utf-8")
    try:
        frontmatter = load_frontmatter(body)
        name = frontmatter.get("name")
        desc = frontmatter.get("description")
        if not isinstance(name, str) or len(name) > 64 or not NAME_RE.fullmatch(name):
            errors.append("frontmatter.name must be lowercase kebab-case, 1-64 characters, no '--'")
        elif name != root.name:
            errors.append(f"frontmatter.name {name!r} != parent directory {root.name!r}")
        if not isinstance(desc, str) or not (1 <= len(desc) <= 1024):
            errors.append("frontmatter.description must be nonempty, at most 1024 characters")
        if "compatibility" in frontmatter:
            val = frontmatter["compatibility"]
            if not isinstance(val, str) or not (1 <= len(val) <= 500):
                errors.append("frontmatter.compatibility must have 1-500 characters")
        if "metadata" in frontmatter and not isinstance(frontmatter["metadata"], dict):
            errors.append("frontmatter.metadata must be a mapping")
        if "allowed-tools" in frontmatter and not isinstance(frontmatter["allowed-tools"], str):
            errors.append("frontmatter.allowed-tools must be a string when present")
    except (ValueError, UnicodeDecodeError) as exc:
        errors.append(str(exc))
    if len(body.splitlines()) >= 500:
        errors.append("SKILL.md is 500+ lines; use progressive disclosure")
    # Files referenced by Markdown links in docs must exist if local.
    for md in sorted(root.rglob("*.md")):
        if any(part in ("__pycache__", ".git") for part in md.parts):
            continue
        txt = md.read_text(encoding="utf-8")
        for target in LINK_RE.findall(txt):
            target = target.strip().split("#", 1)[0].split("?", 1)[0]
            if not target or "://" in target or target.startswith(("mailto:", "#")):
                continue
            rel_path = (md.parent / target).resolve()
            try:
                rel_path.relative_to(root.resolve())
            except ValueError:
                errors.append(f"link escapes skill directory in {md.relative_to(root)}: {target}")
            else:
                if not rel_path.exists():
                    errors.append(f"broken link in {md.relative_to(root)}: {target}")
    return errors


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("skill_dir", nargs="?", default=str(Path(__file__).resolve().parent.parent))
    args = ap.parse_args()
    root = Path(args.skill_dir).resolve()
    errors = validate(root)
    if errors:
        for err in errors:
            print(f"FAIL: {err}", file=sys.stderr)
        return 1
    print(f"OK: {root.name} — SKILL.md/frontmatter, {len(REQUIRED)} recursos e links internos válidos.")
    print("Nota: não substitui 'skills-ref validate' nem testa operação do agente.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
