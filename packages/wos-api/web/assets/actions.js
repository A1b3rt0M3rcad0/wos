import { copy } from "./en-US.js";
export const humanActions = Object.freeze({
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
    states: ["in_progress"],
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
  create_issue: { label: copy.text.reportIssue, contexts: ["outcome"] },
  report_issue_with_blocker: {
    label: copy.text.reportIssueAndBlockWork,
    contexts: ["outcome"],
  },
  create_blocker: {
    label: copy.text.markWorkAsBlocked,
    contexts: ["outcome", "work_item"],
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
});
const legacyLeaseNames = new Set([
  "claim_work_item",
  "complete_work_item",
  "release_work_item",
]);
export function availableActions({
  context,
  entity = {},
  criterion,
  protocol,
  permissions = [],
  principal,
  inventory,
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
      if (a.states && !a.states.includes(entity.lifecycle)) return false;
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
});
