#!/usr/bin/env python3
"""Opt-in schema-30 signed fixture -> additive workspace-note upgrade.

The pinned historical production tree stays unchanged. Only its existing signed
journey test is instrumented in a private archive to retain the validated DB.
Private databases/logs stay in --output; publish only report.json.
"""
import argparse
from contextlib import closing
import hashlib
import importlib.util
import json
import os
import re
import secrets
from pathlib import Path
import subprocess
import tarfile
from urllib.parse import urlsplit, urlunsplit, parse_qsl, urlencode

BASE = '5d945d3081f85927b80c686d695849bd1df73a5d'
ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('operations', ROOT/'tools/operations/backup_restore.py')
ops = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ops)

def production_hash(directory):
    digest = hashlib.sha256()
    for path in sorted(directory.rglob('*')):
        if path.is_file() and (path.suffix == '.sql' or path.suffix == '.go' and not path.name.endswith('_test.go') or path.name in ('go.mod', 'go.sum')):
            digest.update(str(path.relative_to(directory)).encode()+b'\0'+path.read_bytes()+b'\0')
    return digest.hexdigest()

def source_migrations(root):
    manifests={}
    for backend in ('sqlite','postgres'):
        manifest=[]
        for path in sorted((root/'packages/wos-core/storage'/backend/'migrations').glob('*.sql')):
            match=re.fullmatch(r'([0-9]{4})_[a-z0-9_]+\.sql',path.name)
            assert match,'invalid migration filename: '+path.name
            manifest.append({'version':int(match[1]),'name':path.name,'checksum':ops.file_hash(path)})
        assert manifest,'empty migration chain: '+backend
        versions=[item['version'] for item in manifest]
        assert versions==list(range(1,versions[-1]+1)),'non-contiguous migration chain: '+backend
        manifests[backend]=manifest
    assert [(v['version'],v['name']) for v in manifests['sqlite']]==[(v['version'],v['name']) for v in manifests['postgres']],'adapter migration chains differ'
    return manifests

def verify_current_migrations(actual,expected):
    assert actual==expected,'database migrations do not match complete candidate source chain'
    return actual[-1]['version']

def verify_candidate_source(source):
    if source=='working-tree':return False
    assert re.fullmatch(r'[0-9a-f]{40}',source),'qualification source must be a full commit or working-tree'
    expected=set()
    for entry in subprocess.check_output(['git','ls-tree','-rz',source],cwd=ROOT).split(b'\0'):
        if not entry:continue
        metadata,name=entry.split(b'\t',1);name=name.decode()
        if (name.startswith('packages/wos-core/') and (name.endswith('.sql') or name.endswith('.go') and not name.endswith('_test.go'))) or name in ('go.mod','go.sum','tools/acceptance/workspace-notes-upgrade/main.go'):
            expected.add(name);data=(ROOT/name).read_bytes()
            assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()==metadata.split()[2].decode(),'candidate source differs from declared commit: '+name
    actual={str(path.relative_to(ROOT)) for path in (ROOT/'packages/wos-core').rglob('*') if path.is_file() and (path.suffix=='.sql' or path.suffix=='.go' and not path.name.endswith('_test.go'))}
    actual.update(('go.mod','go.sum','tools/acceptance/workspace-notes-upgrade/main.go'))
    assert actual==expected,'candidate production file set differs from declared commit'
    return True

def instrument(archive, backend):
    path = archive/'packages/wos-core/storage'/backend/'signed_review_journey_test.go'
    original = subprocess.check_output(['git', 'show', BASE+':'+str(path.relative_to(archive))], cwd=ROOT)
    assert path.read_bytes() == original, 'historical fixture differs from its pinned Git blob'
    text = original.decode().replace('"path/filepath"', '"path/filepath"\n"os"' + ('\n"os/exec"' if backend=='postgres' else ''))
    if backend == 'sqlite':
        capture = 'if e:=st.Backup(context.Background(),filepath.Join(directory,"source.sqlite3"));e!=nil{t.Fatal(e)}'
    else:
        capture = '''var database string
        if e:=st.db.QueryRowContext(context.Background(),"SELECT current_database()").Scan(&database);e!=nil{t.Fatal(e)}
        data,_:=json.Marshal(map[string]string{"schema":st.Path()})
        if e:=os.WriteFile(filepath.Join(directory,"source.json"),data,0600);e!=nil{t.Fatal(e)}
        dump,e:=os.OpenFile(filepath.Join(directory,"source.dump"),os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if e!=nil{t.Fatal(e)};defer dump.Close()
        command:=exec.Command("docker","exec",os.Getenv("WOS_TEST_POSTGRES_CONTAINER"),"pg_dump","-U","postgres","-d",database,"--schema="+st.Path(),"--format=custom","--no-owner","--no-privileges");command.Stdout=dump;if e=command.Run();e!=nil{t.Fatal(e)}'''
    marker = '\t\t\tctx := context.Background()'
    assert text.count(marker) == 1
    text = text.replace(marker, '''if directory:=os.Getenv("WOS_WORKSPACE_UPGRADE_FIXTURE");directory!=""&&backend=="sql"{
        defer func(){st:=store.(*Store);CAPTURE}()
    }
'''.replace('CAPTURE', capture)+marker)
    path.write_text(text)
    return hashlib.sha256(original).hexdigest()

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--old-binary', required=True)
    parser.add_argument('--helper', required=True)
    parser.add_argument('--source', required=True)
    parser.add_argument('--output', required=True)
    parser.add_argument('--resume', action='store_true', help='reuse this tool’s private source archive; retain prior attempts')
    parser.add_argument('--driver', choices=('sqlite', 'postgres', 'both'), default='both')
    args = parser.parse_args()
    os.umask(0o077)
    candidate_pinned=verify_candidate_source(args.source)
    candidate_migrations=source_migrations(ROOT)
    helper_identity=json.loads(subprocess.check_output([args.helper,'--mode','identity']))
    if candidate_pinned:assert helper_identity['source']==args.source,'helper build does not declare the qualification source'
    assert helper_identity['sqlite_schema']==candidate_migrations['sqlite'][-1]['version'] and helper_identity['postgres_schema']==candidate_migrations['postgres'][-1]['version'],'helper compiled migration chain differs from source'
    directory = Path(args.output).resolve()
    directory.mkdir(parents=True, exist_ok=args.resume)
    archive = directory/'baseline'
    if not args.resume:
        archive.mkdir()
        proc = subprocess.Popen(['git', 'archive', BASE], cwd=ROOT, stdout=subprocess.PIPE)
        with tarfile.open(fileobj=proc.stdout, mode='r|') as stream:
            stream.extractall(archive, filter='data')
        assert proc.wait() == 0
    for entry in subprocess.check_output(['git','ls-tree','-rz',BASE],cwd=ROOT).split(b'\0'):
        if not entry:continue
        metadata,name=entry.split(b'\t',1);name=name.decode()
        if name.endswith('.sql') or name.endswith('.go') and not name.endswith('_test.go') or name in ('go.mod','go.sum'):
            content=(archive/name).read_bytes()
            assert hashlib.sha1(b'blob '+str(len(content)).encode()+b'\0'+content).hexdigest()==metadata.split()[2].decode(),'historical production differs: '+name
    if args.resume:
        for backend in ('sqlite','postgres'):
            path=archive/'packages/wos-core/storage'/backend/'signed_review_journey_test.go'
            path.write_bytes(subprocess.check_output(['git','show',BASE+':'+str(path.relative_to(archive))],cwd=ROOT))
    before = production_hash(archive)
    assert BASE in subprocess.check_output([args.old_binary, 'version'], text=True)
    results = []
    for backend in (('sqlite', 'postgres') if args.driver == 'both' else (args.driver,)):
        target = directory/backend
        if target.exists() and args.resume:
            target.rename(directory/(backend+'-prior-'+secrets.token_hex(4)))
        target.mkdir()
        fixture_hash = instrument(archive, backend)
        assert production_hash(archive) == before, 'historical production files changed'
        env = os.environ.copy()
        env['WOS_WORKSPACE_UPGRADE_FIXTURE'] = str(target)
        env.pop('WOS_TEST_OPERATIONS_TOOL', None)
        with (target/'producer.log').open('wb') as log:
            subprocess.run(['go', 'test', '-count=1', '-run', '^TestSignedReviewCorrectionJourneyWithoutExecutorAuthority/sql$', './packages/wos-core/storage/'+backend], cwd=archive, env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
        path = str(target/'source.sqlite3')
        schema = 'main'
        if backend == 'postgres':
            metadata = json.loads((target/'source.json').read_text())
            schema = metadata['schema']
            parsed = urlsplit(env['WOS_TEST_POSTGRES_DSN'])
            database='wos_workspace_upgrade_'+secrets.token_hex(10)
            subprocess.run(['docker','exec',env['WOS_TEST_POSTGRES_CONTAINER'],'createdb','-U','postgres',database],check=True)
            with (target/'source.dump').open('rb') as dump:
                subprocess.run(['docker','exec','-i',env['WOS_TEST_POSTGRES_CONTAINER'],'pg_restore','-U','postgres','--dbname='+database,'--no-owner','--no-privileges','--exit-on-error'],stdin=dump,check=True)
            env['WOS_TEST_POSTGRES_DSN'] = urlunsplit((parsed.scheme, parsed.netloc, '/'+database, parsed.query, parsed.fragment))
        def rows(query):
            if backend == 'sqlite':
                with closing(ops.sqlite_read(path)) as db:
                    cursor = db.execute(query)
                    return [dict(zip([col[0] for col in cursor.description], row)) for row in cursor]
            previous = os.environ.get('WOS_WORKSPACE_UPGRADE_DSN')
            os.environ['WOS_WORKSPACE_UPGRADE_DSN'] = env['WOS_TEST_POSTGRES_DSN']
            try:
                with ops.PostgresSnapshot(ops.postgres_env('WOS_WORKSPACE_UPGRADE_DSN'), schema) as snapshot:
                    return [json.loads(line) for line in snapshot.run('SELECT row_to_json(t)::text FROM ('+query+') t')]
            finally:
                if previous is None:
                    os.environ.pop('WOS_WORKSPACE_UPGRADE_DSN', None)
                else:
                    os.environ['WOS_WORKSPACE_UPGRADE_DSN'] = previous
        if backend == 'sqlite':
            tables = [row['name'] for row in rows("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")]
        else:
            tables = [row['table_name'] for row in rows("SELECT table_name FROM information_schema.tables WHERE table_schema='"+schema+"' AND table_type='BASE TABLE' ORDER BY table_name")]
        originals = {table: sorted(ops.canonical(row).decode() for row in rows('SELECT * FROM '+ops.identifier(table))) for table in tables}
        assert rows('SELECT MAX(version) AS version FROM schema_migrations')[0]['version'] == 30
        assert len(originals['signed_protocol_facts']) > 0
        command = [args.helper, '--backend', backend, '--sqlite', path, '--schema', schema]
        migrated = json.loads(subprocess.check_output(command+['--mode', 'migrate'], env=env))
        for table, original in originals.items():
            actual = sorted(ops.canonical(row).decode() for row in rows('SELECT * FROM '+ops.identifier(table)))
            if table == 'schema_migrations':
                assert set(original) <= set(actual)
            else:
                assert original == actual, 'migration changed original table '+table
        applied=rows('SELECT version,name,checksum FROM schema_migrations ORDER BY version')
        expected_schema=verify_current_migrations(applied,candidate_migrations[backend])
        assert migrated['schema_version']==expected_schema
        env['WOS_HISTORICAL_BOOTSTRAP_TOKEN'] = 'signed-acquisition-bootstrap-token-at-least-32'
        note = json.loads(subprocess.check_output(command+['--mode', 'note'], env=env))
        assert note['schema_version']==expected_schema
        runtime = env.copy()
        runtime.update(WOS_STORAGE_DRIVER=backend, WOS_SQLITE_PATH=path, WOS_AUTH_MODE='api_token', WOS_LOCAL_PRINCIPAL_ID='admin', WOS_BOOTSTRAP_NAMESPACE_ID='0199a555-0000-7000-8000-000000000001', WOS_BOOTSTRAP_NAMESPACE_NAME='signed-acquisition', WOS_BOOTSTRAP_TOKEN=env['WOS_HISTORICAL_BOOTSTRAP_TOKEN'])
        if backend == 'postgres':
            parsed = urlsplit(env['WOS_TEST_POSTGRES_DSN'])
            options = dict(parse_qsl(parsed.query))
            options['options'] = (options.get('options', '')+' -c search_path='+schema).strip()
            runtime['WOS_POSTGRES_DSN'] = urlunsplit((parsed.scheme, parsed.netloc, parsed.path, urlencode(options), parsed.fragment))
        with (target/'old-refusal.log').open('wb') as log:
            old = subprocess.run([args.old_binary, 'db', 'migrate'], env=runtime, stdout=log, stderr=subprocess.STDOUT, timeout=30)
        assert old.returncode != 0 and 'schema' in (target/'old-refusal.log').read_text().lower()
        results.append({'backend': backend, 'old_schema': 30, 'new_schema': migrated['schema_version'], 'complete_candidate_migrations_verified': True, 'candidate_migration_manifest_sha256': hashlib.sha256(ops.canonical(candidate_migrations[backend])).hexdigest(), 'preexisting_tables_checked': len(originals), 'old_tables_unchanged_by_migration': len(originals)-1, 'old_migration_rows_preserved': True, 'old_migration_rows_checked': len(originals['schema_migrations']), 'original_signed_facts': len(originals['signed_protocol_facts']), 'old_fixture_test_sha256': fixture_hash, 'old_binary_refused_new_schema': True, 'notes': note})
    report = {'baseline_source': BASE, 'source': args.source, 'candidate_source_verified': candidate_pinned, 'helper_identity': helper_identity, 'historical_production_sha256': before, 'old_binary_sha256': ops.file_hash(Path(args.old_binary)), 'helper_sha256': ops.file_hash(Path(args.helper)), 'candidate_production_sha256': production_hash(ROOT/'packages/wos-core'), 'test_only_historical_instrumentation': True, 'results': results, 'complete': len(results) == (2 if args.driver == 'both' else 1)}
    (directory/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(report, indent=2))

if __name__ == '__main__':
    main()
