#!/usr/bin/env python3
"""Actual pinned v0.4.2 signed producer -> operator restore -> explicit Tenant cutover.

Only the private archive's existing test is instrumented to capture its already
validated fixture and separate issuer seed. No historical production file changes.
Private databases/seeds/logs stay outside public evidence; publish report.json only.
"""
import argparse
from contextlib import closing, contextmanager
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import socket
import time
from urllib.request import Request, urlopen
from urllib.error import URLError
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
    marker='\t\t\t\tcred, token, e := sec.IssueCredential(adminCtx, ns, principal, d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: principal}, now.Add(time.Hour))'
    assert text.count(marker)==1
    text=text.replace(marker,marker+'''
                if e == nil && backend == "sql" && principal == "review-administrator" && os.Getenv("WOS_HISTORICAL_FIXTURE_DIR") != "" {
                    if e := os.WriteFile(filepath.Join(os.Getenv("WOS_HISTORICAL_FIXTURE_DIR"),"administrator.token"),[]byte(token),0600); e != nil { t.Fatal(e) }
                }
''')
    path.write_text(text)

TENANT='0199a555-0000-7000-8000-000000000001'
TABLE_RENAMES={'namespaces':'tenants','namespace_grants':'tenant_grants','namespace_work_protocol':'tenant_work_protocol','delivery_namespace_schedule':'delivery_tenant_schedule'}
COLUMN_RENAMES={'namespace_id':'tenant_id','namespace_version':'tenant_version'}
PROOF_TABLES=('signed_protocol_facts','signed_work_contract_submissions','signed_work_contract_checkpoints','signed_operation_results')

def canonical_tables(tables):
    return {table:sorted(ops.canonical(row).decode() for row in data) for table,data in tables.items()}

def proof_snapshot(tables):
    return canonical_tables({name:tables[name] for name in PROOF_TABLES})

def expected_configuration(table,row):
    """Only the explicit current configuration transformations of migration33."""
    row=dict(row)
    if table=='namespace_grants':
        permissions=json.loads(row['permissions_json'])
        if 'namespace:admin' in permissions:
            row['permissions_json']=json.dumps(['tenant:admin' if p=='namespace:admin' else p for p in permissions],separators=(',',':'))
    elif table=='namespace_work_protocol':
        state=json.loads(row['state_json'])
        state['tenant_id']=state.pop('namespace_id')
        if state['phase']=='signed_contracts_v2':
            state['phase']='signed_contracts_v3';state['writer_epoch']=3;state['protocol_version']+=1
        elif state['phase']=='draining_to_signed_v2':
            state['phase']='draining_to_signed_v3';state['protocol_version']+=1
        row.update(phase=state['phase'],version=state['protocol_version'],writer_epoch=state['writer_epoch'],state_json=state)
    elif table=='credential_policies':
        state=json.loads(row['state_json'])
        if 'namespace:admin' in state['permitted_operations']:
            state['tenant_id']=state.pop('namespace_id')
            state['permitted_operations']=['tenant:admin' if p=='namespace:admin' else p for p in state['permitted_operations']]
            state['revision']=str(int(state['revision'])+1)
            row.update(version=int(state['revision']),state_json=state)
    return row

def verify_preserved(originals,actual,fresh_ledger=False):
    for table,before in originals.items():
        after=actual[table]
        if table=='schema_migrations':
            # New Tenant schemas replay the verified chain with new applied_at;
            # the retained shared source still owns the original ledger.
            fields=('version','name','checksum') if fresh_ledger else tuple(before[0])
            wanted={ops.canonical({k:r[k] for k in fields}) for r in before}
            found={ops.canonical({k:r[k] for k in fields}) for r in after}
            assert wanted<=found,'historical migration ledger changed'
            continue
        expected=[expected_configuration(table,row) for row in before]
        normalized=[]
        for row in after:
            row=dict(row)
            # Decode only explicitly rewritten mutable config. Every other TEXT
            # cell, including signatures/receipts/customer JSON, compares exactly.
            if table=='namespace_work_protocol':row['state_json']=json.loads(row['state_json'])
            if table=='credential_policies' and any(isinstance(r['state_json'],dict) and r['credential_id']==row['credential_id'] for r in expected):row['state_json']=json.loads(row['state_json'])
            normalized.append(row)
        assert canonical_tables({table:expected})==canonical_tables({table:normalized}),'historical row bytes or approved configuration differ: '+table

def runtime_environment(backend,database,directory,*,current,seed=True,schema=None):
    env={key:value for key,value in os.environ.items() if not key.startswith('WOS_')}
    with socket.socket() as listener:
        listener.bind(('127.0.0.1',0));port=listener.getsockname()[1]
    env.update(WOS_STORAGE_DRIVER=backend,WOS_SQLITE_PATH=database,WOS_AUTH_MODE='api_token',WOS_SERVER_SIGNING_SEED_ENV='WOS_UPGRADE_ISSUER',WOS_DELIVERY_WORKER_ENABLED='false',WOS_LISTEN=f'127.0.0.1:{port}',WOS_LOCAL_PRINCIPAL_ID='admin',WOS_BOOTSTRAP_TOKEN='signed-acquisition-bootstrap-token-at-least-32')
    if seed:env['WOS_UPGRADE_ISSUER']=(directory/'issuer.seed').read_text()
    word='TENANT' if current else 'NAMESPACE'
    env['WOS_BOOTSTRAP_'+word+'_ID']=TENANT
    env['WOS_BOOTSTRAP_'+word+'_NAME']='signed-acquisition'
    if current:env['WOS_TENANT_ID']=TENANT
    if backend=='postgres':
        parsed=urlsplit(database);options=dict(parse_qsl(parsed.query))
        if schema:options['options']=(options.get('options','')+' -c search_path='+schema).strip()
        env['WOS_POSTGRES_DSN']=urlunsplit((parsed.scheme,parsed.netloc,parsed.path,urlencode(options),parsed.fragment))
    return env

def private_command(command,env,log,check=True,timeout=90):
    with log.open('wb') as stream:
        result=subprocess.run(command,env=env,stdout=subprocess.PIPE,stderr=stream,timeout=timeout)
    if check and result.returncode:raise RuntimeError('command failed; inspect private '+str(log))
    return result

@contextmanager
def running_server(binary,env,log):
    with log.open('wb') as stream:
        process=subprocess.Popen([binary,'server'],env=env,stdout=stream,stderr=subprocess.STDOUT)
        try:
            base='http://'+env['WOS_LISTEN']
            for _ in range(200):
                if process.poll() is not None:raise RuntimeError('runtime exited; inspect private '+str(log))
                try:
                    with urlopen(base+'/readyz',timeout=1) as response:
                        if response.status==200:break
                except (URLError,TimeoutError):pass
                time.sleep(.05)
            else:raise RuntimeError('runtime readiness failed; inspect private '+str(log))
            yield base
        finally:
            process.terminate()
            try:process.wait(timeout=10)
            except subprocess.TimeoutExpired:process.kill();process.wait()

def settle_legacy(base,token,rows):
    count=0
    def command(name,payload):
        nonlocal count
        request=Request(base+'/api/v1/commands/'+name,data=json.dumps({'command':payload}).encode(),headers={'Authorization':'Bearer '+token,'Content-Type':'application/json','Idempotency-Key':'offline-tenant-cutover-'+str(count)},method='POST')
        with urlopen(request,timeout=10) as response:assert response.status==200
        count+=1
    reason='Operator explicitly settles this disposable v0.4.2 fixture before offline Tenant protocol cutover'
    for row in rows("SELECT namespace_id,outcome_id,id,version FROM work_review_cases WHERE status IN ('pending','in_review')"):
        command('intervene_signed_review_case',{'scope':{'namespace_id':row['namespace_id'],'outcome_id':row['outcome_id']},'review_case_id':row['id'],'expected_review_case_version':str(row['version']),'status':'cancelled','reason':reason})
    for table,kind in (('signed_work_contracts','execution'),('work_review_contracts','review')):
        for row in rows('SELECT namespace_id,outcome_id,id,version FROM '+table+" WHERE status='active'"):
            command('revoke_signed_contract',{'scope':{'namespace_id':row['namespace_id'],'outcome_id':row['outcome_id']},'contract_kind':kind,'contract_id':row['id'],'expected_contract_version':str(row['version']),'reason':reason})
    return count

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
            def rows(query,destination=False,query_schema=None):
                if backend=='sqlite':
                    with closing(ops.sqlite_read(target if destination else source)) as db:return [dict(zip([col[0] for col in cursor.description],row)) for cursor in [db.execute(query)] for row in cursor]
                with ops.PostgresSnapshot(ops.postgres_env('WOS_UPGRADE_TARGET_DSN' if destination else 'WOS_UPGRADE_SOURCE_DSN'),query_schema or schema) as session:
                    return [json.loads(line) for line in session.run('SELECT row_to_json(t)::text FROM ('+query+') t')]
            if backend=='sqlite':
                with closing(ops.sqlite_read(source)) as db:manifest=ops.sqlite_inspect(db)
            else:
                with ops.PostgresSnapshot(ops.postgres_env('WOS_UPGRADE_SOURCE_DSN'),schema) as session:manifest=ops.pg_inspect(session)
            assert manifest['schema_version']==26 and manifest['tables']['signed_protocol_facts']['rows']>0
            columns={table:[] for table in manifest['tables']}
            if backend=='sqlite':
                for table in columns:columns[table]=[row['name'] for row in rows('PRAGMA table_info('+ops.identifier(table)+')')]
            else:
                for column in rows("SELECT table_name,column_name FROM information_schema.columns WHERE table_schema='"+schema+"' ORDER BY table_name,ordinal_position"):
                    columns[column['table_name']].append(column['column_name'])
            def snapshot(destination=False,renamed=False,query_schema=None):
                def capture(read):
                    result={}
                    for table,names in columns.items():
                        source_table=TABLE_RENAMES.get(table,table) if renamed else table
                        selection=','.join(ops.identifier(COLUMN_RENAMES.get(col,col) if renamed else col)+' AS '+ops.identifier(col) for col in names)
                        result[table]=read('SELECT '+selection+' FROM '+ops.identifier(source_table))
                    return result
                if backend=='postgres':
                    with ops.PostgresSnapshot(ops.postgres_env('WOS_UPGRADE_TARGET_DSN' if destination else 'WOS_UPGRADE_SOURCE_DSN'),query_schema or schema) as session:
                        return capture(lambda query:[json.loads(line) for line in session.run('SELECT row_to_json(t)::text FROM ('+query+') t')])
                with closing(ops.sqlite_read(target if destination else source)) as db:
                    def read(query):
                        cursor=db.execute(query)
                        return [dict(zip([col[0] for col in cursor.description],row)) for row in cursor]
                    return capture(read)
            def current_env(database,seed=True):
                return runtime_environment(backend,database,directory,current=True,seed=seed)
            def import_shared(database,label,check=True):
                request=directory/(label+'.json')
                request.write_text(json.dumps({'source_schema':schema,'reason':'Explicit offline acceptance cutover from published v0.4.2 after stopping all fixture writers'}))
                env=current_env(database)
                env['WOS_UPGRADE_PLACEMENT_DSN']=database
                return private_command([binaries['current'],'storage','tenant','import-shared','--file',str(request),'--writers-stopped','--dsn-env','WOS_UPGRADE_PLACEMENT_DSN'],env,directory/(label+'.log'),check)
            # Keep the original database on the old schema: exercise refusal on a
            # clean operator restore so the previous release can still settle it.
            before_active=snapshot()
            active_backup=ops.backup(argparse.Namespace(backend=backend,sqlite=source,dsn_env='WOS_UPGRADE_SOURCE_DSN',schema=schema,output=str(directory/'active-backup'),source_commit=BASE,product_version='0.4.2'))
            probe=str(directory/'active-probe.sqlite3') if backend=='sqlite' else new_database()
            os.environ['WOS_UPGRADE_TARGET_DSN']=probe
            ops.restore(argparse.Namespace(backup=str(directory/'active-backup'),sqlite=probe,dsn_env='WOS_UPGRADE_TARGET_DSN',manifest_sha256=active_backup['manifest_sha256'],writers_stopped=True,expected_server_id=manifest['server_id'],issuer_fingerprint=manifest['issuer_fingerprint']))
            saved_target=target;target=probe
            active=sum(len(rows('SELECT id FROM '+table+" WHERE status='active'")) for table in ('signed_work_contracts','work_review_contracts'))
            assert active>0,'published fixture must exercise the unfinished-authority refusal'
            if backend=='sqlite':
                refused=private_command([binaries['current'],'db','migrate'],current_env(probe),directory/'active-refusal.log',False)
            else:refused=import_shared(probe,'active-refusal',False)
            assert refused.returncode!=0,'candidate accepted active historical authority'
            assert 'legacy' in (directory/'active-refusal.log').read_text().lower(),'candidate refusal did not identify unfinished historical work'
            assert canonical_tables(before_active)==canonical_tables(snapshot(True)),'refused cutover mutated historical rows or migration ledger'
            target=saved_target;os.environ['WOS_UPGRADE_TARGET_DSN']=target
            immutable_before=proof_snapshot(before_active)
            old_runtime=runtime_environment(backend,source,directory,current=False,schema=schema)
            with running_server(binaries['old'],old_runtime,directory/'settle-old-runtime.log') as base:
                operations=settle_legacy(base,(directory/'administrator.token').read_text(),rows)
            assert operations>=active,'old runtime did not settle the fixture authority'
            settled_proofs=proof_snapshot(snapshot())
            for table,proofs in immutable_before.items():
                assert set(proofs)<=set(settled_proofs[table]),'old administrative settlement rewrote existing immutable history: '+table
                if table!='signed_operation_results':assert proofs==settled_proofs[table],'administrative revocation unexpectedly produced signed facts'
            assert len(settled_proofs['signed_operation_results'])-len(immutable_before['signed_operation_results'])==operations,'each old administrative settlement must persist its durable receipt'
            originals=snapshot()
            created=ops.backup(argparse.Namespace(backend=backend,sqlite=source,dsn_env='WOS_UPGRADE_SOURCE_DSN',schema=schema,output=str(directory/'backup'),source_commit=BASE,product_version='0.4.2'))
            restored=ops.restore(argparse.Namespace(backup=str(directory/'backup'),sqlite=target,dsn_env='WOS_UPGRADE_TARGET_DSN',manifest_sha256=created['manifest_sha256'],writers_stopped=True,expected_server_id=manifest['server_id'],issuer_fingerprint=manifest['issuer_fingerprint']))
            shared_schema=schema
            placement=None
            if backend=='sqlite':
                private_command([binaries['current'],'db','migrate'],current_env(target),directory/'migration.log')
            else:
                imported=json.loads(import_shared(target,'migration').stdout)['data']
                assert imported['source_write_fenced'] and len(imported['placements'])==1
                placement=imported['placements'][0]
                assert placement['tenant_id']==TENANT and placement['state']=='suspended'
                schema=placement['schema']
                verify_preserved(originals,snapshot(True,True,shared_schema))
                fenced=private_command(['psql','-X','-qAt','-v','ON_ERROR_STOP=1','-c','UPDATE '+ops.identifier(shared_schema)+'.tenants SET name=name WHERE FALSE'],postgres_environment(target),directory/'source-fence.log',False)
                assert fenced.returncode!=0 and 'read-only after Tenant placement' in (directory/'source-fence.log').read_text(),'retained shared source did not enforce its placement fence'
            current_schema=verify_current_migrations(rows('SELECT version,name,checksum FROM schema_migrations ORDER BY version',True),candidate_migrations[backend])
            verify_preserved(originals,snapshot(True,True),fresh_ledger=backend=='postgres')
            documents={'keys':[json.loads(row['state_json']) for row in rows('SELECT state_json FROM signing_keys',True)],'facts':[json.loads(row['state_json']) for row in rows('SELECT state_json FROM signed_protocol_facts',True)]}
            verified=json.loads(subprocess.check_output([binaries['verifier']],input=json.dumps(documents).encode()))
            # Run the genuinely old writer against the migrated data, with its
            # original Namespace configuration (not the new Tenant environment).
            old_target=runtime_environment(backend,target,directory,current=False,schema=schema)
            refused=private_command([binaries['old'],'db','migrate'],old_target,directory/'old-refusal.log',False)
            assert refused.returncode!=0 and 'schema' in (directory/'old-refusal.log').read_text().lower(),'old binary did not safely refuse schema'
            runtime=current_env(target)
            if placement:
                request=directory/'resume.json'
                request.write_text(json.dumps({'tenant_id':TENANT,'expected_version':placement['version'],'reason':'All restored proof bytes verified; explicitly activate this offline acceptance fixture'}))
                runtime['WOS_UPGRADE_PLACEMENT_DSN']=target
                private_command([binaries['current'],'storage','tenant','resume','--file',str(request),'--writers-stopped','--dsn-env','WOS_UPGRADE_PLACEMENT_DSN'],runtime,directory/'resume.log')
            missing=runtime.copy();missing.pop('WOS_UPGRADE_ISSUER')
            failed=private_command([binaries['current'],'server'],missing,directory/'missing-seed.log',False,timeout=15)
            assert failed.returncode!=0 and 'seed' in (directory/'missing-seed.log').read_text().lower(),'missing issuer seed was not explicit'
            with running_server(binaries['current'],runtime,directory/'candidate-runtime.log'):
                pass
            verify_preserved(originals,snapshot(True,True),fresh_ledger=backend=='postgres')
            results.append({'backend':backend,'old_schema':26,'current_schema':current_schema,'candidate_migration_manifest_sha256':hashlib.sha256(ops.canonical(candidate_migrations[backend])).hexdigest(),'all_candidate_migrations_matched_source':True,'old_producer_test_sha256':TEST_HASH[backend],'historical_tables_checked':len(originals),'active_authority_cutover_refused_without_mutation':True,'legacy_administrative_settlement_operations':operations,'exact_immutable_proof_sha256':hashlib.sha256(ops.canonical(settled_proofs)).hexdigest(),'producer_proofs_preserved_during_settlement':True,'mutable_configuration_cutover_checked':True,'current_codec_historical_dsse':verified,'old_binary_new_schema_refused':True,'missing_private_seed_failed_explicitly':True,'candidate_runtime_ready':True,'issuer_identity_preserved':True,'schema_placement':{'source_retained_write_fenced':True,'imported_suspended_then_explicitly_resumed':True} if placement else {'single_tenant_file_bound':True},'backup_restore':restored})
    finally:
        for name in reversed(owned):subprocess.run(['docker','exec',container,'dropdb','-U',dsn.username,'--force',name],check=True)
        (root/'report.json').write_text(json.dumps({'complete':len(results)==len(backends),'baseline_source':BASE,'baseline_production_sha256':before,'candidate_source':args.source,'old_binary_sha256':ops.file_hash(Path(binaries['old'])),'candidate_binary_sha256':ops.file_hash(Path(binaries['current'])),'historical_test_only_capture':True,'private_seed_separate_from_backup':True,'results':results,'production_sla':False},indent=2)+'\n')

def postgres_environment(dsn):
    os.environ['WOS_UPGRADE_INITIAL_DSN']=dsn
    return ops.postgres_env('WOS_UPGRADE_INITIAL_DSN')

if __name__=='__main__':main()
