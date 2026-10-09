import assert from 'node:assert/strict';
import {readFile,writeFile} from 'node:fs/promises';
import {execFileSync} from 'node:child_process';
import {resolve,join} from 'node:path';
import {pathToFileURL,fileURLToPath} from 'node:url';
import {createHash} from 'node:crypto';
const workspace=resolve(process.argv[2]),output=process.argv[3];
const tests=execFileSync('node',['--test','--test-reporter=tap','normalize-title.test.mjs'],{cwd:workspace,encoding:'utf8',timeout:20000});
assert.match(tests,/# fail 0/);assert.match(tests,/# skipped 0/);assert.match(tests,/# tests [1-9][0-9]*/);
const {normalizeTitle:n}=await import(pathToFileURL(join(workspace,'normalize-title.mjs')));
assert.equal(typeof n,'function');let checks=0;
const equal=(input,expected)=>{assert.equal(n(input),expected);checks++};
for(const [value,expected] of [[' \t João\n\n🙂  ','João 🙂'],['',''],['   \n\t',''],['a\u200bb','a\u200bb'],['Lipo São Paulo','Lipo São Paulo']])equal(value,expected);
const spaces=[0x0009,0x000a,0x000b,0x000c,0x000d,0x0020,0x00a0,0x1680,0x2000,0x2001,0x2002,0x2003,0x2004,0x2005,0x2006,0x2007,0x2008,0x2009,0x200a,0x2028,0x2029,0x202f,0x205f,0x3000,0xfeff];
for(const code of spaces){const c=String.fromCodePoint(code);equal(c+'a'+c+c+'🙂'+c,'a 🙂')}
for(const value of [null,undefined,17,false,{},[],new String('abc')]){assert.throws(()=>n(value),TypeError);checks++}
for(const value of ['  a  b  ','João\t🙂','\u00a0a\u00a0','\n\n','a\u200bb','']){assert.equal(n(n(value)),n(value));checks++}
const commit=execFileSync('git',['rev-parse','HEAD'],{cwd:workspace,encoding:'utf8'}).trim();
assert.match(commit,/^[0-9a-f]{40}$/);assert.equal(execFileSync('git',['status','--porcelain'],{cwd:workspace,encoding:'utf8'}),'');
const result={schema:1,kind:'actual-external-controlled-quality',workspace,commit,node:process.version,tests:tests.match(/# tests (\d+)/)[1],external_assertions:checks,quality_passed:true,integration_passed:true,accepted_at:new Date().toISOString(),validator_sha256:createHash('sha256').update(await readFile(fileURLToPath(import.meta.url))).digest('hex')};
await writeFile(output,JSON.stringify(result,null,2)+'\n');console.log(JSON.stringify(result));
