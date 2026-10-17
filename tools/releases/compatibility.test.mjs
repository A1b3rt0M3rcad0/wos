import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {root} from '../distribution/build.mjs';
import {legalEntries, licenseReference, npmLicense, verifyLegalFiles} from '../distribution/legal.mjs';

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

// Reject a removed old field/enum or new required input even after an explicit
// candidate fingerprint review. Byte/shape limits remain identical on old nodes.
function additiveSchema(before,after,where){
 assert.ok(after,where+' was removed');
 for(const [key,value] of Object.entries(before)){
  if(key==='properties'){
   for(const [name,child] of Object.entries(value))additiveSchema(child,after.properties?.[name],where+'.'+name);
  }else if(key==='required'){
   for(const name of after.required??[])assert.ok(value.includes(name),where+' newly requires '+name);
  }else if(key==='enum'){
   for(const item of value)assert.ok(after.enum?.includes(item),where+' removed enum '+item);
  }else if(['anyOf','oneOf','allOf'].includes(key)){
   assert.equal(after[key]?.length,value.length,where+'.'+key);
   value.forEach((child,index)=>additiveSchema(child,after[key][index],where+'.'+key+'['+index+']'));
  }else if(key==='items'&&typeof value==='object'){
   additiveSchema(value,after.items,where+'.items');
  }else assert.deepEqual(after[key],value,where+'.'+key+' changed existing constraint');
 }
 if(!before.required)assert.equal(after.required?.length??0,0,where+' acquired required inputs');
 for(const key of Object.keys(after))assert.ok(key in before||key==='required'&&(after.required?.length??0)===0,where+' added an existing-node constraint '+key);
}
// ADR 044 deliberately breaks addressing and new signed protocol fields. This
// projection is a review/comparison tool only: it must never transform requests,
// stored proofs or released fixture files. All unrelated shape checks still run.
const reviewedSignedPurposes = new Set([
 'contract-spec','contract-authority','work-return','review-return',
 'acceptance-receipt','key-enrollment','return-resolution','return-resolution-receipt',
]);
function tenantIdentifier(value){
 return value.replaceAll('Namespace','Tenant').replaceAll('namespace','tenant').replaceAll('NAMESPACE','TENANT');
}
function reviewedTenantString(value){
 const purpose=value.match(/^application\/vnd\.wos\.([a-z-]+)\.v2\+json$/);
 if(purpose&&reviewedSignedPurposes.has(purpose[1]))return `application/vnd.wos.${purpose[1]}.v3+json`;
 if(value==='signed_contracts_v2')return 'signed_contracts_v3';
 if(value==='draining_to_signed_v2')return 'draining_to_signed_v3';
 return tenantIdentifier(value);
}
function projectReviewedTenantContract(value,field){
 if(typeof value==='string')return reviewedTenantString(value);
 if(Array.isArray(value))return value.map(item=>projectReviewedTenantContract(item));
 if(!value||typeof value!=='object')return value;
 const projected={};
 for(const [name,child] of Object.entries(value)){
  const next=tenantIdentifier(name);
  assert.ok(!(next in projected),'reviewed Tenant projection cannot collapse two fields: '+next);
  projected[next]=projectReviewedTenantContract(child,name);
 }
 // Only the signed protocol discriminant changes. Local workspace schema 2,
 // expected protocol CAS versions and every other numeric bound stay intact.
 if(field==='protocol_version'&&value.const===2)projected.const=3;
 return projected;
}

test('Tenant review projection still rejects unrelated removed fields and new constraints',()=>{
 const before={type:'object',additionalProperties:false,properties:{
  namespace_id:{type:'string',minLength:1},
  retained_field:{type:'string',maxLength:64},
  protocol_version:{type:'integer',const:2},
  workspace_schema:{type:'integer',const:2},
 },required:['namespace_id','retained_field','protocol_version']};
 const expected=projectReviewedTenantContract(before);
 assert.equal(expected.properties.protocol_version.const,3);
 assert.equal(expected.properties.workspace_schema.const,2);
 assert.deepEqual(expected.required,['tenant_id','retained_field','protocol_version']);
 assert.equal(projectReviewedTenantContract('application/vnd.wos.work-return.v2+json'),'application/vnd.wos.work-return.v3+json');
 assert.equal(projectReviewedTenantContract('application/vnd.unrelated.work-return.v2+json'),'application/vnd.unrelated.work-return.v2+json');
 assert.equal(projectReviewedTenantContract('application/vnd.wos.unreviewed.v2+json'),'application/vnd.wos.unreviewed.v2+json');
 additiveSchema(expected,structuredClone(expected),'reviewed fixture');
 const missing=structuredClone(expected);delete missing.properties.retained_field;
 assert.throws(()=>additiveSchema(expected,missing,'reviewed fixture'),/retained_field was removed/);
 const required=structuredClone(expected);required.properties.extra={type:'string'};required.required.push('extra');
 assert.throws(()=>additiveSchema(expected,required,'reviewed fixture'),/newly requires extra/);
 const tighter=structuredClone(expected);tighter.properties.retained_field.maxLength=32;
 assert.throws(()=>additiveSchema(expected,tighter,'reviewed fixture'),/changed existing constraint/);
 const newConstraint=structuredClone(expected);newConstraint.properties.retained_field.pattern='^[a-z]+$';
 assert.throws(()=>additiveSchema(expected,newConstraint,'reviewed fixture'),/added an existing-node constraint/);
 const removedEnum={type:'string',enum:['kept','removed']};
 assert.throws(()=>additiveSchema(removedEnum,{type:'string',enum:['kept']},'reviewed enum'),/removed enum removed/);
});

test('released v0.4.2 shapes retain unrelated invariants after the explicit Tenant major projection',async()=>{
 const directory=path.join(root,'docs/stable/contracts/v0.4.2');
 const historicalBytes=await readFile(path.join(directory,'public-contract.json'));
 assert.equal(digest(historicalBytes),'661320a06ff5d510bfabe0973bdc8ee4a9b1ceddf2b81be5ef62b569518a5266');
 const historical=JSON.parse(historicalBytes);
 for(const filename of ['packages/wos-cli/schemas/profile-v2.schema.json','packages/wos-cli/schemas/contract-v2.schema.json']){
  const bytes=await readFile(path.join(directory,path.basename(filename)));
  assert.equal(digest(normalize(filename,bytes)),historical.files[filename]);
  additiveSchema(projectReviewedTenantContract(JSON.parse(bytes)),JSON.parse(await readFile(path.join(root,filename))),filename);
 }
 const filename='packages/wos-api/commands.openapi.json';
 const bytes=await readFile(path.join(directory,'commands.openapi.json'));
 assert.equal(digest(normalize(filename,bytes)),historical.files[filename]);
 const before=projectReviewedTenantContract(JSON.parse(bytes)),after=JSON.parse(await readFile(path.join(root,filename)));
 for(const [name,schema] of Object.entries(before.components.schemas))additiveSchema(schema,after.components.schemas[name],name);
 for(const [route,operations] of Object.entries(before.paths)){
  assert.ok(after.paths[route],route+' removed after reviewed Tenant addressing projection');
  if(route==='/api/v1/security/signing-identity'){
   assert.deepEqual(after.paths[route].get.parameters,operations.get.parameters);
   assert.deepEqual(Object.keys(after.paths[route].get.responses),Object.keys(operations.get.responses));
  }else if(route==='/api/v1/tenants/{tenant_id}/outcomes/{outcome_id}/signed-state/{resource}'){
   // The pre-existing central-workflow review allows only these optional filters.
   // Preserve every projected old path parameter, response and annotation.
   const current=structuredClone(after.paths[route]);
   assert.deepEqual(current.get.parameters.slice(0,operations.get.parameters.length),operations.get.parameters);
   const additions=current.get.parameters.splice(operations.get.parameters.length);
   const optionalStrings=['holder_principal_id','work_item_id','status','cursor'].map(name=>({description:'Optional executions discovery filter or continuation cursor.',in:'query',name,schema:{type:'string'}}));
   assert.deepEqual(additions,[...optionalStrings,{in:'query',name:'limit',schema:{default:25,maximum:100,minimum:1,type:'integer'}}]);
   assert.deepEqual(current,operations,route+' changed an existing wire property');
  }else assert.deepEqual(after.paths[route],operations,route+' changed an unrelated wire operation');
 }
 const change=JSON.parse(await readFile(path.join(root,'.changes/tenant-schema-isolation.json'),'utf8'));
 assert.equal(change.bump,'major','Tenant addressing and signed protocol changes require a major release');
 const review=JSON.parse(await readFile(path.join(root,'docs/stable/public-contract.json'),'utf8'));
 assert.equal(review.compatibility_review,'docs/stable/tenant-contract-review.md');
 assert.equal(review.contract_transition,'tenant-schema-protocol-v3');
 assert.equal(review.architecture_decision,'docs/adr/0044-tenant-schemas-and-protocol-cutover.md');
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
