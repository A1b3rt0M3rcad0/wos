#!/usr/bin/env python3
"""Opt-in disposable PostgreSQL two-API crash/restore and receiver dedupe pilot."""
import argparse
import base64
from concurrent.futures import ThreadPoolExecutor
import hashlib
import hmac
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
import os
from pathlib import Path
import secrets
import socket
import sqlite3
import subprocess
import threading
import time
from urllib.parse import urlparse

spec=importlib.util.spec_from_file_location('wos_ops',Path(__file__).parents[1]/'operations'/'backup_restore.py')
ops=importlib.util.module_from_spec(spec);spec.loader.exec_module(ops)

def free_port():
    with socket.socket() as listener:
        listener.bind(('127.0.0.1',0))
        return listener.getsockname()[1]

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary',required=True)
    parser.add_argument('--source-commit',required=True)
    parser.add_argument('--dsn-env',required=True)
    parser.add_argument('--restore-dsn-env')
    parser.add_argument('--output',required=True)
    args=parser.parse_args()
    binary=Path(args.binary).resolve(strict=True)
    build=subprocess.check_output([str(binary),'version'],text=True).strip()
    if '(commit '+args.source_commit+',' not in build:
        parser.error('binary build metadata must identify the exact supplied source commit')
    root=Path(args.output).resolve();root.mkdir(mode=0o700,exist_ok=False)
    os.umask(0o077)
    dsns=[os.environ[args.dsn_env]]
    if args.restore_dsn_env:dsns.append(os.environ[args.restore_dsn_env])
    for name,dsn in zip([args.dsn_env,args.restore_dsn_env],dsns):
        if not urlparse(dsn).path.lstrip('/').startswith('wos_qualification_'):
            parser.error('pilot refuses a database without the disposable wos_qualification_ prefix')
        with ops.PostgresSnapshot(ops.postgres_env(name),'public') as session:
            count=int(session.run("SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND c.relkind IN ('r','p','v','m','S','f')")[0])
            if count:parser.error('pilot refuses an existing nonempty database')
    ns='0199f016-0000-7000-8000-000000000101'
    endpoint='0199f016-0000-7000-8000-000000000102'
    token=secrets.token_urlsafe(48);seed=base64.b64encode(secrets.token_bytes(32)).decode();secret=secrets.token_urlsafe(48)
    gate=threading.Event();eight=threading.Event();lock=threading.Lock()
    ledger=root/'consumer.sqlite3'
    with sqlite3.connect(ledger) as db:
        db.executescript('CREATE TABLE effects(event_id TEXT PRIMARY KEY,body_sha256 TEXT NOT NULL); CREATE TABLE requests(id INTEGER PRIMARY KEY,event_id TEXT NOT NULL);')
    state={'inflight':0,'max_inflight':0,'errors':[],'first_receive_at':None,'held_at_eight':None}
    class Receiver(BaseHTTPRequestHandler):
        def log_message(self,*_):pass
        def do_POST(self):
            length=int(self.headers.get('Content-Length','0'));body=self.rfile.read(min(length,256*1024+1))
            event=self.headers.get('X-WOS-Event-ID','');stamp=self.headers.get('X-WOS-Timestamp','')
            try:
                expected='sha256='+hmac.new(secret.encode(),(stamp+'.'+event+'.').encode()+body,hashlib.sha256).hexdigest()
                assert length<=256*1024 and abs(time.time()-int(stamp))<=300 and hmac.compare_digest(expected,self.headers.get('X-WOS-Signature',''))
                message=json.loads(body);assert message['integration_event_id']==event and message['source']['namespace_id']==ns
                digest=hashlib.sha256(body).hexdigest()
                with sqlite3.connect(ledger,timeout=10) as db:
                    db.execute('INSERT OR IGNORE INTO effects VALUES(?,?)',(event,digest))
                    assert db.execute('SELECT body_sha256 FROM effects WHERE event_id=?',(event,)).fetchone()[0]==digest
                    db.execute('INSERT INTO requests(event_id) VALUES(?)',(event,))
                    count=db.execute('SELECT COUNT(*) FROM requests').fetchone()[0]
                with lock:
                    state['inflight']+=1;state['max_inflight']=max(state['max_inflight'],state['inflight'])
                    if state['first_receive_at'] is None:state['first_receive_at']=time.monotonic()
                    if count>=8 and not eight.is_set():state['held_at_eight']=time.monotonic();eight.set()
                if not gate.wait(20):raise RuntimeError('pilot receiver hold exceeded its own bound')
                self.send_response(204);self.end_headers()
            except (BrokenPipeError,ConnectionResetError):
                pass # The killed API cannot receive its acknowledgement.
            except Exception as error:
                with lock:state['errors'].append(type(error).__name__+': '+str(error))
                self.send_response(400);self.end_headers()
            finally:
                with lock:state['inflight']=max(0,state['inflight']-1)
    receiver=ThreadingHTTPServer(('127.0.0.1',0),Receiver)
    receiver.daemon_threads=True
    receiver_thread=threading.Thread(target=receiver.serve_forever,daemon=True);receiver_thread.start()
    processes=[];logs=[];ports=[free_port(),free_port()]
    base_env=os.environ.copy()
    for key in list(base_env):
        if key.startswith('WOS_'):base_env.pop(key)
    base_env.update(WOS_AUTH_MODE='api_token',WOS_LOCAL_PRINCIPAL_ID='qualification-operator',
        WOS_BOOTSTRAP_TOKEN=token,WOS_BOOTSTRAP_NAMESPACE_ID=ns,WOS_BOOTSTRAP_NAMESPACE_NAME='Qualification',
        WOS_STORAGE_DRIVER='postgres',WOS_MCP_ENABLED='true',WOS_SERVER_SIGNING_SEED_ENV='WOS_QUALIFICATION_SEED',
        WOS_QUALIFICATION_SEED=seed,WOS_SIGNED_ACCEPTANCE_FLOOR='independent_review',
        WOS_DELIVERY_WORKER_ENABLED='true',WOS_DELIVERY_WORKER_CONCURRENCY='4',WOS_WEBHOOK_ALLOW_LOOPBACK='true',
        WOS_WEBHOOK_ENDPOINTS=json.dumps([{'id':endpoint,'namespace_id':ns,'url':f'http://127.0.0.1:{receiver.server_port}/signal','secret_ref':'qualification-hmac','key_id':'qualification-v1'}]),
        WOS_WEBHOOK_SECRETS=json.dumps({'qualification-hmac':secret}))
    def request(port,path,method='GET',payload=None,key=None,authorized=True):
        connection=http.client.HTTPConnection('127.0.0.1',port,timeout=20)
        headers={'Authorization':'Bearer '+token} if authorized else {}
        if key:headers['Idempotency-Key']=key
        if payload is not None:headers['Content-Type']='application/json'
        connection.request(method,path,body=json.dumps(payload) if payload is not None else None,headers=headers)
        response=connection.getresponse();body=response.read();status=response.status;connection.close()
        return status,json.loads(body) if body else {}
    def wait_ready(port):
        for _ in range(120):
            try:
                if request(port,'/readyz')[0]==200:return
            except (OSError,http.client.HTTPException):pass
            time.sleep(.25)
        raise RuntimeError('owned API did not become ready; private process logs retained')
    def start(index,dsn):
        log=(root/('api-'+str(index)+'.log')).open('wb');logs.append(log)
        env=base_env.copy();env.update(WOS_LISTEN=f'127.0.0.1:{ports[index%2]}',WOS_POSTGRES_DSN=dsn)
        process=subprocess.Popen([str(binary),'server'],env=env,stdout=log,stderr=subprocess.STDOUT)
        processes.append(process);wait_ready(ports[index%2]);return process
    def trust(port):
        status,result=request(port,f'/api/v1/namespaces/{ns}/signed-trust');assert status==200
        return result['server']
    report={'source_commit':args.source_commit,'binary_build':build,'binary_sha256':ops.file_hash(binary),'production_sla':False,
            'scenario':'restore' if args.restore_dsn_env else 'two-server-failover','complete':False}
    try:
        first=start(0,dsns[0]);second=start(1,dsns[0]);approved=trust(ports[0]);assert approved==trust(ports[1])
        report.update(server_id=approved['server_id'],issuer_fingerprint=approved['fingerprint'])
        path=f'/api/v1/namespaces/{ns}/outcomes'
        status,result=request(ports[0],path,'POST',{'title':'Two server delivery pilot','desired_state':'authoritative queue and external dedupe','priority':'normal'},'qualification-outcome-create');assert status==201,(status,result)
        oid=result['value']['id'];scope={'namespace_id':ns,'outcome_id':oid}
        status,result=request(ports[0],'/api/v1/commands/configure_trigger','POST',{'command':{'scope':scope,'name':'Qualification signal','event_types':['work_item.created'],'target_endpoint_ids':[endpoint],'signal_type':'qualification.created'}},'qualification-trigger-configure');assert status==200,(status,result)
        health=f'/api/v1/namespaces/{ns}/operations/delivery-health?outcome_id={oid}'
        for port in ports:
            status,value=request(port,health);assert status==200 and value['worker']['worker_running'] and value['worker']['concurrency']==4
        messages=8 if args.restore_dsn_env else 100
        failure=[]
        def crash_after_both_pools_claimed():
            if not eight.wait(15):failure.append('both API pools did not reach eight held requests');gate.set();return
            # Before ten-second client timeout, eight held requests prove the
            # two independent four-lane processes both dispatched real HTTP.
            if state['held_at_eight']-state['first_receive_at']>=8:failure.append('two-process dispatch was not established before HTTP timeout')
            first.kill();first.wait(timeout=5)
            if not args.restore_dsn_env:gate.set()
        crasher=threading.Thread(target=crash_after_both_pools_claimed);crasher.start()
        def create(i):
            payload={'title':f'Qualification signal {i}','priority':'normal'};key=f'qualification-create-work-{i}'
            for attempt in range(150):
                port=ports[i%2] if first.poll() is None else ports[1]
                try:
                    status,value=request(port,path+'/'+oid+'/work-items','POST',payload,key)
                    if status==201:return value['value']['id']
                    code=value.get('error',{}).get('code')
                    if status!=409 or code!='transaction_conflict':raise RuntimeError('unexpected creation result '+str(status)+' '+str(code))
                except (OSError,http.client.HTTPException):
                    # Reconcile the identical original intention through the
                    # surviving API; no new payload/key or expected version.
                    pass
                time.sleep(min(.02*(attempt+1),.2))
            raise RuntimeError('exact creation retry budget exhausted')
        started=time.monotonic()
        with ThreadPoolExecutor(max_workers=8) as pool:
            ids=list(pool.map(create,range(messages)))
        assert len(set(ids))==messages
        crasher.join(timeout=20);assert not crasher.is_alive() and not failure,failure
        if args.restore_dsn_env:
            second.kill();second.wait(timeout=5)
            # External effects were committed before this backup, but their
            # HTTP acknowledgements were held. WOS is now stopped for cutover.
            options=argparse.Namespace(backend='postgres',dsn_env=args.dsn_env,schema='public',output=str(root/'backup'),source_commit=args.source_commit,product_version='dev')
            created=ops.backup(options)
            options=argparse.Namespace(backup=str(root/'backup'),dsn_env=args.restore_dsn_env,sqlite=None,
                manifest_sha256=created['manifest_sha256'],writers_stopped=True,
                expected_server_id=approved['server_id'],issuer_fingerprint=approved['fingerprint'])
            restored=ops.restore(options)
            report['backup_restore']=restored
            gate.set()
            second=start(2,dsns[1]);third=start(3,dsns[1])
            assert approved==trust(ports[0])==trust(ports[1])
        deadline=time.monotonic()+100
        latest=None
        while time.monotonic()<deadline:
            port=ports[1]
            status,latest=request(port,health)
            if status==200 and latest['queue']['succeeded']==messages:break
            time.sleep(.25)
        assert latest and latest['queue']['succeeded']==messages and latest['queue']['pending']==0 and latest['queue']['leased']==0,latest
        assert latest['queue']['exhausted']==0 and latest['queue']['attempts_total']>messages
        with sqlite3.connect(ledger) as db:
            effects=db.execute('SELECT COUNT(*) FROM effects').fetchone()[0]
            calls=db.execute('SELECT COUNT(*) FROM requests').fetchone()[0]
        assert effects==messages and calls>effects and not state['errors'],state
        # Inspect the actual database, not the receiver, for authoritative ACKs.
        database_env=args.restore_dsn_env or args.dsn_env
        with ops.PostgresSnapshot(ops.postgres_env(database_env),'public') as session:
            duplicates=int(session.run("SELECT COUNT(*) FROM (SELECT delivery_id FROM delivery_attempts WHERE result='http_204' GROUP BY delivery_id HAVING COUNT(*)>1) t")[0])
            successes=int(session.run("SELECT COUNT(*) FROM delivery_attempts WHERE result='http_204'")[0])
            restored_work_ids=json.loads(session.run("SELECT COALESCE(json_agg(id ORDER BY id),'[]'::json)::text FROM work_items")[0])
        assert duplicates==0 and successes==messages,(duplicates,successes)
        assert sorted(ids)==restored_work_ids,'committed work identities changed or were lost'
        report.update(complete=True,messages=messages,two_real_api_processes=True,lanes_per_api=4,
            held_requests_before_kill=8,external_effects_before_ack=True,original_api_sigkill=True,
            unchanged_creation_retries=True,durable_succeeded=messages,authoritative_successes=successes,
            duplicate_authoritative_successes=duplicates,receiver_requests=calls,consumer_unique_effects=effects,
            external_duplicates=calls-effects,receiver_dedupe_key='integration_event_id',
            attempts_total=latest['queue']['attempts_total'],elapsed_seconds=time.monotonic()-started,
            queue_health=latest,identity_preserved=True)
        if args.restore_dsn_env:
            report.update(source_writers_stopped_before_snapshot=True,
                rpo_fixture_committed_work_lost=0,fixture_snapshot_window='backup after all eight accepted work commands and external receiver effects; writers stopped',
                rto_restore_and_verify_seconds=restored['rto_restore_and_verify_seconds'])
    finally:
        gate.set()
        for process in processes:
            if process.poll() is None:
                process.terminate()
                try:process.wait(timeout=20)
                except subprocess.TimeoutExpired:process.kill();process.wait(timeout=5)
        receiver.shutdown();receiver.server_close();receiver_thread.join(timeout=2)
        for log in logs:log.close()
        (root/'report.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps({'complete':report['complete'],'scenario':report['scenario'],'report':str(root/'report.json')}))
    return 0

if __name__=='__main__':raise SystemExit(main())
