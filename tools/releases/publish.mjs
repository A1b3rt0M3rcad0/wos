import * as fs from 'node:fs/promises';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {verifyNativePlatforms} from '../distribution/platform.mjs';
import { root, run, sha256, validVersion } from '../distribution/build.mjs';

export const repository='A1b3rt0M3rcad0/wos';
export const imageName='ghcr.io/a1b3rt0m3rcad0/wos';

export async function loadRelease(output=path.join(root,'dist')) {
  const manifest=JSON.parse(await fs.readFile(path.join(output,'distribution.json'),'utf8'));
  validVersion(manifest.version);
  if(manifest.schema!==1 || !/^[a-f0-9]{40}$/.test(manifest.commit) || manifest.artifacts.length!==5)throw new Error('Invalid distribution manifest');
  const expected={wos:`wos-${manifest.version}.tgz`,'wos-skill':`wos-skill-${manifest.version}.tgz`,archive:`wos_${manifest.version}_linux_amd64.tar.gz`,"client-linux":`wosctl_${manifest.version}_linux_amd64.tar.gz`,"client-windows":`wosctl_${manifest.version}_windows_amd64.zip`};
  const seen=new Set();
  for(const a of manifest.artifacts) {
    if(seen.has(a.kind) || a.filename!==expected[a.kind])throw new Error('Invalid/duplicate distribution artifact');seen.add(a.kind);
    const bytes=await fs.readFile(path.join(output,a.filename));
    if(sha256(bytes)!==a.sha256)throw new Error(`Corrupt artifact: ${a.filename}`);
    if(a.kind.startsWith('client-')&&(!/^[a-f0-9]{64}$/.test(a.binary_sha256)||a.platform!==a.kind.slice(7)||a.arch!=='amd64'))throw new Error('Invalid client binary identity');
    if(['wos','wos-skill'].includes(a.kind) && (a.name!==`@a1b3rt0m3rcad0/${a.kind}` || a.version!==manifest.version || a.integrity!==`sha512-${createHash('sha512').update(bytes).digest('base64')}`))throw new Error(`Invalid npm identity/integrity: ${a.kind}`);
  }
  const version=(await fs.readFile(path.join(root,'VERSION'),'utf8')).trim();
  const prepared=JSON.parse(await fs.readFile(path.join(root,'release-manifest.json'),'utf8'));
  if(version!==manifest.version || prepared.version!==version || prepared.schema!==1)throw new Error('Release must match prepared source version');
  if(run('git',['rev-parse','HEAD'])!==manifest.commit)throw new Error('Artifacts must come from current source commit');
  if(run('git',['status','--porcelain','--untracked-files=normal']))throw new Error('Release source must be clean');
  if((await fs.readdir(path.join(root,'.changes'))).some(f=>f!=='README.md'))throw new Error('Unconsumed changes must go through a release PR');
  const notes=await fs.readFile(path.join(root,'docs/releases',`${version}.md`),'utf8');
  const assets=[...manifest.artifacts.map(a=>({filename:a.filename,sha256:a.sha256})),...await Promise.all(['distribution.json','SHA256SUMS'].map(async filename=>({filename,sha256:sha256(await fs.readFile(path.join(output,filename)))})))];
  // Verify the human-readable checksum file, not only JSON declarations.
  const checksums=await fs.readFile(path.join(output,'SHA256SUMS'),'utf8');
  const expectedChecksums=assets.filter(a=>a.filename!=='SHA256SUMS').map(a=>`${a.sha256}  ${a.filename}`).join('\n')+'\n';
  if(checksums!==expectedChecksums)throw new Error('Checksum file does not match release assets');
  return {manifest,notes,assets,output};
}

export async function publishRelease(release,{store,npm,image}) {
  const {manifest,notes,assets}=release;
  const tag=`v${manifest.version}`;
  // Check all identities before writing; registry versions are immutable.
  const published=new Set();
  for(const a of manifest.artifacts.filter(a=>['wos','wos-skill'].includes(a.kind))) {
    const integrity=await npm.integrity(a.name,a.version);
    if(integrity && integrity!==a.integrity)throw new Error(`Registry content differs for ${a.name}@${a.version}; never overwrite/reuse this version`);
    if(integrity)published.add(a.name);
  }
  await image.check(manifest);
  await store.ensureTag(tag,manifest.commit);
  const draft=await store.ensureDraft(tag,manifest.commit,notes);
  for(const asset of assets)await store.ensureAsset(draft,asset,release.output);
  for(const a of manifest.artifacts.filter(a=>['wos','wos-skill'].includes(a.kind))) {
    if(!published.has(a.name))await npm.publish(a,release.output);
    if(await npm.integrity(a.name,a.version)!==a.integrity)throw new Error(`Published npm integrity could not be verified: ${a.name}`);
  }
  await image.publish(manifest);
  await image.check(manifest,true);
  await store.finalize(draft);
  return {version:manifest.version,commit:manifest.commit,tag,npm:manifest.artifacts.filter(a=>['wos','wos-skill'].includes(a.kind)).map(a=>a.name),image:`${imageName}:${manifest.version}`};
}

function command(program,args,options={}) {
  const r=spawnSync(program,args,{cwd:root,encoding:'utf8',maxBuffer:8<<20,...options});
  if(r.error)throw r.error;
  return r;
}
function notFound(r) {return r.status!==0 && /HTTP 404/.test(r.stderr);}

export function realAdapters({provenance=false,execute=command}={}) {
  function checked(program,args,options={}) {
    const r=execute(program,args,options);
    if(r.status!==0)throw new Error(`${program} operation failed (exit ${r.status}); check authentication, permissions and service availability`);
    return r.stdout.trim();
  }
  function ghJson(args) {return JSON.parse(checked('gh',args));}
  const api=`repos/${repository}`;
  const store={
    async ensureTag(tag,commit) {
      const args=['api',`${api}/git/ref/tags/${tag}`];
      const r=execute('gh',args);
      if(notFound(r))checked('gh',['api','--method','POST',`${api}/git/refs`,'-f',`ref=refs/tags/${tag}`,'-f',`sha=${commit}`]);
      else {
        if(r.status!==0)throw new Error('Cannot read release tag; authentication/network error');
        const ref=JSON.parse(r.stdout);
        if(ref.object.type!=='commit'||ref.object.sha!==commit)throw new Error('Release tag points to another commit; refusing replacement');
      }
    },
    async ensureDraft(tag,commit,notes) {
      // The by-tag endpoint only guarantees published releases; list drafts too.
      for(let page=1;;page++) {
        const releases=ghJson(['api',`${api}/releases?per_page=100&page=${page}`]);
        const matches=releases.filter(r=>r.tag_name===tag);
        if(matches.length>1)throw new Error('Multiple releases use this version tag');
        if(matches.length) {
          const release=matches[0];
          if(release.target_commitish!==commit)throw new Error('Release identity differs from source commit');
          return release;
        }
        if(releases.length<100)break;
      }
      return ghJson(['api','--method','POST',`${api}/releases`,'-f',`tag_name=${tag}`,'-f',`target_commitish=${commit}`,'-f',`name=WOS ${tag}`,'-f',`body=${notes}`,'-F','draft=true']);
    },
    async ensureAsset(release,asset,output) {
      let matches=ghJson(['api',`${api}/releases/${release.id}/assets`]).filter(a=>a.name===asset.filename);
      if(matches.length>1)throw new Error('Duplicate release assets');
      if(!matches.length) {
        const upload=new URL(release.upload_url.replace(/\{.*$/,''));
        if(upload.protocol!=='https:' || upload.hostname!=='uploads.github.com' || upload.pathname!==`/repos/${repository}/releases/${release.id}/assets`)throw new Error('Unexpected release upload endpoint');
        upload.searchParams.set('name',asset.filename);
        const created=ghJson(['api','--method','POST',upload.href,'-H','Content-Type: application/octet-stream','--input',path.join(output,asset.filename)]);
        matches=[created];
      }
      // Read exact bytes by ID, including drafts, without tag lookup or clobber.
      const downloaded=execute('gh',['api',`${api}/releases/assets/${matches[0].id}`,'-H','Accept: application/octet-stream'],{encoding:null,maxBuffer:64<<20});
      if(downloaded.status!==0)throw new Error(`Cannot verify uploaded asset: ${asset.filename}`);
      if(sha256(downloaded.stdout)!==asset.sha256)throw new Error(`Release asset mismatch: ${asset.filename}`);
    },
    async finalize(release) {
      if(release.draft)checked('gh',['api','--method','PATCH',`${api}/releases/${release.id}`,'-F','draft=false','-f','make_latest=true']);
    },
  };
  const npm={
    async integrity(name,version) {
      const r=execute('npm',['view',`${name}@${version}`,'dist.integrity','--json','--registry=https://registry.npmjs.org/']);
      if(r.status===0) {
        const value=JSON.parse(r.stdout||'null');if(typeof value!=='string')throw new Error('npm version exists without integrity metadata');return value;
      }
      let error;try{error=JSON.parse(r.stdout);}catch{}
      if(error?.error?.code==='E404')return null;
      throw new Error('Cannot inspect npm registry; only explicit E404 means absent');
    },
    async publish(artifact,output) {
      const args=['publish',path.join(output,artifact.filename),'--ignore-scripts','--access=public','--tag=latest','--registry=https://registry.npmjs.org/'];
      if(provenance)args.push('--provenance');
      checked('npm',args);
    },
  };
  let remoteExists=false;
  const image={
    async check(manifest,required=false) {
      const tag=`${imageName}:${manifest.version}`;
      const r=execute('docker',['manifest','inspect',tag]);
      if(r.status!==0) {
        if(!required && /manifest unknown|no such manifest|not found/i.test(r.stderr)){remoteExists=false;return;}
        throw new Error('Cannot inspect GHCR image (missing image, access or network failure)');
      }
      checked('docker',['pull',tag]);
      const commit=checked('docker',['inspect','--format','{{index .Config.Labels "org.opencontainers.image.revision"}}',tag]);
      const version=checked('docker',['inspect','--format','{{index .Config.Labels "org.opencontainers.image.version"}}',tag]);
      if(commit!==manifest.commit||version!==manifest.version)throw new Error('GHCR version already belongs to another source; refusing replacement');
      remoteExists=true;
    },
    async publish(manifest) {
      const tag=`${imageName}:${manifest.version}`;
      if(remoteExists)return;
      const local='wos:release-tested';
      // Built and smoked by the workflow before this mutation stage.
      if(checked('docker',['inspect','--format','{{index .Config.Labels "org.opencontainers.image.revision"}}',local])!==manifest.commit)throw new Error('Local image source mismatch');
      if(checked('docker',['inspect','--format','{{index .Config.Labels "org.opencontainers.image.version"}}',local])!==manifest.version)throw new Error('Local image version mismatch');
      checked('docker',['tag',local,tag]);checked('docker',['push',tag]);
    },
  };
  return {store,npm,image};
}

if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try {
    const dry=process.argv.length===3 && process.argv[2]==='--dry-run';
    if(process.argv.length>2&&!dry)throw new Error('Usage: node tools/releases/publish.mjs [--dry-run]');
    const release=await loadRelease();
    if(dry)console.log(JSON.stringify({dryRun:true,version:release.manifest.version,commit:release.manifest.commit,assets:release.assets.map(a=>a.filename),steps:['require matching native Linux and Windows acceptance receipts','verify immutable tag/registry/image identities','ensure draft and verified assets','publish/reconcile both npm tarballs','publish/reconcile tested GHCR image','finalize GitHub Release']},null,2));
    else {
      if(process.env.WOS_RELEASE_PUBLISH!=='1'||process.env.GITHUB_REPOSITORY?.toLowerCase()!==repository.toLowerCase()||!process.env.GH_TOKEN)throw new Error('Publication requires the authorized repository workflow and WOS_RELEASE_PUBLISH=1');
      if(process.env.NPM_AUTH_MODE!=='oidc'&&!process.env.NODE_AUTH_TOKEN)throw new Error('Configure npm trusted publishing or NPM_TOKEN in GitHub Actions; never put credentials in source');
      await verifyNativePlatforms(release.manifest);
      console.log(JSON.stringify(await publishRelease(release,realAdapters({provenance:process.env.NPM_PROVENANCE==='true'})),null,2));
    }
  }catch(e){console.error(e.message);process.exitCode=1;}
}
