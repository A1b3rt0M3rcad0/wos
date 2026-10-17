#!/usr/bin/env python3
"""Source-bound signed SQL scalability matrix; no production capacity claim.

Independent application/store replicas run in one Go test process. Real HTTP
replicas, worker failure and receiver effects are qualified by separate pilots.
"""
import argparse
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import re
import signal
import subprocess
import time

TOPOLOGIES = {'same-outcome': (1, 1), 'separate-outcomes': (1, 4), 'separate-namespaces': (4, 4)}


def source_hashes(root):
    files = [p for folder in ('packages', 'tools') for p in (root/folder).rglob('*')
             if p.is_file() and p.suffix in ('.go', '.py', '.sql')]
    files += [root/name for name in ('go.mod', 'go.sum', '.go-version')]
    return {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(files)}


def parse_result(log, expected):
    rows, passed, failed = [], False, False
    for line in log.splitlines():
        try:
            item = json.loads(line)
        except ValueError:
            continue
        if item.get('Action') in ('skip', 'fail'):
            failed = True
        if item.get('Action') == 'pass' and '/namespaces=' in item.get('Test', ''):
            passed = True
        if 'SIGNED_LOAD ' in item.get('Output', ''):
            rows.append(json.loads(item['Output'].split('SIGNED_LOAD ', 1)[1]))
    if failed or not passed or len(rows) != 1:
        raise ValueError('exactly one executed passing fixture required; skips, failures and duplicates cannot qualify')
    row = rows[0]
    for key, value in expected.items():
        if row.get(key) != value:
            raise ValueError('measured fixture differs: '+key)
    for key in ('accepted_per_second', 'delivery_p50_ms', 'delivery_p95_ms', 'delivery_p99_ms',
                'read_p50_ms', 'read_p95_ms', 'read_p99_ms', 'transaction_wait_p95_ms',
                'guard_wait_p95_ms', 'pool_wait_ms', 'pool_wait_count', 'conflicts', 'retries', 'read_transaction_conflicts'):
        value = row.get(key)
        if not isinstance(value, (int, float)) or not math.isfinite(value) or value < 0:
            raise ValueError('complete finite nonnegative metrics required: '+key)
    for prefix in ('delivery', 'read'):
        if not row[prefix+'_p50_ms'] <= row[prefix+'_p95_ms'] <= row[prefix+'_p99_ms']:
            raise ValueError('unordered empirical quantiles')
    if row.get('concurrent_reads', 0) < 1 or not 0 < row.get('read_max_bytes', 0) <= 256*1024:
        raise ValueError('bounded concurrent reads were not exercised')
    if not row.get('approval_replay_deduplicated') or row.get('version_collision_winners') != 1:
        raise ValueError('independent acceptance/replay/version collision verification missing')
    if row.get('version_collision_rejections') != expected['consumers']-1:
        raise ValueError('each losing original version must be rejected')
    if row.get('transport_measured') is not False:
        raise ValueError('SQL fixture cannot qualify HTTP or process throughput')
    return row


def run_child(command, root, env, log, timeout):
    began = time.monotonic()
    with log.open('wb') as stream:
        child = subprocess.Popen(command, cwd=root, env=env, stdout=stream,
                                 stderr=subprocess.STDOUT, start_new_session=True)
        while True:
            pid, status, usage = os.wait4(child.pid, os.WNOHANG)
            if pid:
                child.returncode = os.waitstatus_to_exitcode(status)
                if child.returncode:
                    raise RuntimeError('fixture failed; retained private log: '+str(log))
                return {'wall_seconds_including_fixture': time.monotonic()-began,
                        'user_cpu_seconds': usage.ru_utime, 'system_cpu_seconds': usage.ru_stime,
                        'peak_rss_kib': usage.ru_maxrss}
            if time.monotonic()-began > timeout:
                os.killpg(child.pid, signal.SIGTERM)
                os.wait4(child.pid, 0)
                raise RuntimeError('bounded fixture timed out; partial report retained')
            time.sleep(.1)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, default=Path('.'))
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--archive-commit', help='exact source SHA of an immutable Git archive; test harness replacement is separately hashed')
    parser.add_argument('--allow-dirty', action='store_true', help='development only: report dirty source hashes, never immutable qualification')
    parser.add_argument('--backend', choices=('sqlite', 'postgres'), default='postgres')
    parser.add_argument('--replicas', nargs='+', type=int, choices=(1, 2, 4), default=[1, 2, 4])
    parser.add_argument('--history', nargs='+', type=int, choices=(0, 16, 100, 1000), default=[0, 16])
    parser.add_argument('--consumers', type=int, choices=(2, 4, 8, 50), default=4)
    parser.add_argument('--topology', nargs='+', choices=tuple(TOPOLOGIES), default=list(TOPOLOGIES))
    parser.add_argument('--repetitions', type=int, default=3)
    parser.add_argument('--timeout', type=int, default=600)
    args = parser.parse_args()
    if any(len(values) != len(set(values)) for values in (args.replicas, args.history, args.topology)):
        parser.error('matrix dimensions must be unique; repetitions have their own argument')
    if platform.system() != 'Linux':
        parser.error('Linux runner required for wait4 resource units')
    if not 1 <= args.repetitions <= 10 or not 30 <= args.timeout <= 1800:
        parser.error('repetitions must be 1..10 and per-case timeout 30..1800 seconds')
    root = args.root.resolve()
    if args.archive_commit:
        if (root/'.git').exists() or not re.fullmatch('[a-f0-9]{40}', args.archive_commit):
            parser.error('archive mode needs an exact SHA and no .git directory')
        commit, dirty = args.archive_commit, False
    else:
        commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
        dirty = bool(subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=normal'], cwd=root, text=True))
        if dirty and not args.allow_dirty:
            parser.error('commit the source or use explicitly non-qualifying --allow-dirty')
    if args.backend == 'postgres' and not os.environ.get('WOS_TEST_POSTGRES_DSN'):
        parser.error('real PostgreSQL DSN required; skipped tests cannot qualify')
    hashes = source_hashes(root)
    output = args.output.resolve()
    if output == root or root in output.parents:
        parser.error('keep private measurement output outside the source tree')
    output.mkdir(mode=0o700, parents=True, exist_ok=False)
    report = {'schema': 1, 'source_commit': commit, 'source_dirty': dirty,
              'source_files_sha256': hashes, 'complete': False, 'production_sla': False,
              'recorded_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'qualified_immutable_source': not dirty,
              'archive_harness_overlay': bool(args.archive_commit),
              'environment': {'host': platform.platform(), 'go': subprocess.check_output(['go', 'version'], text=True).strip(),
                              'gomaxprocs': os.environ.get('GOMAXPROCS', 'runtime default')},
              'measurement_boundary': 'independent application/store replicas with separate SQL pools in one precompiled Go process; whole-child CPU/RSS include fixture setup, exclude database CPU/RSS, HTTP, load balancing and external delivery workers',
              'history_boundary': 'real accepted signed tasks per Outcome, each with independent review; immutable history grows while timed task count stays fixed',
              'quantiles': 'empirical complete-journey and concurrent read samples, not production tail/SLO estimates',
              'runs': []}
    try:
        binary = output/'scalability.test'
        with (output/'build.log').open('wb') as stream:
            subprocess.run(['go', 'test', '-c', '-o', str(binary), './packages/wos-core/storage/'+args.backend],
                           cwd=root, stdout=stream, stderr=subprocess.STDOUT, check=True)
        for topology in args.topology:
            namespaces, outcomes = TOPOLOGIES[topology]
            for replicas in args.replicas:
                for history in args.history:
                    for repetition in range(1, args.repetitions+1):
                        name = f'{topology}-replicas-{replicas}-history-{history}-run-{repetition}'
                        env = os.environ.copy()
                        env.update(WOS_SCALE_QUALIFICATION='1', WOS_SCALE_OUTCOMES=str(outcomes),
                                   WOS_SCALE_NAMESPACES=str(namespaces), WOS_SCALE_REPLICAS=str(replicas),
                                   WOS_SCALE_HISTORY=str(history), WOS_SCALE_CONSUMERS=str(args.consumers))
                        log = output/(name+'.jsonl')
                        # test2json preserves exact executed/skipped/pass provenance for the precompiled binary.
                        command = ['go', 'tool', 'test2json', '-t', str(binary), '-test.v',
                                   '-test.run=^TestScalableSignedWorkMatrix$', '-test.count=1', '-test.timeout='+str(args.timeout)+'s']
                        resources = run_child(command, root, env, log, args.timeout+15)
                        expected = {'namespaces': namespaces, 'outcomes': outcomes, 'active_outcomes': outcomes,
                                    'service_replicas': replicas, 'consumers': args.consumers,
                                    'accepted_historical_tasks_per_outcome': history, 'accepted_deliveries': args.consumers*2,
                                    'verified_outbox_deliveries': outcomes*history+args.consumers*2}
                        metrics = parse_result(log.read_text(), expected)
                        report['runs'].append({'topology': topology, 'backend': args.backend,
                                               'repetition': repetition, 'metrics': metrics, 'resources': resources, 'log': log.name})
                        (output/'report.json').write_text(json.dumps(report, indent=2)+'\n')
                        print(json.dumps({'completed': name, 'accepted': metrics['accepted_deliveries']}), flush=True)
        if hashes != source_hashes(root):
            raise RuntimeError('source changed during measurement; immutable qualification refused')
        report['complete'] = True
    finally:
        (output/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
