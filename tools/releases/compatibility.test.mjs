import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {root} from '../distribution/build.mjs';

function digest(bytes){return createHash('sha256').update(bytes).digest('hex');}
function normalize(filename,bytes) {
 if(filename==='packages/wos-api/commands.openapi.json')return bytes.toString().replace(/("version": )"[0-9]+\.[0-9]+\.[0-9]+"/,'$1"PRODUCT_VERSION"');
 return bytes;
}
test('frozen public wire schemas reject unreviewed drift while allowing product version changes',async()=>{
 const frozen=JSON.parse(await readFile(path.join(root,'docs/stable/public-contract.json'),'utf8'));
 assert.ok(Object.keys(frozen.files).length>=5);
 for(const [filename,expected] of Object.entries(frozen.files))assert.equal(digest(normalize(filename,await readFile(path.join(root,filename)))),expected,filename+' requires an explicit compatibility review');
});
test('one public package and both OpenAPI specifications follow the product release/license',async()=>{
 const version=(await readFile(path.join(root,'VERSION'),'utf8')).trim();
 const pkg=JSON.parse(await readFile(path.join(root,'packages/wos-cli-npm/package.json'),'utf8'));assert.equal(pkg.name,'wosctl');assert.equal(pkg.version,version);assert.equal(pkg.license,'Apache-2.0');
 for(const fixture of ['wos-npm','wos-skill'])assert.equal(JSON.parse(await readFile(path.join(root,'packages',fixture,'package.json'),'utf8')).private,true);
 assert.match(await readFile(path.join(root,'LICENSE'),'utf8'),/Apache License/);
 assert.match(await readFile(path.join(root,'Dockerfile'),'utf8'),/org\.opencontainers\.image\.licenses="Apache-2\.0"/);
 assert.equal(JSON.parse(await readFile(path.join(root,'packages/wos-api/commands.openapi.json'),'utf8')).info.version,version);
 assert.equal((await readFile(path.join(root,'packages/wos-api/openapi.yaml'),'utf8')).match(/^  version: (.+)$/m)?.[1],version);
});
