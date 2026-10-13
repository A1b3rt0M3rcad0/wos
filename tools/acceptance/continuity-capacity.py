"""Measure dense reads on one clean source; never turn reference timings into an SLA."""
import argparse
import datetime
import json
import os
from pathlib import Path
import platform
import re
import resource
import subprocess
import time


def parse_measurements(log, history, samples):
    if not re.search(r'^PASS$', log, re.M) or re.search(r'^FAIL', log, re.M):
        raise ValueError('nonzero completed benchmark execution required')
    rows = []
    for line in log.splitlines():
        if not line.startswith('BenchmarkContinuity/'):
            continue
        parts = line.split()
        if len(parts) < 4 or int(parts[1]) != samples or len(parts[2:]) % 2:
            raise ValueError('complete fixed-sample benchmark metrics required')
        metrics = {parts[i+1]: float(parts[i]) for i in range(2, len(parts), 2)}
        name = re.sub(r'-\d+$', '', parts[0].split('/', 1)[1])
        if not all(metrics.get(k, -1) >= 0 for k in ('ns/op', 'B/op', 'allocs/op', 'pool_wait_ns/op')):
            raise ValueError('timing, allocation and pool measurements required')
        if not name.startswith('parallel_'):
            if not (0 < metrics.get('response_bytes', 0) <= 256*1024):
                raise ValueError('bounded response measurement required')
            if not (0 < metrics.get('p50_ns', 0) <= metrics.get('p95_ns', 0) <= metrics.get('p99_ns', 0)):
                raise ValueError('ordered empirical quantile measurements required')
        elif metrics.get('snapshots/s', 0) <= 0:
            raise ValueError('actual parallel throughput required')
        rows.append({'benchmark': name, 'samples': samples, 'metrics': metrics,
                     'sql_statement_count_available': 'sql_statements/op' in metrics})
    expected = {f'snapshot_{history}_work_items', f'snapshot_{history}_work_items_with_proof',
                f'work_context_{history}_work_items_with_proof', f'parallel_snapshot_{history}_work_items_with_proof'}
    if len(rows) != 4 or {r['benchmark'] for r in rows} != expected:
        raise ValueError('all four actual dense-read cases required')
    return rows


def source_is_clean():
    # CI logs at repository root are output, not compiler inputs. Reject tracked
    # changes and untracked files in all source/command trees.
    return not subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=no'], text=True) and not subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard', '--', 'packages', 'tools', 'tests', '.github', 'go.mod', 'go.sum'], text=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--history', type=int, nargs='+', default=[100, 1000])
    parser.add_argument('--samples', type=int, default=100)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if any(h not in (100, 1000, 10000) for h in args.history) or len(set(args.history)) != len(args.history) or not 20 <= args.samples <= 1000:
        raise ValueError('explicit supported history and bounded samples required')
    if not os.environ.get('WOS_TEST_POSTGRES_DSN'):
        raise ValueError('real PostgreSQL required; no skipped acceptance')
    source = subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip()
    if not re.fullmatch('[a-f0-9]{40}', source) or not source_is_clean():
        raise ValueError('one clean immutable source required')
    args.output.mkdir(parents=True, exist_ok=True)
    report = {'schema': 1, 'source_commit': source, 'recorded_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'go': subprocess.check_output(['go', 'version'], text=True).strip(), 'host': platform.platform(),
              'gomaxprocs': os.environ.get('GOMAXPROCS', 'runtime default'), 'cases': [],
              'limitations': ['Reference fixture, not production SLA or billing evidence.',
                              'Go process CPU/RSS and profiles include fixture seeding, compilation and reads; exclude the independent PostgreSQL server.',
                              'Empirical p99 of 100 samples is not a production tail estimate.',
                              'Full continuity still reconstructs the entire Outcome; section byte caps do not bound that cost.',
                              'Unavailable SQL counts remain explicit; no value is inferred.']}
    for history in args.history:
        for storage in ('sqlite', 'postgres'):
            basename = f'{storage}-{history}'
            command = ['go', 'test', f'./packages/wos-core/storage/{storage}', '-run', '^$',
                       '-bench', '^BenchmarkContinuity/(snapshot|work_context|parallel_snapshot)',
                       '-benchtime', f'{args.samples}x', '-count', '1', '-benchmem',
                       '-o', str((args.output/(basename+'.test')).resolve()),
                       '-cpuprofile', str((args.output/(basename+'.cpu')).resolve()),
                       '-memprofile', str((args.output/(basename+'.heap')).resolve())]
            before = resource.getrusage(resource.RUSAGE_CHILDREN)
            started = time.monotonic()
            result = subprocess.run(command, text=True, capture_output=True, env={**os.environ, 'WOS_CONTINUITY_BENCHMARK_HISTORY': str(history)})
            elapsed = time.monotonic()-started
            after = resource.getrusage(resource.RUSAGE_CHILDREN)
            log = result.stdout+result.stderr
            (args.output/(basename+'.txt')).write_text(log)
            if result.returncode:
                raise RuntimeError(f'{basename} failed; inspect its owned log')
            rows = parse_measurements(log, history, args.samples)
            report['cases'].append({'storage': storage, 'work_items_per_outcome': history,
                                   'rich_history': 'one required criterion and one explicit assessment per WorkItem, created through Application',
                                   'whole_fixture_wall_seconds': elapsed,
                                   'go_process_user_cpu_seconds': after.ru_utime-before.ru_utime,
                                   'go_process_system_cpu_seconds': after.ru_stime-before.ru_stime,
                                   'child_peak_rss_kib_high_watermark': after.ru_maxrss,
                                   'measurements': rows})
            print(f'{basename}: four completed cases, {args.samples} samples each', flush=True)
    if subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip() != source or not source_is_clean():
        raise ValueError('source changed during measurements')
    (args.output/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    print('Dense continuity capacity evidence recorded; no production SLA inferred')


if __name__ == '__main__':
    main()
