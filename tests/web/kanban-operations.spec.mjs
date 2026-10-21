import {test,expect,openOutcome} from './ux-fixture.mjs';
import {seedCanvas,provisionCanvasViewer} from './canvas-fixture.mjs';
import {mkdir,writeFile} from 'node:fs/promises';
import {execFileSync} from 'node:child_process';
import AxeBuilder from '@axe-core/playwright';
async function capture(page,name){
 if(!process.env.WOS_CAPTURE_OPERATIONAL_KANBAN)return;
 const dir=process.env.WOS_CAPTURE_OPERATIONAL_KANBAN;await mkdir(dir,{recursive:true});await page.screenshot({path:`${dir}/${name}.png`});
 await writeFile(`${dir}/${name}.json`,JSON.stringify({source:execFileSync('git',['rev-parse','HEAD'],{encoding:'utf8'}).trim(),workingTree:execFileSync('git',['status','--porcelain','--','packages','tests/web'],{encoding:'utf8'}),viewport:page.viewportSize(),browser:page.context().browser().version()},null,2));
}
const board=page=>page.locator('#workspace-kanban');
const modal=page=>page.locator('body:has(#workspace-kanban-operation[open]) #workspace-kanban-operation, body:not(:has(#workspace-kanban-operation[open])) #workspace-kanban');
async function move(page,title,stage){
 const card=board(page).locator('.workspace-kanban-card').filter({hasText:title});
 await card.dragTo(board(page).locator(`[data-column="${stage}"]`));
}
async function open(page,wos,outcome,title){
 await wos.login(page);await openOutcome(page,outcome.title,{area:null});
 await page.getByRole('combobox',{name:'Search Outcome Board',exact:true}).fill(title);
 await page.getByRole('option').filter({hasText:title}).first().click();
 await page.locator('#canvas-details').getByRole('button',{name:title===outcome.title?'Open Objectives Kanban':'Open Tasks Kanban',exact:true}).click();
}
async function action(page,name){
 if(!(await page.locator("#workspace-kanban-operation").evaluate(node=>node.open))) await expect(board(page).locator(".workspace-kanban-inspection")).toBeVisible();
 const target=modal(page).locator(`[data-action-id="${name}"]`).first();
 for(const group of await target.locator('xpath=ancestor::details').all()) if(!(await group.evaluate(n=>n.open))) await group.locator(':scope > summary').click();
 await modal(page).locator(`[data-action-id="${name}"]`).filter({visible:true}).first().click();
 await expect(modal(page).locator('#command-form')).toBeVisible();
 await expect(page.locator('dialog[open]')).toHaveCount(1);
}
async function save(page,name,{nextCommand,proof=false}={}){
 const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/commands/'+name));
 await page.locator('#submit-command').click();const r=await response;const json=await r.json();expect(r.ok(),JSON.stringify(json)).toBe(true);
 if(proof){await expect(modal(page).getByRole('heading',{name:'Review proof',exact:true})).toBeVisible();return json.value;}
 if(nextCommand){await expect(modal(page).locator('#command-select')).toHaveValue(nextCommand);return json.value;}
 await expect(modal(page).locator('#command-form')).toHaveCount(0);
 await expect(modal(page)).toBeVisible();await expect(modal(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');return json.value;
}
async function fixture(wos,{startObjective=true}={}){
 const {value:outcome,scope}=await wos.outcome('An operational result');
 await wos.command('add_criterion',{owner:{...scope,kind:'outcome',id:outcome.id},expected_version:outcome.version,title:'A checked delivery',required:true,verification_mode:'attestation'});
 await wos.command('activate_outcome',{scope,expected_version:outcome.version+1});
 let objective=await wos.command('create_objective',{scope,title:'A reviewed installation guide',priority:'normal',required_for_outcome:false});
 if(startObjective){
  await wos.command('add_criterion',{owner:{...scope,kind:'objective',id:objective.id},expected_version:objective.version,title:'The installation guide has a verifiable success condition',required:true,verification_mode:'attestation'});
  objective=(await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/objectives/${objective.id}`)).value;
  objective=await wos.command('start_objective',{scope,objective_id:objective.id,expected_version:objective.version});
 }
 return {outcome,scope,objective};
}
async function startCanvasObjectives(wos,canvas){
 const base=`/namespaces/${wos.namespace}/outcomes/${canvas.outcome.id}`;
 for(const objective of [canvas.objective,canvas.child]){
  let current=(await wos.api(`${base}/objectives/${objective.id}`)).value;
  await wos.command('add_criterion',{owner:{...canvas.scope,kind:'objective',id:objective.id},expected_version:current.version,title:`${objective.title} has an explicit success condition`,required:true,verification_mode:'attestation'});
  current=(await wos.api(`${base}/objectives/${objective.id}`)).value;
  await wos.command('start_objective',{scope:canvas.scope,objective_id:objective.id,expected_version:current.version});
 }
}
for(const viewport of [{width:1280,height:720},{width:2560,height:1440},{width:3840,height:2160},{width:390,height:844}]){
 test(`create, assign, acquire, prove, assess and conclude in one Kanban at ${viewport.width}px`,async({page,wos})=>{
  await page.setViewportSize(viewport);const {outcome,scope,objective}=await fixture(wos);
  const base=`/namespaces/${wos.namespace}/outcomes/${outcome.id}`;
  await open(page,wos,outcome,objective.title);
  const addBacklogTask=modal(page).locator('button[data-target-lifecycle="backlog"]');
  const addTodoTask=modal(page).locator('button[data-target-lifecycle="todo"]');
  await expect(addBacklogTask).toBeVisible();
  await expect(addTodoTask).toBeVisible();
  await expect(modal(page).locator('.workspace-kanban-card-shell select')).toHaveCount(0);
  await addBacklogTask.click();
  await expect(modal(page).locator('#command-form')).toBeVisible();
  await page.getByLabel('Task title',{exact:true}).fill('Verify installation commands');
  const task=await save(page,'create_work_item');expect(task.lifecycle).toBe('backlog');expect(task.objective_id).toBe(objective.id);
  await modal(page).getByRole('button',{name:/Verify installation commands/}).click();
  await expect(modal(page).locator('.workspace-kanban-columns')).toBeVisible();
  const inspector=modal(page).locator('.workspace-kanban-inspection');
  const inspectorHeader=inspector.locator('.workspace-kanban-inspection-header');
  await inspector.locator('[data-operational-group=people] > summary').click();
  await inspector.evaluate(node=>{node.scrollTop=node.scrollHeight;});
  const headerBounds=await inspectorHeader.boundingBox(),inspectorBounds=await inspector.boundingBox();
  expect(headerBounds.y).toBeGreaterThanOrEqual(inspectorBounds.y-1);
  expect(headerBounds.y).toBeLessThan(inspectorBounds.y+50);
  await expect(inspectorHeader.getByRole('button',{name:'Back to Kanban',exact:true})).toBeVisible();
  await expect(modal(page).getByRole('heading',{name:'Responsibility and execution authority',exact:true})).toBeVisible();
  await capture(page,`item-${viewport.width}`);
  await action(page,'set_work_item_assignees');
  await page.getByRole('button',{name:'Add participant',exact:true}).click();
  await page.getByLabel('Participant type',{exact:true}).selectOption('human');
  await page.getByLabel('Participant provider',{exact:true}).fill('team');
  await page.getByLabel('Participant identity',{exact:true}).fill('guide-owner');
  await save(page,'set_work_item_assignees');
  let record=(await wos.api(`${base}/work-items/${task.id}`)).value;
  expect(record.assignee_refs[0].id).toBe('guide-owner');expect(record.current_lease).toBeUndefined();
  await action(page,'set_work_item_execution_spec');
  for(const [label,value] of [['Instructions','Run the installation commands in an isolated environment'],['Deliverables','A reviewable installation output']]){
   const group=page.getByRole('group',{name:label,exact:true});await group.getByRole('button',{name:`Add ${label.toLowerCase()} item`,exact:true}).click();await group.getByLabel(`${label} item`,{exact:true}).last().fill(value);
  }
  await save(page,'set_work_item_execution_spec');
  await expect(modal(page).locator('[data-workspace-facet=work]')).toContainText('Run the installation commands');
  await action(page,'add_criterion');
  await page.getByLabel('What must be true to consider this item satisfied?',{exact:true}).fill('The documented command succeeds');
  await page.getByLabel('How will it be verified?',{exact:true}).selectOption('evidence_review');
  const criterion=await save(page,'add_criterion');
  await action(page,'activate_work_item');await page.getByLabel('Reason',{exact:true}).fill('The bounded guide check is ready to acquire.');await save(page,'activate_work_item');
  await action(page,'claim_work_item');const claimed=await save(page,'claim_work_item');expect(claimed.current_lease.principal_id).toBe('owner');
  await expect(modal(page).locator('.workspace-kanban-controls')).toBeVisible();
  await expect(modal(page).getByRole('button',{name:'Review acceptance criteria',exact:true})).toBeVisible();
  await expect(modal(page).locator('button[data-target-lifecycle="backlog"]')).toBeVisible();
  await action(page,'renew_work_item_lease');const renewed=await save(page,'renew_work_item_lease');expect(renewed.current_lease.claim_id).toBe(claimed.current_lease.claim_id);expect(renewed.current_lease.fencing_token).toBe(claimed.current_lease.fencing_token);
  await capture(page,`executing-${viewport.width}`);
  await modal(page).getByRole('button',{name:'Review acceptance criteria',exact:true}).click();
  await modal(page).getByRole('button',{name:'Add artifact',exact:true}).click();
  await page.getByLabel('Relationship to this item',{exact:true}).selectOption('produces');
  await page.getByLabel('Artifact name',{exact:true}).fill('Installation output');await page.getByLabel('Artifact type',{exact:true}).fill('test_log');await page.getByLabel('URL',{exact:true}).fill('https://example.test/build/42');
  await save(page,'register_artifact_in_context',{proof:true});
  await modal(page).getByRole('button',{name:'Add or link evidence',exact:true}).click();
  await page.getByLabel('Evidence source',{exact:true}).selectOption('observation');await page.getByLabel('Description',{exact:true}).fill('The documented installation completed in the isolated environment.');await page.getByLabel('Why is this relevant?',{exact:true}).fill('This records the criterion-specific command observation.');
  const proof=await save(page,'register_evidence_with_links',{proof:true});
  record=(await wos.api(`${base}/work-items/${task.id}`)).value;expect(record.lifecycle).toBe('in_progress');expect(record.criteria.current_assessments||{}).toEqual({});
  await modal(page).getByRole('checkbox',{name:/The documented command succeeds/}).click();
  await page.getByLabel('Rationale',{exact:true}).fill('Reviewed the actual output and the current criterion.');
  const picker=page.getByRole('combobox',{name:'Evidence used',exact:true});await picker.fill('documented installation');
  await page.getByRole('option').filter({hasText:'documented installation'}).first().click();
  const assessment=await save(page,'record_criterion_assessment',{proof:true});expect(assessment.result).toBe('met');expect(assessment.evidence_ids).toContain(proof.evidence.id);
  await modal(page).getByRole('button',{name:'Back to item',exact:true}).click();
  await expect(board(page).locator('.workspace-kanban-inspection-header > h3')).toHaveText(task.title);
  await action(page,'complete_work_item');await page.getByLabel('Task result',{exact:true}).fill('The installation guide was exercised and its output is linked.');await page.getByLabel('Reason',{exact:true}).fill('The current required criterion was assessed with actual evidence.');
  const footerBounds=await page.locator('#command-form > footer').boundingBox();
  const reasonBounds=await page.getByLabel('Reason',{exact:true}).boundingBox();
  expect(reasonBounds.y+reasonBounds.height,'The footer must not cover the focused completion field').toBeLessThanOrEqual(footerBounds.y+1);
  await capture(page,`completion-${viewport.width}`);
  const done=await save(page,'complete_work_item');expect(done.lifecycle).toBe('done');expect(done.conclusion).toBeTruthy();
  expect((await wos.api(`${base}/objectives/${objective.id}`)).value.lifecycle).toBe('in_progress');
  await page.keyboard.press('Escape');await expect(modal(page).locator('[data-column=accepted]')).toContainText(task.title);
  await capture(page,`accepted-${viewport.width}`);
  const issues=await new AxeBuilder({page}).include('#workspace-kanban').analyze();expect(issues.violations).toEqual([]);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 });
}
test('drag prepares actual operations; blocked and read-only work never moves',async({page,wos,browser})=>{
 const c=await seedCanvas(wos);
 const base=`/namespaces/${wos.namespace}/outcomes/${c.outcome.id}`;
 await startCanvasObjectives(wos,c);
 await open(page,wos,c.outcome,c.objective.title);
 const moves=[];page.on('request',r=>{if(r.url().includes('/commands/'))moves.push(r.url());});
 const blockedBefore=(await wos.api(`${base}/work-items/${c.tasks.blocked.id}`)).value;
 const blockedCard=modal(page).locator(`[data-kanban-ref="work_item/${c.tasks.blocked.id}"]`);
 await expect(blockedCard).toHaveAttribute('draggable','true');
 await expect(blockedCard.locator('select')).toHaveCount(0);
 await expect(modal(page).locator('.workspace-kanban-card-shell select')).toHaveCount(0);
 await move(page,c.tasks.blocked.title,'in_progress');
 await expect(modal(page).locator('.workspace-move-feedback')).toContainText('No state changed');expect(moves).toEqual([]);
 await expect(modal(page).locator('.workspace-requirement').first()).toBeVisible();
 await expect(board(page).locator('[data-column=todo]')).toContainText(c.tasks.blocked.title);
 await expect(board(page).locator('[data-column=in_progress]')).not.toContainText(c.tasks.blocked.title);
 expect((await wos.api(`${base}/work-items/${c.tasks.blocked.id}`)).value).toMatchObject({lifecycle:blockedBefore.lifecycle,version:blockedBefore.version});
 await expect(modal(page).locator('[data-workspace-facet=attention]')).toContainText(/Blocked|blocked/);
 await page.keyboard.press('Escape');
 const card=modal(page).locator(`[data-kanban-ref="work_item/${c.tasks.advisory.id}"]`);
 await card.dragTo(modal(page).locator('[data-column=in_progress]'));
 await expect(modal(page).locator('#command-select')).toHaveValue('claim_work_item');expect(moves).toEqual([]);
 await expect(modal(page).locator('#command-form')).toBeVisible();
 await page.keyboard.press('Escape');await expect(modal(page).locator('.workspace-kanban-operation')).toHaveCount(0);
 await expect(modal(page)).toBeVisible();
 await modal(page).getByRole('button',{name:/^Back to (Kanban|previous item)$/}).click();
 await move(page,c.tasks.advisory.title,'verification');
 await expect(modal(page).getByRole('heading',{name:'Review proof',exact:true})).toBeVisible();expect(moves).toEqual([]);
 const viewer=await provisionCanvasViewer(wos),context=await browser.newContext();
 try{
  const p=await context.newPage();await wos.login(p,viewer.token);await openOutcome(p,c.outcome.title,{area:null});
  await p.getByRole('combobox',{name:'Search Outcome Board',exact:true}).fill(c.objective.title);await p.getByRole('option').filter({hasText:c.objective.title}).first().click();await p.locator('#canvas-details').getByRole('button',{name:'Open Tasks Kanban',exact:true}).click();
  await move(p,c.tasks.advisory.title,'in_progress');await expect(modal(p).locator('.workspace-move-feedback')).toContainText('authorized operation');await expect(modal(p).locator('#command-form')).toHaveCount(0);
 }finally{await context.close();}
});

test('embedded forms preserve conflicting input and recover the same uncertain creation intent',async({page,wos})=>{
 const {outcome,scope,objective}=await fixture(wos),base=`/namespaces/${wos.namespace}/outcomes/${outcome.id}`;
 const task=await wos.command('create_work_item',{scope,title:'Original scoped task',priority:'normal',lifecycle:'backlog',objective_id:objective.id});
 await open(page,wos,outcome,objective.title);await modal(page).getByRole('button',{name:/Original scoped task/}).click();
 await action(page,'update_work_item');await page.getByLabel('Task title',{exact:true}).fill('My retained title');
 await wos.command('update_work_item',{scope,work_item_id:task.id,expected_version:task.version,title:'Concurrent title'});
 const rejected=page.waitForResponse(r=>r.url().endsWith('/commands/update_work_item'));
 await page.locator('#submit-command').click();expect((await (await rejected).json()).error.code).toBe('version_conflict');
 await expect(page.getByLabel('Task title',{exact:true})).toHaveValue('My retained title');await expect(page.locator('#command-error')).toContainText(/newer version/i);
 await page.locator('#close-command').click();await expect(modal(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');
 await expect(modal(page).locator('.workspace-kanban-inspection-header > h3')).toHaveText('Concurrent title');
 await modal(page).getByRole('button',{name:/^Back to (Kanban|previous item)$/}).click();
 await modal(page).locator('button[data-target-lifecycle="backlog"]').click();await page.getByLabel('Task title',{exact:true}).fill('Recover exactly once');
 let first=true;const writes=[];
 await page.route('**/commands/create_work_item',async route=>{
  writes.push({key:route.request().headers()['idempotency-key'],body:route.request().postData()});
  if(first){first=false;const result=await route.fetch();expect(result.ok()).toBe(true);await route.abort('failed');}else await route.continue();
 });
 await page.locator('#submit-command').click();await expect(page.locator('#retry-command')).toBeVisible();
 await page.keyboard.press('Escape');await expect(modal(page).locator('#command-form')).toBeVisible();await expect(page.getByLabel('Task title',{exact:true})).toHaveValue('Recover exactly once');
 const recovered=page.waitForResponse(r=>r.url().endsWith('/commands/create_work_item'));
 await page.locator('#retry-command').click();expect((await recovered).ok()).toBe(true);
 await expect(modal(page).locator('.workspace-kanban-operation')).toHaveCount(0);await expect(modal(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');
 expect(writes).toHaveLength(2);expect(writes[1]).toEqual(writes[0]);
 const cards=await wos.api(`${base}/workspace-cards`,{parent_ref:{...scope,kind:'objective',id:objective.id},limit:50});
 expect(cards.items.filter(i=>i.title==='Recover exactly once')).toHaveLength(1);
});

test('Objective moves use StartObjective and explicit assessed achievement; hierarchy is not completion',async({page,wos})=>{
 const {outcome,scope,objective}=await fixture(wos,{startObjective:false}),owner={...scope,kind:'objective',id:objective.id};
 const criterion=await wos.command('add_criterion',{owner,expected_version:objective.version,title:'Guide checked against agreed condition',required:true,verification_mode:'attestation'});
 await open(page,wos,outcome,outcome.title);
 await move(page,objective.title,'in_progress');
 await expect(modal(page).locator('#command-select')).toHaveValue('start_objective');await page.getByLabel('Reason',{exact:true}).fill('Begin the agreed objective with its current required criterion.');await save(page,'start_objective');
 await modal(page).getByRole('button',{name:/^Back to (Kanban|previous item)$/}).click();
 await move(page,objective.title,'accepted');
 await expect(modal(page).locator('.workspace-move-feedback')).toContainText('No state changed');
 await modal(page).getByRole('button',{name:criterion.title,exact:true}).click();await expect(page.locator('#workspace-kanban-operation .workspace-transition-header')).toContainText(criterion.title);await action(page,'record_criterion_assessment');await page.getByLabel('Rationale',{exact:true}).fill('The agreed condition was inspected and confirmed.');await save(page,'record_criterion_assessment',{nextCommand:'achieve_objective'});
 await expect(modal(page).locator('#command-select')).toHaveValue('achieve_objective');await page.getByLabel('Reason',{exact:true}).fill('Accept the objective against the current assessed condition.');const achieved=await save(page,'achieve_objective');expect(achieved.lifecycle).toBe('achieved');
 expect((await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}`)).value.lifecycle).toBe('active');
});

test('contract-backed movement stays an explicit external handoff without browser acquisition or delivery',async({page,wos})=>{
 const {outcome,scope,objective}=await fixture(wos);
 await wos.command('set_namespace_work_protocol',{scope,expected_protocol_version:1,phase:'draining',reason:'Drain legacy writers for the supported contract protocol.'});
 await wos.command('set_namespace_work_protocol',{scope,expected_protocol_version:2,phase:'contracts_v1',writers_drained:true,reason:'Use the supported external contract host.'});
 const task=await wos.command('create_work_item',{scope,title:'Externally authorized execution',priority:'normal',lifecycle:'todo',objective_id:objective.id,execution_spec:{instructions:['Execute only in the authorized host']}});
 await open(page,wos,outcome,objective.title);
 const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url());});
 await move(page,task.title,'in_progress');
 await expect(modal(page).locator('.workspace-handoff')).toBeVisible();
 await expect(modal(page).locator('#command-form')).not.toBeVisible();
 await expect(page.locator('dialog[open]')).toHaveCount(1);expect(writes).toEqual([]);
 const actual=(await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
 expect(actual.lifecycle).toBe('todo');expect(actual.current_lease).toBeUndefined();expect(actual.current_contract_id).toBeUndefined();
});

test('related Issues and Blockers remain operable in place and resolution never silently releases an impediment',async({page,wos})=>{
 const c=await seedCanvas(wos),base=`/namespaces/${wos.namespace}/outcomes/${c.outcome.id}`;
 await startCanvasObjectives(wos,c);
 await open(page,wos,c.outcome,c.objective.title);await modal(page).getByRole('button',{name:new RegExp(c.tasks.blocked.title)}).click();
 await modal(page).locator('[data-operational-group=impediments] > summary').click();
 await modal(page).locator(`[data-reference-kind=blocker][data-reference-id="${c.blocker.id}"]`).first().click();
 await expect(board(page).locator(".workspace-kanban-inspection")).toHaveAttribute("data-subject",new RegExp(c.blocker.id));
 await modal(page).locator('[data-operational-group=impediments] > summary').click();
 await modal(page).locator(`[data-reference-kind=issue][data-reference-id="${c.issue.id}"]`).first().click();
 await expect(modal(page).locator('.workspace-kanban-inspection-header > h3')).toHaveText(c.issue.title);
 await action(page,'resolve_issue');await page.getByLabel('Resolution summary',{exact:true}).fill('Review environment access has been restored.');await save(page,'resolve_issue');
 expect((await wos.api(`${base}/blockers/${c.blocker.id}`)).value.lifecycle).toBe('active');
 await modal(page).getByRole('button',{name:/^Back to (Kanban|previous item)$/}).click();
 let inspection=modal(page).locator('.workspace-kanban-inspection');
 if (!(await inspection.count())) {
  await modal(page).getByRole('button',{name:new RegExp(c.tasks.blocked.title)}).click();
  inspection=modal(page).locator('.workspace-kanban-inspection');
 }
 await expect(inspection).toContainText('Wait for verified environment access');
 const impediments=inspection.locator('[data-operational-group=impediments]');
 if (!(await impediments.evaluate(node=>node.open))) await impediments.locator(':scope > summary').click();
 await inspection.locator(`[data-reference-kind=blocker][data-reference-id="${c.blocker.id}"]`).first().click();
 await expect(modal(page).locator('.workspace-kanban-inspection-header > h3')).toHaveText('Blocker');
 await expect(modal(page).locator('.workspace-kanban-inspection')).toContainText('Wait for verified environment access');
 await action(page,'resolve_blocker');await page.getByLabel('Resolution summary',{exact:true}).fill('Explicitly release this impediment after checking access.');await page.getByLabel('Confirm work can proceed',{exact:true}).check();await save(page,'resolve_blocker');
 await modal(page).getByRole('button',{name:/^Back to (Kanban|previous item)$/}).click();
 inspection=modal(page).locator('.workspace-kanban-inspection');
 if (!(await inspection.count())) await modal(page).getByRole('button',{name:new RegExp(c.tasks.blocked.title)}).click();
 await expect(modal(page).locator('[data-workspace-facet=attention]')).not.toContainText(/work is blocked/i);
 await action(page,'claim_work_item');const claimed=await save(page,'claim_work_item');expect(claimed.current_lease.principal_id).toBe('owner');
});

test('an already acquired contract has a direct readonly continuation without a parallel browser workflow',async({page,wos})=>{
 const {outcome,scope,objective}=await fixture(wos);
 await wos.command('set_namespace_work_protocol',{scope,expected_protocol_version:1,phase:'draining',reason:'Drain legacy acquisition before contract operation.'});
 await wos.command('set_namespace_work_protocol',{scope,expected_protocol_version:2,phase:'contracts_v1',writers_drained:true,reason:'Operate the existing contract protocol.'});
 const task=await wos.command('create_work_item',{scope,title:'Continue acquired external work',priority:'normal',lifecycle:'todo',objective_id:objective.id,execution_spec:{instructions:['Continue only in the authorized host and submit actual material']}});
 const acquired=await wos.command('acquire_work_contract',{scope,work_item_id:task.id,expected_work_item_version:task.version,ttl_seconds:900});
 await open(page,wos,outcome,objective.title);await modal(page).getByRole('button',{name:new RegExp(task.title)}).click();
 const writes=[];page.on('request',r=>{if(r.url().includes('/commands/'))writes.push(r.url());});
 await modal(page).getByRole('button',{name:'Continue in authorized host',exact:true}).click();
 await expect(modal(page).locator('.contract-section')).toContainText('Active reservation');
 await expect(modal(page).locator('.contract-section')).toContainText(acquired.contract.id);
 await expect(modal(page).locator('.workspace-full-record')).toHaveCount(0);
 await expect(page.locator('dialog[open]')).toHaveCount(1);expect(writes).toEqual([]);
 const actual=(await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
 expect(actual.current_contract_id).toBe(acquired.contract.id);expect(actual.lifecycle).toBe('in_progress');expect(actual.current_lease).toBeUndefined();
 const cameraBeforeClose=await page.locator('.canvas-world').getAttribute('style');
 await modal(page).locator('#close-detail').click();
 await expect(modal(page).locator('.workspace-kanban-operation')).toHaveCount(0);
 await expect(modal(page).locator('.workspace-kanban-inspection-header > h3')).toHaveText(task.title);
 await expect(page.locator('.canvas-world')).toHaveAttribute('style',cameraBeforeClose);
 await expect(page.locator('dialog[open]')).toHaveCount(1);
});

for(const origin of ['outcome','objective']){
 test(`delayed origin restoration returns focus to the rebuilt ${origin} trigger`,async({page,wos})=>{
  const {outcome,scope,objective}=await fixture(wos);
  let ref={...scope,kind:'outcome',id:outcome.id},name='Open Objectives Kanban';
  if(origin==='outcome') await open(page,wos,outcome,outcome.title);
  else{
   ref={...scope,kind:'objective',id:objective.id};name='Open Tasks Kanban';
   await open(page,wos,outcome,objective.title);

  }
  await expect(modal(page)).toBeVisible();
  if(origin==='outcome')await expect(modal(page).locator('.workspace-kanban-card')).toHaveCount(1);
  const trigger=page.locator('#canvas-details').getByRole('button',{name,exact:true}),oldTrigger=await page.locator('#canvas-details').getByRole('button',{name,exact:true,includeHidden:true}).elementHandle();
  let release,begin,first=true;
  const gate=new Promise(resolve=>{release=resolve;}),started=new Promise(resolve=>{begin=resolve;});
  await page.route('**/workspace-view',async route=>{
   const subject=route.request().postDataJSON()?.subject;
   if(first&&subject?.kind==='entity'&&subject.ref.id===ref.id){
    first=false;const response=await route.fetch();expect(response.ok()).toBe(true);begin();await gate;await route.fulfill({response});
   }else await route.continue();
  });
  try{
   await modal(page).getByRole('button',{name:'Back to Canvas',exact:true}).click();await started;await expect(modal(page)).not.toBeVisible();
   release();await expect.poll(()=>oldTrigger.evaluate(node=>node.isConnected)).toBe(false);
   await expect(trigger).toBeFocused();
  }finally{release();}
 });
}
