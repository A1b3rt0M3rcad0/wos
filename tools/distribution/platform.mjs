import * as fs from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {root,run,sha256,validVersion} from './build.mjs';

export async function recordNativePlatform(){
 const platform=process.platform==='win32'?'windows':process.platform;
 if(process.env.WOS_TEST_NATIVE_KEYRING!=='1')throw new Error('Native release gate requires the actual OS credential service fixture');
 if(!['linux','windows'].includes(platform)||process.arch!=='x64')throw new Error('Native acceptance requires Linux/Windows amd64 host');
 const version=validVersion((await fs.readFile(path.join(root,'VERSION'),'utf8')).trim());const commit=run('git',['rev-parse','HEAD']);const epoch=run('git',['show','-s','--format=%ct',commit]);const builtAt=new Date(Number(epoch)*1000).toISOString().replace('.000Z','Z');
 const events=run('go',['test','-race','-count=1','-json','./packages/wos-cli/...','./packages/wos-sdk-go/...','./tests/acceptance']).split('\n').filter(Boolean).map(line=>JSON.parse(line));
 const goTests={passed:events.filter(event=>event.Action==='pass'&&event.Test).length,skipped:events.filter(event=>event.Action==='skip'&&event.Test).map(event=>`${event.Package}:${event.Test}`)};
 if(!goTests.passed)throw new Error('Native gate executed no passing tests');
 run(process.execPath,['--test','packages/wos-skill/test/*.test.mjs']);
 const metadata='github.com/A1b3rt0M3rcad0/wos/packages/wos-cli';const flags=`-s -w -X ${metadata}.Version=${version} -X ${metadata}.Commit=${commit} -X ${metadata}.BuiltAt=${builtAt}`;
 const executable=path.join(root,'dist','native',platform,platform==='windows'?'wosctl.exe':'wosctl');await fs.mkdir(path.dirname(executable),{recursive:true});
 run('go',['build','-trimpath','-buildvcs=false',`-ldflags=${flags}`,'-o',executable,'./packages/wos-cli/cmd/wosctl'],{env:{...process.env,CGO_ENABLED:'0',GOOS:platform,GOARCH:'amd64'}});
 const identity=JSON.parse(run(executable,['version','--output','json'])).data;assert.equal(identity.version,version);assert.equal(identity.commit,commit);assert.equal(identity.built_at,builtAt);
 const proof={schema:1,platform,arch:'amd64',version,commit,built_at:builtAt,binary_sha256:sha256(await fs.readFile(executable)),native_host:process.platform,native_keyring_executed:true,go_tests:goTests,go:run('go',['version']),passed:true,checks:['native Go race workspace/SDK/protocol acceptance','native five-skill installer tests','native client source identity']};
 await fs.writeFile(path.join(root,'dist',`platform-${platform}-amd64.json`),JSON.stringify(proof,null,2)+'\n');return proof;
}
export async function verifyNativePlatforms(manifest,directory=path.join(root,'dist','acceptance')){
 for(const platform of ['linux','windows']){
  const proof=JSON.parse(await fs.readFile(path.join(directory,`platform-${platform}-amd64.json`),'utf8'));
  const binary=manifest.artifacts.find(a=>a.kind===`client-${platform}`);if(!binary)throw new Error('Missing client artifact');
  if(proof.schema!==1||!proof.passed||proof.native_keyring_executed!==true||!Number.isInteger(proof.go_tests?.passed)||proof.go_tests.passed<1||!Array.isArray(proof.go_tests.skipped)||proof.platform!==platform||proof.arch!=='amd64'||proof.native_host!==(platform==='windows'?'win32':'linux')||proof.version!==manifest.version||proof.commit!==manifest.commit||proof.built_at!==manifest.built_at||proof.binary_sha256!==binary.binary_sha256||!Array.isArray(proof.checks)||proof.checks.length!==3)throw new Error(`Missing or mismatched native ${platform} acceptance; cross-build is not acceptance`);
 }
 return true;
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 try{if(process.argv.length!==3||process.argv[2]!=='record')throw new Error('Usage: node tools/distribution/platform.mjs record');console.log(JSON.stringify(await recordNativePlatform(),null,2));}catch(e){console.error(e.message);process.exitCode=1;}
}
