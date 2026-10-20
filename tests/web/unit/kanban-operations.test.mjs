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
