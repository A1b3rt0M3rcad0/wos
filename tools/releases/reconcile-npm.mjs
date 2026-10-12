import assert from 'node:assert/strict';
import {readFile,writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {realAdapters,awaitNpmIntegrity} from './publish.mjs';
import {verifyNativePlatforms} from '../distribution/platform.mjs';

// Read-only registry recovery: artifacts may come from an older accepted source.
// This deliberately does not build, publish, retag or otherwise mutate a registry.
export async function reconcileNpm(directory,source,npm=realAdapters().npm) {
 assert.match(source,/^[a-f0-9]{40}$/);
 const manifest=JSON.parse(await readFile(path.join(directory,'distribution.json'),'utf8'));
 assert.equal(manifest.schema,1);assert.equal(manifest.commit,source);
 assert.match(manifest.version,/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/);
 const expected={'wos-cli':`wosctl-${manifest.version}.tgz`,archive:`wos_${manifest.version}_linux_amd64.tar.gz`,'client-linux':`wosctl_${manifest.version}_linux_amd64.tar.gz`,'client-windows':`wosctl_${manifest.version}_windows_amd64.zip`};
 assert.equal(manifest.artifacts.length,4);const seen=new Set();const sums=[];
 for(const artifact of manifest.artifacts) {
  assert.equal(artifact.filename,expected[artifact.kind]);assert.ok(!seen.has(artifact.kind));seen.add(artifact.kind);
  const bytes=await readFile(path.join(directory,artifact.filename));
  assert.equal(createHash('sha256').update(bytes).digest('hex'),artifact.sha256);sums.push(`${artifact.sha256}  ${artifact.filename}`);
  if(artifact.kind==='wos-cli') {
   assert.equal(artifact.name,'wosctl');assert.equal(artifact.version,manifest.version);
   assert.equal('sha512-'+createHash('sha512').update(bytes).digest('base64'),artifact.integrity);
  }
 }
 sums.push(`${createHash('sha256').update(await readFile(path.join(directory,'distribution.json'))).digest('hex')}  distribution.json`);
 assert.equal(await readFile(path.join(directory,'SHA256SUMS'),'utf8'),sums.join('\n')+'\n');
 await verifyNativePlatforms(manifest,path.join(directory,'acceptance'));
 const entry=manifest.artifacts.find(a=>a.kind==='wos-cli');
 await awaitNpmIntegrity(npm,entry);
 const receipt={schema:1,published:true,name:entry.name,version:entry.version,source_sha:source,integrity:entry.integrity,sha256:entry.sha256,integrity_verified:true,registry_mutations:0};
 await writeFile(path.join(directory,'npm-publication.json'),JSON.stringify(receipt,null,2)+'\n');return receipt;
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
 try { console.log(JSON.stringify(await reconcileNpm('dist',process.env.WOS_EXPECTED_COMMIT))); }
 catch(error) {console.error('Read-only npm reconciliation: '+error.message);process.exitCode=1;}
}
