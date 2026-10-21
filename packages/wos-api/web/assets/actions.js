import { copy } from "./en-US.js";
export const humanActions = Object.freeze({
  append_entity_note: { label: copy.text.workspaceAddUpdate, contexts: ["outcome", "objective", "work_item", "issue", "blocker", "evidence", "artifact", "decision", "roadmap", "criterion", "relation", "evidence_link"] },
  create_plan_items: { label: copy.text.workspaceCreatePlan, contexts: ["outcome", "objective"] },
  set_work_item_execution_spec: { label: copy.text.workspaceEditExecution, contexts: ["work_item"] },
  set_work_item_not_before: { label: copy.text.workspaceEditSchedule, contexts: ["work_item"], advanced: true },
  set_work_item_objective: { label: copy.text.workspaceMoveTask, contexts: ["work_item"], advanced: true },
  set_objective_parent: { label: copy.text.workspaceMoveObjective, contexts: ["objective"], advanced: true },
  set_outcome_owners: { label: copy.text.editParticipants, contexts: ["outcome"], advanced: true },
  set_objective_owners: { label: copy.text.editParticipants, contexts: ["objective"], advanced: true },
  register_evidence_with_links: { label: copy.text.workspaceAddEvidence, contexts: ["outcome", "objective", "work_item", "issue", "decision", "criterion"] },
  register_artifact_in_context: { label: copy.text.workspaceAddArtifact, contexts: ["outcome", "objective", "work_item", "issue", "blocker", "decision", "evidence"] },
  propose_decision_in_context: { label: copy.text.workspaceAddDecision, contexts: ["outcome", "objective", "work_item", "issue", "blocker", "decision", "evidence"] },
  replace_external_context: {
    label: copy.text.editExternalContext,
    contexts: ["outcome"],
    advanced: true,
  },
  set_work_item_assignees: {
    label: copy.text.editParticipants,
    contexts: ["work_item"],
    advanced: true,
  },
  revoke_work_contract: {
    label: copy.text.revokeContract,
    contexts: ["work_item"],
    advanced: true,
    requiresUnsignedContract: true,
  },
  create_outcome: { label: copy.text.newOutcome, contexts: ["workspace"] },
  update_outcome: { label: copy.text.editOutcome, contexts: ["outcome"] },
  create_objective: {
    label: copy.text.addObjective,
    contexts: ["outcome", "objective"],
  },
  update_objective: {
    label: copy.text.editObjective,
    contexts: ["objective"],
  },
  create_work_item: {
    label: copy.text.newTask3e9922,
    contexts: ["outcome", "objective"],
  },
  update_work_item: {
    label: copy.text.editTask,
    contexts: ["work_item"],
    states: ["backlog", "todo", "in_progress"],
  },
  add_criterion: {
    label: copy.text.addSuccessCriterion,
    contexts: ["outcome", "objective", "work_item"],
  },
  record_criterion_assessment: {
    label: copy.text.assessCriterion,
    contexts: ["criterion"],
  },
  revise_criterion: {
    label: copy.text.reviseCriterion,
    contexts: ["criterion"],
  },
  retire_criterion: {
    label: copy.text.retireCriterion,
    contexts: ["criterion"],
  },
  activate_outcome: {
    label: copy.text.activateOutcome,
    contexts: ["outcome"],
    states: ["draft"],
  },
  achieve_outcome: {
    label: copy.text.certifyOutcome,
    contexts: ["outcome"],
    states: ["active"],
  },
  fail_outcome: {
    label: copy.text.markOutcomeFailed,
    contexts: ["outcome"],
    states: ["active"],
  },
  abandon_outcome: {
    label: copy.text.abandonOutcome,
    contexts: ["outcome"],
    states: ["draft", "active"],
  },
  reopen_outcome: {
    label: copy.text.reopenOutcome,
    contexts: ["outcome"],
    states: ["achieved", "failed", "abandoned"],
  },
  archive_outcome: {
    label: copy.text.archiveOutcome,
    contexts: ["outcome"],
    notArchived: true,
  },
  unarchive_outcome: {
    label: copy.text.restoreOutcome,
    contexts: ["outcome"],
    archived: true,
  },
  start_objective: {
    label: copy.text.startObjective,
    contexts: ["objective"],
    states: ["planned"],
  },
  achieve_objective: {
    label: copy.text.certifyObjective,
    contexts: ["objective"],
    states: ["planned", "in_progress"],
  },
  cancel_objective: {
    label: copy.text.cancelObjective,
    contexts: ["objective"],
    states: ["planned", "in_progress"],
  },
  reopen_objective: {
    label: copy.text.reopenObjective,
    contexts: ["objective"],
    states: ["achieved", "cancelled"],
  },
  activate_work_item: {
    label: copy.text.moveTaskToDo,
    contexts: ["work_item"],
    states: ["backlog"],
  },
  defer_work_item: {
    label: copy.text.moveTaskToBacklog,
    contexts: ["work_item"],
    states: ["todo"],
  },
  cancel_work_item: {
    label: copy.text.cancelTask,
    contexts: ["work_item"],
    states: ["backlog", "todo", "in_progress"],
  },
  reopen_work_item: {
    label: copy.text.reopenTask,
    contexts: ["work_item"],
    states: ["done", "cancelled"],
  },
  add_dependency: {
    label: copy.text.addDependency,
    contexts: ["work_item", "objective"],
  },
  remove_dependency: {
    label: "Remove dependency",
    contexts: ["relation"],
    states: ["active"],
    relationType: "depends_on",
  },
  create_issue: { label: copy.text.reportIssue, contexts: ["outcome", "objective", "work_item", "artifact", "evidence", "decision"] },
  report_issue_with_blocker: {
    label: copy.text.reportIssueAndBlockWork,
    contexts: ["outcome"],
  },
  create_blocker: {
    label: copy.text.markWorkAsBlocked,
    contexts: ["outcome", "work_item"],
  },
  relate_issue_to_work_item: {
    label: copy.text.linkIssueAndTask,
    contexts: ["issue", "work_item"],
    requiresActiveOutcome: true,
  },
  remove_issue_work_item_relation: {
    label: copy.text.removeIssueTaskLink,
    contexts: ["relation"],
    states: ["active"],
    relationType: "relates_to",
    requiresActiveOutcome: true,
  },
  update_issue: { label: copy.text.editIssue, contexts: ["issue"] },
  investigate_issue: {
    label: copy.text.investigateIssue,
    contexts: ["issue"],
    states: ["open"],
  },
  resolve_issue: {
    label: copy.text.resolveIssue,
    contexts: ["issue"],
    states: ["open", "investigating"],
  },
  resolve_issue_and_blockers: {
    label: copy.text.resolveIssueAndReleaseSelectedBlockers,
    contexts: ["issue"],
    states: ["open", "investigating"],
  },
  reopen_issue: {
    label: copy.text.reopenIssue,
    contexts: ["issue"],
    states: ["resolved", "wont_fix", "duplicate"],
  },
  update_blocker_description: {
    label: copy.text.editBlockingImpact,
    contexts: ["blocker"],
  },
  resolve_blocker: {
    label: copy.text.releaseBlocker,
    contexts: ["blocker"],
    states: ["active"],
  },
  cancel_blocker: {
    label: copy.text.cancelBlocker,
    contexts: ["blocker"],
    states: ["active"],
  },
  register_evidence: {
    label: copy.text.registerEvidence,
    contexts: ["outcome"],
  },
  retract_evidence: {
    label: copy.text.retractEvidence,
    contexts: ["evidence"],
    states: ["registered"],
  },
  create_evidence_link: {
    label: copy.text.linkEvidence,
    contexts: ["evidence"],
  },
  register_artifact: {
    label: copy.text.registerArtifact,
    contexts: ["outcome"],
  },
  withdraw_artifact: {
    label: copy.text.withdrawArtifact,
    contexts: ["artifact"],
    states: ["registered"],
  },
  propose_decision: { label: copy.text.proposeDecision, contexts: ["outcome"] },
  accept_decision: {
    label: copy.text.acceptDecision,
    contexts: ["decision"],
    states: ["proposed"],
  },
  reject_decision: {
    label: copy.text.rejectDecision,
    contexts: ["decision"],
    states: ["proposed"],
  },
  create_roadmap: { label: copy.text.createRoadmap, contexts: ["outcome"] },
  open_roadmap_draft: {
    label: copy.text.editRoadmap,
    contexts: ["roadmap"],
    requiresNoDraft: true,
  },
  replace_roadmap_draft: {
    label: copy.text.editDraft,
    contexts: ["roadmap"],
    requiresDraft: true,
  },
  discard_roadmap_draft: {
    label: copy.text.discardDraft,
    contexts: ["roadmap"],
    requiresDraft: true,
  },
  publish_roadmap_draft: {
    label: copy.text.reviewAndPublish,
    contexts: ["roadmap"],
    requiresDraft: true,
  },
  activate_roadmap_revision: {
    label: copy.text.activateRevision,
    contexts: ["roadmap"],
    requiresRevision: true,
  },
  deactivate_roadmap_revision: {
    label: copy.text.deactivateRevision,
    contexts: ["roadmap"],
    requiresRevision: true,
  },
  archive_roadmap: { label: copy.text.archiveRoadmap, contexts: ["roadmap"] },
  reopen_roadmap: {
    label: copy.text.reopenRoadmap,
    contexts: ["roadmap"],
    states: ["archived"],
  },
  claim_work_item: {
    label: copy.text.reserveTask,
    contexts: ["work_item"],
    states: ["todo"],
    lease: "empty",
  },
  complete_work_item: {
    label: copy.text.completeTask,
    contexts: ["work_item"],
    states: ["in_progress"],
    lease: "holder",
  },
  release_work_item: {
    label: copy.text.releaseTaskReservation,
    contexts: ["work_item"],
    states: ["in_progress"],
    lease: "holder",
  },
  renew_work_item_lease: { label: 'Extend my reservation', contexts: ['work_item'], states: ['in_progress'], lease: 'holder' },
  reclaim_work_item: { label: 'Reacquire expired reservation', contexts: ['work_item'], states: ['in_progress'], lease: 'expired' },
});
const legacyLeaseNames = new Set([
  "claim_work_item",
  "complete_work_item",
  "release_work_item",
  "renew_work_item_lease",
  "reclaim_work_item",
]);
export function availableActions({
  context,
  entity = {},
  criterion,
  protocol,
  permissions = [],
  principal,
  inventory,
  outcome,
}) {
  return Object.entries(humanActions)
    .filter(([name, a]) => {
      const entry = inventory[name];
      if (
        !entry ||
        !permissions.includes(entry.permission) ||
        !a.contexts.includes(context)
      )
        return false;
      if (a.requiresActiveOutcome && outcome?.lifecycle !== "active") return false;
      if (a.states && !a.states.includes(entity.lifecycle)) return false;
      if (
        a.relationType &&
        (entity._kind !== "relation" || entity.relation_type !== a.relationType)
      )
        return false;
      if (
        (a.archived && !entity.archived_at) ||
        (a.notArchived && entity.archived_at)
      )
        return false;
      if (entity.archived_at && !a.archived) return false;
      if (
        a.requiresUnsignedContract &&
        (!entity.current_contract_id ||
          protocol?.phase === "signed_contracts_v2")
      )
        return false;
      if (
        (a.requiresDraft && !entity.draft) ||
        (a.requiresNoDraft && entity.draft)
      )
        return false;
      if (
        entity.lifecycle === "archived" &&
        context === "roadmap" &&
        name !== "reopen_roadmap"
      )
        return false;
      if (a.requiresRevision && !entity.revisions?.length) return false;
      if (
        legacyLeaseNames.has(name) &&
        (protocol?.phase !== "legacy" || entity.contracts_enabled)
      )
        return false;
      if (
        a.lease === "holder" &&
        (!entity.current_lease ||
          entity.current_lease.principal_id !== principal ||
          entity._operational_state?.lease_status !== "active")
      )
        return false;
      if (
        a.lease === 'expired' && (!entity.current_lease || entity._operational_state?.lease_status !== 'expired')
      ) return false;
      if (
        a.lease === "empty" &&
        (entity.current_lease ||
          entity._operational_state?.display_state !== "ready")
      )
        return false;
      if (context === "criterion" && !criterion) return false;
      if (
        context === "criterion" &&
        entity._kind === "work_item" &&
        protocol?.phase === "signed_contracts_v2"
      )
        return false;
      return true;
    })
    .map(([name, a]) => ({ name, ...a }));
}

export const permissionLabels = Object.freeze({
  "work.contract.return": copy.text.workContractReturn,
  "work.contract.complete_direct": copy.text.workContractCompleteDirect,
  "work.review.acquire": copy.text.workReviewAcquire,
  "work.review.decide": copy.text.workReviewDecide,
  "identity.signing_key.enroll": copy.text.identitySigningKeyEnroll,
  "identity.signing_key.rotate": copy.text.identitySigningKeyRotate,
  "identity.signing_key.revoke": copy.text.identitySigningKeyRevoke,
  "work.contract.acquire": copy.text.workContractAcquire,
  "work.contract.revoke": copy.text.workContractRevoke,
  "state:read": copy.text.stateRead,
  "outcome:write": copy.text.outcomeWrite,
  "planning:write": copy.text.planningWrite,
  "work:write": copy.text.workWrite,
  "records:write": copy.text.recordsWrite,
  "assessment:write": copy.text.assessmentWrite,
  "conclusion:write": copy.text.conclusionWrite,
  "namespace:admin": copy.text.namespaceAdmin,
  "integration:write": copy.text.integrationWrite,
  "actor:delegate": copy.text.actorDelegate,
  "work:admin_cancel": copy.text.workAdminCancel,
  "work:admin_complete": copy.text.workAdminComplete,
  "assessment:waive": copy.text.assessmentWaive,
  "credentials:read": copy.text.credentialsRead,
  "audit:read": copy.text.auditRead,
  "objective:create": copy.text.objectiveCreate,
  "objective:update": copy.text.objectiveUpdate,
  "objective:start": copy.text.objectiveStart,
  "objective:conclude": copy.text.objectiveConclude,
  "objective:reopen": copy.text.objectiveReopen,
  "objective:cancel": copy.text.objectiveCancel,
  "objective:configure_completion": copy.text.objectiveConfigureCompletion,
  "task:create": copy.text.taskCreate,
  "task:update_definition": copy.text.taskUpdateDefinition,
  "task:activate": copy.text.taskActivate,
  "task:defer": copy.text.taskDefer,
  "task:assign": copy.text.taskAssign,
  "task:claim": copy.text.taskClaim,
  "task:renew": copy.text.taskRenew,
  "task:release": copy.text.taskRelease,
  "task:deliver": copy.text.taskDeliver,
  "task:review": copy.text.taskReview,
  "task:accept": copy.text.taskAccept,
  "task:request_changes": copy.text.taskRequestChanges,
  "task:withdraw_delivery": copy.text.taskWithdrawDelivery,
  "task:cancel": copy.text.taskCancel,
  "task:reopen": copy.text.taskReopen,
  "task:intervene": copy.text.taskIntervene,
  "criterion:define": copy.text.criterionDefine,
  "criterion:assess": copy.text.criterionAssess,
  "criterion:waive": copy.text.criterionWaive,
  "dependency:write": copy.text.dependencyWrite,
  "issue:create": copy.text.issueCreate,
  "issue:update": copy.text.issueUpdate,
  "issue:assign": copy.text.issueAssign,
  "issue:investigate": copy.text.issueInvestigate,
  "issue:resolve": copy.text.issueResolve,
  "issue:reopen": copy.text.issueReopen,
  "blocker:create": copy.text.blockerCreate,
  "blocker:release": copy.text.blockerRelease,
  "blocker:cancel": copy.text.blockerCancel,
  "credentials:issue": copy.text.credentialsIssue,
  "credentials:update": copy.text.credentialsUpdate,
  "credentials:revoke": copy.text.credentialsRevoke,
  "credentials:rotate": copy.text.credentialsRotate,
  "permissions:manage": copy.text.permissionsManage,
  "policy:manage": copy.text.policyManage,
});

export const permissionDescriptions = Object.freeze({
  "work.contract.return": "Submit a signed work return for receipt and validation.",
  "work.contract.complete_direct": "Request direct completion under the configured signed protocol.",
  "work.review.acquire": "Acquire a review contract for an eligible delivery.",
  "work.review.decide": "Submit a signed review decision under its review contract.",
  "identity.signing_key.enroll": "Authorize enrollment of a signing key for a credential.",
  "identity.signing_key.rotate": "Authorize rotation of an active signing key.",
  "identity.signing_key.revoke": "Revoke an enrolled signing key.",
  "work.contract.acquire": "Acquire an eligible work contract.",
  "work.contract.revoke": "Revoke a work contract through the existing protocol.",
  "state:read": "Read state within the granted scope.",
  "outcome:write": "Update Outcome state and its existing settings.",
  "planning:write": "Create, edit and publish Roadmaps.",
  "work:write": "Use the published legacy work-writing capability.",
  "records:write": "Add or update records, notes, Evidence and Artifacts.",
  "assessment:write": "Use the published legacy assessment capability.",
  "conclusion:write": "Use the published legacy conclusion capability.",
  "namespace:admin": "Create a Namespace and its initial grant.",
  "integration:write": "Configure integrations and delivery settings.",
  "actor:delegate": "Act for another Actor where delegation is explicitly supported.",
  "work:admin_cancel": "Cancel work using the legacy administrative override.",
  "work:admin_complete": "Complete work using the legacy administrative override.",
  "assessment:waive": "Use the published legacy assessment waiver capability.",
  "credentials:read": "Read credential metadata and grants in the Namespace.",
  "audit:read": "Read security audit and operational diagnostic records.",
  "objective:create": "Create Objectives in the granted scope.",
  "objective:update": "Edit Objective definitions, ownership and hierarchy.",
  "objective:start": "Start a planned Objective after domain readiness checks.",
  "objective:conclude": "Record an authorized Objective conclusion.",
  "objective:reopen": "Reopen an Objective with its required reason and checks.",
  "objective:cancel": "Cancel an Objective through its explicit lifecycle command.",
  "objective:configure_completion": "Configure the delegated Objective completion policy.",
  "task:create": "Create Tasks in an Objective context.",
  "task:update_definition": "Edit a Task definition while domain rules allow it.",
  "task:activate": "Move an eligible Task from planning into active work.",
  "task:defer": "Return an eligible Task from active planning to Backlog.",
  "task:assign": "Change assigned participants without granting execution authority.",
  "task:claim": "Acquire execution authority for an eligible Task.",
  "task:renew": "Renew or resume an authorized Task execution lease.",
  "task:release": "Release the caller's Task execution authority.",
  "task:deliver": "Submit Task work for the configured delivery protocol.",
  "task:review": "Acquire or participate in the authorized Task review protocol.",
  "task:accept": "Accept a Task only through its valid conclusion path.",
  "task:request_changes": "Return a reviewed Task delivery for correction.",
  "task:withdraw_delivery": "Withdraw a delivery where the protocol allows it.",
  "task:cancel": "Cancel a Task through its explicit lifecycle command.",
  "task:reopen": "Reopen an eligible Task with its required reason and checks.",
  "task:intervene": "Intervene in an active Task or signed review case with audit.",
  "criterion:define": "Create or revise success criteria where allowed.",
  "criterion:assess": "Record an assessment against the current criterion revision.",
  "criterion:waive": "Waive a criterion through the explicit waiver decision.",
  "dependency:write": "Add, remove or apply dependency relations.",
  "issue:create": "Report an Issue in its owning Objective.",
  "issue:update": "Edit Issue details and relationships.",
  "issue:assign": "Assign or unassign an Issue owner.",
  "issue:investigate": "Start or update an Issue investigation.",
  "issue:resolve": "Resolve, mark duplicate or close an Issue as not planned.",
  "issue:reopen": "Reopen a resolved Issue.",
  "blocker:create": "Create an explicit Blocker relation.",
  "blocker:release": "Release a Blocker through its explicit lifecycle command.",
  "blocker:cancel": "Cancel a Blocker through its explicit lifecycle command.",
  "credentials:issue": "Issue a scoped credential for a granted Principal.",
  "credentials:update": "Update credential metadata and policy bindings.",
  "credentials:revoke": "Revoke a credential in this Namespace.",
  "credentials:rotate": "Rotate credential authority through the supported flow.",
  "permissions:manage": "Create or update explicit Principal permission grants.",
  "policy:manage": "Manage operational, identity and protocol policies.",
});
