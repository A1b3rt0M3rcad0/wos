import {test,expect,openOutcome} from './ux-fixture.mjs';
import {mkdir,readFile,readdir,writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {resolve} from 'node:path';
const kanban=page=>page.locator('#workspace-kanban');
const operation=page=>page.locator('#workspace-kanban-operation');
async function capture(page,wos,name){
 const dir=process.env.WOS_CAPTURE_SIMPLIFIED_KANBAN;if(!dir)return;
 const root=resolve('../..'),source=execFileSync('git',['rev-parse','HEAD'],{cwd:root,encoding:'utf8'}).trim();
 expect(execFileSync('git',['status','--porcelain','--','packages','tests/web','tools'],{cwd:root,encoding:'utf8'})).toBe('');
 const binary=JSON.parse(execFileSync(resolve(root,'bin/wos'),['--output','json','version'],{encoding:'utf8'})).data;
 expect(binary.commit).toBe(source);
 const hash=bytes=>createHash('sha256').update(bytes).digest('hex');
 const assetRoot=resolve(root,'packages/wos-api/web/assets');const assets={};
 for(const file of (await readdir(assetRoot)).filter(name=>/\.(js|css|html)$/.test(name))){
  const response=await page.request.get(`${wos.url}/app/${file}`);expect(response.ok()).toBe(true);
  const expected=hash(await readFile(resolve(assetRoot,file))),actual=hash(await response.body());expect(actual,`Embedded ${file}`).toBe(expected);assets[file]=actual;
 }
 await mkdir(dir,{recursive:true});await page.evaluate(()=>document.fonts.ready);
 const png=await page.screenshot({path:`${dir}/${name}.png`,animations:'disabled'});
 await writeFile(`${dir}/${name}.json`,JSON.stringify({source,binary,viewport:page.viewportSize(),browser:page.context().browser().version(),assets,pngSha256:hash(png)},null,2)+'\n');
}
async function setup(wos){
 const {value:outcome,scope}=await wos.outcome('Focused planning and proof');
 await wos.command('add_criterion',{owner:{...scope,kind:'outcome',id:outcome.id},expected_version:outcome.version,title:'A verified result',required:true,verification_mode:'attestation'});
 await wos.command('activate_outcome',{scope,expected_version:outcome.version+1});
 const objective=await wos.command('create_objective',{scope,title:'A focused work context',priority:'normal'});
 const task=await wos.command('create_work_item',{scope,title:'Deliver the agreed login',description:'Implement the login and demonstrate the agreed behavior.',priority:'normal',objective_id:objective.id,lifecycle:'backlog'});
 const criterion=await wos.command('add_criterion',{owner:{...scope,kind:'work_item',id:task.id},expected_version:task.version,title:'An expired token is rejected',required:true,verification_mode:'evidence_review'});
 return {outcome,scope,objective,task,criterion};
}
async function open(page,wos,outcome,title){
 await wos.login(page);await openOutcome(page,outcome.title,{area:null});
 await page.getByRole('combobox',{name:'Search Outcome Board',exact:true}).fill(title);
 await page.getByRole('option').filter({hasText:title}).first().click();
 await page.locator('#canvas-details').getByRole('button',{name:title===outcome.title?'Open Objectives Kanban':'Open Tasks Kanban',exact:true}).click();
 await expect(kanban(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');
}
for(const viewport of [{width:1280,height:720},{width:2560,height:1440},{width:3840,height:2160},{width:390,height:844}]){
 test(`focused modal, contextual controls and clean inspection at ${viewport.width}`,async({page,wos})=>{
  await page.setViewportSize(viewport);const {outcome,objective,task}=await setup(wos);
  await open(page,wos,outcome,outcome.title);
  await expect(kanban(page)).toHaveJSProperty('tagName','DIALOG');await expect(page.locator('dialog[open]')).toHaveCount(1);
  await expect(kanban(page).getByRole('button',{name:'Add Objective',exact:true})).toBeVisible();
  await expect(kanban(page).getByRole('button',{name:/Add.*Task/})).toHaveCount(0);
  await expect(kanban(page).locator('.workspace-kanban-breadcrumb')).toHaveCount(0);
  await kanban(page).locator('.workspace-kanban-card').filter({hasText:objective.title}).click();
  const objectiveInspector=kanban(page).locator('.workspace-kanban-inspection');
  for(const text of ['Verify criteria','Record details and audit','More actions','Add material']) await expect(objectiveInspector.getByText(text,{exact:true})).toHaveCount(0);
  if(viewport.width===2560) await capture(page,wos,'objectives-2560');
  await kanban(page).getByRole('button',{name:'Open Tasks Kanban',exact:true}).click();
  await expect(kanban(page).getByRole('heading',{name:'Tasks Kanban',exact:true})).toBeVisible();
  await expect(kanban(page).getByRole('button',{name:'Add Task',exact:true})).toBeVisible();
  await expect(kanban(page).getByRole('button',{name:/Add.*[Oo]bjective/})).toHaveCount(0);
  await kanban(page).locator('.workspace-kanban-card').filter({hasText:task.title}).click();
  const inspector=kanban(page).locator('.workspace-kanban-inspection');
  for(const text of ['Verify criteria','Record details and audit','More actions','Add material']) await expect(inspector.getByText(text,{exact:true})).toHaveCount(0);
  await expect(inspector.locator('[data-operational-group=work] > summary')).toBeVisible();
  const rect=await kanban(page).boundingBox();expect(rect.x).toBeGreaterThanOrEqual(0);expect(rect.y).toBeGreaterThanOrEqual(0);expect(rect.x+rect.width).toBeLessThanOrEqual(viewport.width);expect(rect.y+rect.height).toBeLessThanOrEqual(viewport.height);
  await expect(kanban(page).locator('.workspace-card-menu, .workspace-kanban-card-shell select')).toHaveCount(0);
  await capture(page,wos,`kanban-${viewport.width}`);
  await page.keyboard.press('Escape');await page.keyboard.press('Escape');
  await expect(kanban(page)).not.toBeVisible();await expect(page.locator('#area-tabs [data-area=overview]')).toBeVisible();
 });
}
test('proof checklist reuses existing Evidence, requires explicit assessment and preserves the agreed Task definition',async({page,wos})=>{
 const {outcome,scope,objective,task,criterion}=await setup(wos);
 let actual=await wos.command('activate_work_item',{scope,work_item_id:task.id,expected_version:task.version+1});
 actual=await wos.command('claim_work_item',{scope,work_item_id:task.id,expected_version:actual.version});
 const evidence=await wos.command('register_evidence',{scope,evidence_type:'test_result',description:'Expired-token test passed',source_ref:{provider:'tests',id:'expired-token'},captured_at:new Date().toISOString()});
 await open(page,wos,outcome,objective.title);
 const shell=kanban(page).locator('.workspace-kanban-card-shell').filter({hasText:task.title});
 await shell.locator('.workspace-kanban-card').click();
 const inspector=kanban(page).locator('.workspace-kanban-inspection');
 await expect(inspector).toContainText('Planning is locked outside Backlog.');
 for(const action of ['set_work_item_execution_spec','update_work_item','add_criterion']) await expect(inspector.locator(`[data-action-id=${action}]`)).toHaveCount(0);
 await inspector.getByRole('button',{name:'Back to Kanban',exact:true}).click();
 await shell.locator('.workspace-kanban-card').dragTo(kanban(page).locator('[data-column=verification]'));
 await expect(operation(page).getByRole('heading',{name:'What to do',exact:true})).toBeVisible();
 await expect(operation(page).getByText('Available preparation actions',{exact:true})).toHaveCount(0);
 for(const text of ['Responsibility & authority','Dependencies & impediments','Activity history','Record details and audit']) await expect(operation(page).getByText(text,{exact:true})).toHaveCount(0);
 let check=operation(page).getByRole('checkbox',{name:/An expired token is rejected/});await expect(check).not.toBeChecked();
 await operation(page).getByRole('button',{name:'Add or link evidence',exact:true}).click();
 await expect(page.locator('dialog[open]')).toHaveCount(1);
 await operation(page).getByLabel('Evidence source',{exact:true}).selectOption('existing');
 await operation(page).getByRole('combobox',{name:'Evidence',exact:true}).fill(evidence.description);
 await operation(page).getByRole('option').filter({hasText:evidence.description}).click();
 await operation(page).getByLabel('Why is this relevant?',{exact:true}).fill('This test exercises the exact expired-token criterion.');
 const linked=page.waitForResponse(r=>r.url().endsWith('/commands/register_evidence_with_links'));
 await operation(page).locator('#submit-command').click();const link=await linked;expect(link.ok(),await link.text()).toBe(true);
 await expect(operation(page).getByRole('heading',{name:'Review proof',exact:true})).toBeVisible();
 check=operation(page).getByRole('checkbox',{name:/An expired token is rejected/});await expect(check).not.toBeChecked();
 await check.click();await expect(operation(page).locator('#command-select')).toHaveValue('record_criterion_assessment');
 await operation(page).getByLabel('Rationale',{exact:true}).fill('Reviewed the expired-token test and its actual result.');
 const evidencePicker=operation(page).getByRole('combobox',{name:/Evidence/});
 if(await evidencePicker.count()){await evidencePicker.fill(evidence.description);await operation(page).getByRole('option').filter({hasText:evidence.description}).click();}
 const assessed=page.waitForResponse(r=>r.url().endsWith('/commands/record_criterion_assessment'));
 await operation(page).locator('#submit-command').click();const response=await assessed;expect(response.ok(),await response.text()).toBe(true);
 await expect(operation(page).getByRole('checkbox',{name:/An expired token is rejected/})).toBeChecked();
 await capture(page,wos,'review-proof');
 const current=(await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
 expect(current.lifecycle).toBe('in_progress');expect(current.criteria.items[0].id).toBe(criterion.id);expect(current.title).toBe(task.title);
});

test('proof preserves complete long instructions without restoring the audit interface',async({page,wos})=>{
 const {outcome,scope,objective,task}=await setup(wos);
 const instruction='Inspect each documented step. '.repeat(100)+'CRITICAL FINAL INSTRUCTION';
 await wos.command('set_work_item_execution_spec',{scope,work_item_id:task.id,expected_version:String(task.version+1),execution_spec:{instructions:[instruction],constraints:[],deliverables:[],scope_hints:[],context_refs:[]}});
 await open(page,wos,outcome,objective.title);
 const shell=kanban(page).locator('.workspace-kanban-card-shell').filter({hasText:task.title});
 await shell.locator('.workspace-kanban-card').dragTo(kanban(page).locator('[data-column=verification]'));
 await expect(operation(page)).not.toContainText('CRITICAL FINAL INSTRUCTION');
 await operation(page).getByRole('button',{name:'Read complete instructions',exact:true}).click();
 await expect(operation(page)).toContainText('CRITICAL FINAL INSTRUCTION');
 await expect(operation(page).locator('.workspace-audit,.workspace-full-record')).toHaveCount(0);
 await expect(page.locator('dialog[open]')).toHaveCount(1);
});
