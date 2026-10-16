#!/usr/bin/env python3
"""Test-only libpq clients from the explicitly selected disposable PG container.

Production operator procedures use native matching clients on PATH. This adapter
keeps CI on the service image's client version and refuses connection URIs in argv.
"""
import os
from pathlib import Path
import shlex
import subprocess
import sys
import uuid

def main():
    if len(sys.argv)==3 and sys.argv[1]=='install':
        root=Path(sys.argv[2]);root.mkdir(mode=0o700,parents=True,exist_ok=True)
        for tool in ('psql','pg_dump','pg_restore'):
            path=root/tool
            path.write_text('#!/bin/sh\nexec '+shlex.quote(sys.executable)+' '+shlex.quote(str(Path(__file__).resolve()))+' '+tool+' "$@"\n')
            path.chmod(0o700)
        return 0
    if len(sys.argv)<2 or sys.argv[1] not in ('psql','pg_dump','pg_restore'):
        raise SystemExit('select psql, pg_dump or pg_restore; or install BIN_DIRECTORY')
    tool,args=sys.argv[1],sys.argv[2:]
    container=os.environ.get('WOS_TEST_POSTGRES_CONTAINER','')
    if not container or container.startswith('-'):
        raise SystemExit('explicit disposable WOS_TEST_POSTGRES_CONTAINER required')
    if any('postgres://' in arg or 'postgresql://' in arg for arg in args):
        raise SystemExit('connection URI in test client argv refused')
    base=['docker'];temporary=None
    try:
        for index,arg in enumerate(args):
            if arg.startswith('--use-list='):
                temporary='/tmp/wos-qualification-restore-list-'+uuid.uuid4().hex
                subprocess.run(base+['cp',arg.split('=',1)[1],container+':'+temporary],check=True,stdout=subprocess.DEVNULL)
                args[index]='--use-list='+temporary
        command=base+['exec','-i','-e','PGHOST=127.0.0.1','-e','PGPORT=5432']
        for key in ('PGDATABASE','PGUSER','PGPASSWORD','PGOPTIONS','PGCLIENTENCODING','PGCONNECT_TIMEOUT','PGSSLMODE'):
            command+=['-e',key]
        return subprocess.run(command+[container,tool]+args).returncode
    finally:
        if temporary:
            subprocess.run(base+['exec',container,'rm','-f',temporary],stdout=subprocess.DEVNULL,check=True)

if __name__=='__main__':raise SystemExit(main())
