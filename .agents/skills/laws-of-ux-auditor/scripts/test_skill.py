#!/usr/bin/env python3
"""Offline tests for the portable UX audit skill and accompanying scripts."""
from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCRIPTS = ROOT / "scripts"


def run(*args: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        [sys.executable, str(SCRIPTS / args[0]), *map(str, args[1:])],
        capture_output=True, text=True, timeout=15, check=False
    )


class SkillTests(unittest.TestCase):
    def test_structure_and_links(self):
        r = run("validate_skill.py", str(ROOT))
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)

    def test_example_json_can_be_created_and_validated(self):
        with tempfile.TemporaryDirectory() as workdir:
            out = Path(workdir) / "audit"
            r = run("scaffold_audit.py", "--product", "Produto Exemplo", "--scope",
                    "Cadastro e board", "--mode", "visual", "--date", "2026-10-19", "--out", str(out))
            self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
            self.assertIn("Produto Exemplo", (out / "audit-report.md").read_text(encoding="utf-8"))
            self.assertTrue((out / "evidencias" / "README.md").exists())
            r = run("validate_audit.py", str(out / "audit-record.json"))
            self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
            r = run("scaffold_audit.py", "--product", "X", "--scope", "Y", "--mode", "visual", "--out", str(out))
            self.assertNotEqual(r.returncode, 0, "Overwriting should require --force")

    def test_packaged_json_example(self):
        r = run("validate_audit.py", str(ROOT / "assets/examples/sample-audit.json"))
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        data = json.loads((ROOT / "assets/examples/sample-audit.json").read_text(encoding="utf-8"))
        self.assertEqual(data["findings"][0]["classification"], "hypothesis")
        self.assertEqual(data["findings"][0]["confidence"], "baixo")

    def test_finding_score_and_evidence(self):
        with tempfile.TemporaryDirectory() as workdir:
            path = Path(workdir) / "audit.json"
            data = json.loads((ROOT / "assets/templates/audit-record.json").read_text(encoding="utf-8"))
            f = {
                "id": "UX-001", "title": "Painel bloqueia salvar", "classification": "observed",
                "area": "F", "screen": "/board", "user_goal": "Concluir atualização",
                "observed": "Botão sobreposto a rodapé", "expected": "Salvar visível",
                "limitations": "Sem testes com usuários; alcance estimado", 
                "repro_steps": ["Abrir 1280x720", "Selecionar card", "Rolar"],
                "evidence": [{"type": "screenshot", "reference": "evidencias/x.png",
                              "observation": "CTA encoberta"}],
                "principles": ["layout", "wcag-2.4.11"],
                "confidence": "medio", "proposal": {
                    "change": "Corrigir sticky footer",
                    "acceptance": ["CTA permanece visível no viewport de teste"],
                    "verification": ["Repetir fluxo após mudança"]},
                "impact": {"severity": 4, "criticality": 5, "reach": 3,
                           "score": 4.1, "rationale": "Jornada central; alcance estimado"},
                "effort": {"complexity": 2, "dependencies": 2, "verification": 2,
                           "score": 2, "rationale": "Componente isolado"},
                "priority": {"tier": "P1", "index": 2.05, "rationale": "Alto impacto e baixo esforço"}
            }
            data["findings"] = [f]
            data["audit"]["product"] = "Exemplo"
            data["audit"]["scope"] = "Board"
            data["audit"]["date"] = "2026-10-19"
            data["audit"]["mode"] = "visual"
            path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
            valid = run("validate_audit.py", str(path))
            self.assertEqual(valid.returncode, 0, valid.stdout + valid.stderr)
            f["impact"]["score"] = 1.0
            path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
            invalid = run("validate_audit.py", str(path))
            self.assertNotEqual(invalid.returncode, 0, invalid.stdout + invalid.stderr)
            f["impact"]["score"] = 4.1
            f["evidence"] = []
            path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
            invalid = run("validate_audit.py", str(path))
            self.assertNotEqual(invalid.returncode, 0, invalid.stdout + invalid.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
