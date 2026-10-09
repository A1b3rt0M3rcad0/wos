"""Validate complete, actually passing signed workload JSON events."""
import datetime
import json
import math
from pathlib import Path
import re
import subprocess
import sys

if len(sys.argv) != 4:
    raise SystemExit('usage: signed-load-report.py GO_JSON_LOG FULL_SOURCE_COMMIT OUTPUT')
log, source, output = sys.argv[1:]
if not re.fullmatch(r'[a-f0-9]{40}', source):
    raise SystemExit('full immutable source commit required')
rows, passed, packages = [], set(), set()
for line in Path(log).read_text().splitlines():
    event = json.loads(line)
    if event['Action'] == 'fail':
        raise SystemExit('a failed signed workload cannot produce acceptance evidence')
    package = event.get('Package', '')
    if event['Action'] == 'pass':
        if 'Test' in event:
            passed.add((package, event['Test']))
        else:
            packages.add(package)
    text = event.get('Output', '')
    if 'SIGNED_LOAD {' in text:
        row = json.loads(text.split('SIGNED_LOAD ', 1)[1])
        rows.append(dict(package=package, test=event['Test'], **row))
expected = {(backend, outcomes, history, consumers) for backend in ('sqlite', 'postgres')
            for outcomes in (1, 4) for history in (0, 16) for consumers in (2, 8)}
observed = set()
for row in rows:
    package = row['package']
    backend = package.rsplit('/', 1)[-1]
    key = (backend, row['outcomes'], row['accepted_historical_tasks_per_outcome'], row['consumers'])
    if key in observed or (package, row['test']) not in passed or package not in packages:
        raise SystemExit('duplicate or unconfirmed workload result')
    observed.add(key)
    if row['accepted_deliveries'] != row['consumers'] * 2:
        raise SystemExit('incomplete independent accepted deliveries')
    if row['verified_outbox_deliveries'] != row['outcomes'] * row['accepted_historical_tasks_per_outcome'] + row['accepted_deliveries']:
        raise SystemExit('incomplete committed DONE outbox evidence')
    if row['protocol'] != 'signed_contracts_v2' or row['acceptance'] != 'independent_review' or row['approval_replay_deduplicated'] is not True:
        raise SystemExit('incorrect protocol, review or replay evidence')
    for field in ('elapsed_ms', 'accepted_per_second', 'delivery_p50_ms', 'delivery_p95_ms', 'delivery_p99_ms'):
        if not isinstance(row[field], (int, float)) or not math.isfinite(row[field]) or row[field] <= 0:
            raise SystemExit('invalid actual measurement')
    row['backend'] = backend
for backend in ('sqlite', 'postgres'):
    if not any(row['backend'] == backend and row.get('worker_recovery_experiment') is True for row in rows):
        raise SystemExit('missing signed completion worker restart/fencing experiment')
if observed != expected or len(rows) != 16:
    raise SystemExit('incomplete signed SQLite/PostgreSQL matrix')
report = dict(schema=1, source_commit=source, measured_at=datetime.datetime.now(datetime.timezone.utc).isoformat(),
              go=subprocess.check_output(['go', 'version'], text=True).strip(),
              scope='Local real application/storage signing, authenticated authority, independent required attestation review, original approval replay and DONE/outbox assertions; excludes process/network/model latency.',
              retry_policy='Only transaction_conflict, same immutable original command ID/idempotency key/envelope; bounded jitter/backoff, 100 retries maximum.',
              limitations=['16 accepted historical tasks per Outcome, not an arbitrary dense production dataset',
                           'No production SLA, distributed hosts, model tokens or billing savings',
                           'SQL statement counts are unavailable; reported honestly',
                           'Latency includes history/event verification reads and deduplication replay',
                           'Worker absence/expiry and storage connection restart are exercised; no OS kill or network partition is claimed'], rows=rows)
Path(output).write_text(json.dumps(report, indent=2) + '\n')
print('16 actually passing signed workload measurements saved')
