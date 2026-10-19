import {test, expect, openOutcome} from './ux-fixture.mjs';
import {revealMenuAction} from './workspace-action-fixture.mjs';
async function select(page,title){
 await page.getByRole('combobox',{name:'Search Outcome Board',exact:true}).fill(title);
 await page.getByRole('option').filter({hasText:title}).first().click();
 await expect(page.locator('#canvas-details #detail-title')).toHaveText(title);
}
test('strict cancellation acknowledgement in the browser persists its exact excluded scope',async({page,wos})=>{
 const outcome=await wos.command('create_outcome',{namespace_id:wos.namespace,title:'Accept the agreed scope',desired_state:'No unacknowledged cancelled work',priority:'normal',completion_policy:'strict_work'});
 const scope={namespace_id:wos.namespace,outcome_id:outcome.id}, owner={...scope,kind:'outcome',id:outcome.id};
 const base=`/namespaces/${wos.namespace}/outcomes/${outcome.id}`,read=async()=>(await wos.api(base)).value;
 const criterion=await wos.command('add_criterion',{owner,expected_version:outcome.version,title:'The agreed result was checked',required:true,verification_mode:'attestation'});
 let current=await read();
 await wos.command('attest_criterion',{owner,expected_version:current.version,criterion_id:criterion.id,criterion_revision:criterion.criterion_revision,result:'met',rationale:'The final agreed result was independently checked.'});
 current=await read();await wos.command('activate_outcome',{scope,expected_version:current.version});
 const task=await wos.command('create_work_item',{scope,title:'Explicitly removed experiment',priority:'normal',lifecycle:'backlog'});
 await wos.command('cancel_work_item',{scope,work_item_id:task.id,expected_version:task.version,reason:'The owner removed this experiment from the agreed result.'});
 await wos.login(page);await openOutcome(page,outcome.title,{area:null});await select(page,outcome.title);
 await page.locator('#canvas-details #entity-actions').click();
 const action=page.locator('#action-options [data-action-id="achieve_outcome"]');await revealMenuAction(action);await action.click();
 await expect(page.locator('.workspace-conclusion-impact')).toContainText('Strict work');
 await expect(page.locator('.workspace-conclusion-impact')).toContainText(task.title);
 const acknowledgement=page.getByLabel('I reviewed these cancellations and accept their exclusion from the result.',{exact:true});
 await expect(acknowledgement).not.toBeChecked();
 await page.getByLabel('Reason',{exact:true}).fill('Accepted after checking the result and explicitly reviewing the cancelled scope.');
 const attempts=[];page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/commands/achieve_outcome'))attempts.push(r.postDataJSON().command);});
 await page.locator('#submit-command').click();await expect(page.locator('#command-form')).toBeVisible();expect(attempts).toHaveLength(0);expect((await read()).lifecycle).toBe('active');
 await acknowledgement.check();
 const accepted=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/commands/achieve_outcome'));
 await page.locator('#submit-command').click();const response=await accepted;expect(response.ok(),JSON.stringify(await response.json())).toBe(true);
 await expect(page.locator('#command-form')).not.toBeVisible();
 const concluded=await read(),ref={...scope,kind:'work_item',id:task.id};
 expect(concluded.lifecycle).toBe('achieved');expect(concluded.conclusion.obligations.acknowledged_cancelled_work).toEqual([ref]);
 expect(attempts).toHaveLength(1);expect(attempts[0].acknowledged_cancelled_work).toEqual([ref]);expect(typeof attempts[0].observed_outcome_revision).toBe('string');
 expect((await wos.api(`${base}/work-items/${task.id}`)).value.lifecycle).toBe('cancelled');
});
test('real credential revocation closes an open Kanban and clears its private content',async({page,wos})=>{
 const {value:outcome,scope}=await wos.outcome('Private planning');await wos.command('create_objective',{scope,title:'Private Objective',priority:'normal',required_for_outcome:false});
 let admin=await wos.api(`/namespaces/${wos.namespace}/administration`);
 const grant=await wos.api('/security/commands',{namespace_id:wos.namespace,expected_namespace_version:admin.namespace_version,operation:'set_grant',principal_id:'kanban-reader',permissions:['state:read']});
 const issued=await wos.api('/security/commands',{namespace_id:wos.namespace,expected_namespace_version:grant.result.namespace_version,operation:'issue_credential',principal_id:'kanban-reader',actor_ref:{kind:'human',provider:'ux-test',id:'kanban-reader'},expires_at:new Date(Date.now()+3600000).toISOString()});
 await wos.login(page,issued.token);await openOutcome(page,outcome.title,{area:null});await select(page,outcome.title);
 await page.locator('#canvas-details').getByRole('button',{name:'Open Objectives Kanban',exact:true}).click();
 const modal=page.locator('#workspace-kanban');await expect(modal.getByRole('button',{name:/Private Objective/})).toBeVisible();
 admin=await wos.api(`/namespaces/${wos.namespace}/administration`);const credential=admin.credentials.find(c=>c.principal_id==='kanban-reader');
 await wos.api('/security/commands',{namespace_id:wos.namespace,expected_namespace_version:admin.namespace_version,operation:'revoke_credential',credential_id:credential.id});
 await modal.getByRole('button',{name:'Refresh',exact:true}).click();
 await expect(modal).not.toBeVisible();await expect(page.locator('#outcome')).toBeHidden();await expect(page.locator('#outcomes')).toBeEmpty();await expect(page.locator('#canvas-details')).not.toBeVisible();await expect(modal).not.toContainText('Private Objective');
});
