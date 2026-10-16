#!/usr/bin/env python3
"""Exporta os tokens próprios da chess-style para CSS, sem dependências externas."""
from __future__ import annotations
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "tokens" / "DESIGN_TOKENS.json"
TARGET = ROOT / "tokens" / "tokens.css"

def render_css(data: dict) -> str:
    lines = [
        "/* Gerado por scripts/build_tokens.py. Editar DESIGN_TOKENS.json. */",
        "/* M = pixels da captura; P = proposta, não CSS da referência. */",
        ":root {",
    ]
    names: set[str] = set()
    for token in data["tokens"]:
        css = token.get("css")
        if css is None:
            continue
        if not isinstance(css, str) or not css.startswith("--ch-") or css in names:
            raise ValueError(f"Nome CSS inválido ou duplicado: {css!r}")
        names.add(css)
        value = token["value"]
        if not isinstance(value, str) or any(c in value for c in [";", "{", "}", "\n"]):
            raise ValueError(f"Valor CSS inválido: {token['name']}")
        lines.append(f"  {css}: {value}; /* {token['evidence']} */")
    lines.append("}")
    return "\n".join(lines) + "\n"

def main() -> None:
    data = json.loads(SOURCE.read_text(encoding="utf-8"))
    TARGET.parent.mkdir(parents=True, exist_ok=True)
    TARGET.write_text(render_css(data), encoding="utf-8")
    print(f"Gerado: {TARGET.relative_to(ROOT)}")

if __name__ == "__main__":
    main()
