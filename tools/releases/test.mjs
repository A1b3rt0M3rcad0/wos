import test from 'node:test';
import assert from 'node:assert/strict';
import * as fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import {releasePlan,prepareRelease,nextVersion} from './version.mjs';
import {publishRelease,realAdapters} from './publish.mjs';
import {sha256} from '../distribution/build.mjs';

async function fixture(t) {
  const dir=await fs.mkdtemp(path.join(os.tmpdir(),'wos-version-test-'));t.after(()=>fs.rm(dir,{recursive:true,force:true}));
  await fs.mkdir(path.join(dir,'.changes'));
  await fs.writeFile(path.join(dir,'.changes/README.md'),'queue');
  await fs.writeFile(path.join(dir,'VERSION'),'0.1.0\n');
  await fs.writeFile(path.join(dir,'CHANGELOG.md'),'# Changelog\n\nExisting history\n');
  for(const name of ['wos-npm','wos-skill']) {
    await fs.mkdir(path.join(dir,'packages',name),{recursive:true});
    await fs.writeFile(path.join(dir,'packages',name,'package.json'),JSON.stringify({name,version:'0.1.0',...(name==='wos-skill'?{wosCompatibility:{service:'0.1.0',httpApi:'v1',snapshotSchema:1}}:{})}));
  }
  return dir;
}
async function change(dir,name,bump,summary='A bounded change') {
  await fs.writeFile(path.join(dir,'.changes',`${name}.json`),JSON.stringify({bump,summary}));
}

test('explicit stable SemVer chooses highest bump and first release keeps baseline', () => {
  assert.equal(nextVersion('0.1.0','minor',true),'0.1.0');
  assert.equal(nextVersion('0.1.0','patch'),'0.1.1');
  assert.equal(nextVersion('0.1.0','minor'),'0.2.0');
  assert.equal(nextVersion('0.1.0','major'),'1.0.0');
  assert.throws(()=>nextVersion('0.1.0','guess'),/Invalid bump/);
});

test('preparation is repeatable, consumes changes and synchronizes both packages and notes', async t => {
  const dir=await fixture(t);
  assert.equal(await releasePlan(dir),null);
  await change(dir,'a','patch');await change(dir,'b','minor','Add delegation');
  assert.equal((await releasePlan(dir)).version,'0.1.0');
  assert.equal((await prepareRelease(dir)).bump,'minor');
  assert.equal(await prepareRelease(dir),null);
  await change(dir,'c','patch','Fix claim handoff');
  assert.equal((await prepareRelease(dir)).version,'0.1.1');
  for(const folder of ['wos-npm','wos-skill'])assert.equal(JSON.parse(await fs.readFile(path.join(dir,'packages',folder,'package.json'),'utf8')).version,'0.1.1');
  assert.equal(JSON.parse(await fs.readFile(path.join(dir,'packages/wos-skill/package.json'),'utf8')).wosCompatibility.service,'0.1.1');
  assert.match(await fs.readFile(path.join(dir,'docs/releases/0.1.1.md'),'utf8'),/Fix claim handoff/);
  assert.match(await fs.readFile(path.join(dir,'CHANGELOG.md'),'utf8'),/Existing history/);
  await change(dir,'d','patch');await change(dir,'e','major');
  assert.equal((await releasePlan(dir)).version,'1.0.0');
});

test('bad changes and unsynchronized packages fail before writing any release state', async t => {
  const dir=await fixture(t);await change(dir,'bad','wild');
  await assert.rejects(prepareRelease(dir),/Invalid change/);
  assert.equal(await fs.readFile(path.join(dir,'VERSION'),'utf8'),'0.1.0\n');
  await fs.unlink(path.join(dir,'.changes/bad.json'));
  await fs.symlink(path.join(dir,'VERSION'),path.join(dir,'.changes/link.json'));
  await assert.rejects(releasePlan(dir),/Invalid change entry/);
  await fs.unlink(path.join(dir,'.changes/link.json'));await change(dir,'good','minor');
  await fs.writeFile(path.join(dir,'packages/wos-npm/package.json'),'{"version":"9.9.9"}');
  await assert.rejects(prepareRelease(dir),/synchronized/);
  assert.equal(await fs.readFile(path.join(dir,'VERSION'),'utf8'),'0.1.0\n');
});

function publicationFixture() {
  const manifest={version:'0.1.0',commit:'a'.repeat(40),artifacts:[{kind:'wos',name:'@a1b3rt0m3rcad0/wos',version:'0.1.0',integrity:'service'},{kind:'wos-skill',name:'@a1b3rt0m3rcad0/wos-skill',version:'0.1.0',integrity:'skills'},{kind:'archive'},{kind:'client-linux'},{kind:'client-windows'}]};
  const release={manifest,notes:'release notes',assets:[{filename:'asset'}],output:'/artifacts'};
  const state={registry:new Map(),calls:[],image:false,final:false,failSkills:true};
  const adapters={
    store:{async ensureTag(){state.calls.push('tag');},async ensureDraft(){state.calls.push('draft');return {id:1};},async ensureAsset(){state.calls.push('asset');},async finalize(){state.final=true;state.calls.push('final');}},
    npm:{async integrity(name){return state.registry.get(name)??null;},async publish(a){state.calls.push(a.name);if(a.kind==='wos-skill'&&state.failSkills)throw new Error('Registry unavailable');state.registry.set(a.name,a.integrity);}},
    image:{async check(m,required=false){if(required&&!state.image)throw new Error('Image not available');},async publish(){state.image=true;state.calls.push('image');}},
  };
  return {release,state,adapters};
}

test('partial registry failure keeps release unfinished and retry reconciles the first immutable package', async () => {
  const {release,state,adapters}=publicationFixture();
  await assert.rejects(publishRelease(release,adapters),/Registry unavailable/);
  assert.equal(state.final,false);assert.equal(state.image,false);
  assert.equal(state.registry.get('@a1b3rt0m3rcad0/wos'),'service');
  state.failSkills=false;
  await publishRelease(release,adapters);
  assert.equal(state.calls.filter(x=>x==='@a1b3rt0m3rcad0/wos').length,1);
  assert.equal(state.final,true);assert.equal(state.calls.at(-1),'final');
});

test('registry collisions and inspection failures stop before publication mutations', async () => {
  const {release,state,adapters}=publicationFixture();
  state.registry.set('@a1b3rt0m3rcad0/wos','other bytes');
  await assert.rejects(publishRelease(release,adapters),/never overwrite/);
  assert.deepEqual(state.calls,[]);
  state.registry.clear();adapters.npm.integrity=async()=>{throw new Error('Unauthorized');};
  await assert.rejects(publishRelease(release,adapters),/Unauthorized/);
  assert.deepEqual(state.calls,[]);
});

test('tag, asset and image failures never finalize a release', async () => {
  for(const failure of ['ensureTag','ensureAsset','image']) {
    const {release,state,adapters}=publicationFixture();state.failSkills=false;
    if(failure==='image')adapters.image.publish=async()=>{throw new Error('Image failure');};
    else adapters.store[failure]=async()=>{throw new Error('Identity or asset failure');};
    await assert.rejects(publishRelease(release,adapters),/failure/);
    assert.equal(state.final,false);
  }
});

test('real GitHub adapter resumes drafts by release list and verifies exact asset bytes by ID', async () => {
  const commit='b'.repeat(40);const bytes=Buffer.from('published asset bytes');const calls=[];
  const draft={id:9,tag_name:'v0.1.0',target_commitish:commit,draft:true};
  const execute=(program,args)=>{
    calls.push(args);
    const endpoint=args[1];
    if(endpoint.endsWith('/releases?per_page=100&page=1'))return {status:0,stdout:JSON.stringify([draft]),stderr:''};
    if(endpoint.endsWith('/releases/9/assets'))return {status:0,stdout:JSON.stringify([{id:10,name:'asset.tgz'}]),stderr:''};
    if(endpoint.endsWith('/releases/assets/10'))return {status:0,stdout:bytes,stderr:''};
    throw new Error(`Unexpected mutation/lookup: ${args.join(' ')}`);
  };
  const {store}=realAdapters({execute});
  assert.equal((await store.ensureDraft('v0.1.0',commit,'notes')).id,9);
  await store.ensureAsset(draft,{filename:'asset.tgz',sha256:sha256(bytes)},'/assets');
  assert.ok(!calls.some(a=>a.includes('POST')));
  await assert.rejects(store.ensureAsset(draft,{filename:'asset.tgz',sha256:sha256('different')},'/assets'),/mismatch/);
});

test('npm adapter treats only explicit E404 as absent and propagates permission/network failures', async () => {
  let response={status:1,stdout:'{"error":{"code":"E404"}}',stderr:''};
  const {npm}=realAdapters({execute:()=>response});
  assert.equal(await npm.integrity('@scope/package','0.1.0'),null);
  response={status:1,stdout:'{"error":{"code":"E403"}}',stderr:''};
  await assert.rejects(npm.integrity('@scope/package','0.1.0'),/only explicit E404/);
  response={status:0,stdout:'"sha512-integrity"',stderr:''};
  assert.equal(await npm.integrity('@scope/package','0.1.0'),'sha512-integrity');
});
