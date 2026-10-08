"""Check the explicit v1 integration catalogue against the current event mapping.

New public event names require a reviewed catalogue update. This check does not
infer or publish internal DomainEvent payloads.
"""
from pathlib import Path
import json
import re

root = Path(__file__).resolve().parents[2]
pipeline = (root / "packages/wos-core/application/pipeline.go").read_text()
mapping = pipeline.split("func eventTypesForCommand(", 1)[1].split("func conclusionRecordedPayload", 1)[0]
current = set(re.findall(r'"([a-z_]+\.[a-z_]+)"', mapping))
for suffix in re.findall(r'ownerPrefix \+ "(\.[a-z_]+)"', mapping):
    current.update(owner + suffix for owner in ("outcome", "objective", "work_item"))
current.update(re.findall(r'"(work_contract\.[a-z_]+)"', (root / "packages/wos-core/application/work_contract_service.go").read_text()))
current.update(re.findall(r'"(outcome\.work_protocol_changed)"', (root / "packages/wos-core/application/work_protocol.go").read_text()))
current.update(re.findall(r'"([a-z_]+\.[a-z_]+)"', (root / "packages/wos-core/application/signed_return.go").read_text()))
catalog = json.loads((root / "docs/integration-events-v1.json").read_text())
published = [entry["event_type"] for entry in catalog["events"]]
if catalog["schema_version"] != 1 or len(published) != len(set(published)):
    raise SystemExit("invalid or duplicate public integration catalogue")
if set(published) != current:
    raise SystemExit(f"catalogue mismatch: missing={sorted(current-set(published))}, stale={sorted(set(published)-current)}")
print(f"integration catalogue v1: {len(published)} public event types, mapping checked")
