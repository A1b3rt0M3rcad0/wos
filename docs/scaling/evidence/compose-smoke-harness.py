#!/usr/bin/env python3
"""Disposable actual public-profile smoke, using a supplied candidate binary.

The temporary image replaces the binary/secret entrypoint in a previously
available runtime image; this does not qualify the production Dockerfile build.
"""
import base64
import hashlib
import http.client
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import ssl
import subprocess
import sys
import time

binary=Path(sys.argv[1]).resolve(strict=True)
root=Path(sys.argv[2]).resolve()
root.mkdir(mode=0o700,exist_ok=False)
os.umask(0o077)
repo=Path('/workspace/wos')
project='wos-qualification-compose'
source_hash=hashlib.sha256(binary.read_bytes()).hexdigest()
image='wos-qualification-runtime:'+source_hash[:12]
base='ghcr.io/a1b3rt0m3rcad0/wos:0.4.2'
report={'schema':1,'complete':False,'production_sla':False,'binary_sha256':source_hash,
        'temporary_runtime_base':base,'measurement_boundary':'actual Docker Compose processes, TLS ingress, pooled PostgreSQL, browser session and restart; no load/SLA/HA claim',
        'production_dockerfile_build_qualified':False,'checks':[],
        'workspace_go_sql_sha256_observed_at_pilot':{str(p.relative_to(repo)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted((repo/'packages').rglob('*')) if p.is_file() and p.suffix in ('.go','.sql') and not p.name.endswith('_test.go')}}
log=(root/'commands.log').open('wb')

def run(command,check=True,timeout=180):
    result=subprocess.run(command,cwd=repo,stdout=subprocess.PIPE,stderr=subprocess.PIPE,
                          env=env,timeout=timeout)
    # Commands never contain tokens or private key bytes. Keep output private.
    log.write(('> '+' '.join(command)+'\n').encode()+result.stdout+result.stderr);log.flush()
    if check and result.returncode:
        raise RuntimeError('private command failed: '+command[0]+' '+command[1])
    return result

def compose(*args,scaled=False,check=True):
    cmd=['docker','compose','-p',project,'--env-file',str(root/'deployment.env'),
         '-f',str(repo/'deploy/scaling/compose.yaml')]
    if scaled:cmd+=['-f',str(repo/'deploy/scaling/compose.replicas.yaml')]
    cmd+=['-f',str(root/'image-override.yaml'),*args]
    return run(cmd,check=check)

def ids(service):return compose('ps','-q',service,scaled=True).stdout.decode().split()

def internal(container,path):
    result=run(['docker','exec',container,'wget','-Y','off','-q','-S','-O','-',
                'http://127.0.0.1:8080'+path],check=False)
    return result.returncode,result.stdout.decode(),result.stderr.decode()

def ready(container):
    began=time.monotonic()
    while time.monotonic()-began<60:
        code,body,_=internal(container,'/readyz')
        if code==0:return json.loads(body)
        time.sleep(.25)
    raise RuntimeError('owned process not ready')

with socket.socket() as listener:
    listener.bind(('127.0.0.1',0));port=listener.getsockname()[1]
private=root/'secrets';private.mkdir(mode=0o700)
values={'database_password':secrets.token_hex(32),'bootstrap_token':secrets.token_urlsafe(48),
        'issuer_seed':base64.b64encode(secrets.token_bytes(32)).decode()}
for name,value in values.items():
    p=private/name;p.write_text(value+'\n');p.chmod(0o444)
(root/'deployment.env').write_text('\n'.join([
    'WOS_SHARED_SECRETS_DIR='+str(private),'WOS_BOOTSTRAP_NAMESPACE_ID=0199ff99-0000-7000-8000-000000000001',
    'WOS_BOOTSTRAP_NAMESPACE_NAME=Disposable qualification','WOS_SHARED_HTTPS_PORT='+str(port),
    'WOS_PUBLIC_HOST=wos.localhost'])+'\n')
ns='0199ff99-0000-7000-8000-000000000001'
(root/'image-override.yaml').write_text('services:\n  postgres:\n    image: postgres:18.6\n  wos:\n    image: '+image+'\n  worker:\n    image: '+image+'\n')
# Base-only use must not introduce the overlay's worker service.
base_override=root/'base-image-override.yaml'
base_override.write_text('services:\n  postgres:\n    image: postgres:18.6\n  wos:\n    image: '+image+'\n')
env={k:v for k,v in os.environ.items() if not k.startswith('WOS_')}
(root/'buildx').mkdir()
env['BUILDX_CONFIG']=str(root/'buildx')
build=root/'image';build.mkdir()
shutil.copyfile(binary,build/'wos')
assert hashlib.sha256((build/'wos').read_bytes()).hexdigest()==source_hash, 'candidate binary changed during copy'
shutil.copyfile(repo/'deploy/shared/entrypoint.sh',build/'entrypoint.sh')
(build/'Dockerfile').write_text('FROM '+base+'\nCOPY --chmod=0555 wos /usr/local/bin/wos\nCOPY --chmod=0555 entrypoint.sh /usr/local/lib/wos/shared-entrypoint.sh\n')

class LocalTLS(http.client.HTTPSConnection):
    def connect(self):
        self.sock=self._context.wrap_socket(socket.create_connection((self.host,self.port),self.timeout),server_hostname='wos.localhost')

cookie=''
context=None

def request(path,method='GET',body=None,key=None,correlation=None,bearer=False):
    c=LocalTLS('127.0.0.1',port,timeout=15,context=context)
    headers={'Host':'wos.localhost:'+str(port)}
    if bearer:headers['Authorization']='Bearer '+values['bootstrap_token']
    elif cookie:headers['Cookie']=cookie
    if key:headers['Idempotency-Key']=key
    if correlation:headers['X-Correlation-ID']=correlation
    if body is not None:headers['Content-Type']='application/json'
    c.request(method,path,json.dumps(body) if body is not None else None,headers)
    response=c.getresponse();raw=response.read();h=dict(response.getheaders());status=response.status;c.close()
    try:decoded=json.loads(raw)
    except ValueError:decoded=raw.decode()
    return status,decoded,h

started=False
try:
    run(['docker','build','-t',image,str(build)])
    report['image_id']=run(['docker','image','inspect',image,'--format','{{.Id}}']).stdout.decode().strip()
    report['binary_version']=run(['docker','run','--rm',image,'version']).stdout.decode().strip()
    report['profile_sha256']={str(p.relative_to(repo)):hashlib.sha256(p.read_bytes()).hexdigest() for p in [repo/'deploy/shared/entrypoint.sh',repo/'deploy/scaling/compose.yaml',repo/'deploy/scaling/compose.replicas.yaml',repo/'deploy/scaling/Caddyfile']}
    # Start the one-process profile first without declaring an extra worker.
    cmd=['docker','compose','-p',project,'--env-file',str(root/'deployment.env'),'-f',str(repo/'deploy/scaling/compose.yaml'),'-f',str(base_override),'up','-d','--no-build','--wait','--wait-timeout','120']
    assert not run(['docker','ps','-aq','--filter','label=com.docker.compose.project='+project]).stdout.strip(), 'disposable project already exists'
    started=True
    run(cmd)
    first=run(['docker','compose','-p',project,'--env-file',str(root/'deployment.env'),'-f',str(repo/'deploy/scaling/compose.yaml'),'-f',str(base_override),'ps','-q','wos']).stdout.decode().strip()
    assert ready(first)['process_role']=='complete'
    report['checks'].append('single complete process starts against PostgreSQL')
    compose('cp','ingress:/data/caddy/pki/authorities/local/root.crt',str(root/'public-root.crt'))
    context=ssl.create_default_context(cafile=str(root/'public-root.crt'))
    status,session,headers=request('/session','POST',{'token':values['bootstrap_token']})
    assert status==200,(status,session)
    cookie=headers['Set-Cookie'].split(';',1)[0]
    path='/api/v1/namespaces/'+ns+'/outcomes'
    payload={'title':'Profile continuity qualification','desired_state':'Same persisted state across API roles','priority':'normal'}
    status,created,_=request(path,'POST',payload,'qualification-profile-original-create')
    assert status==201,(status,created)
    outcome=created['value']['id']
    status,trust,_=request('/api/v1/namespaces/'+ns+'/signed-trust')
    assert status==200,(status,trust)
    approved=trust['server']
    report['checks'].append('TLS browser session, persistent Outcome and signed issuer are established')
    compose('up','-d','--no-build','--wait','--wait-timeout','120',scaled=True)
    apis,workers=ids('wos'),ids('worker')
    assert len(apis)==2 and len(workers)==2
    for c in apis:assert ready(c)['process_role']=='api'
    for c in workers:
        health=ready(c);assert health['process_role']=='worker' and health['delivery_worker']['worker_running']
        for path_ in ['/app/','/api/v1/identity','/session','/mcp']:
            code,_,stderr=internal(c,path_);assert code!=0 and '404' in stderr,(path_,stderr)
        mounts=json.loads(run(['docker','inspect',c,'--format','{{json .Mounts}}']).stdout)
        targets={m['Destination'] for m in mounts}
        assert '/run/secrets/database_password' in targets
        assert not {'/run/secrets/bootstrap_token','/run/secrets/issuer_seed'} & targets
    report['checks'].append('two API and two worker processes are healthy; workers expose no product routes or bootstrap/issuer secret mounts')
    marker='qualification-routing-'+secrets.token_hex(6)
    for i in range(32):
        status,value,_=request('/api/v1/identity',correlation=marker+'-'+str(i))
        assert status==200,(status,value)
    counts=[]
    for c in apis:
        output=run(['docker','logs',c]).stdout.decode()+run(['docker','logs',c]).stderr.decode()
        counts.append(output.count(marker))
    assert all(counts),counts
    report['ingress_requests_by_api']=counts
    report['checks'].append('dynamic DNS ingress routes authenticated cookie requests to both API processes')
    run(['docker','restart',apis[0]])
    assert ready(apis[0])['process_role']=='api'
    for i in range(8):assert request('/api/v1/identity')[0]==200
    status,replayed,_=request(path,'POST',payload,'qualification-profile-original-create')
    assert status==201 and replayed['value']['id']==outcome and replayed['idempotent_replay'],(status,replayed)
    status,after,_=request('/api/v1/namespaces/'+ns+'/signed-trust')
    assert status==200 and after['server']==approved
    run(['docker','restart',workers[0]])
    assert ready(workers[0])['delivery_worker']['worker_running']
    report['checks'].append('API and worker restarts preserve the browser session, issuer and exact original idempotent result')
    report['issuer_public']=approved
    report['complete']=True
finally:
    if started:
        compose('logs','--no-color',scaled=True,check=False)
        compose('down','-v','--remove-orphans',scaled=True,check=False)
    (root/'report.json').write_text(json.dumps(report,indent=2)+'\n')
    log.close()
print(json.dumps({'complete':report['complete'],'report':str(root/'report.json')}))
