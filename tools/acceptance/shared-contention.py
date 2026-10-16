#!/usr/bin/env python3
"""Opt-in Linux/POSIX shared-backend contention reference, never a production SLA."""
import argparse
import json
import hashlib
import re
import os
from pathlib import Path
import platform
import signal
import subprocess
import time


def run_case(command, env, root, log, timeout):
    started = time.monotonic()
    with log.open('wb') as output:
        process = subprocess.Popen(command, cwd=root, env=env, stdout=output,
                                   stderr=subprocess.STDOUT, start_new_session=True)
        while True:
            pid, status, usage = os.wait4(process.pid, os.WNOHANG)
            if pid:
                process.returncode = os.waitstatus_to_exitcode(status)
                return process.returncode, usage, time.monotonic()-started
            if time.monotonic()-started > timeout:
                os.killpg(process.pid, signal.SIGTERM)
                _, status, usage = os.wait4(process.pid, 0)
                process.returncode = os.waitstatus_to_exitcode(status)
                raise RuntimeError('bounded contention process timed out; private partial log retained')
            time.sleep(.2)


def parse_load(path):
    rows = []
    failed, passed_case = False, False
    for line in path.read_text().splitlines():
        try:
            item = json.loads(line)
        except ValueError:
            continue
        if item.get('Action') in ('fail', 'skip'):
            failed = True
        if item.get('Action') == 'pass' and '/consumers=' in item.get('Test',''):
            passed_case = True
        output = item.get('Output', '')
        if 'SIGNED_LOAD ' in output:
            rows.append(json.loads(output.split('SIGNED_LOAD ', 1)[1]))
    if failed or not passed_case or len(rows) != 1:
        raise ValueError('one executed passing signed load result is required; failures/skips cannot qualify')
    row = rows[0]
    required = ['delivery_p95_ms','conflicts','retries','accepted_per_second','pool_wait_ms']
    if any(row.get(k, -1) < 0 for k in required) or not row.get('approval_replay_deduplicated'):
        raise ValueError('incomplete signed contention metrics')
    return row


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--root', default='.')
    p.add_argument('--output', required=True)
    p.add_argument('--archive-commit', help='exact source of an immutable git archive without .git')
    p.add_argument('--backend', choices=['sqlite','postgres','both'], default='both')
    args = p.parse_args()
    root = Path(args.root).resolve()
    if platform.system() != 'Linux':
        p.error('Linux reference runner required for the recorded resource units')
    if args.archive_commit:
        if (root/'.git').exists() or not re.fullmatch('[0-9a-f]{40}',args.archive_commit):
            p.error('--archive-commit requires a Git archive without .git and an exact 40-character SHA')
        source=args.archive_commit
    else:
        source = subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
        dirty = subprocess.check_output(['git','status','--porcelain','--untracked-files=normal'],cwd=root,text=True)
        if dirty:
            p.error('commit the benchmark source before measurement; keep output outside the repository')
    hashes={str(path.relative_to(root)):hashlib.sha256(path.read_bytes()).hexdigest() for folder in ['packages','tools'] for path in sorted((root/folder).rglob('*')) if path.is_file() and (path.suffix in ['.go','.py','.sql','.mod'] or path.name in ['go.mod','go.sum'])}
    for name in ['go.mod','go.sum','.go-version']:
        hashes[name]=hashlib.sha256((root/name).read_bytes()).hexdigest()
    output = Path(args.output).resolve()
    output.mkdir(mode=0o700, exist_ok=False)
    backends = ['sqlite','postgres'] if args.backend == 'both' else [args.backend]
    if 'postgres' in backends and not os.environ.get('WOS_TEST_POSTGRES_DSN'):
        p.error('real PostgreSQL DSN required; no skip is accepted')
    report = {'source_commit':source,'measured_source_sha256':hashes,'host':platform.platform(),
              'go_version':subprocess.check_output(['go','version'],cwd=root,text=True).strip(),
              'gomaxprocs':os.environ.get('GOMAXPROCS','runtime default'),
              'production_sla':False,'profiled':False,
              'resource_scope':'whole Go test process tree, including fixture setup and compilation; database CPU/RSS are not included',
              'rows':[],'complete':False}
    try:
        for backend in backends:
            for outcomes in [1,4]:
                for consumers in [2,8,50]:
                    name = f'{backend}-outcomes-{outcomes}-consumers-{consumers}'
                    log = output/(name+'.jsonl')
                    env = os.environ.copy()
                    env.update(WOS_SIGNED_LOAD='1',WOS_SHARED_LOAD='1')
                    pattern=f'^TestSignedWorkReviewLoadMatrix$/^outcomes={outcomes}$/^history=0$/^consumers={consumers}$'
                    command=['go','test','./packages/wos-core/storage/'+backend,'-run',pattern,'-count=1','-timeout=20m','-json']
                    code,usage,elapsed=run_case(command,env,root,log,1250)
                    if code:
                        raise RuntimeError(name+' failed; see retained JSONL')
                    row=parse_load(log)
                    if row['outcomes']!=outcomes or row.get('active_outcomes')!=outcomes or row['consumers']!=consumers or row['accepted_deliveries']!=consumers*2 or row.get('version_collision_winners')!=1 or row.get('version_collision_rejections')!=consumers-1:
                        raise ValueError('fixture dimensions or accepted result count differ')
                    row.update(backend=backend,wall_seconds_including_setup=elapsed,
                               process_tree_user_cpu_seconds=usage.ru_utime,
                               process_tree_system_cpu_seconds=usage.ru_stime,
                               process_tree_peak_rss_kib=usage.ru_maxrss,
                               log=log.name)
                    report['rows'].append(row)
                    print(json.dumps({'completed_case':name,'accepted':row['accepted_deliveries']}),flush=True)
        report['complete']=True
    finally:
        (output/'report.json').write_text(json.dumps(report,indent=2)+'\n')
    return 0

if __name__=='__main__':
    raise SystemExit(main())
