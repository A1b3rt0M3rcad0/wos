import {test,expect,openOutcome} from './ux-fixture.mjs';
const board=page=>page.locator('#workspace-kanban');
const operation=page=>page.locator('#workspace-kanban-operation');
async function open(page,wos,outcome,title){
 await wos.login(page);await openOutcome(page,outcome.title,{area:null});
 await page.getByRole('combobox',{name:'Search Outcome Board',exact:true}).fill(title);
 await page.getByRole('option').filter({hasText:title}).first().click();
 await page.locator('#canvas-details').getByRole('button',{name:title===outcome.title?'Open Objectives Kanban':'Open Tasks Kanban',exact:true}).click();
 await expect(board(page)).toBeVisible();await expect(board(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');
}
async function setup(wos){
 const {value:outcome,scope}=await wos.outcome('Contextual operational board');
 await wos.command('add_criterion',{owner:{...scope,kind:'outcome',id:outcome.id},expected_version:outcome.version,title:'A verified outcome',required:true,verification_mode:'attestation'});
 await wos.command('activate_outcome',{scope,expected_version:outcome.version+1});
 const objective=await wos.command('create_objective',{scope,title:'A verifiable objective',priority:'normal',required_for_outcome:false});
 const task=await wos.command('create_work_item',{scope,title:'Prepare a reviewed delivery',priority:'normal',lifecycle:'backlog',objective_id:objective.id});
 return {outcome,scope,objective,task};
}
async function move(page,title,lane){
 const card=board(page).locator('.workspace-kanban-card-shell').filter({has:page.getByRole('button',{name:new RegExp(title)})});
 await card.locator('summary').click();await card.getByRole('button',{name:`Move to ${lane}`,exact:true}).click();
 await expect(operation(page)).toBeVisible();
}
for(const size of [{width:1280,height:720},{width:2560,height:1440},{width:390,height:844}]){
 test(`Kanban is a workspace with persistent lanes and one actual operation modal at ${size.width}`,async({page,wos})=>{
  await page.setViewportSize(size);const {outcome,objective,task}=await setup(wos);
  await open(page,wos,outcome,objective.title);
  const camera=await page.locator('.canvas-world').getAttribute('style');
  await expect(board(page)).not.toHaveJSProperty('tagName','DIALOG');
  await expect(page.locator('dialog[open]')).toHaveCount(0);
  await expect(board(page).locator('[data-column]')).toHaveCount(5);
  await expect(board(page).locator('[data-column=backlog]')).toContainText(task.title);
  await expect(board(page).locator('.workspace-kanban-move select')).toHaveCount(0);
  const mountedLane=await board(page).locator('[data-column=backlog]').elementHandle();
  await board(page).locator('.workspace-kanban-card').filter({hasText:task.title}).click();
  expect(await mountedLane.evaluate(node=>node.isConnected),'Inspection retains the same mounted lane node').toBe(true);
  await expect(board(page).locator('.workspace-kanban-columns')).toBeVisible();
  await expect(board(page).getByRole('complementary',{name:'Item inspector'})).toBeVisible();
  await expect(page.locator('dialog[open]')).toHaveCount(0);
  if(size.width>1600){
   const a=await board(page).locator('.workspace-kanban-inspection').boundingBox();const b=await board(page).locator('.workspace-kanban-columns').boundingBox();expect(a.x).toBeGreaterThanOrEqual(b.x+b.width-1);
  }
  await page.keyboard.press('Escape');await expect(board(page).locator('.workspace-kanban-inspection')).toHaveCount(0);
  await move(page,task.title,'To do');
  await expect(operation(page).locator('#command-select')).toHaveValue('activate_work_item');
  await expect(page.locator('dialog[open]')).toHaveCount(1);
  await expect(board(page).locator('[data-column=backlog]')).toContainText(task.title);
  await expect(board(page)).toHaveJSProperty('inert',false);
  const response=page.waitForResponse(r=>r.url().endsWith('/commands/activate_work_item'));
  if(await operation(page).getByLabel('Reason',{exact:true}).count()) await operation(page).getByLabel('Reason',{exact:true}).fill('Confirm the requested transition against the current state.');
  await operation(page).locator('#submit-command').click();expect((await response).ok()).toBe(true);
  await expect(operation(page)).not.toBeVisible();
  await expect(board(page).locator('[data-column=todo]')).toContainText(task.title);
  await page.keyboard.press('Escape');await board(page).getByRole('button',{name:'Back to Canvas',exact:true}).click();
  await expect(board(page)).not.toBeVisible();
  await expect(page.locator('.canvas-world')).toHaveAttribute('style',camera);
 });
}
test('Backlog to In progress requires two explicit confirmed commands and a fresh projection',async({page,wos})=>{
 const {outcome,objective,task,scope}=await setup(wos);
 await wos.command('add_criterion',{owner:{...scope,kind:'work_item',id:task.id},expected_version:task.version,title:'A current Task condition',required:true,verification_mode:'attestation'});
 await open(page,wos,outcome,objective.title);
 const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url().split('/').at(-1));});
 await move(page,task.title,'In progress');
 await expect(operation(page)).toContainText('Two explicit steps');expect(writes).toEqual([]);
 await operation(page).getByRole('button',{name:'Move task to do',exact:true}).click();
 await expect(operation(page).locator('#command-select')).toHaveValue('activate_work_item');
 if(await operation(page).getByLabel('Reason',{exact:true}).count()) await operation(page).getByLabel('Reason',{exact:true}).fill('Confirm the requested transition against the current state.');
  await operation(page).locator('#submit-command').click();
 await expect(operation(page).locator('#command-select')).toHaveValue('claim_work_item');
 expect(writes).toEqual(['activate_work_item']);
 const base=`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`;
 expect((await wos.api(base)).value.lifecycle).toBe('todo');
 if(await operation(page).getByLabel('Reason',{exact:true}).count()) await operation(page).getByLabel('Reason',{exact:true}).fill('Confirm the requested transition against the current state.');
  await operation(page).locator('#submit-command').click();
 await expect(operation(page)).not.toBeVisible();
 await expect(board(page).locator('[data-column=in_progress]')).toContainText(task.title);
 expect(writes).toEqual(['activate_work_item','claim_work_item']);expect((await wos.api(base)).value.current_lease.principal_id).toBe('owner');
});
test('Objectives use planned, started and achieved lanes without a fabricated review queue',async({page,wos})=>{
 const {outcome,objective}=await setup(wos);await open(page,wos,outcome,outcome.title);
 await expect(board(page).locator('[data-column]')).toHaveCount(3);
 await expect(board(page).locator('[data-column=planned]')).toContainText(objective.title);
 await expect(board(page).locator('[data-column=verification]')).toHaveCount(0);
 await board(page).locator('.workspace-card-menu summary').click();
 await expect(board(page).getByRole('button',{name:'Move to Achieved',exact:true})).toBeVisible();
});
test('an eligible planned Objective is achieved directly without an invented Start command',async({page,wos})=>{
 const {outcome,objective,scope}=await setup(wos),owner={...scope,kind:'objective',id:objective.id};
 const criterion=await wos.command('add_criterion',{owner,expected_version:objective.version,title:'An inspected condition',required:true,verification_mode:'attestation'});
 await wos.command('record_criterion_assessment',{owner,criterion_id:criterion.id,criterion_revision:criterion.criterion_revision,expected_version:objective.version+1,result:'met',rationale:'The current condition was explicitly inspected.'});
 await open(page,wos,outcome,outcome.title);const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url().split('/').at(-1));});
 await board(page).locator('.workspace-kanban-card').filter({hasText:objective.title}).click();
 await expect(board(page).locator('.workspace-next-operation [data-action-id=achieve_objective]')).toBeVisible();
 await board(page).getByRole('button',{name:'Back to Kanban',exact:true}).click();
 await move(page,objective.title,'Achieved');
 await expect(operation(page).locator('#command-select')).toHaveValue('achieve_objective');expect(writes).toEqual([]);
 await operation(page).getByLabel('I understand that this open work is not completed by this conclusion',{exact:true}).check();
 await operation(page).getByLabel('Reason',{exact:true}).fill('Accept the inspected planned condition explicitly.');
 await operation(page).locator('#submit-command').click();
 await expect(operation(page)).not.toBeVisible();
 await expect(board(page).locator('[data-column=accepted]')).toContainText(objective.title);
 expect(writes).toEqual(['achieve_objective']);expect((await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}`)).value.lifecycle).toBe('active');
});
test('native Backlog drag prepares activation without writing and the real modal retains keyboard focus',async({page,wos})=>{
 const {outcome,objective,task}=await setup(wos);await open(page,wos,outcome,objective.title);
 const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url());});
 await board(page).locator('.workspace-kanban-card').filter({hasText:task.title}).dragTo(board(page).locator('[data-column=todo]'));
 await expect(operation(page).locator('#command-select')).toHaveValue('activate_work_item');expect(writes).toEqual([]);
 for(let i=0;i<12;i++){await page.keyboard.press('Tab');expect(await operation(page).evaluate(node=>node.contains(document.activeElement))).toBe(true);}
 await page.keyboard.press('Escape');await expect(operation(page)).not.toBeVisible();
 await expect(board(page).locator('[data-column=backlog]')).toContainText(task.title);expect(writes).toEqual([]);
 await expect(board(page).locator('.workspace-kanban-card')).toBeFocused();
});
test('a confirmed command with a failed refresh retries reads without duplicating the write',async({page,wos})=>{
 const {outcome,objective,task}=await setup(wos);await open(page,wos,outcome,objective.title);
 let committed=false,failed=false;const writes=[];
 await page.route('**/commands/activate_work_item',async route=>{
  const response=await route.fetch();expect(response.ok()).toBe(true);committed=true;writes.push(route.request().postData());await route.fulfill({response});
 });
 await page.route('**/continuity?limit=25',async route=>{
  if(committed&&!failed){failed=true;await route.abort('failed');}else await route.continue();
 });
 await move(page,task.title,'To do');await operation(page).getByLabel('Reason',{exact:true}).fill('Confirm availability before the interrupted read.');
 await operation(page).locator('#submit-command').click();
 await expect(operation(page)).not.toBeVisible();await expect(board(page).locator('[data-column=todo]')).toContainText(task.title);
 expect(failed).toBe(true);expect(writes).toHaveLength(1);
 await board(page).getByRole('button',{name:'Refresh',exact:true}).click();
 await expect(board(page).locator('[data-column=todo]')).toContainText(task.title);expect(writes).toHaveLength(1);
});
test('a contract v1 submission does not invent a signed review fact or a Verification lifecycle',async({page,wos})=>{
 const {outcome,objective,scope}=await setup(wos);
 await wos.command('set_namespace_work_protocol',{scope,expected_protocol_version:1,phase:'draining',reason:'Drain the legacy acquisition protocol.'});
 await wos.command('set_namespace_work_protocol',{scope,expected_protocol_version:2,phase:'contracts_v1',writers_drained:true,reason:'Use the existing contract host.'});
 const task=await wos.command('create_work_item',{scope,title:'An actual delivery awaiting review',priority:'normal',lifecycle:'todo',objective_id:objective.id,execution_spec:{instructions:['Produce a reviewable delivery']}});
 const criterion=await wos.command('add_criterion',{owner:{...scope,kind:'work_item',id:task.id},expected_version:task.version,title:'An inspected current criterion',required:true,verification_mode:'attestation'});
 const acquired=await wos.command('acquire_work_contract',{scope,work_item_id:task.id,expected_work_item_version:task.version+1,ttl_seconds:900}),contract=acquired.contract;
 await wos.command('submit_work_result',{scope,contract_id:contract.id,expected_contract_version:contract.version,authority:{execution_id:contract.execution_id,fencing_token:contract.fencing_token,spec_digest:contract.spec_digest},material:{contract_id:contract.id,work_item_id:task.id,spec_digest:contract.spec_digest,summary:'A delivery for independent review',artifacts:[],evidence_ids:[],criterion_evidence:[]}});
 await open(page,wos,outcome,objective.title);
 await expect(board(page).locator('[data-column=verification]')).not.toContainText(task.title);
 await expect(board(page).locator('[data-column=in_progress]')).toContainText(task.title);
 await expect(board(page).locator('[data-column=accepted]')).not.toContainText(task.title);
 const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url());});
 await board(page).locator('.workspace-kanban-card').filter({hasText:task.title}).click();
 await expect(board(page).locator('.workspace-kanban-columns')).toBeVisible();expect(writes).toEqual([]);
});
test('blocked achievement can add its criterion and assessment in context before explicit acceptance',async({page,wos})=>{
 const {outcome,objective}=await setup(wos);await open(page,wos,outcome,outcome.title);
 const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url().split('/').at(-1));});
 await move(page,objective.title,'Achieved');
 await expect(operation(page)).toContainText('No state changed');expect(writes).toEqual([]);
 await operation(page).locator('.workspace-transition-remedies [data-action-id=add_criterion]').click();
 await operation(page).getByLabel('What must be true to consider this item satisfied?',{exact:true}).fill('The current condition was checked');
 await operation(page).getByLabel('How will it be verified?',{exact:true}).selectOption('attestation');
 await operation(page).locator('#submit-command').click();
 await expect(operation(page).locator('.workspace-transition-body')).toBeVisible();
 await operation(page).getByRole('button',{name:'The current condition was checked',exact:true}).click();
 await operation(page).locator('.workspace-transition-remedies [data-action-id=record_criterion_assessment]').click();
 await operation(page).getByLabel('Rationale',{exact:true}).fill('The current definition was inspected before explicit acceptance.');
 await operation(page).locator('#submit-command').click();
 await expect(operation(page).locator('#command-select')).toHaveValue('achieve_objective');
 expect(writes).toEqual(['add_criterion','record_criterion_assessment']);
 await operation(page).getByLabel('I understand that this open work is not completed by this conclusion',{exact:true}).check();
 await operation(page).getByLabel('Reason',{exact:true}).fill('Explicitly accept the inspected condition; child work keeps its own lifecycle.');
 await operation(page).locator('#submit-command').click();
 await expect(operation(page)).not.toBeVisible();await expect(board(page).locator('[data-column=accepted]')).toContainText(objective.title);
 expect(writes).toEqual(['add_criterion','record_criterion_assessment','achieve_objective']);
});
test('emulated touch can use the card action menu without drag or hover',async({browser,wos})=>{
 const context=await browser.newContext({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});
 try{
  const page=await context.newPage(),{outcome,objective,task}=await setup(wos);await open(page,wos,outcome,objective.title);
  const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url());});
  await board(page).locator('.workspace-card-menu summary').tap();
  await board(page).getByRole('button',{name:'Move to To do',exact:true}).tap();
  await expect(operation(page).locator('#command-select')).toHaveValue('activate_work_item');expect(writes).toEqual([]);
  await operation(page).getByRole('button',{name:'Back to item',exact:true}).tap();
  await expect(operation(page)).not.toBeVisible();await expect(board(page).locator('[data-column=backlog]')).toContainText(task.title);
 }finally{await context.close();}
});
