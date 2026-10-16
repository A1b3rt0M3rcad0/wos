#!/usr/bin/env python3
"""Opt-in WOS backup/restore. Standard library + PostgreSQL client tools.
No service, scheduler, remote control plane or private signing key management.
"""
import argparse
import base64
import contextlib
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import shutil
import sqlite3
import subprocess
import sys
import tempfile
import threading
import time
from urllib.parse import urlparse, unquote, parse_qsl

class OperationError(Exception):
    pass

def utc():
    return dt.datetime.now(dt.timezone.utc).isoformat()

def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(',', ':'), allow_nan=False).encode()

def file_hash(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()

def identifier(value):
    if not re.fullmatch(r'[a-z][a-z0-9_]{0,62}', value):
        raise OperationError('unsupported database identifier')
    return '"' + value + '"'

def sync_directory(path):
    if os.name == 'posix':
        descriptor = os.open(path, os.O_RDONLY | getattr(os, 'O_DIRECTORY', 0))
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)

def private_json(path, value):
    with path.open('x', encoding='utf-8') as stream:
        json.dump(value, stream, indent=2, ensure_ascii=False)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    os.chmod(path, 0o600)

def postgres_env(name):
    if not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_]*', name) or not os.environ.get(name):
        raise OperationError('the named PostgreSQL DSN environment variable is required')
    uri = urlparse(os.environ[name])
    if uri.scheme not in ('postgres', 'postgresql') or not uri.hostname:
        raise OperationError('operational scripts require a PostgreSQL URI in the named environment variable')
    database = unquote(uri.path.lstrip('/'))
    if not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_-]{0,62}', database):
        raise OperationError('operational scripts require a plain explicit database name')
    env = os.environ.copy()
    for key in ['PGSERVICE','PGSERVICEFILE','PGHOSTADDR','PGHOST','PGPORT','PGUSER','PGPASSWORD','PGDATABASE','PGOPTIONS','PGSSLMODE','PGSSLROOTCERT','PGSSLCERT','PGSSLKEY','PGPASSFILE']:
        env.pop(key, None)
    env.update(PGHOST=unquote(uri.hostname), PGPORT=str(uri.port or 5432), PGDATABASE=database,
               PGCONNECT_TIMEOUT='5')
    if uri.username:
        env['PGUSER'] = unquote(uri.username)
    if uri.password is not None:
        env['PGPASSWORD'] = unquote(uri.password)
    options = {'sslmode':'PGSSLMODE','sslrootcert':'PGSSLROOTCERT','sslcert':'PGSSLCERT',
               'sslkey':'PGSSLKEY','passfile':'PGPASSFILE','application_name':'PGAPPNAME',
               'target_session_attrs':'PGTARGETSESSIONATTRS','connect_timeout':'PGCONNECT_TIMEOUT'}
    extra_options = ''
    for key,value in parse_qsl(uri.query, keep_blank_values=True):
        if key == 'options':
            extra_options = value
        elif key in options:
            env[options[key]] = value
        else:
            raise OperationError('unsupported PostgreSQL URI option for operational client tools: '+key)
    env['PGCLIENTENCODING'] = 'UTF8'
    env['PGOPTIONS'] = extra_options + ' -c statement_timeout=30000 -c lock_timeout=5000'
    # Passwords remain environment-only, never argv or public result metadata.

    return env

def pg_command(tool, args, env, *, output=None, input_file=None):
    try:
        with tempfile.TemporaryFile() as errors:
            result = subprocess.run([tool] + args, env=env, stdout=output or subprocess.PIPE,
                                    stdin=input_file, stderr=errors, timeout=900, check=False)
            if result.returncode:
                raise OperationError(tool + ' failed; inspect client connectivity, permissions and the private backup locally')
            return result.stdout
    except (OSError, subprocess.TimeoutExpired) as e:
        raise OperationError(tool + ' unavailable or timed out') from e

class PostgresSnapshot:
    def __init__(self, env, schema):
        self.env, self.schema = env, schema
        identifier(schema)
        self.errors = tempfile.TemporaryFile()
        self.proc = subprocess.Popen(['psql', '-X', '-qAt', '-v', 'ON_ERROR_STOP=1'], env=env,
                                     stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                     stderr=self.errors, text=True, bufsize=1)
        self.lines = queue.Queue()
        def drain():
            try:
                for line in self.proc.stdout:
                    self.lines.put(line.rstrip('\n'))
            finally:
                self.lines.put(None)
        self.reader = threading.Thread(target=drain, daemon=True)
        self.reader.start()
        try:
            self.run('BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY')
            self.run('SET LOCAL search_path TO ' + identifier(schema) + ',pg_catalog')
        except Exception:
            self.close()
            raise
    def run(self, sql, consume=None):
        marker = '_wos_end_' + os.urandom(16).hex()
        try:
            self.proc.stdin.write(sql.rstrip(';') + ";\nSELECT '" + marker + "';\n")
            self.proc.stdin.flush()
        except (OSError, BrokenPipeError) as e:
            raise OperationError('psql session unavailable') from e
        result = []
        while True:
            try:
                line = self.lines.get(timeout=60)
            except queue.Empty as e:
                raise OperationError('bounded read-only database inspection timed out') from e
            if line is None:
                raise OperationError('read-only PostgreSQL inspection failed')
            if line == marker:
                return result
            if consume:
                consume(line)
            else:
                result.append(line)
    def close(self):
        with contextlib.suppress(Exception):
            self.proc.stdin.write('ROLLBACK;\n\\q\n')
            self.proc.stdin.close()
        try:
            self.proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.proc.kill()
            self.proc.wait(timeout=5)
        self.reader.join(timeout=1)
        self.proc.stdout.close()
        self.errors.close()
    def __enter__(self):
        return self
    def __exit__(self, *_):
        self.close()

def row_digest(rows):
    h, count = hashlib.sha256(), 0
    for row in rows:
        converted = {key: {'base64': base64.b64encode(value).decode()} if isinstance(value, bytes) else value
                     for key, value in row.items()}
        h.update(canonical(converted) + b'\n')
        count += 1
    return {'rows': count, 'sha256': h.hexdigest()}

def summary(tables, schema_version, issuer, watermark):
    if schema_version < 1:
        raise OperationError('source is not a migrated WOS database')
    return {'schema_version': schema_version,
            'server_id': issuer.get('server_id'), 'issuer_fingerprint': issuer.get('fingerprint'),
            'outcome_watermark': watermark, 'tables': tables}

def sqlite_inspect(db):
    if db.execute('PRAGMA quick_check').fetchone()[0] != 'ok':
        raise OperationError('SQLite quick_check failed')
    if db.execute('PRAGMA foreign_key_check').fetchone():
        raise OperationError('SQLite foreign_key_check failed')
    tables = {}
    names = [row[0] for row in db.execute("SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")]
    if 'schema_migrations' not in names:
        raise OperationError('source is not a WOS database')
    for table in names:
        ident = identifier(table)
        columns = [row[1] for row in db.execute('PRAGMA table_info(' + ident + ')')]
        order = ','.join(identifier(c) for c in columns)
        cursor = db.execute('SELECT * FROM ' + ident + ' ORDER BY ' + order)
        tables[table] = row_digest(dict(zip(columns, row)) for row in cursor)
    version = db.execute('SELECT MAX(version) FROM schema_migrations').fetchone()[0]
    issuer = {}
    if 'server_protocol_identity' in names:
        row = db.execute('SELECT state_json FROM server_protocol_identity WHERE singleton=1').fetchone()
        if row:
            issuer = json.loads(row[0])
    revisions = [int(row[0]) for row in db.execute('SELECT state_revision FROM outcome_coordination')]
    row = (len(revisions), max(revisions, default=0), sum(revisions))
    return summary(tables, int(version or 0), issuer,
                   dict(zip(['outcomes', 'max_revision', 'revision_sum'], row)))

def pg_inspect(session):
    schema = session.schema
    names = session.run("SELECT table_name FROM information_schema.tables WHERE table_schema='" + schema + "' AND table_type='BASE TABLE' ORDER BY table_name")
    if 'schema_migrations' not in names:
        raise OperationError('source is not a migrated WOS database')
    tables = {}
    for table in names:
        ident = identifier(table)
        h, count = hashlib.sha256(), 0
        def consume(line):
            nonlocal count
            h.update(canonical(json.loads(line)) + b'\n')
            count += 1
        session.run('SELECT row_to_json(t)::text FROM ' + identifier(schema) + '.' + ident + ' t ORDER BY row_to_json(t)::text COLLATE "C"', consume)
        tables[table] = {'rows': count, 'sha256': h.hexdigest()}
    version = int(session.run('SELECT COALESCE(MAX(version),0) FROM schema_migrations')[0])
    issuer = {}
    if 'server_protocol_identity' in names:
        values = session.run('SELECT state_json FROM server_protocol_identity WHERE singleton=1')
        if values:
            issuer = json.loads(values[0])
    watermark = json.loads(session.run("SELECT json_build_object('outcomes',COUNT(*),'max_revision',COALESCE(MAX(state_revision),0),'revision_sum',COALESCE(SUM(state_revision),0))::text FROM outcome_coordination")[0])
    return summary(tables, version, issuer, watermark)

def sqlite_read(path):
    if not path:
        raise OperationError('--sqlite must name the explicit source or new restore file')
    path = Path(path).resolve(strict=True)
    db = sqlite3.connect(path.as_uri() + '?mode=ro', uri=True)
    db.execute('PRAGMA query_only=ON')
    db.execute('BEGIN')
    return db

def check_backup(directory, expected_manifest=None):
    directory = Path(directory).resolve(strict=True)
    path = directory / 'manifest.json'
    if path.is_symlink() or not path.is_file():
        raise OperationError('backup manifest must be a regular file')
    digest = file_hash(path)
    if expected_manifest and digest != expected_manifest:
        raise OperationError('approved manifest SHA256 mismatch')
    sums = json.loads((directory / 'checksums.json').read_text())
    if sums.get('manifest.json') != digest:
        raise OperationError('manifest checksum mismatch')
    manifest = json.loads(path.read_text())
    backend = manifest.get('backend')
    expected = 'database.sqlite3' if backend == 'sqlite' else 'database.dump' if backend == 'postgres' else None
    if manifest.get('manifest_schema') != 1 or manifest.get('data_file') != expected:
        raise OperationError('unsupported backup format')
    data = directory / expected
    if data.is_symlink() or not data.is_file() or sums.get(expected) != file_hash(data):
        raise OperationError('backup data checksum mismatch')
    if manifest.get('data_sha256') != sums[expected]:
        raise OperationError('backup data binding mismatch')
    identifier(manifest.get('database_schema', 'public'))
    return directory, manifest, digest

def backup(args):
    if not re.fullmatch(r'[0-9a-f]{40}', args.source_commit):
        raise OperationError('source_commit must be the reviewed 40-character Git SHA')
    output = Path(args.output)
    output.mkdir(mode=0o700, parents=False, exist_ok=False)
    began = utc()
    data = output / ('database.sqlite3' if args.backend == 'sqlite' else 'database.dump')
    if args.backend == 'sqlite':
        with contextlib.closing(sqlite_read(args.sqlite)) as source:
            observed = sqlite_inspect(source)
            with sqlite3.connect(data) as target:
                source.backup(target)
        snapshot_at = began
    else:
        env = postgres_env(args.dsn_env)
        with PostgresSnapshot(env, args.schema) as session:
            exported = session.run('SELECT pg_export_snapshot()')[0]
            snapshot_at = json.loads(session.run('SELECT to_json(transaction_timestamp())::text')[0])
            observed = pg_inspect(session)
            with data.open('xb') as stream:
                pg_command('pg_dump', ['--format=custom', '--no-owner', '--no-privileges', '--schema=' + args.schema, '--snapshot=' + exported], env, output=stream)
                stream.flush()
                os.fsync(stream.fileno())
    os.chmod(data, 0o600)
    with data.open('rb') as durable:
        os.fsync(durable.fileno())
    manifest = {'manifest_schema': 1, 'backend': args.backend, 'database_schema': 'main' if args.backend == 'sqlite' else args.schema,
                'source_commit': args.source_commit, 'source_commit_origin': 'operator-declared deployed source',
                'product_version': args.product_version, 'backup_started_at': began,
                'snapshot_at': snapshot_at, 'backup_completed_at': utc(),
                'data_file': data.name, 'data_sha256': file_hash(data), 'snapshot': observed,
                'issuer_seed_collected': False}
    private_json(output / 'manifest.json', manifest)
    sums = {data.name: file_hash(data), 'manifest.json': file_hash(output / 'manifest.json')}
    private_json(output / 'checksums.json', sums)
    sync_directory(output)
    sync_directory(output.parent)
    return {'operation': 'backup', 'completed': True, 'manifest_sha256': sums['manifest.json'],
            'schema_version': observed['schema_version'], 'server_id': observed['server_id'],
            'snapshot_at': snapshot_at, 'data_bytes': data.stat().st_size}

def inspect_target(args, manifest):
    if manifest['backend'] == 'sqlite':
        with contextlib.closing(sqlite_read(args.sqlite)) as db:
            return sqlite_inspect(db)
    with PostgresSnapshot(postgres_env(args.dsn_env), manifest['database_schema']) as session:
        return pg_inspect(session)

def check_approved_pin(args, manifest, required=False):
    snapshot = manifest['snapshot']
    expected_id = getattr(args, 'expected_server_id', None)
    expected_pin = getattr(args, 'issuer_fingerprint', None)
    if required and snapshot['server_id'] and (not expected_id or not expected_pin):
        raise OperationError('signed-state restore requires the independently approved --expected-server-id and --issuer-fingerprint')
    if expected_id or expected_pin:
        if expected_id != snapshot['server_id'] or expected_pin != snapshot['issuer_fingerprint']:
            raise OperationError('approved server identity/issuer pin differs from the backup')
        return True
    return False

def verify(args):
    _, manifest, digest = check_backup(args.backup, args.manifest_sha256)
    pin_checked = check_approved_pin(args, manifest)
    observed = inspect_target(args, manifest)
    if observed != manifest['snapshot']:
        raise OperationError('restored logical state differs: schema, identity, authority, revocations, proof or outbox mismatch')
    return {'operation': 'verify', 'verified': True, 'read_only': True,
            'manifest_sha256': digest, 'approved_pin_checked': pin_checked, 'schema_version': observed['schema_version'],
            'server_id': observed['server_id'], 'table_count': len(observed['tables']), 'verified_at': utc()}

def restore(args):
    if not args.writers_stopped:
        raise OperationError('restore requires --writers-stopped for the explicit cutover procedure')
    directory, manifest, digest = check_backup(args.backup, args.manifest_sha256)
    check_approved_pin(args, manifest, required=True)
    start = time.monotonic()
    if manifest['backend'] == 'sqlite':
        if not args.sqlite:
            raise OperationError('--sqlite must name a new restore file')
        destination = Path(args.sqlite)
        # O_EXCL refuses existing/live destinations, including symlinks.
        fd = os.open(destination, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as target, (directory / manifest['data_file']).open('rb') as source:
            shutil.copyfileobj(source, target)
            target.flush()
            os.fsync(target.fileno())
        sync_directory(destination.parent)
    else:
        env = postgres_env(args.dsn_env)
        with PostgresSnapshot(env, manifest['database_schema']) as target:
            objects = int(target.run("SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND c.relkind IN ('r','p','v','m','S','f')")[0])
            if objects:
                raise OperationError('PostgreSQL restore destination must be a new empty database; existing user objects found')
            schema_exists = target.run("SELECT COUNT(*) FROM pg_namespace WHERE nspname='" + manifest['database_schema'] + "'")[0] == '1'
        with (directory / manifest['data_file']).open('rb') as stream, tempfile.TemporaryDirectory(prefix='wos-restore-') as temporary:
            # A pristine PostgreSQL database already has public. Do not DROP it
            # or use --clean: omit only that existing schema's CREATE/comment
            # entries; every table/data/constraint entry remains mandatory.
            contents = pg_command('pg_restore', ['--list'], env, input_file=stream).decode('utf-8')
            schema = re.escape(manifest['database_schema'])
            if schema_exists:
                contents = '\n'.join(line for line in contents.splitlines()
                    if not re.match(r'^\d+; \d+ \d+ (?:SCHEMA - ' + schema + r' |COMMENT - SCHEMA ' + schema + r' )', line)) + '\n'
            listing = Path(temporary) / 'restore.list'
            listing.write_text(contents, encoding='utf-8')
            os.chmod(listing, 0o600)
            stream.seek(0)
            pg_command('pg_restore', ['--exit-on-error', '--single-transaction', '--no-owner', '--no-privileges', '--use-list=' + str(listing), '--dbname=' + env['PGDATABASE']], env, input_file=stream)
    result = verify(args)
    result.update(operation='restore_and_verify', new_destination_only=True, read_only=False, validation_read_only=True,
                  rto_restore_and_verify_seconds=time.monotonic()-start,
                  snapshot_at=manifest['snapshot_at'], production_sla=False)
    return result

def parser():
    p = argparse.ArgumentParser(description=__doc__)
    subs = p.add_subparsers(dest='command', required=True)
    b = subs.add_parser('backup')
    b.add_argument('--backend', choices=['sqlite', 'postgres'], required=True)
    b.add_argument('--sqlite')
    b.add_argument('--dsn-env', default='WOS_POSTGRES_DSN')
    b.add_argument('--schema', default='public')
    b.add_argument('--output', required=True)
    b.add_argument('--source-commit', required=True)
    b.add_argument('--product-version', required=True)
    for name in ['inspect', 'verify', 'restore']:
        q = subs.add_parser(name)
        q.add_argument('--backup', required=True)
        q.add_argument('--manifest-sha256')
        if name != 'inspect':
            q.add_argument('--sqlite')
            q.add_argument('--dsn-env', default='WOS_RESTORE_POSTGRES_DSN')
            q.add_argument('--expected-server-id')
            q.add_argument('--issuer-fingerprint')
        if name == 'restore':
            q.add_argument('--writers-stopped', action='store_true')
    return p

def main(argv=None):
    os.umask(0o077)
    args = parser().parse_args(argv)
    try:
        if args.command == 'backup':
            result = backup(args)
        elif args.command == 'restore':
            result = restore(args)
        elif args.command == 'verify':
            result = verify(args)
        else:
            _, manifest, digest = check_backup(args.backup, args.manifest_sha256)
            result = {'operation': 'inspect', 'read_only': True, 'checksums_verified': True,
                      'manifest_sha256': digest, 'backend': manifest['backend'],
                      'snapshot': manifest['snapshot'], 'snapshot_at': manifest['snapshot_at']}
        print(json.dumps(result, sort_keys=True))
        return 0
    except (OperationError, OSError, sqlite3.Error, ValueError, KeyError, TypeError) as e:
        print(json.dumps({'operation': args.command, 'completed': False, 'error': str(e)}), file=sys.stderr)
        return 1

if __name__ == '__main__':
    sys.exit(main())
