import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { root, validVersion, run } from './build.mjs';
import path from 'node:path';

test('distribution rejects prerelease/untrusted metadata and failed commands', () => {
  for(const value of ['1.2.3','0.1.0','10.20.30']) assert.equal(validVersion(value),value);
  for(const value of ['v1.2.3','1.2.3; echo unsafe','01.2.3','1.2','1.2.3-rc.1']) assert.throws(()=>validVersion(value),/Invalid stable SemVer/);
  assert.throws(()=>run(process.execPath,['-e','process.exit(7)']),/failed \(7\)/);
});

test('single wosctl package identity/version are synchronized without install hooks or dependencies', async () => {
  const version=(await readFile(path.join(root,'VERSION'),'utf8')).trim();
  for(const folder of ['wos-cli-npm']) {
    const pkg=JSON.parse(await readFile(path.join(root,'packages',folder,'package.json'),'utf8'));
    assert.equal(pkg.version,version); assert.equal(pkg.license,'Apache-2.0');
    assert.equal(pkg.name, 'wosctl');
    assert.equal(pkg.dependencies,undefined);
    for(const hook of ['preinstall','install','postinstall','prepare']) assert.equal(pkg.scripts?.[hook],undefined);
  }
});

test('publication needs actual matching native hosts and binary checksums', async t => {
  const {verifyNativePlatforms}=await import('./platform.mjs');
  const fs=await import('node:fs/promises');const os=await import('node:os');
  const directory=await fs.mkdtemp(path.join(os.tmpdir(),'wos-platform-proof-'));t.after(()=>fs.rm(directory,{recursive:true,force:true}));
  const manifest={version:'0.2.0',commit:'a'.repeat(40),built_at:'2026-10-08T19:00:00Z',artifacts:['linux','windows'].map(platform=>({kind:`client-${platform}`,binary_sha256:'b'.repeat(64)}))};
  await assert.rejects(verifyNativePlatforms(manifest,directory),/ENOENT/);
  for(const platform of ['linux','windows'])await fs.writeFile(path.join(directory,`platform-${platform}-amd64.json`),JSON.stringify({schema:1,platform,arch:'amd64',native_host:platform==='windows'?'win32':'linux',version:manifest.version,commit:manifest.commit,built_at:manifest.built_at,binary_sha256:'b'.repeat(64),passed:true,native_keyring_executed:true,go_tests:{passed:1,skipped:[]},checks:['race','installer','version']}));
  assert.equal(await verifyNativePlatforms(manifest,directory),true);
  const windows=path.join(directory,'platform-windows-amd64.json');const proof=JSON.parse(await fs.readFile(windows,'utf8'));proof.native_keyring_executed=false;await fs.writeFile(windows,JSON.stringify(proof));await assert.rejects(verifyNativePlatforms(manifest,directory),/mismatched/);proof.native_keyring_executed=true;proof.native_host='linux';await fs.writeFile(windows,JSON.stringify(proof));await assert.rejects(verifyNativePlatforms(manifest,directory),/cross-build/);
  proof.native_host='win32';proof.commit='c'.repeat(40);await fs.writeFile(windows,JSON.stringify(proof));await assert.rejects(verifyNativePlatforms(manifest,directory),/mismatched/);
});

test('Docker and CI use the reviewed Go patch from the shared version file', async () => {
 const goVersion=(await readFile(path.join(root,'.go-version'),'utf8')).trim();
 assert.match(goVersion,/^1\.[0-9]+\.[0-9]+$/);
 const docker=await readFile(path.join(root,'Dockerfile'),'utf8');
 assert.ok(docker.includes('FROM golang:'+goVersion+'-alpine AS build'),'Docker Go patch drifted from .go-version');
 const module=await readFile(path.join(root,'go.mod'),'utf8');
 assert.ok(module.includes('go '+goVersion.split('.').slice(0,2).join('.')),'Go language baseline differs');
 for(const name of await readdir(path.join(root,'.github','workflows'))) {
  if(!name.endsWith('.yml'))continue;
  const source=await readFile(path.join(root,'.github','workflows',name),'utf8');
  if(source.includes('actions/setup-go@')) {
   assert.ok(source.includes('go-version-file: .go-version'),name+' does not pin the shared Go version');
   assert.ok(!source.includes('go-version-file: go.mod'),name+' can drift to another patch');
  }
 }
});
