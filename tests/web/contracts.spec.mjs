import {test,expect} from 'playwright/test';
import assert from 'node:assert/strict';
import {spawn} from 'node:child_process';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {resolve,join} from 'node:path';
import {createServer} from 'node:net';
import {once} from 'node:events';
import {randomUUID,randomBytes} from 'node:crypto';

test('explicit cutover, contract progress, human review and privileged revocation',async({browser})=>{
 const directory=await mkdtemp(join(tmpdir(),'wos-contract-ui-'));const socket=createServer();socket.listen(0,'127.0.0.1');await once(socket,'listening');const port=socket.address().port;await new Promise(done=>socket.close(done));
 const url=`http://127.0.0.1:${port}`,namespace='01a11990-0000-7000-8000-000000000001',token=randomBytes(32).toString('base64url');
 let logs='';const child=spawn(resolve('../../bin/wos'),['server'],{env:{...process.env,WOS_LISTEN:`127.0.0.1:${port}`,WOS_SQLITE_PATH:join(directory,'wos.db'),WOS_AUTH_MODE:'api_token',WOS_BOOTSTRAP_TOKEN:token,WOS_BOOTSTRAP_NAMESPACE_ID:namespace,WOS_BOOTSTRAP_NAMESPACE_NAME:'Contract UI',WOS_LOCAL_PRINCIPAL_ID:'human'},stdio:['ignore','ignore','pipe']});child.stderr.on('data',b=>logs=(logs+b.toString()).slice(-8000));
 const context=await browser.newContext(),page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 async function command(name,command){const response=await fetch(url+'/api/v1/commands/'+name,{method:'POST',headers:{Authorization:`Bearer ${token}`,'Content-Type':'application/json','Idempotency-Key':randomUUID()},body:JSON.stringify({command})});const data=await response.json();assert.ok(response.ok,JSON.stringify(data));return data.value}
 try {
  let ready=false;for(let i=0;i<100;i++){try{if((await fetch(url+'/readyz')).ok){ready=true;break}}catch{}await new Promise(done=>setTimeout(done,50))}assert.ok(ready,logs);
  const outcome=await command('create_outcome',{namespace_id:namespace,title:'Contratos verificados',desired_state:'Entrega revisada',priority:'normal'}),scope={namespace_id:namespace,outcome_id:outcome.id};
  await command('add_criterion',{owner:{...scope,kind:'outcome',id:outcome.id},expected_version:outcome.version,title:'Entrega aceita',required:true,verification_mode:'attestation'});
  await command('activate_outcome',{scope,expected_version:outcome.version+1});
  await command('set_namespace_work_protocol',{scope,expected_protocol_version:1,phase:'draining',reason:'Parar os servidores antigos'});
  await command('set_namespace_work_protocol',{scope,expected_protocol_version:2,phase:'contracts_v1',writers_drained:true,reason:'Migração validada'});
  const work=await command('create_work_item',{scope,title:'Implementar fluxo contratual',priority:'normal',lifecycle:'todo',execution_spec:{instructions:['Executar somente a tarefa contratada']}});
  const owner={...scope,kind:'work_item',id:work.id};
  await command('add_criterion',{owner,expected_version:work.version,title:'Fluxo aprovado',required:true,verification_mode:'attestation'});
  const acquired=await command('acquire_work_contract',{scope,work_item_id:work.id,expected_work_item_version:work.version+1,ttl_seconds:900});let contract=acquired.contract;
  const authority={execution_id:contract.execution_id,fencing_token:contract.fencing_token,spec_digest:contract.spec_digest};
  const synced=await command('sync_work_contract',{scope,contract_id:contract.id,authority,expected_contract_version:contract.version,checkpoint:{summary:'Implementação pronta para avaliação',next_action:'Revisão humana',completed:['Fluxo principal'],pending:['Aceite'],dirty:false}});contract=synced.contract;
  const submitted=await command('submit_work_result',{scope,contract_id:contract.id,authority,expected_contract_version:contract.version,material:{contract_id:contract.id,work_item_id:work.id,spec_digest:contract.spec_digest,summary:'Entrega imutável para revisão',artifacts:[],evidence_ids:[],criterion_evidence:[]}});
  await page.goto(url+'/app/');await page.getByLabel('Access credential',{exact:true}).fill(token);await page.getByRole('button',{name:'Sign in',exact:true}).click();await page.locator('#outcomes').getByRole('button').filter({hasText:'Contratos verificados'}).click();
  await page.getByRole('button',{name:'Board',exact:true}).click();await page.getByRole('button').filter({hasText:'Implementar fluxo contratual'}).first().click();
  await expect(page.getByText('Active reservation',{exact:true})).toBeVisible();await expect(page.getByText('Implementação pronta para avaliação',{exact:true})).toBeVisible();await expect(page.getByText('Entrega imutável para revisão',{exact:true})).toBeVisible();
  await page.getByRole('button',{name:'Review submission',exact:true}).click();await expect(page.locator('#fields')).toContainText('Reviewing submission:');await page.getByLabel('Rationale',{exact:true}).fill('Revisão humana explícita do material submetido');await page.locator('#submit-command').click();await expect(page.locator('#command-dialog')).not.toBeVisible();
  await page.getByRole('button').filter({hasText:'Implementar fluxo contratual'}).first().click();
  await page.locator('.contract-section').getByRole('button',{name:'Revoke contract',exact:true}).click();await page.getByLabel('Reason',{exact:true}).fill('Interromper execução após revisão');await page.locator('#submit-command').click();await expect(page.locator('#command-dialog')).not.toBeVisible();
  await page.getByRole('button').filter({hasText:'Implementar fluxo contratual'}).first().click();
  await page.getByText('Contract history',{exact:true}).click();await page.getByRole('button').filter({hasText:'revoked'}).click();await expect(page.getByText('Revoked contract',{exact:true})).toBeVisible();
  if(process.env.WOS_BROWSER_ARTIFACT_DIR)await page.screenshot({path:join(process.env.WOS_BROWSER_ARTIFACT_DIR,'09-contract-review.png'),fullPage:true});assert.deepEqual(errors,[]);
 }finally{await context.close();if(child.exitCode===null){const ended=once(child,'exit');child.kill('SIGTERM');await ended}await rm(directory,{recursive:true,force:true})}
});
