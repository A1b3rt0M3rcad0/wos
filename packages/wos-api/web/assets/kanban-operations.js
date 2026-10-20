import { workspaceActions, workspaceCommandName } from './workspace-view.js';

// Stages are projections. A move only chooses an existing authorized operation;
// neither this module nor a drop writes lifecycle or guesses a command payload.
export function kanbanOperation(view, stage) {
  const actions = workspaceActions(view);
  const find = (...names) => actions.find(c => names.includes(workspaceCommandName(c.action_id)));
  const kind = view.subject.ref.kind;
  if (stage === 'verification') return { review: true,
    message: 'Review the current criteria and delivery. Opening review does not submit work or accept the result.' };
  const capability = stage === 'accepted'
    ? find(kind === 'objective' ? 'achieve_objective' : 'complete_work_item')
    : stage === 'in_progress'
      ? find('start_objective', 'claim_work_item', 'acquire_work_contract', 'acquire_signed_work_contract', 'reclaim_work_item')
      : stage === 'todo' ? find('activate_work_item', 'release_work_item') : null;
  if (capability) return { capability };
  return { message: stage === 'in_progress' && view.header.lifecycle === 'backlog'
    ? 'Make this Task available first, then acquire execution authority. Use Activate Task below.'
    : 'This move is not available in the current state or with your permissions. Inspect the work, authority and criteria below; no state was changed.' };
}

export function operationGroup(action) {
  const name = workspaceCommandName(action.action_id);
  if (/claim|acquire|renew_work|activate_work|start_objective|release_work|complete_work|achieve_/.test(name)) return 'Execute and conclude';
  if (/criterion|assessment|attest|evidence|artifact/.test(name)) return 'Prove and verify';
  if (/owner|assignee/.test(name)) return 'Responsibility';
  if (/issue|blocker|dependency/.test(name)) return 'Resolve impediments';
  return 'Plan and document';
}
