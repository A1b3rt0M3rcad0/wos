#!/usr/bin/env python3
"""Provision and remove only owned disposable databases for the API pilots."""
import argparse
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
from urllib.parse import urlsplit,urlunsplit

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary',required=True)
    parser.add_argument('--source',required=True)
    parser.add_argument('--output',required=True)
    args=parser.parse_args()
    container=os.environ.get('WOS_TEST_POSTGRES_CONTAINER','')
    dsn=urlsplit(os.environ.get('WOS_TEST_POSTGRES_DSN',''))
    if not container or container.startswith('-') or dsn.scheme not in ('postgres','postgresql') or not dsn.username:
        parser.error('explicit disposable PostgreSQL test container and URI required')
    root=Path(args.output).resolve();root.mkdir(mode=0o700,exist_ok=False)
    databases=[]
    def database():
        name='wos_qualification_'+secrets.token_hex(12)
        subprocess.run(['docker','exec',container,'createdb','-U',dsn.username,name],check=True)
        databases.append(name)
        return urlunsplit((dsn.scheme,dsn.netloc,'/'+name,dsn.query,dsn.fragment))
    try:
        reports=[]
        for scenario in ('failover','restore'):
            env=os.environ.copy();env['WOS_PILOT_SOURCE_DSN']=database()
            command=[sys.executable,str(Path(__file__).with_name('two-server-delivery.py')),
                '--binary',args.binary,'--source-commit',args.source,'--dsn-env','WOS_PILOT_SOURCE_DSN',
                '--output',str(root/scenario)]
            if scenario=='restore':
                env['WOS_PILOT_TARGET_DSN']=database();command+=['--restore-dsn-env','WOS_PILOT_TARGET_DSN']
            subprocess.run(command,env=env,check=True)
            report=json.loads((root/scenario/'report.json').read_text())
            assert report['complete'] and report['source_commit']==args.source
            reports.append(report)
        (root/'reports.json').write_text(json.dumps({'complete':True,'source_commit':args.source,'results':reports},indent=2)+'\n')
    finally:
        for name in reversed(databases):
            subprocess.run(['docker','exec',container,'dropdb','-U',dsn.username,'--force',name],check=True)

if __name__=='__main__':main()
