import * as fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { setTimeout as pause } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import { root, run, sha256, validVersion } from './build.mjs';

export async function verifyDistribution(output = path.join(root,'dist')) {
  const manifest=JSON.parse(await fs.readFile(path.join(output,'distribution.json'),'utf8'));
  validVersion(manifest.version);
  assert.equal(manifest.schema,1);
  for(const a of manifest.artifacts) {
    assert.equal(path.basename(a.filename),a.filename);
    assert.equal(sha256(await fs.readFile(path.join(output,a.filename))),a.sha256);
  }
  const consumer=await fs.mkdtemp(path.join(os.tmpdir(),'wos-npm-consumer-'));
  let server;
  try {
    const packs=manifest.artifacts.filter(a=>['wos','wos-skill'].includes(a.kind));
    assert.equal(packs.length,2);
    run('npm',['install','--offline','--ignore-scripts','--no-audit','--no-fund','--package-lock=false','--prefix',consumer,...packs.map(a=>path.join(output,a.filename))]);
    const cli=path.join(consumer,'node_modules/.bin/wos');
    const version=run(cli,['version']);
    assert.ok(version.includes(`wos ${manifest.version} (commit ${manifest.commit}, built ${manifest.built_at})`));
    run(cli,['config','validate']);
    assert.throws(()=>run(cli,['not-a-command']),/failed/);
    const skill=path.join(consumer,'node_modules/.bin/wos-skill');
    assert.equal(run(skill,['--version']),manifest.version);
    assert.equal(JSON.parse(run(skill,['install','--agent','all','--project',consumer])).length,16);
    run(skill,['doctor','--agent','all','--project',consumer]);
    run(skill,['uninstall','--agent','all','--project',consumer]);
    const log=[];
    // Reserve and release an ephemeral TCP port immediately before startup.
    const {createServer}=await import('node:net');
    const reservation=createServer(); await new Promise(resolve=>reservation.listen(0,'127.0.0.1',resolve));
    const port=reservation.address().port; await new Promise(resolve=>reservation.close(resolve));
    server=spawn(cli,['server'],{env:{...process.env,WOS_LISTEN:`127.0.0.1:${port}`,WOS_SQLITE_PATH:path.join(consumer,'smoke.db'),WOS_LOCAL_PRINCIPAL_ID:'npm-smoke',WOS_MCP_ENABLED:'true'},stdio:['ignore','pipe','pipe']});
    server.stdout.on('data',b=>log.push(b.toString()));server.stderr.on('data',b=>log.push(b.toString()));
    let ready=false; const base=`http://127.0.0.1:${port}`;
    for(let i=0;i<100;i++) {
      if(server.exitCode!==null) throw new Error(`Packaged server exited: ${log.join('')}`);
      try{const r=await fetch(base+'/readyz',{signal:AbortSignal.timeout(1000)});ready=r.ok;if(ready)break;}catch{}
      await pause(100);
    }
    assert.ok(ready,`Packaged service did not become ready: ${log.join('')}`);
    assert.equal((await fetch(base+'/livez')).status,200);
    assert.match(await (await fetch(base+'/app/')).text(),/id="view-tabs"/);
    const headers={'content-type':'application/json','accept':'application/json, text/event-stream','MCP-Protocol-Version':'2025-06-18'};
    const init=await fetch(base+'/mcp',{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'npm-distribution-smoke',version:'1'}}})});
    assert.equal(init.status,200); assert.equal((await init.json()).result.protocolVersion,'2025-06-18');
    const tools=await fetch(base+'/mcp',{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:2,method:'tools/list',params:{}})});
    assert.equal(tools.status,200);assert.ok((await tools.json()).result.tools.some(t=>t.name==='wos_claim_work_item'));
    const actor=await fetch(base+'/api/v1/namespaces/0199d000-0000-7000-8000-000000000001/outcomes');
    assert.equal(actor.status,200);
    await stop(server); assert.equal(server.exitCode,0,'SIGTERM reaches Go graceful shutdown');server=null;
    const binary=path.join(consumer,'node_modules/@a1b3rt0m3rcad0/wos/native/wos');
    await fs.appendFile(binary,'corruption');
    assert.throws(()=>run(cli,['version']),/integrity\/version check failed/);
    return {version:manifest.version,commit:manifest.commit,checks:['artifact hashes','offline npm install','version/exit propagation','four skill destinations','HTTP/MCP/UI readiness','SIGTERM graceful shutdown','corruption rejection']};
  } finally {if(server)await stop(server);await fs.rm(consumer,{recursive:true,force:true});}
}

async function stop(child) {
  if(child.exitCode!==null||child.signalCode!==null)return;
  const finished=new Promise((resolve,reject)=>{child.once('exit',resolve);child.once('error',reject);});
  child.kill('SIGTERM');
  const timer=setTimeout(()=>child.kill('SIGKILL'),15000);
  try{await finished;}finally{clearTimeout(timer);}
}

if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try{console.log(JSON.stringify(await verifyDistribution(),null,2));}catch(e){console.error(e);process.exitCode=1;}
}
