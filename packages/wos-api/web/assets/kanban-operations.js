import { workspaceActions, workspaceCommandName } from './workspace-view.js';

// Local presentation only. The public `stage` contract remains unchanged.
export function kanbanLanes(kind) {
  return kind === 'objective'
    ? [['planned', 'Planned'], ['in_progress', 'In progress'], ['accepted', 'Achieved']]
    : [['backlog', 'Backlog'], ['todo', 'To do'], ['in_progress', 'In progress'],
      ['verification', 'Verification'], ['accepted', 'Accepted']];
}
export function kanbanLane(item) {
  if (['done', 'achieved'].includes(item.lifecycle)) return 'accepted';
  if (['cancelled', 'abandoned', 'failed'].includes(item.lifecycle)) return 'cancelled';
  if (item.ref.kind === 'objective') return item.lifecycle === 'in_progress' ? 'in_progress' : 'planned';
  if (item.display_state === 'awaiting_review') return 'verification';
  return ['backlog', 'todo', 'in_progress'].includes(item.lifecycle) ? item.lifecycle : 'backlog';
}

// Intent selection never creates a command body, authority, signature or status.
export function kanbanOperation(view, stage) {
  const actions = workspaceActions(view), kind = view.subject.ref.kind;
  const find = (...names) => names.map(name => actions.find(c => workspaceCommandName(c.action_id) === name)).find(Boolean);
  const unsupported = {kind:'unsupported', message:'This move has no authorized operation in the current state. No state changed. Inspect the work and its available actions.'};
  if (!kanbanLanes(kind).some(([lane]) => lane === stage)) return unsupported;
  if (['done','achieved','cancelled','abandoned','failed'].includes(view.header.lifecycle)) return unsupported;
  if (kind === 'work_item' && stage === 'verification') return {kind:'inspection', review:true,
    message:'Review the current criteria and delivery. Opening review does not submit work, create a verification state or accept the result.'};
  let capability, message, sequential = false;
  if (stage === 'accepted') capability = find(kind === 'objective' ? 'achieve_objective' : 'complete_work_item');
  else if (stage === 'in_progress') {
    if (kind === 'work_item' && view.header.lifecycle === 'backlog') {
      capability = find('activate_work_item'); sequential = true;
      message = 'Two explicit steps: make this Task available first, then acquire execution authority. Each command requires your confirmation; assignment grants no authority.';
    } else capability = find('start_objective','reclaim_work_item','claim_work_item','acquire_signed_work_contract','acquire_work_contract');
  } else if (stage === 'todo' && kind === 'work_item') capability = find('activate_work_item','release_work_item');
  if (!capability) return {...unsupported, message: message || unsupported.message};
  const external = capability.transport === 'signed_host';
  return {kind: external ? 'external_handoff' : capability.availability !== 'available' || sequential ? 'needs_preparation' : 'command',
    capability, sequential, reasons:capability.reason_codes || [], remedies:capability.remedies || [],
    message: message || (external ? 'Continue in the authorized host. The browser does not acquire signed authority, execute work or sign a delivery.'
      : capability.availability === 'available' ? 'Review and confirm the existing operation. Only a persisted server response changes the lane.'
        : 'No state changed. Resolve the reported requirements, then recheck this intention against the current state.')};
}

export function operationGroup(action) {
  const name = workspaceCommandName(action.action_id);
  if (/claim|acquire|renew_work|activate_work|start_objective|release_work|complete_work|achieve_/.test(name)) return 'Execute and conclude';
  if (/criterion|assessment|attest|evidence|artifact/.test(name)) return 'Prove and verify';
  if (/owner|assignee/.test(name)) return 'Responsibility';
  if (/issue|blocker|dependency/.test(name)) return 'Resolve impediments';
  return 'Plan and document';
}
