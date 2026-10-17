import * as fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {randomBytes} from 'node:crypto';
import { spawn } from 'node:child_process';
import { setTimeout as pause } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import { root, run, sha256, validVersion } from './build.mjs';
import {verifySignedPackage} from './signed-smoke.mjs';
import { legalEntries, npmLicense, verifyArchiveLegalFiles, verifyLegalFiles } from './legal.mjs';

export async function verifyDistribution(output = path.join(root,'dist')) {
  const manifest=JSON.parse(await fs.readFile(path.join(output,'distribution.json'),'utf8'));
  validVersion(manifest.version);
  assert.equal(manifest.schema,1);
  for(const a of manifest.artifacts) {
    assert.equal(path.basename(a.filename),a.filename);
    assert.equal(sha256(await fs.readFile(path.join(output,a.filename))),a.sha256);
    if(a.kind==='archive'||a.kind==='client-linux'||a.kind==='client-windows') await verifyArchiveLegalFiles(root,path.join(output,a.filename));
  }
  for(const platform of ['linux','windows']){
    const a=manifest.artifacts.find(x=>x.kind===`client-${platform}`);assert.ok(a);
    const stage=path.join(output,`client-${platform}`),filename=platform==='windows'?'wosctl.exe':'wosctl';
    await verifyLegalFiles(root, stage);
    const metadata=JSON.parse(await fs.readFile(path.join(stage,'release.json'),'utf8'));
    assert.equal(metadata.version,manifest.version);assert.equal(metadata.commit,manifest.commit);assert.equal(metadata.built_at,manifest.built_at);assert.equal(metadata.sha256,a.binary_sha256);assert.equal(sha256(await fs.readFile(path.join(stage,filename))),a.binary_sha256);
    const primary=path.join(stage,platform==='windows'?'wos.exe':'wos');
    assert.equal(sha256(await fs.readFile(primary)),manifest.cli_binaries[platform+'-amd64'].wos.sha256);
    assert.equal(metadata.wos_sha256,manifest.cli_binaries[platform+'-amd64'].wos.sha256);
    if(platform==='linux')assert.equal(JSON.parse(run(primary,['version','--output','json'])).data.commit,manifest.commit);
    const bytes=await fs.readFile(path.join(stage,filename));assert.equal(platform==='windows'?bytes.subarray(0,2).toString():bytes.subarray(1,4).toString(),platform==='windows'?'MZ':'ELF');
    if(platform==='linux'){assert.equal(JSON.parse(run(path.join(stage,filename),['version','--output','json'])).data.commit,manifest.commit)}
  }
  await verifyLegalFiles(root, path.join(output, 'archive'));
  await verifyLegalFiles(root, path.join(output, 'npm', 'wos-cli'));
  const consumer=await fs.mkdtemp(path.join(os.tmpdir(),'wos-npm-consumer-'));
  let server;
  try {
    const packs=manifest.artifacts.filter(a=>a.kind==='wos-cli');
    assert.equal(packs.length,1);
    assert.equal(packs[0].name,'wosctl');
    run('npm',['install','--offline','--ignore-scripts','--no-audit','--no-fund','--package-lock=false','--prefix',consumer,...packs.map(a=>path.join(output,a.filename))]);
    const cli=path.join(consumer,'node_modules/.bin/wos');
    const installedRoot=path.join(consumer,'node_modules/wosctl');
    const installedPackage=JSON.parse(await fs.readFile(path.join(installedRoot,'package.json'),'utf8'));
    assert.equal(installedPackage.license,npmLicense);
    assert.ok(legalEntries.every(filename=>installedPackage.files.includes(filename)));
    await verifyLegalFiles(root, installedRoot);
    const notices=path.join(consumer,'node_modules/wosctl/third-party-notices');
    assert.ok((await fs.readFile(path.join(notices,'Go-LICENSE.txt'),'utf8')).includes('Copyright'));
    assert.match(await fs.readFile(path.join(notices,'index.md'),'utf8'),/github.com\/modelcontextprotocol\/go-sdk@v1.8.0/);
    const version=run(cli,['version']);
    assert.ok(version.includes(`wos ${manifest.version} (commit ${manifest.commit}, built ${manifest.built_at})`));
    run(cli,['config','validate']);
    const client=cli;
    const clientVersion=JSON.parse(run(client,['version','--output','json'])).data;
    assert.equal(clientVersion.version,manifest.version);assert.equal(clientVersion.commit,manifest.commit);assert.equal(clientVersion.built_at,manifest.built_at);
    const installedRelease=JSON.parse(await fs.readFile(path.join(consumer,'node_modules/wosctl/release.json'),'utf8'));
    assert.equal(installedRelease.work_protocol,clientVersion.work_protocol);
    assert.equal(installedRelease.workspace_schema,clientVersion.workspace_schema);
    assert.ok(clientVersion.supported_work_protocols.includes('signed_contracts_v3'));
    const draftSchema=JSON.parse(await fs.readFile(path.join(consumer,'node_modules/wosctl/schemas/contract-v2.schema.json'),'utf8'));
    assert.equal(draftSchema.properties.schema_version.const,2);
    assert.match(await fs.readFile(path.join(notices,'index.md'),'utf8'),/go.yaml.in\/yaml\/v3@v3.0.5/);
    assert.throws(()=>run(cli,['not-a-command']),/failed/);
    // Use the native Go binary directly. These calls must not require Node/npm.
    const nativeBinary=path.join(consumer,'node_modules/wosctl/native/linux-amd64/wos');
    assert.equal(run(nativeBinary,['skill','version']),manifest.version);
    const nativeRoot=path.join(consumer,'native-skill-fixture');
    const preview=JSON.parse(run(nativeBinary,['skill','install','--agent','all','--project',nativeRoot,'--dry-run']));
    assert.equal(preview.length,20); assert.ok(preview.every(x=>x.dryRun===true));
    assert.equal(JSON.parse(run(nativeBinary,['skill','install','--agent','all','--project',nativeRoot])).length,20);
    for(const folder of ['.claude/skills','.agents/skills','.hermes/skills','skills']) {
      for(const name of ['wos-coordination','wos-delegation','wos-continuity','wos-review','wos-workspace']) await verifyLegalFiles(root,path.join(nativeRoot,folder,name));
    }
    assert.ok(JSON.parse(run(nativeBinary,['skill','install','--agent','all','--project',nativeRoot])).every(x=>x.action==='unchanged'));
    assert.ok(JSON.parse(run(nativeBinary,['skill','doctor','--agent','all','--project',nativeRoot])).every(x=>x.action==='installed'));
    assert.ok(JSON.parse(run(nativeBinary,['skill','update','--agent','all','--project',nativeRoot])).every(x=>x.action==='unchanged'));
    const genericRoot=path.join(consumer,'generic-skill-fixture');
    assert.equal(JSON.parse(run(nativeBinary,['skill','install','--agent','generic','--dir',genericRoot])).length,5);
    assert.ok(JSON.parse(run(nativeBinary,['skill','doctor','--agent','generic','--dir',genericRoot])).every(x=>x.action==='installed'));
    assert.equal(JSON.parse(run(nativeBinary,['skill','uninstall','--agent','generic','--dir',genericRoot])).length,5);
    assert.ok(JSON.parse(run(nativeBinary,['skill','uninstall','--agent','all','--project',nativeRoot])).every(x=>x.action==='remove'));

    assert.deepEqual(installedRelease.binaries,manifest.cli_binaries);
    const log=[];
    // Reserve and release an ephemeral TCP port immediately before startup.
    const {createServer}=await import('node:net');
    const reservation=createServer(); await new Promise(resolve=>reservation.listen(0,'127.0.0.1',resolve));
    const port=reservation.address().port; await new Promise(resolve=>reservation.close(resolve));
    const credential=randomBytes(32).toString('base64url'),ns='0199d000-0000-7000-8000-000000000001';
    const clientEnv={...process.env,WOS_DIST_CREDENTIAL:credential};
    server=spawn(cli,['server'],{env:{...process.env,WOS_AUTH_MODE:'api_token',WOS_BOOTSTRAP_TOKEN:credential,WOS_BOOTSTRAP_TENANT_ID:ns,WOS_BOOTSTRAP_TENANT_NAME:'Offline distribution',WOS_LISTEN:`127.0.0.1:${port}`,WOS_SQLITE_PATH:path.join(consumer,'smoke.db'),WOS_LOCAL_PRINCIPAL_ID:'npm-smoke',WOS_MCP_ENABLED:'true'},stdio:['ignore','pipe','pipe']});
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
    const headers={authorization:`Bearer ${credential}`,'content-type':'application/json','accept':'application/json, text/event-stream','MCP-Protocol-Version':'2025-06-18'};
    const init=await fetch(base+'/mcp',{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'npm-distribution-smoke',version:'1'}}})});
    assert.equal(init.status,200); const initialized=(await init.json()).result;assert.equal(initialized.protocolVersion,'2025-06-18');assert.equal(initialized.serverInfo.version,manifest.version);
    const tools=await fetch(base+'/mcp',{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:2,method:'tools/list',params:{}})});
    assert.equal(tools.status,200);assert.ok((await tools.json()).result.tools.some(t=>t.name==='wos_claim_work_item'));
    const actor=await fetch(base+'/api/v1/tenants/0199d000-0000-7000-8000-000000000001/outcomes',{headers});
    assert.equal(actor.status,200);
    let commandIndex=0;
    async function mutate(name,command){const r=await fetch(base+'/api/v1/commands/'+name,{method:'POST',headers:{Authorization:`Bearer ${credential}`,'Content-Type':'application/json','Idempotency-Key':`offline-package-command-${++commandIndex}`},body:JSON.stringify({command})});const value=await r.json();assert.ok(r.ok,JSON.stringify(value));return value.value}
    const out=await mutate('create_outcome',{tenant_id:ns,title:'Offline package contract',desired_state:'Typed client and server agree',priority:'normal'}),scope={tenant_id:ns,outcome_id:out.id};
    await mutate('add_criterion',{owner:{...scope,kind:'outcome',id:out.id},expected_version:1,title:'Verified package',required:true,verification_mode:'attestation'});await mutate('activate_outcome',{scope,expected_version:2});
    await mutate('set_tenant_work_protocol',{scope,expected_protocol_version:1,phase:'draining',reason:'Fresh verification instance'});await mutate('set_tenant_work_protocol',{scope,expected_protocol_version:2,phase:'contracts_v1',writers_drained:true,reason:'No older writers in fixture'});
    const work=await mutate('create_work_item',{scope,title:'Use packaged client',priority:'normal',lifecycle:'todo'});
    run(client,['init','--workspace',consumer,'--server',base,'--tenant',ns,'--outcome',out.id,'--credential-env','WOS_DIST_CREDENTIAL','--output','json'],{env:clientEnv});
    const checked=JSON.parse(run(client,['work','checkout',work.id,'--version','1','--workspace',consumer,'--output','json'],{env:clientEnv}));assert.equal(checked.committed,true);assert.equal(checked.work_item_id,work.id);
    run(client,['work','status',work.id,'--workspace',consumer,'--output','json'],{env:clientEnv});run(client,['work','recover',work.id,'--workspace',consumer,'--output','json'],{env:clientEnv});
    await stop(server); assert.equal(server.exitCode,0,'SIGTERM reaches Go graceful shutdown');server=null;
    const signed=await verifySignedPackage(cli,client);
    assert.equal(signed.protocol,'signed_contracts_v3');
    const binary=path.join(consumer,'node_modules/wosctl/native/linux-amd64/wos');
    await fs.writeFile(binary+'.corrupt',Buffer.concat([await fs.readFile(binary),Buffer.from('corruption')]));await fs.rename(binary+'.corrupt',binary);
    assert.throws(()=>run(cli,['version']),/checksum mismatch/);
    const clientBinary=path.join(consumer,'node_modules/wosctl/native/linux-amd64/wosctl');await fs.writeFile(clientBinary+'.corrupt',Buffer.concat([await fs.readFile(clientBinary),Buffer.from('corruption')]));await fs.rename(clientBinary+'.corrupt',clientBinary);assert.throws(()=>run(path.join(consumer,'node_modules/.bin/wosctl'),['version']),/checksum mismatch/);
    return {version:manifest.version,commit:manifest.commit,checks:['artifact hashes','source-identical legal files and npm license metadata','offline npm install','server/client source versions and exit propagation','packaged CLI init/checkout/status/recover','signed protected profile checkout/finish/independent review/recover','four skill destinations with source-identical legal files','native Go offline skill install/update/doctor/uninstall and generic destination','HTTP/MCP/UI readiness','SIGTERM graceful shutdown','corruption rejection','cross-platform standalone Go CLI npm package']};
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
