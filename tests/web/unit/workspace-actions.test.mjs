import test from 'node:test';
import assert from 'node:assert/strict';
import {commandForSchema, workspaceActionName, workspaceActions} from '../../../packages/wos-api/web/assets/workspace-actions.js';
import {createIntent} from '../../../packages/wos-api/web/assets/intents.js';

test('workspace exact decimal versions preserve old numeric wire and new quoted fields', () => {
  const max = '18446744073709551615';
  const schema = {type:'object',properties:{expected_version:{type:'integer'},new_version:{type:'string'},nested:{type:'array',items:{type:'object',properties:{criterion_revision:{type:'integer'}}}}}};
  const body = createIntent('edit', commandForSchema({expected_version:max,new_version:max,nested:[{criterion_revision:max}]},schema),{},'fixed').body;
  assert.equal(body,'{"command":{"expected_version":18446744073709551615,"new_version":"18446744073709551615","nested":[{"criterion_revision":18446744073709551615}]}}');
  assert.equal(body, createIntent('edit',commandForSchema({expected_version:max,new_version:max,nested:[{criterion_revision:max}]},schema),{},'fixed').body);
});
test('workspace capabilities retain blocked reason and explicit transport without granting writes', () => {
  const actions=workspaceActions({capabilities:[{action_id:'ActivateOutcome',availability:'blocked',reason_codes:['missing_required_criterion'],transport:'browser_command'}]});
  assert.equal(actions.length,1);
  assert.equal(actions[0].name,'activate_outcome');
  assert.equal(actions[0].availability,'blocked');
  assert.deepEqual(actions[0].reason_codes,['missing_required_criterion']);
  assert.equal(workspaceActionName('RegisterEvidenceWithLinks'),'register_evidence_with_links');
});
