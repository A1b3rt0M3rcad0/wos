import * as fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { root, validVersion } from '../distribution/build.mjs';

const rank={patch:1,minor:2,major:3};
export function nextVersion(current,bump,first=false) {
  validVersion(current);
  if(!rank[bump])throw new Error(`Invalid bump: ${bump}`);
  if(first)return current;
  let [major,minor,patch]=current.split('.').map(Number);
  if(bump==='major'){major++;minor=0;patch=0;}else if(bump==='minor'){minor++;patch=0;}else patch++;
  return validVersion(`${major}.${minor}.${patch}`);
}

export async function releasePlan(source=root) {
  const version=validVersion((await fs.readFile(path.join(source,'VERSION'),'utf8')).trim());
  let prior=null;
  try{prior=JSON.parse(await fs.readFile(path.join(source,'release-manifest.json'),'utf8'));}catch(e){if(e.code!=='ENOENT')throw e;}
  if(prior && (prior.schema!==1 || prior.version!==version))throw new Error('Prepared release/version mismatch');
  const directory=path.join(source,'.changes');
  const changes=[];
  for(const entry of (await fs.readdir(directory,{withFileTypes:true})).sort((a,b)=>a.name.localeCompare(b.name))) {
    if(entry.name==='README.md')continue;
    if(!entry.isFile() || !/^[a-z0-9][a-z0-9-]*\.json$/.test(entry.name))throw new Error(`Invalid change entry: ${entry.name}`);
    const item=JSON.parse(await fs.readFile(path.join(directory,entry.name),'utf8'));
    if(!rank[item.bump] || typeof item.summary!=='string' || !item.summary.trim() || item.summary.length>1500 || /[\r\n]/.test(item.summary) || Object.keys(item).some(k=>!['bump','summary'].includes(k)))throw new Error(`Invalid change declaration: ${entry.name}`);
    changes.push({file:entry.name,bump:item.bump,summary:item.summary.trim()});
  }
  if(!changes.length)return null;
  const bump=changes.reduce((best,x)=>rank[x.bump]>rank[best]?x.bump:best,'patch');
  return {schema:1,version:nextVersion(version,bump,!prior),previous_prepared_version:prior?.version??null,bump,changes};
}

export async function prepareRelease(source=root) {
  const plan=await releasePlan(source);if(!plan)return null;
  // All semantic inputs are validated before any writes.
  const packages=[];
  for(const folder of ['wos-npm','wos-skill']) {
    const filename=path.join(source,'packages',folder,'package.json');
    const pkg=JSON.parse(await fs.readFile(filename,'utf8'));
    if(pkg.version!==(await fs.readFile(path.join(source,'VERSION'),'utf8')).trim())throw new Error('Package versions must be synchronized before preparing release');
    const updated={...pkg,version:plan.version};
    if(folder==='wos-skill' && pkg.wosCompatibility)updated.wosCompatibility={...pkg.wosCompatibility,service:plan.version};
    packages.push({filename,pkg:updated});
  }
  const openapiFile=path.join(source,'packages/wos-api/commands.openapi.json');
  const openapi=JSON.parse(await fs.readFile(openapiFile,'utf8'));
  if(openapi.info?.version!==(await fs.readFile(path.join(source,'VERSION'),'utf8')).trim())throw new Error('OpenAPI release/version mismatch');
  openapi.info.version=plan.version;
  const notes=`# WOS ${plan.version}\n\n${plan.changes.map(x=>`- ${x.summary}`).join('\n')}\n\nPackages: WOS service with wosctl and five portable skills. Service target: Linux amd64. Client assets: Linux amd64 and Windows amd64; actual matching native test receipts are required before publication. Skills require Node >=22 and a compatible runtime. Cross-building and release preparation do not certify Windows or registry publication.\n`;
  const changelogFile=path.join(source,'CHANGELOG.md');
  const changelog=await fs.readFile(changelogFile,'utf8');
  if(!changelog.startsWith('# Changelog\n'))throw new Error('Unexpected changelog format');
  await fs.writeFile(path.join(source,'VERSION'),plan.version+'\n');
  await fs.writeFile(openapiFile,JSON.stringify(openapi,null,2)+'\n');
  for(const {filename,pkg} of packages)await fs.writeFile(filename,JSON.stringify(pkg,null,2)+'\n');
  await fs.writeFile(path.join(source,'release-manifest.json'),JSON.stringify(plan,null,2)+'\n');
  await fs.mkdir(path.join(source,'docs/releases'),{recursive:true});
  await fs.writeFile(path.join(source,'docs/releases',`${plan.version}.md`),notes);
  await fs.writeFile(changelogFile,changelog.replace('# Changelog\n',`# Changelog\n\n## ${plan.version} — prepared for release\n\n${plan.changes.map(x=>`- ${x.summary}`).join('\n')}\n`));
  for(const c of plan.changes)await fs.unlink(path.join(source,'.changes',c.file));
  return plan;
}

if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try {
    const mode=process.argv[2];if(!['plan','prepare'].includes(mode)||process.argv.length!==3)throw new Error('Usage: node tools/releases/version.mjs plan|prepare');
    console.log(JSON.stringify(mode==='prepare'?await prepareRelease():await releasePlan(),null,2));
  }catch(e){console.error(e.message);process.exitCode=1;}
}
