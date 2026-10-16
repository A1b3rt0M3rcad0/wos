#!/usr/bin/env python3
"""Valida estrutura, tokens, recursos locais e ausência de fontes binárias.

Não avalia estética, não executa navegador e não certifica acessibilidade.
"""
from __future__ import annotations
import hashlib
import json
import re
import sys
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import unquote, urlsplit
from build_tokens import render_css

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = [
    "SKILL.md", "README.md", "tokens/DESIGN_TOKENS.json", "tokens/tokens.css",
    "references/DESIGN_DNA.md", "references/VISUAL_FORENSICS.md",
    "references/TYPOGRAPHY_SYSTEM.md", "references/GRAPHIC_LANGUAGE.md",
    "references/LAYOUT_AND_COMPOSITION.md", "references/MOTION_AND_INTERACTIONS.md",
    "references/SECTION_ATLAS.md", "references/ASSET_INVENTORY.md",
    "references/RESPONSIVE_ACCESSIBILITY.md", "references/REPRODUCTION_PLAYBOOK.md",
    "references/IMPLEMENTATION_ARCHITECTURE.md", "references/QUALITY_CHECKLIST.md",
    "references/EVIDENCE_AND_LIMITS.md", "references/TECHNICAL_SOURCES.md",
    "evidence/manifest.json", "templates/ASSET_MANIFEST.json",
    "examples/starter/index.html", "examples/starter/styles.css",
    "examples/starter/script.js",
]

class ResourceParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.refs: list[str] = []
        self.ids: set[str] = set()
        self.duplicate_ids: list[str] = []
        self.h1s = 0
    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        data = dict(attrs)
        if tag == "h1":
            self.h1s += 1
        if data.get("id"):
            ident = data["id"]
            if ident in self.ids:
                self.duplicate_ids.append(ident)
            self.ids.add(ident)
        for key in ("src", "href"):
            if data.get(key):
                self.refs.append(data[key])

def local_target(base: Path, ref: str) -> Path | None:
    parts = urlsplit(ref)
    if parts.scheme or parts.netloc or not parts.path:
        return None
    return (base / unquote(parts.path)).resolve()

def main() -> int:
    errors: list[str] = []
    for rel in REQUIRED:
        if not (ROOT / rel).is_file():
            errors.append(f"Arquivo obrigatório ausente: {rel}")
    if errors:
        print("FALHOU:\n- " + "\n- ".join(errors), file=sys.stderr)
        return 1
    skill = (ROOT / "SKILL.md").read_text(encoding="utf-8")
    if not skill.startswith("---\n") or not re.search(r"^name: chess-style$", skill, re.M):
        errors.append("Frontmatter/name inválido em SKILL.md")
    if not re.search(r"^description:", skill, re.M):
        errors.append("Descrição da skill ausente")
    for path in ROOT.rglob("*"):
        if not path.is_file():
            continue
        if path.suffix.lower() in {".ttf", ".otf", ".woff", ".woff2", ".ttc"}:
            errors.append(f"Fonte binária não permitida: {path.relative_to(ROOT)}")
        if path.suffix == ".json":
            try:
                json.loads(path.read_text(encoding="utf-8"))
            except (ValueError, OSError) as exc:
                errors.append(f"JSON inválido em {path.name}: {exc}")
        if path.suffix == ".md":
            text = path.read_text(encoding="utf-8")
            for ref in re.findall(r"\]\(([^)]+)\)", text):
                target = local_target(path.parent, ref)
                if target and not target.is_file():
                    errors.append(f"Link local ausente em {path.name}: {ref}")
    if errors:
        print("FALHOU:\n- " + "\n- ".join(errors), file=sys.stderr)
        return 1
    data = json.loads((ROOT / "tokens/DESIGN_TOKENS.json").read_text(encoding="utf-8"))
    if render_css(data) != (ROOT / "tokens/tokens.css").read_text(encoding="utf-8"):
        errors.append("tokens.css divergente. Execute scripts/build_tokens.py")
    token_names = [item["name"] for item in data["tokens"]]
    if len(set(token_names)) != len(token_names):
        errors.append("Há nomes de tokens duplicados")
    for item in data["tokens"]:
        if item.get("evidence") not in {"O", "M", "I", "P", "N"}:
            errors.append(f"Procedência ausente: {item['name']}")
    manifest = json.loads((ROOT / "evidence/manifest.json").read_text(encoding="utf-8"))
    for crop in manifest["crops"]:
        if not (ROOT / crop["file"]).is_file():
            errors.append(f"Recorte ausente: {crop['id']}")
    page = ROOT / "examples/starter/index.html"
    parser = ResourceParser()
    parser.feed(page.read_text(encoding="utf-8"))
    if parser.h1s != 1:
        errors.append(f"Starter deve ter um H1; encontrados {parser.h1s}")
    for ident in parser.duplicate_ids:
        errors.append(f"ID duplicado: {ident}")
    for ref in parser.refs:
        if ref == "#":
            errors.append("Link placeholder '#' no starter")
        parts = urlsplit(ref)
        if not parts.path and parts.fragment and unquote(parts.fragment) not in parser.ids:
            errors.append(f"Âncora ausente: {ref}")
        target = local_target(page.parent, ref)
        if target and not target.is_file():
            errors.append(f"Recurso local ausente no starter: {ref}")
    package_manifest = ROOT / "PACKAGE_MANIFEST.json"
    if package_manifest.exists():
        package = json.loads(package_manifest.read_text(encoding="utf-8"))
        for item in package["files"]:
            p = ROOT / item["path"]
            if not p.is_file():
                errors.append(f"Arquivo do manifesto ausente: {item['path']}")
            elif hashlib.sha256(p.read_bytes()).hexdigest() != item["sha256"]:
                errors.append(f"Hash divergente: {item['path']}")
    if errors:
        print("FALHOU:\n- " + "\n- ".join(errors), file=sys.stderr)
        return 1
    print(f"OK: estrutura, {len(data['tokens'])} tokens, {len(manifest['crops'])} recortes e recursos locais.")
    print("Essa validação não substitui teste de navegador ou revisão visual.")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
