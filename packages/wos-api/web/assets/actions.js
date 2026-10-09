export const humanActions=Object.freeze({
 revoke_work_contract:{label:"Revoke contract",contexts:["work_item"],advanced:true,requiresUnsignedContract:true},
 create_outcome:{label:'New outcome',contexts:['workspace']},
 update_outcome:{label:'Edit outcome',contexts:['outcome']},
 create_objective:{label:'Add objective',contexts:['outcome','objective']},
 update_objective:{label:'Edit objective',contexts:['objective']},
 create_work_item:{label:'New task',contexts:['outcome','objective']},
 update_work_item:{label:'Edit task',contexts:['work_item'],states:['backlog','todo','in_progress']},
 add_criterion:{label:'Add success criterion',contexts:['outcome','objective','work_item']},
 record_criterion_assessment:{label:'Assess criterion',contexts:['criterion']},
 revise_criterion:{label:'Revise criterion',contexts:['criterion']},
 retire_criterion:{label:'Retire criterion',contexts:['criterion']},
 activate_outcome:{label:'Activate outcome',contexts:['outcome'],states:['draft']},
 achieve_outcome:{label:'Certify outcome',contexts:['outcome'],states:['active']},
 fail_outcome:{label:'Mark outcome failed',contexts:['outcome'],states:['active']},
 abandon_outcome:{label:'Abandon outcome',contexts:['outcome'],states:['draft','active']},
 reopen_outcome:{label:'Reopen outcome',contexts:['outcome'],states:['achieved','failed','abandoned']},
 archive_outcome:{label:'Archive outcome',contexts:['outcome'],notArchived:true},
 unarchive_outcome:{label:'Restore outcome',contexts:['outcome'],archived:true},
 start_objective:{label:'Start objective',contexts:['objective'],states:['planned']},
 achieve_objective:{label:'Certify objective',contexts:['objective'],states:['in_progress']},
 cancel_objective:{label:'Cancel objective',contexts:['objective'],states:['planned','in_progress']},
 reopen_objective:{label:'Reopen objective',contexts:['objective'],states:['achieved','cancelled']},
 activate_work_item:{label:'Move task to do',contexts:['work_item'],states:['backlog']},
 defer_work_item:{label:'Move task to backlog',contexts:['work_item'],states:['todo']},
 cancel_work_item:{label:'Cancel task',contexts:['work_item'],states:['backlog','todo','in_progress']},
 reopen_work_item:{label:'Reopen task',contexts:['work_item'],states:['done','cancelled']},
 add_dependency:{label:'Add dependency',contexts:['work_item','objective']},
 create_issue:{label:'Report issue',contexts:['outcome']},
 report_issue_with_blocker:{label:'Report issue and block work',contexts:['outcome']},
 create_blocker:{label:'Mark work as blocked',contexts:['outcome','work_item']},
 update_issue:{label:'Edit issue',contexts:['issue']},
 investigate_issue:{label:'Investigate issue',contexts:['issue'],states:['open']},
 resolve_issue:{label:'Resolve issue',contexts:['issue'],states:['open','investigating']},
 resolve_issue_and_blockers:{label:'Resolve issue and release selected blockers',contexts:['issue'],states:['open','investigating']},
 reopen_issue:{label:'Reopen issue',contexts:['issue'],states:['resolved','wont_fix','duplicate']},
 update_blocker_description:{label:'Edit blocking impact',contexts:['blocker']},
 resolve_blocker:{label:'Release blocker',contexts:['blocker'],states:['active']},
 cancel_blocker:{label:'Cancel blocker',contexts:['blocker'],states:['active']},
 register_evidence:{label:'Register evidence',contexts:['outcome']},
 retract_evidence:{label:'Retract evidence',contexts:['evidence'],states:['registered']},
 create_evidence_link:{label:'Link evidence',contexts:['evidence']},
 register_artifact:{label:'Register artifact',contexts:['outcome']},
 withdraw_artifact:{label:'Withdraw artifact',contexts:['artifact'],states:['registered']},
 propose_decision:{label:'Propose decision',contexts:['outcome']},
 accept_decision:{label:'Accept decision',contexts:['decision'],states:['proposed']},
 reject_decision:{label:'Reject decision',contexts:['decision'],states:['proposed']},
 create_roadmap:{label:'Create roadmap',contexts:['outcome']},
 open_roadmap_draft:{label:'Edit roadmap',contexts:['roadmap'],requiresNoDraft:true},
 replace_roadmap_draft:{label:'Edit draft',contexts:['roadmap'],requiresDraft:true},
 discard_roadmap_draft:{label:'Discard draft',contexts:['roadmap'],requiresDraft:true},
 publish_roadmap_draft:{label:'Review and publish',contexts:['roadmap'],requiresDraft:true},
 activate_roadmap_revision:{label:'Activate revision',contexts:['roadmap'],requiresRevision:true},
 deactivate_roadmap_revision:{label:'Deactivate revision',contexts:['roadmap'],requiresRevision:true},
 archive_roadmap:{label:'Archive roadmap',contexts:['roadmap']},
 reopen_roadmap:{label:'Reopen roadmap',contexts:['roadmap'],states:['archived']},
 claim_work_item:{label:'Reserve task',contexts:['work_item'],states:['todo'],lease:'empty'},
 complete_work_item:{label:'Complete task',contexts:['work_item'],states:['in_progress'],lease:'holder'},
 release_work_item:{label:'Release task reservation',contexts:['work_item'],states:['in_progress'],lease:'holder'},
});
const legacyLeaseNames=new Set(['claim_work_item','complete_work_item','release_work_item']);
export function availableActions({context,entity={},criterion,protocol,permissions=[],principal,inventory}) {
 return Object.entries(humanActions).filter(([name,a])=>{
  const entry=inventory[name];if(!entry||!permissions.includes(entry.permission)||!a.contexts.includes(context))return false;
  if(a.states&&!a.states.includes(entity.lifecycle))return false;
  if(a.archived&&!entity.archived_at||a.notArchived&&entity.archived_at)return false;
  if(entity.archived_at&&!a.archived)return false;
  if(a.requiresUnsignedContract&&(!entity.current_contract_id||protocol?.phase==='signed_contracts_v2'))return false;
  if(a.requiresDraft&&!entity.draft||a.requiresNoDraft&&entity.draft)return false;
  if(entity.lifecycle==='archived'&&context==='roadmap'&&name!=='reopen_roadmap')return false;
  if(a.requiresRevision&&!entity.revisions?.length)return false;
  if(legacyLeaseNames.has(name)&&(protocol?.phase!=='legacy'||entity.contracts_enabled))return false;
  if(a.lease==='holder'&&(!entity.current_lease||entity.current_lease.principal_id!==principal||entity._operational_state?.lease_status!=='active'))return false;
  if(a.lease==='empty'&&(entity.current_lease||entity._operational_state?.display_state!=='ready'))return false;
  if(context==='criterion'&&!criterion)return false;
  if(context==='criterion'&&entity._kind==='work_item'&&protocol?.phase==='signed_contracts_v2')return false;
  return true;
 }).map(([name,a])=>({name,...a}));
}

export const permissionLabels=Object.freeze({
  "work.contract.return": "Work contract return",
  "work.contract.complete_direct": "Work contract complete direct",
  "work.review.acquire": "Work review acquire",
  "work.review.decide": "Work review decide",
  "identity.signing_key.enroll": "Identity signing key enroll",
  "identity.signing_key.rotate": "Identity signing key rotate",
  "identity.signing_key.revoke": "Identity signing key revoke",
  "work.contract.acquire": "Work contract acquire",
  "work.contract.revoke": "Work contract revoke",
  "state:read": "State read",
  "outcome:write": "Outcome write",
  "planning:write": "Planning write",
  "work:write": "Work write",
  "records:write": "Records write",
  "assessment:write": "Assessment write",
  "conclusion:write": "Conclusion write",
  "namespace:admin": "Namespace admin",
  "integration:write": "Integration write",
  "actor:delegate": "Actor delegate",
  "work:admin_cancel": "Work admin cancel",
  "work:admin_complete": "Work admin complete",
  "assessment:waive": "Assessment waive"
});
