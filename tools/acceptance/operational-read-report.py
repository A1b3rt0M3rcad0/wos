#!/usr/bin/env python3
"""Controlled same-host before/after rich-read benchmark; opt-in POSIX laboratory.

Both archives receive the identical reviewed v2 test harness. Production sources
stay unchanged. Fixture validation/setup and ANALYZE precede request timing;
whole-child CPU/peak RSS include setup and are labelled separately.
"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import re
import secrets
import statistics
import subprocess
import time
from urllib.parse import urlsplit,urlunsplit

CASES=('section_work_items','snapshot_with_proof','focal_with_proof')

def parse(log,size,samples,repetitions):
    result={name:[] for name in CASES}
    for line in log.splitlines():
        match=re.match(r'^BenchmarkOperationalReads/('+ '|'.join(CASES)+r')/'+str(size)+r'-\d+\s+(\d+)\s+(.+)$',line)
        if not match:continue
        assert int(match[2])==samples,'sample count differs'
        fields=match[3].split();assert len(fields)%2==0
        metrics={fields[i+1]:float(fields[i]) for i in range(0,len(fields),2)}
        assert all(math.isfinite(value) and value>=0 for value in metrics.values())
        assert {'p50_ns','p95_ns','p99_ns','B/op','allocs/op','response_bytes','sql_statements/op','pool_waits/op','pool_wait_ns/op'}<=metrics.keys()
        assert metrics['p50_ns']<=metrics['p95_ns']<=metrics['p99_ns']
        assert metrics['response_bytes']<=256*1024
        result[match[1]].append(metrics)
    assert all(len(values)==repetitions for values in result.values()),'missing or duplicate benchmark repetitions'
    assert re.search(r'^PASS$',log,re.M) and not re.search(r'^FAIL',log,re.M),'benchmark did not pass'
    return result

def hash_files(root,predicate):
    h=hashlib.sha256()
    for path in sorted(root.rglob('*')):
        if path.is_file() and '.git' not in path.parts and predicate(path):
            h.update(str(path.relative_to(root)).encode()+b'\0'+path.read_bytes()+b'\0')
    return h.hexdigest()

def run_child(command,env,log):
    began=time.monotonic()
    with log.open('wb') as stream:
        child=subprocess.Popen(command,env=env,stdout=stream,stderr=subprocess.STDOUT)
        _,status,usage=os.wait4(child.pid,0)
        child.returncode=os.waitstatus_to_exitcode(status)
    assert child.returncode==0,'benchmark failed; raw log retained at '+str(log)
    return {'wall_seconds':time.monotonic()-began,'user_cpu_seconds':usage.ru_utime,
        'system_cpu_seconds':usage.ru_stime,'peak_rss_bytes':usage.ru_maxrss*(1 if platform.system()=='Darwin' else 1024),
        'boundary':'whole precompiled Go test child, including domain fixture/setup/ANALYZE and all timed reads; excludes database process'}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for field in ('baseline','candidate','baseline-commit','candidate-commit','output'):p.add_argument('--'+field,required=True)
    p.add_argument('--sizes',nargs='+',type=int,choices=(100,1000,10000),default=[100,1000,10000])
    p.add_argument('--backend',choices=('sqlite','postgres','both'),default='both')
    p.add_argument('--repetitions',type=int,default=3)
    args=p.parse_args()
    if not hasattr(os,'wait4'):p.error('POSIX wait4 is required for measured CPU/RSS')
    assert args.repetitions>=3
    for commit in (args.baseline_commit,args.candidate_commit):assert re.fullmatch('[0-9a-f]{40}',commit)
    roots={'baseline':Path(args.baseline).resolve(),'candidate':Path(args.candidate).resolve()}
    output=Path(args.output).resolve();output.mkdir(mode=0o700,exist_ok=False)
    backends=('sqlite','postgres') if args.backend=='both' else (args.backend,)
    report={'schema':1,'complete':False,'production_sla':False,'baseline_source':args.baseline_commit,'candidate_source':args.candidate_commit,
        'harness':'v2: same ANALYZE outside timing; SQL tracing and pool deltas on both sources',
        'aggregation':'median of all three or more equal-sample p95 repetitions; raw repetitions retained',
        'targets_declared_before_run':{'simple_10000_p95_ratio':.7,'simple_10000_bytes_ratio':.5,'full_10000_p95_ratio':.7,'full_10000_bytes_ratio':.5,'focal_1000_p95_ratio':1.1},
        'environment':{'platform':platform.platform(),'go':subprocess.check_output(['go','version'],text=True).strip(),'gomaxprocs':os.environ.get('GOMAXPROCS','default')},'sources':{},'runs':[]}
    for label,root in roots.items():
        report['sources'][label]={'production_go_sql_sha256':hash_files(root,lambda q:q.suffix=='.sql' or q.suffix=='.go' and not q.name.endswith('_test.go')),
            'read_harness_sha256':hash_files(root,lambda q:q.name in ('operational_read_benchmark_test.go','operational_trace_test.go'))}
    assert report['sources']['baseline']['read_harness_sha256']==report['sources']['candidate']['read_harness_sha256'],'harness files differ'
    container=os.environ.get('WOS_TEST_POSTGRES_CONTAINER','');dsn=urlsplit(os.environ.get('WOS_TEST_POSTGRES_DSN',''))
    if 'postgres' in backends:assert container and not container.startswith('-') and dsn.scheme in ('postgres','postgresql') and dsn.username
    try:
        binaries={}
        for backend in backends:
            for label,root in roots.items():
                binary=output/(label+'-'+backend+'.test')
                with (output/(label+'-'+backend+'-build.log')).open('wb') as log:
                    subprocess.run(['go','test','-c','-o',str(binary),'./packages/wos-core/storage/'+backend],cwd=root,stdout=log,stderr=subprocess.STDOUT,check=True)
                binaries[(backend,label)]=binary
        for backend in backends:
            for size in args.sizes:
                for label in ('baseline','candidate'):
                    env=os.environ.copy();env.update(WOS_OPERATIONAL_READ_SIZE=str(size),WOS_OPERATIONAL_ANALYZE='1')
                    database=None
                    try:
                        if backend=='postgres':
                            database='wos_read_qualification_'+secrets.token_hex(12)
                            subprocess.run(['docker','exec',container,'createdb','-U',dsn.username,database],check=True)
                            env['WOS_TEST_POSTGRES_DSN']=urlunsplit((dsn.scheme,dsn.netloc,'/'+database,dsn.query,dsn.fragment))
                        samples=40 if size==10000 else 80
                        log=output/(label+'-'+backend+'-'+str(size)+'.log')
                        resource=run_child([str(binaries[(backend,label)]),'-test.run=^$','-test.bench=^BenchmarkOperationalReads$','-test.benchtime='+str(samples)+'x','-test.count='+str(args.repetitions)],env,log)
                        metrics=parse(log.read_text(),size,samples,args.repetitions)
                        report['runs'].append({'source':label,'backend':backend,'items':size,'samples_per_repetition':samples,'repetitions':args.repetitions,'metrics':metrics,'resource':resource})
                        (output/'report.json').write_text(json.dumps(report,indent=2)+'\n')
                    finally:
                        if database:subprocess.run(['docker','exec',container,'dropdb','-U',dsn.username,'--force',database],check=True)
        targets=[]
        for backend in backends:
            for size,case,key,maximum in ((10000,'section_work_items','p95_ns',.7),(10000,'section_work_items','B/op',.5),(10000,'snapshot_with_proof','p95_ns',.7),(10000,'snapshot_with_proof','B/op',.5),(1000,'focal_with_proof','p95_ns',1.1)):
                if size not in args.sizes:continue
                values={run['source']:statistics.median(m[key] for m in run['metrics'][case]) for run in report['runs'] if run['backend']==backend and run['items']==size}
                ratio=values['candidate']/values['baseline']
                targets.append({'backend':backend,'items':size,'case':case,'metric':key,'ratio':ratio,'maximum':maximum,'passed':ratio<=maximum})
        report.update(complete=True,targets=targets,targets_passed=all(v['passed'] for v in targets))
    finally:
        (output/'report.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps({'complete':report['complete'],'targets_passed':report['targets_passed'],'report':str(output/'report.json')}))
    return 0 if report['targets_passed'] else 1

if __name__=='__main__':raise SystemExit(main())
