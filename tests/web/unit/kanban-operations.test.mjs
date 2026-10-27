import test from 'node:test';
import assert from 'node:assert/strict';
import {kanbanOperation} from '../../../packages/wos-api/web/assets/kanban-operations.js';
const subject={kind:'entity',ref:{namespace_id:'ns',outcome_id:'out',kind:'work_item',id:'task'}};
const view=(caps=[],lifecycle='todo')=>({subject,header:{lifecycle},capabilities:caps.map(c=>({subject,transport:'browser_command',availability:'available',...c}))});
test('moving prepares only an actual subject-scoped capability and never invents lifecycle operations',()=>{
 const cap={action_id:'ClaimWorkItem'};assert.equal(kanbanOperation(view([cap]),'in_progress').capability.action_id,'ClaimWorkItem');
 assert.equal(kanbanOperation(view([{action_id:'ClaimWorkItem',subject:{...subject,ref:{...subject.ref,id:'other'}}}]),'in_progress').capability,undefined);
 assert.equal(kanbanOperation(view([],'backlog'),'in_progress').capability,undefined);
 assert.match(kanbanOperation(view([],'backlog'),'in_progress').message,/available first/);
 assert.equal(kanbanOperation(view([{action_id:'CompleteWorkItem',availability:'blocked',reason_codes:['dependency_not_satisfied']}]),'accepted').capability.availability,'blocked');
});
test('verification is a read, signed authority remains external, and Objective achievement stays explicit',()=>{
 assert.equal(kanbanOperation(view([{action_id:'CompleteWorkItem'}]),'verification').review,true);
 assert.equal(kanbanOperation(view([{action_id:'AcquireSignedWorkContract',transport:'signed_host',availability:'unknown'}]),'in_progress').capability.transport,'signed_host');
 const s={kind:'entity',ref:{...subject.ref,kind:'objective'}};
 const v={subject:s,header:{lifecycle:'planned'},capabilities:[{subject:s,action_id:'AchieveObjective',transport:'browser_command',availability:'blocked'}]};
 assert.equal(kanbanOperation(v,'accepted').capability.action_id,'AchieveObjective');
 assert.equal(kanbanOperation(v,'in_progress').capability,undefined);
});

test('typed lanes preserve the public projection while separating lifecycle facts',async()=>{
 const {kanbanLane,kanbanLanes}=await import('../../../packages/wos-api/web/assets/kanban-operations.js');
 assert.deepEqual(kanbanLanes('objective').map(([id])=>id),['planned','in_progress','accepted']);
 assert.deepEqual(kanbanLanes('work_item').map(([id])=>id),['backlog','todo','in_progress','verification','accepted']);
 const item=(kind,lifecycle,display_state,stage='todo')=>({ref:{kind},lifecycle,display_state,stage});
 for(const [kind,life,shown,lane] of [
  ['work_item','backlog','backlog','backlog'],['work_item','todo','ready','todo'],['work_item','in_progress','blocked','in_progress'],
  ['work_item','in_progress','awaiting_review','verification'],['work_item','done','awaiting_review','accepted'],
  ['work_item','cancelled','awaiting_review','cancelled'],['objective','planned','awaiting_review','planned'],['objective','achieved','awaiting_review','accepted']])
  assert.equal(kanbanLane(item(kind,life,shown)),lane);
});
test('two-step acquisition is explicit, Objective achievement needs no artificial start and terminal moves cannot reopen',()=>{
 const backlog=kanbanOperation(view([{action_id:'ActivateWorkItem'}],'backlog'),'in_progress');
 assert.equal(backlog.kind,'needs_preparation');assert.equal(backlog.sequential,true);assert.equal(backlog.capability.action_id,'ActivateWorkItem');
 const s={kind:'entity',ref:{...subject.ref,kind:'objective'}};
 const objective={subject:s,header:{lifecycle:'planned'},capabilities:[{subject:s,action_id:'AchieveObjective',transport:'browser_command',availability:'available'}]};
 assert.equal(kanbanOperation(objective,'accepted').kind,'command');
 assert.equal(kanbanOperation(objective,'verification').kind,'unsupported');
 assert.equal(kanbanOperation(view([{action_id:'ClaimWorkItem'}],'done'),'in_progress').kind,'unsupported');
});

test('a pending native delivery moves to acceptance only through its exact review capability',()=>{
 const delivery={id:'delivery-1',digest:'sha256:current',round:'2'};
 const taskSubject={kind:'entity',ref:{...subject.ref}};
 const pending={subject:taskSubject,header:{lifecycle:'in_progress',display_state:'awaiting_review'},
  verification:{pending:true,current:delivery,source_kind:'task_delivery',source_id:delivery.id,
   delivery_digest:delivery.digest,round:delivery.round},
  capabilities:[{subject:taskSubject,action_id:'AcceptTaskDelivery',transport:'browser_command',availability:'available'}]};
 const intent=kanbanOperation(pending,'accepted');
 assert.equal(intent.kind,'command');
 assert.equal(intent.capability.action_id,'AcceptTaskDelivery');
 const blocked={...pending,capabilities:[{subject:taskSubject,action_id:'CompleteWorkItem',transport:'browser_command',availability:'available'},
  {subject:taskSubject,action_id:'AcceptTaskDelivery',transport:'browser_command',availability:'blocked',reason_codes:['criterion_not_met']}]};
 const blockedIntent=kanbanOperation(blocked,'accepted');
 assert.equal(blockedIntent.kind,'needs_preparation');
 assert.equal(blockedIntent.capability.action_id,'AcceptTaskDelivery');
 assert.deepEqual(blockedIntent.reasons,['criterion_not_met']);
 const signed={...pending,verification:{...pending.verification,source_kind:'signed_submission'},capabilities:[]};
 assert.equal(kanbanOperation(signed,'accepted').kind,'unsupported');
});
