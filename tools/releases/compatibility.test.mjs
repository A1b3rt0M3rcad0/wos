import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {root} from '../distribution/build.mjs';
import {legalEntries, licenseReference, npmLicense, verifyLegalFiles} from '../distribution/legal.mjs';
import {renderReadmeRelease,updateReadmeRelease} from './version.mjs';

function digest(bytes){return createHash('sha256').update(bytes).digest('hex');}
test('README source metadata follows VERSION and published badges stay live',async()=>{
 const version=(await readFile(path.join(root,'VERSION'),'utf8')).trim();
 const readme=await readFile(path.join(root,'README.md'),'utf8');
 const section=readme.match(/<!-- wos-release:start -->([\s\S]*?)<!-- wos-release:end -->/)?.[1];
 assert.ok(section,'README release section is missing');
 assert.equal(section.match(/\*\*Source version: \[([^\]]+)\]/)?.[1],version,'README source version differs from VERSION');
 assert.ok(section.includes('`wosctl@'+version+'`'),'README npm metadata version differs from VERSION');
 assert.ok(section.includes(`](docs/releases/${version}.md)`),'README release-notes link differs from VERSION');
 await readFile(path.join(root,'docs/releases',`${version}.md`));
 assert.equal(updateReadmeRelease(readme,version),readme,'README version metadata is stale');
 assert.ok(readme.includes(renderReadmeRelease(version)));
 assert.ok(readme.includes('https://img.shields.io/github/v/release/A1b3rt0M3rcad0/wos?'));
 assert.ok(readme.includes('https://img.shields.io/npm/v/wosctl?'));
 assert.ok(!readme.includes('docs/images/badges/release.svg'));
 assert.ok(!readme.includes('docs/images/badges/npm.svg'));
});
function normalize(filename,bytes) {
 if(filename==='packages/wos-api/commands.openapi.json')return bytes.toString().replace(/("version": )"[0-9]+\.[0-9]+\.[0-9]+"/,'$1"PRODUCT_VERSION"');
 return bytes;
}
test('frozen public wire schemas reject unreviewed drift while allowing product version changes',async()=>{
 const frozen=JSON.parse(await readFile(path.join(root,'docs/stable/public-contract.json'),'utf8'));
 assert.ok(Object.keys(frozen.files).length>=5);
 for(const [filename,expected] of Object.entries(frozen.files))assert.equal(digest(normalize(filename,await readFile(path.join(root,filename)))),expected,filename+' requires an explicit compatibility review');
});

// Reject a removed old field/enum or new required input even after an explicit
// candidate fingerprint review. Byte/shape limits remain identical on old nodes.
function additiveSchema(before,after,where,reviewedRequiredAdds=new Set(),reviewedConstraintValues=new Map()){
 assert.ok(after,where+' was removed');
 for(const [key,value] of Object.entries(before)){
  if(key==='properties'){
   for(const [name,child] of Object.entries(value))additiveSchema(child,after.properties?.[name],where+'.'+name,reviewedRequiredAdds,reviewedConstraintValues);
  }else if(key==='required'){
   for(const name of after.required??[])assert.ok(value.includes(name)||reviewedRequiredAdds.has(`${where}.${name}`),where+' newly requires '+name);
  }else if(key==='enum'){
   for(const item of value)assert.ok(after.enum?.includes(item),where+' removed enum '+item);
  }else if(['anyOf','oneOf','allOf'].includes(key)){
   assert.equal(after[key]?.length,value.length,where+'.'+key);
   value.forEach((child,index)=>additiveSchema(child,after[key][index],where+'.'+key+'['+index+']',reviewedRequiredAdds,reviewedConstraintValues));
  }else if(key==='items'&&typeof value==='object'){
   additiveSchema(value,after.items,where+'.items',reviewedRequiredAdds,reviewedConstraintValues);
  }else if(key==='maxItems'&&reviewedConstraintValues.has(`${where}.maxItems`)){
   assert.equal(after.maxItems,reviewedConstraintValues.get(`${where}.maxItems`),where+'.maxItems differs from reviewed value');
  }else assert.deepEqual(after[key],value,where+'.'+key+' changed existing constraint');
 }
 if(!before.required)assert.equal(after.required?.length??0,0,where+' acquired required inputs');
 for(const key of Object.keys(after))assert.ok(key in before||key==='required'&&(after.required?.length??0)===0||key==='maxItems'&&reviewedConstraintValues.get(`${where}.maxItems`)===after.maxItems,where+' added an existing-node constraint '+key);
}
test('released v0.4.2 baseline stays immutable and candidate breaking inputs are exact and reviewed',async()=>{
 const directory=path.join(root,'docs/stable/contracts/v0.4.2');
 const historicalBytes=await readFile(path.join(directory,'public-contract.json'));
 assert.equal(digest(historicalBytes),'661320a06ff5d510bfabe0973bdc8ee4a9b1ceddf2b81be5ef62b569518a5266');
 const historical=JSON.parse(historicalBytes);
 for(const filename of ['packages/wos-cli/schemas/profile-v2.schema.json','packages/wos-cli/schemas/contract-v2.schema.json']){
  const bytes=await readFile(path.join(directory,path.basename(filename)));
  assert.equal(digest(normalize(filename,bytes)),historical.files[filename]);
  additiveSchema(JSON.parse(bytes),JSON.parse(await readFile(path.join(root,filename))),filename);
 }
 const filename='packages/wos-api/commands.openapi.json';
 const bytes=await readFile(path.join(directory,'commands.openapi.json'));
 assert.equal(digest(normalize(filename,bytes)),historical.files[filename]);
 const before=JSON.parse(bytes),after=JSON.parse(await readFile(path.join(root,filename)));
 // The operational hierarchy plan makes Issue ownership mandatory. These are
 // the only reviewed request-shape breaks relative to published v0.4.2; the
 // immutable released snapshot remains untouched and every other old shape is
 // checked with the additive compatibility guard below.
 const reviewedRequiredAdds=new Set([
  'Catalog_create_issue.objective_id',
  'Catalog_report_issue_with_blocker.objective_id',
 ]);
 const reviewedConstraintValues=new Map([
  ['SigningSecurityIntentV2.credential_policy.permitted_operations.maxItems',128],
 ]);
 for(const [name,schema] of Object.entries(before.components.schemas))additiveSchema(schema,after.components.schemas[name],name,reviewedRequiredAdds,reviewedConstraintValues);
 for(const [route,operations] of Object.entries(before.paths)){
  assert.ok(after.paths[route],route+' removed');
  if(route==='/api/v1/security/signing-identity'){
   assert.deepEqual(after.paths[route].get.parameters,operations.get.parameters);
   assert.deepEqual(Object.keys(after.paths[route].get.responses),Object.keys(operations.get.responses));
  }else if(route==='/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}'){
   // Reviewed central-workflow addition: only these optional query filters.
   // Preserve every old path parameter, response and operation annotation.
   const current=structuredClone(after.paths[route]);
   assert.deepEqual(current.get.parameters.slice(0,operations.get.parameters.length),operations.get.parameters);
   const additions=current.get.parameters.splice(operations.get.parameters.length);
   const optionalStrings=['holder_principal_id','work_item_id','status','cursor'].map(name=>({description:'Optional executions discovery filter or continuation cursor.',in:'query',name,schema:{type:'string'}}));
   assert.deepEqual(additions,[...optionalStrings,{in:'query',name:'limit',schema:{default:25,maximum:100,minimum:1,type:'integer'}}]);
   assert.deepEqual(current,operations,route+' changed an existing wire property');
  }else assert.deepEqual(after.paths[route],operations,route+' changed existing wire operation');
 }
});
test('one public package and both OpenAPI specifications follow the product release/license',async()=>{
 const version=(await readFile(path.join(root,'VERSION'),'utf8')).trim();
 const pkg=JSON.parse(await readFile(path.join(root,'packages/wos-cli-npm/package.json'),'utf8'));assert.equal(pkg.name,'wosctl');assert.equal(pkg.version,version);assert.equal(pkg.license,npmLicense);
 for(const fixture of ['wos-npm','wos-skill'])assert.equal(JSON.parse(await readFile(path.join(root,'packages',fixture,'package.json'),'utf8')).private,true);
 const license=await readFile(path.join(root,'LICENSE'),'utf8');
 assert.match(license,/WOS Sustainable Use License/);
 assert.doesNotMatch(license,/Enterprise Edition/);
 await verifyLegalFiles(root,path.join(root,'packages/wos-skill'));
 const docker=await readFile(path.join(root,'Dockerfile'),'utf8');
 assert.ok(docker.includes(`org.opencontainers.image.licenses="${licenseReference}"`));
 assert.ok(docker.includes(`COPY ${legalEntries.filter(filename=>filename!=='LICENSES').join(' ')} /usr/share/doc/wos/`));
 assert.ok(docker.includes('COPY LICENSES /usr/share/doc/wos/LICENSES'));
 assert.equal(JSON.parse(await readFile(path.join(root,'packages/wos-api/commands.openapi.json'),'utf8')).info.version,version);
 assert.equal((await readFile(path.join(root,'packages/wos-api/openapi.yaml'),'utf8')).match(/^  version: (.+)$/m)?.[1],version);
});
