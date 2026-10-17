#!/usr/bin/env python3
"""Actual pinned v0.4.2 signed producer -> operator restore -> additive upgrade.

Only the private archive's existing test is instrumented to capture its already
validated fixture and separate issuer seed. No historical production file changes.
Private databases/seeds/logs stay outside public evidence; publish report.json only.
"""
import argparse
from contextlib import closing
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
from urllib.parse import urlsplit,urlunsplit,parse_qsl,urlencode

BASE='b572269085494ddd4d6370716ff1cb360b46065f'
TEST_HASH={'sqlite':'1fbb8b202fde5ee3d669e72e65ff9a249aefd4afa70104ef93bf0a56186df0c8','postgres':'d596ea648aeac68669aa3c7169197f6491f5357dd2c207c2ec5c9bccb7d86db7'}
spec=importlib.util.spec_from_file_location('ops',Path(__file__).parents[1]/'operations'/'backup_restore.py')
ops=importlib.util.module_from_spec(spec);spec.loader.exec_module(ops)

def source_migrations(root):
    """Read the complete candidate chain; SQL checksums remain dialect-specific."""
    manifests={}
    for backend in ('sqlite','postgres'):
        directory=root/'packages/wos-core/storage'/backend/'migrations'
        manifest=[]
        for path in sorted(directory.glob('*.sql')):
            match=re.fullmatch(r'([0-9]{4})_[a-z0-9_]+\.sql',path.name)
            assert match,'invalid candidate migration filename: '+str(path)
            manifest.append({'version':int(match[1]),'name':path.name,'checksum':ops.file_hash(path)})
        assert manifest,'candidate migration chain is empty: '+backend
        versions=[item['version'] for item in manifest]
        assert versions==list(range(1,versions[-1]+1)),'candidate migration chain has missing or duplicate versions: '+backend
        manifests[backend]=manifest
    assert [(item['version'],item['name']) for item in manifests['sqlite']]==[(item['version'],item['name']) for item in manifests['postgres']],'candidate adapter migration chains differ'
    return manifests

def verify_current_migrations(actual,expected):
    assert actual==expected,'migrated schema does not match the complete candidate source migration manifest'
    return actual[-1]['version']

def production_hash(root):
    h=hashlib.sha256()
    for path in sorted(root.rglob('*')):
        if path.is_file() and (path.suffix=='.sql' or path.suffix=='.go' and not path.name.endswith('_test.go')):
            h.update(str(path.relative_to(root)).encode()+b'\0'+path.read_bytes()+b'\0')
    return h.hexdigest()

def instrument(archive,backend):
    path=archive/'packages/wos-core/storage'/backend/'signed_review_journey_test.go'
    raw=path.read_bytes();assert hashlib.sha256(raw).hexdigest()==TEST_HASH[backend],'historical test differs from pinned source'
    text=raw.decode().replace('"path/filepath"','"path/filepath"\n"os"'+ ('\n"os/exec"\n"net/url"' if backend=='postgres' else ''))
    capture='''
            if directory := os.Getenv("WOS_HISTORICAL_FIXTURE_DIR"); directory != "" && backend == "sql" {
                defer func() {
                    st := store.(*Store)
                    CAPTURE
                }()
            }
'''
    if backend=='sqlite':
        body='if e := st.Backup(context.Background(),filepath.Join(directory,"source.sqlite3")); e != nil { t.Fatal(e) }'
    else:
        body='''
                    var database string
                    if e := st.db.QueryRowContext(context.Background(),"SELECT current_database()").Scan(&database); e != nil { t.Fatal(e) }
                    dsn,e := url.Parse(os.Getenv("WOS_TEST_POSTGRES_DSN")); if e != nil { t.Fatal(e) }
                    metadata,_ := json.Marshal(map[string]string{"schema":st.Path()})
                    if e=os.WriteFile(filepath.Join(directory,"source.json"),metadata,0600); e != nil { t.Fatal(e) }
                    output,e := os.OpenFile(filepath.Join(directory,"source.dump"),os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600); if e != nil { t.Fatal(e) }
                    defer output.Close()
                    dump := exec.Command("docker","exec","-e","PGOPTIONS=-c max_parallel_workers_per_gather=0",os.Getenv("WOS_TEST_POSTGRES_CONTAINER"),"pg_dump","-U",dsn.User.Username(),"-d",database,"--schema="+st.Path(),"--format=custom","--no-owner","--no-privileges")
                    dump.Stdout=output
                    if e=dump.Run(); e != nil { t.Fatal(e) }
'''
    marker='\t\t\tctx := context.Background()'
    assert text.count(marker)==1;text=text.replace(marker,capture.replace('CAPTURE',body)+marker)
    marker='\t\t\tpub, private, err := ed25519.GenerateKey(rand.Reader)'
    assert text.count(marker)==1
    text=text.replace(marker,marker+'''
            if err == nil && backend == "sql" && os.Getenv("WOS_HISTORICAL_FIXTURE_DIR") != "" {
                if e := os.WriteFile(filepath.Join(os.Getenv("WOS_HISTORICAL_FIXTURE_DIR"),"issuer.seed"),[]byte(base64.StdEncoding.EncodeToString(private.Seed())),0600); e != nil { t.Fatal(e) }
            }
''')
    path.write_text(text)

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('archive','old-binary','binary','verifier','source','output'):p.add_argument('--'+name,required=True)
    p.add_argument('--driver',choices=('sqlite','postgres','both'),default='both')
    args=p.parse_args();os.umask(0o077)
    candidate_migrations=source_migrations(Path(__file__).resolve().parents[2])
    archive=Path(args.archive).resolve();assert not (archive/'.git').exists(),'use a private archive, never the working checkout'
    tree=subprocess.check_output(['git','ls-tree','-rz',BASE],cwd=Path(__file__).parents[2])
    for entry in tree.split(b'\0'):
        if not entry:continue
        metadata,name=entry.split(b'\t',1);name=name.decode()
        if (name.endswith('.go') and not name.endswith('_test.go')) or name.endswith('.sql') or name in ('go.mod','go.sum'):
            contents=(archive/name).read_bytes()
            assert hashlib.sha1(b'blob '+str(len(contents)).encode()+b'\0'+contents).hexdigest()==metadata.split()[2].decode(),'historical production archive differs: '+name
    root=Path(args.output).resolve();root.mkdir(mode=0o700,exist_ok=False)
    binaries={name:str(Path(value).resolve()) for name,value in [('old',args.old_binary),('current',args.binary),('verifier',args.verifier)]}
    assert '0.4.2 (commit '+BASE+',' in subprocess.check_output([binaries['old'],'version'],text=True)
    assert '(commit '+args.source+',' in subprocess.check_output([binaries['current'],'version'],text=True)
    backends=('sqlite','postgres') if args.driver=='both' else (args.driver,)
    before=production_hash(archive)
    for backend in backends:instrument(archive,backend)
    assert before==production_hash(archive),'historical production sources changed'
    container=os.environ.get('WOS_TEST_POSTGRES_CONTAINER','');dsn=urlsplit(os.environ.get('WOS_TEST_POSTGRES_DSN',''))
    owned=[];results=[]
    def new_database():
        assert container and not container.startswith('-') and dsn.scheme in ('postgres','postgresql') and dsn.username
        name='wos_qualification_upgrade_'+secrets.token_hex(10)
        subprocess.run(['docker','exec',container,'createdb','-U',dsn.username,name],check=True);owned.append(name)
        return urlunsplit((dsn.scheme,dsn.netloc,'/'+name,dsn.query,dsn.fragment))
    try:
        for backend in backends:
            directory=root/backend;directory.mkdir(mode=0o700)
            env=os.environ.copy();env['WOS_HISTORICAL_FIXTURE_DIR']=str(directory)
            env.pop('WOS_TEST_OPERATIONS_TOOL',None)
            if backend=='postgres':env['WOS_TEST_POSTGRES_DSN']=new_database()
            with (directory/'producer.log').open('wb') as log:
                subprocess.run(['go','test','-count=1','-v','-run','^TestSignedReviewCorrectionJourneyWithoutExecutorAuthority/sql$','./packages/wos-core/storage/'+backend],cwd=archive,env=env,stdout=log,stderr=subprocess.STDOUT,check=True)
            schema='main';source=str(directory/'source.sqlite3');target=str(directory/'restored.sqlite3')
            if backend=='postgres':
                source=new_database();target=new_database();schema=json.loads((directory/'source.json').read_text())['schema']
                env['WOS_UPGRADE_SOURCE_DSN']=source
                with (directory/'source.dump').open('rb') as stream:
                    ops.pg_command('pg_restore',['--exit-on-error','--single-transaction','--no-owner','--no-privileges','--dbname='+urlsplit(source).path.lstrip('/')],postgres_environment(source),input_file=stream)
            os.environ['WOS_UPGRADE_SOURCE_DSN']=source;os.environ['WOS_UPGRADE_TARGET_DSN']=target
            def rows(query,destination=False):
                if backend=='sqlite':
                    with closing(ops.sqlite_read(target if destination else source)) as db:return [dict(zip([col[0] for col in cursor.description],row)) for cursor in [db.execute(query)] for row in cursor]
                with ops.PostgresSnapshot(ops.postgres_env('WOS_UPGRADE_TARGET_DSN' if destination else 'WOS_UPGRADE_SOURCE_DSN'),schema) as session:
                    return [json.loads(line) for line in session.run('SELECT row_to_json(t)::text FROM ('+query+') t')]
            if backend=='sqlite':
                with closing(ops.sqlite_read(source)) as db:manifest=ops.sqlite_inspect(db)
            else:
                with ops.PostgresSnapshot(ops.postgres_env('WOS_UPGRADE_SOURCE_DSN'),schema) as session:manifest=ops.pg_inspect(session)
            assert manifest['schema_version']==26 and manifest['tables']['signed_protocol_facts']['rows']>0
            columns={}
            for table in manifest['tables']:
                if backend=='sqlite':columns[table]=[row['name'] for row in rows('PRAGMA table_info('+ops.identifier(table)+')')]
                else:columns[table]=[row['column_name'] for row in rows("SELECT column_name FROM information_schema.columns WHERE table_schema='"+schema+"' AND table_name='"+table+"' ORDER BY ordinal_position")]
            originals={table:sorted(ops.canonical(row).decode() for row in rows('SELECT '+','.join(ops.identifier(col) for col in names)+' FROM '+ops.identifier(table))) for table,names in columns.items()}
            created=ops.backup(argparse.Namespace(backend=backend,sqlite=source,dsn_env='WOS_UPGRADE_SOURCE_DSN',schema=schema,output=str(directory/'backup'),source_commit=BASE,product_version='0.4.2'))
            restored=ops.restore(argparse.Namespace(backup=str(directory/'backup'),sqlite=target,dsn_env='WOS_UPGRADE_TARGET_DSN',manifest_sha256=created['manifest_sha256'],writers_stopped=True,expected_server_id=manifest['server_id'],issuer_fingerprint=manifest['issuer_fingerprint']))
            runtime=os.environ.copy()
            for name in list(runtime):
                if name.startswith('WOS_'):runtime.pop(name)
            with socket.socket() as listener:
                listener.bind(('127.0.0.1',0));port=listener.getsockname()[1]
            runtime.update(WOS_STORAGE_DRIVER=backend,WOS_SQLITE_PATH=target,WOS_AUTH_MODE='api_token',WOS_SERVER_SIGNING_SEED_ENV='WOS_UPGRADE_ISSUER',WOS_UPGRADE_ISSUER=(directory/'issuer.seed').read_text(),WOS_DELIVERY_WORKER_ENABLED='false',WOS_LISTEN=f'127.0.0.1:{port}',
                WOS_LOCAL_PRINCIPAL_ID='admin',WOS_BOOTSTRAP_TENANT_ID='0199a555-0000-7000-8000-000000000001',WOS_BOOTSTRAP_TENANT_NAME='signed-acquisition',WOS_BOOTSTRAP_TOKEN='signed-acquisition-bootstrap-token-at-least-32')
            if backend=='postgres':
                parsed=urlsplit(target);options=dict(parse_qsl(parsed.query));options['options']=(options.get('options','')+' -c search_path='+schema).strip()
                runtime['WOS_POSTGRES_DSN']=urlunsplit((parsed.scheme,parsed.netloc,parsed.path,urlencode(options),parsed.fragment))
            with (directory/'migration.log').open('wb') as log:subprocess.run([binaries['current'],'db','migrate'],env=runtime,stdout=log,stderr=subprocess.STDOUT,check=True)
            current_schema=verify_current_migrations(rows('SELECT version,name,checksum FROM schema_migrations ORDER BY version',True),candidate_migrations[backend])
            def preserved():
                for table,names in columns.items():
                    actual=sorted(ops.canonical(row).decode() for row in rows('SELECT '+','.join(ops.identifier(col) for col in names)+' FROM '+ops.identifier(table),True))
                    if table=='schema_migrations':assert set(originals[table])<=set(actual)
                    else:assert originals[table]==actual,'historical row bytes changed: '+table
            preserved()
            documents={'keys':[json.loads(row['state_json']) for row in rows('SELECT state_json FROM signing_keys',True)],'facts':[json.loads(row['state_json']) for row in rows('SELECT state_json FROM signed_protocol_facts',True)]}
            verified=json.loads(subprocess.check_output([binaries['verifier']],input=json.dumps(documents).encode()))
            with (directory/'old-refusal.log').open('wb') as log:refused=subprocess.run([binaries['old'],'db','migrate'],env=runtime,stdout=log,stderr=subprocess.STDOUT)
            assert refused.returncode!=0 and 'schema' in (directory/'old-refusal.log').read_text().lower(),'old binary did not safely refuse schema'
            missing=runtime.copy();missing.pop('WOS_UPGRADE_ISSUER')
            with (directory/'missing-seed.log').open('wb') as log:failed=subprocess.run([binaries['current'],'server'],env=missing,stdout=log,stderr=subprocess.STDOUT,timeout=15)
            assert failed.returncode!=0 and 'seed' in (directory/'missing-seed.log').read_text().lower(),'missing issuer seed was not explicit'
            preserved()
            results.append({'backend':backend,'old_schema':26,'current_schema':current_schema,'candidate_migration_manifest_sha256':hashlib.sha256(ops.canonical(candidate_migrations[backend])).hexdigest(),'all_candidate_migrations_matched_source':True,'old_producer_test_sha256':TEST_HASH[backend],'all_original_tables_preserved':len(originals),'old_migration_rows_preserved':True,'current_codec_historical_dsse':verified,'old_binary_new_schema_refused':True,'missing_private_seed_failed_explicitly':True,'issuer_identity_preserved':True,'backup_restore':restored})
    finally:
        for name in reversed(owned):subprocess.run(['docker','exec',container,'dropdb','-U',dsn.username,'--force',name],check=True)
        (root/'report.json').write_text(json.dumps({'complete':len(results)==len(backends),'baseline_source':BASE,'baseline_production_sha256':before,'candidate_source':args.source,'old_binary_sha256':ops.file_hash(Path(binaries['old'])),'candidate_binary_sha256':ops.file_hash(Path(binaries['current'])),'historical_test_only_capture':True,'private_seed_separate_from_backup':True,'results':results,'production_sla':False},indent=2)+'\n')

def postgres_environment(dsn):
    os.environ['WOS_UPGRADE_INITIAL_DSN']=dsn
    return ops.postgres_env('WOS_UPGRADE_INITIAL_DSN')

if __name__=='__main__':main()
