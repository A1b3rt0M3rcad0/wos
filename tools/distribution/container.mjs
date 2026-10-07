import { randomBytes } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as pause } from 'node:timers/promises';
import assert from 'node:assert/strict';
import { root, run } from './build.mjs';

export async function verifyContainer(image='wos:release-tested') {
  const manifest=JSON.parse(await readFile(path.join(root,'dist/distribution.json'),'utf8'));
  const info=JSON.parse(run('docker',['inspect',image]))[0];
  assert.equal(info.Config.User,'10001:10001');
  assert.equal(info.Config.Labels['org.opencontainers.image.version'],manifest.version);
  assert.equal(info.Config.Labels['org.opencontainers.image.revision'],manifest.commit);
  assert.ok(run('docker',['run','--rm',image,'version']).includes(`wos ${manifest.version} (commit ${manifest.commit}, built ${manifest.built_at})`));
  run('docker',['run','--rm',image,'config','validate']);
  const secret=randomBytes(32).toString('base64url');
  const namespace='0199d330-0000-7000-8000-000000000001';
  const container=run('docker',['run','-d','--read-only','--tmpfs','/data:uid=10001,gid=10001,mode=0700','-p','127.0.0.1::8080','--env','WOS_LISTEN=0.0.0.0:8080','--env','WOS_AUTH_MODE=api_token','--env','WOS_BOOTSTRAP_TOKEN','--env',`WOS_BOOTSTRAP_NAMESPACE_ID=${namespace}`,'--env','WOS_BOOTSTRAP_NAMESPACE_NAME=Release-smoke','--env','WOS_LOCAL_PRINCIPAL_ID=release-smoke','--env','WOS_MCP_ENABLED=true',image],{env:{...process.env,WOS_BOOTSTRAP_TOKEN:secret}});
  try {
    const runtime=JSON.parse(run('docker',['inspect',container]))[0];
    assert.equal(runtime.HostConfig.ReadonlyRootfs,true);
    const address=runtime.NetworkSettings.Ports['8080/tcp'][0];assert.equal(address.HostIp,'127.0.0.1');
    const base=`http://127.0.0.1:${address.HostPort}`;let ready=false;
    for(let i=0;i<100;i++) {
      try{ready=(await fetch(base+'/readyz',{signal:AbortSignal.timeout(1000)})).ok;if(ready)break;}catch{}
      await pause(100);
    }
    assert.ok(ready,'Versioned read-only container readiness');
    assert.equal((await fetch(base+'/livez')).status,200);
    assert.equal((await fetch(base+'/api/v1/commands')).status,401,'Remote catalog requires authentication');
    const headers={authorization:`Bearer ${secret}`,'content-type':'application/json',accept:'application/json, text/event-stream','MCP-Protocol-Version':'2025-06-18'};
    const catalog=await fetch(base+'/api/v1/commands',{headers});assert.equal(catalog.status,200);
    assert.ok((await catalog.json()).commands.length>=78);
    const mcp=await fetch(base+'/mcp',{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'versioned-container-smoke',version:'1'}}})});
    assert.equal(mcp.status,200);assert.equal((await mcp.json()).result.protocolVersion,'2025-06-18');
    run('docker',['exec',container,'test','-s','/etc/ssl/certs/ca-certificates.crt']);
    run('docker',['exec',container,'test','-s','/usr/share/doc/wos/third-party-notices/Go-LICENSE.txt']);
    return {version:manifest.version,commit:manifest.commit,checks:['OCI/source identity','non-root/read-only','version/config','health','HTTP authentication/catalog','MCP negotiation','runtime CA bundle']};
  }finally{run('docker',['rm','-f',container]);}
}

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try{console.log(JSON.stringify(await verifyContainer(),null,2));}catch(e){console.error(e.message);process.exitCode=1;}
}
