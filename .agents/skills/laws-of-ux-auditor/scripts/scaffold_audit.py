#!/usr/bin/env python3
"""Create a new evidence-oriented UX audit workspace from bundled templates."""
from __future__ import annotations

import argparse
from datetime import date
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
TEMPLATES = ROOT / "assets" / "templates"
FILES = (
    "audit-report.md", "flow-inventory.md", "implementation-plan.md",
    "verification-checklist.md", "audit-record.json"
)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--product", required=True, help="Product name")
    ap.add_argument("--scope", required=True, help="Audited journeys/screens")
    ap.add_argument("--mode", required=True, choices=("visual", "interativo", "codigo", "hibrido"))
    ap.add_argument("--out", required=True, type=Path, help="Destination directory")
    ap.add_argument("--date", default=date.today().isoformat(), help="YYYY-MM-DD; default local date")
    ap.add_argument("--force", action="store_true", help="Allow overwriting generated files")
    args = ap.parse_args()
    try:
        date.fromisoformat(args.date)
    except ValueError:
        ap.error("--date must be YYYY-MM-DD")
    out = args.out.expanduser().resolve()
    if out.exists() and not out.is_dir():
        ap.error("--out must refer to a directory")
    conflicts = [name for name in FILES if (out / name).exists()]
    if conflicts and not args.force:
        ap.error(f"refusing to overwrite files: {', '.join(conflicts)} (use --force deliberately)")
    replacements = {
        "{{PRODUCT}}": args.product,
        "{{SCOPE}}": args.scope,
        "{{MODE}}": args.mode,
        "{{DATE}}": args.date,
    }
    out.mkdir(parents=True, exist_ok=True)
    for name in FILES:
        src = TEMPLATES / name
        text = src.read_text(encoding="utf-8")
        for before, after in replacements.items():
            text = text.replace(before, after)
        (out / name).write_text(text, encoding="utf-8")
    evidence = out / "evidencias"
    evidence.mkdir(exist_ok=True)
    readme = evidence / "README.md"
    if not readme.exists():
        readme.write_text(
            "# Evidências da auditoria\n\nSalve prints, vídeos, logs e extratos de DOM "
            "anonimizados aqui. Use caminhos relativos nos achados. Não inclua segredos "
            "ou dados pessoais. Registre viewport, data, role, estado e passos de reprodução.\n",
            encoding="utf-8",
        )
    print(f"OK: auditoria criada em {out}")
    print("Próximo passo: adicione evidências, findings e execute scripts/validate_audit.py no JSON.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
