import assert from 'node:assert/strict';
import {mkdtemp,readFile,rm,mkdir,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {spawnSync} from 'node:child_process';

const version=process.env.WOS_PUBLISHED_VERSION,commit=process.env.WOS_EXPECTED_COMMIT;
assert.match(version??'',/^\d+\.\d+\.\d+$/);
assert.match(commit??'',/^[a-f0-9]{40}$/);
const prefix=await mkdtemp(path.join(tmpdir(),'wos-published-consumer-'));
function execute(program,args,{shell=false}={}) {
 const result=spawnSync(program,args,{encoding:'utf8',maxBuffer:4<<20,shell});
 if(result.error)throw result.error;
 assert.equal(result.status,0,`${path.basename(program)} failed: ${result.stderr}`);
 return result.stdout.trim();
}
try {
 // Only one registry package; the native Go commands install the skill bundle.
 execute(process.platform==='win32'?'npm.cmd':'npm',['install','-g','wosctl@latest','--prefix',prefix,'--ignore-scripts','--no-audit','--no-fund','--registry=https://registry.npmjs.org/'],{shell:process.platform==='win32'});
 const pkgRoot=path.join(prefix,process.platform==='win32'?'node_modules':'lib/node_modules','wosctl');
 const pkg=JSON.parse(await readFile(path.join(pkgRoot,'package.json'),'utf8'));assert.equal(pkg.name,'wosctl');assert.equal(pkg.version,version);
 function cli(name,args){return execute(process.execPath,[path.join(pkgRoot,'bin',name+'.mjs'),...args]);}
 assert.ok(cli('wos',['version']).includes(`wos ${version} (commit ${commit}`));
 const client=JSON.parse(cli('wos',['version','--output','json'])).data;assert.equal(client.version,version);assert.equal(client.commit,commit);
 const workspace=path.join(prefix,'consumer-project');await mkdir(workspace);
 for(const agent of ['codex','all']) {
  const count=agent==='all'?20:5;
  assert.equal(JSON.parse(cli('wos',['skill','install','--agent',agent,'--project',workspace])).length,count);
  assert.ok(JSON.parse(cli('wos',['skill','doctor','--agent',agent,'--project',workspace])).every(x=>x.action==='installed'));
  assert.ok(JSON.parse(cli('wos',['skill','update','--agent',agent,'--project',workspace])).every(x=>x.action==='unchanged'));
  assert.equal(JSON.parse(cli('wos',['skill','uninstall','--agent',agent,'--project',workspace])).length,count);
 }
 const generic=path.join(prefix,'generic-skills');
 assert.equal(JSON.parse(cli('wos',['skill','install','--agent','generic','--dir',generic])).length,5);
 assert.ok(JSON.parse(cli('wos',['skill','doctor','--agent','generic','--dir',generic])).every(x=>x.action==='installed'));
 assert.equal(JSON.parse(cli('wos',['skill','uninstall','--agent','generic','--dir',generic])).length,5);
 const receipt={schema:1,name:'wosctl',version,commit,native_host:process.platform,arch:process.arch,registry_install:'npm install -g wosctl@latest',checks:['global public registry installation','wos/wosctl source versions','native Go skills install/doctor/update/uninstall','codex/all/generic destinations'],passed:true};
 await mkdir('dist',{recursive:true});await writeFile(`dist/published-consumer-${process.platform}.json`,JSON.stringify(receipt,null,2)+'\n');console.log(JSON.stringify(receipt));
} finally {await rm(prefix,{recursive:true,force:true});}
