"""Record active fuzz execution; passing seed-only tests are insufficient."""
import datetime
import json
from pathlib import Path
import re
import subprocess
import sys

if len(sys.argv) != 6 or not re.fullmatch(r'[a-f0-9]{40}', sys.argv[1]):
    raise SystemExit('usage: fuzz-report.py FULL_SOURCE CANONICAL_JSONL ENVELOPE_JSONL YAML_JSONL OUTPUT')
source, *logs, output = sys.argv[1:]
rows = []
for name, filename in zip(('FuzzCanonical', 'FuzzEnvelope', 'FuzzYAMLJSON'), logs):
    executions = 0
    passed = False
    for line in Path(filename).read_text().splitlines():
        event = json.loads(line)
        if event['Action'] == 'fail':
            raise SystemExit('failed fuzz campaign; preserve and investigate its corpus')
        if event['Action'] == 'pass' and event.get('Test') == name:
            passed = True
        counts = re.findall(r'execs:\s*(\d+)', event.get('Output', ''))
        executions = max(executions, *(int(n) for n in counts), 0)
    if not passed or executions == 0:
        raise SystemExit('campaign did not execute active fuzzing: ' + name)
    rows.append(dict(target=name, executions=executions, passed=True))
report = dict(schema=1, source_commit=source, measured_at=datetime.datetime.now(datetime.timezone.utc).isoformat(),
              go=subprocess.check_output(['go', 'version'], text=True).strip(), parallel_workers=2,
              limitations=['Bounded active campaigns, not exhaustive proof or coverage percentage',
                           'Counts include actual mutations/baseline cases; no security claim from count alone'], campaigns=rows)
Path(output).write_text(json.dumps(report, indent=2) + '\n')
print('3 active fuzz campaigns recorded')
