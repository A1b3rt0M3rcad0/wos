import { copy, message } from "./en-US.js";
import { el, button, details } from "./ui.js";
import { display, kindName, date } from "./presentation.js";
import { humanActions } from "./actions.js";

export const workspaceCommandName = (value) => String(value || "")
  .replace(/([a-z0-9])([A-Z])/g, "$1_$2").toLowerCase();
const signedOperations = new Set([
  "acquire_signed_work_contract", "acquire_work_contract", "acquire_signed_review_contract", "acquire_review_contract",
]);
const sameRef = (a, b) => Boolean(a && b && a.namespace_id === b.namespace_id
  && a.outcome_id === b.outcome_id && a.kind === b.kind && a.id === b.id);
const scoped = (value, subject) => value?.id && value.namespace_id === subject?.namespace_id
  && value.outcome_id === subject?.outcome_id;
const sameSubject = (a, b) => sameRef(a?.ref, b?.ref) && a?.kind === b?.kind
  && a?.criterion_id === b?.criterion_id && a?.revision_number === b?.revision_number
  && a?.record_kind === b?.record_kind && a?.record_id === b?.record_id;

export function workspaceActions(view) {
  const result = new Map();
  for (const capability of view?.capabilities || []) {
    const name = workspaceCommandName(capability.action_id);
    if (!humanActions[name] && !signedOperations.has(name)) continue;
    if (!["browser_command", "signed_host"].includes(capability.transport)) continue;
    if (!sameSubject(capability.subject, view.subject)) continue;
    const existing = result.get(name);
    if (!existing || (capability.presentation_priority ?? 100) < (existing.presentation_priority ?? 100))
      result.set(name, capability);
  }
  return [...result.values()].sort((a, b) =>
    (a.presentation_priority ?? 100) - (b.presentation_priority ?? 100)
    || workspaceCommandName(a.action_id).localeCompare(workspaceCommandName(b.action_id)));
}

function actionButton(label, handler, actionId, className = "quiet") {
  const control = button(label, handler, className);
  control.dataset.actionId = workspaceCommandName(actionId);
  return control;
}

export function renderWorkspaceActions(host, {
  view, onAction, onBlocked, onMoreActions,
} = {}) {
  host.replaceChildren();
  host.classList.add("workspace-primary-actions");
  const actions = workspaceActions(view);
  for (const [index, capability] of actions.slice(0, 3).entries()) {
    const name = workspaceCommandName(capability.action_id);
    const signed = capability.transport === "signed_host";
    const label = signed ? copy.text.workspaceSignedAction : humanActions[name]?.label;
    if (!label) continue;
    const control = actionButton(label, () => {
      if (capability.availability !== "available" && onBlocked) onBlocked(capability);
      else onAction?.(capability.action_id, capability);
    }, name, index === 0 ? "workspace-primary-action" : "quiet");
    control.dataset.availability = capability.availability;
    if (capability.availability !== "available") {
      control.append(el("small", copy.text.workspaceBlockedAction));
      control.setAttribute("aria-description", copy.text.workspaceBlockedAction);
    } else if (signed) control.setAttribute("aria-description", copy.text.workspaceSignedHelp);
    host.append(control);
  }
  if (onMoreActions) host.append(actionButton(copy.text.workspaceMore, onMoreActions, "more_actions"));
  return actions;
}

const reasonCopy = (value) => ({
  required_active_criterion_missing: copy.text.workspaceReasonRequiredCriterion,
  missing_required_criterion: copy.text.workspaceReasonRequiredCriterion,
  definition_frozen: copy.text.workspaceReasonFrozen,
  active_contract: copy.text.workspaceReasonFrozen,
  review_open: copy.text.workspaceReasonFrozen,
  structure_preview_changed: copy.text.workspaceReasonStructureChanged,
  blocked: copy.text.workspaceReasonBlocked,
  dependency_not_satisfied: copy.text.workspaceReasonDependency,
  signed_authority_required: copy.text.workspaceReasonSigned,
})[value] || display(value);

function coverageText(coverage) {
  if (!coverage || coverage.status === "not_loaded") return copy.text.workspaceUnknown;
  if (coverage.status === "unavailable") return copy.text.workspaceUnavailable;
  if (coverage.status === "partial") return copy.text.workspacePartial;
  if (coverage.status === "not_applicable") return copy.text.workspaceNotApplicable;
  return "";
}

const roleCopy = (role) => ({
  declared_owner: copy.text.workspaceDeclaredOwner, declared_assignee: copy.text.workspaceDeclaredAssignee,
  executor: copy.text.workspaceRecordedExecutor, reviewer: copy.text.workspaceCurrentReviewer,
  last_executor: copy.text.workspaceLastExecutor, author: copy.text.workspaceAuthor,
  producer: copy.text.workspaceProducer,
})[role] || display(role);
const authorityCopy = (state) => ({
  current_authority: copy.text.workspaceAuthorityCurrent, authority_suspended: copy.text.workspaceAuthoritySuspended,
  expired: copy.text.workspaceAuthorityExpired,
})[state] || copy.text.workspaceAuthorityUnknown;

export function renderWorkspaceView(host, {
  view, onReference, onAction, onBlocked, onCriterion, onTimelineMore, onNote,
  onFullRecord, onFacetMore, onSubject, onPlan, principalId,
} = {}) {
  host.replaceChildren();
  host.classList.add("workspace-view");
  if (!view?.subject?.ref || !view.header) return;
  const subject = view.subject.ref;
  const kind = view.subject.kind === "criterion" ? "criterion" : subject.kind;
  const record = view.entity || {};
  const actions = workspaceActions(view);
  const capabilities = new Map(actions.map((value) => [workspaceCommandName(value.action_id), value]));
  const headers = new Map((view.relationships?.nodes || []).filter((item) => scoped(item.ref, subject))
    .map((item) => [`${item.ref.kind}/${item.ref.id}`, item]));
  const contextHeaders = [view.placement?.outcome, view.placement?.parent, ...(view.placement?.path || [])];
  for (const item of contextHeaders) if (item && scoped(item.ref, subject))
    headers.set(`${item.ref.kind}/${item.ref.id}`, item);

  function reference(ref, title) {
    if (!scoped(ref, subject)) return el("span", copy.text.workspaceScopeUnknown, "field-help");
    const name = title || headers.get(`${ref.kind}/${ref.id}`)?.title || kindName(ref.kind);
    const control = actionButton(name, () => onReference?.(ref), "open_reference", "quiet workspace-reference");
    control.dataset.referenceKind = ref.kind;
    control.dataset.referenceId = ref.id;
    return control;
  }
  function section(key, title, empty) {
    const node = el("section", undefined, "detail-section workspace-section");
    node.dataset.workspaceFacet = key;
    node.append(el("h3", title));
    const coverage = view.coverage?.[key];
    const note = coverageText(coverage);
    const fallback = () => node.append(el("p",
      coverage?.status === "complete" ? empty : note || copy.text.workspaceUnknown, "workspace-empty"));
    const finish = () => {
      if (note && node.querySelector(".workspace-empty")?.textContent !== note)
        node.append(el("p", note, "workspace-coverage field-help"));
      if (coverage?.next_cursor && onFacetMore && key !== "activity") {
        const label = key === "criteria" ? copy.text.workspaceMoreCriteria
          : key === "roadmap" ? copy.text.workspaceMoreRevisions
          : key === "required_objectives" ? copy.text.workspaceMoreRequiredObjectives : copy.text.workspaceMoreRelations;
        node.append(actionButton(label, () => onFacetMore(key, coverage.next_cursor), `load_more_${key}`));
      }
    };
    host.append(node);
    return { node, fallback, finish };
  }
  function invoke(name) {
    const capability = capabilities.get(name);
    if (!capability) return;
    if (capability.availability !== "available" && onBlocked) onBlocked(capability);
    else onAction?.(capability.action_id, capability);
  }
  function localAction(parent, name, label) {
    if (capabilities.has(name)) parent.append(actionButton(label || humanActions[name]?.label,
      () => invoke(name), name));
  }
  function texts(parent, title, values) {
    if (!values?.length) return;
    parent.append(el("h4", title));
    const list = el("ul", undefined, "workspace-text-list");
    for (const value of values) list.append(el("li", value));
    parent.append(list);
  }

  const placement = el("nav", undefined, "workspace-path");
  placement.setAttribute("aria-label", copy.text.workspaceContext);
  const visited = new Set();
  for (const item of contextHeaders) {
    if (!item?.ref || !scoped(item.ref, subject)
      || (view.subject.kind !== "criterion" && sameRef(item.ref, subject))) continue;
    const key = `${item.ref.kind}/${item.ref.id}`;
    if (visited.has(key)) continue;
    visited.add(key);
    placement.append(reference(item.ref, item.title));
  }
  if (placement.children.length) host.append(placement);
  const state = el("p", undefined, "workspace-state");
  if (view.header.lifecycle) state.append(el("span", display(view.header.lifecycle), "status-badge"));
  if (view.header.display_state && view.header.display_state !== view.header.lifecycle)
    state.append(el("span", display(view.header.display_state), "workspace-operational-state"));
  if (kind === "objective") state.append(el("span", record.required_for_outcome
    ? copy.text.workspaceOutcomeRequired : copy.text.workspaceOutcomeOptional));
  host.append(state);
  const meaning = {
    evidence: copy.text.workspaceEvidenceMeaning, issue: copy.text.workspaceIssueMeaning,
    blocker: copy.text.workspaceBlockerMeaning, artifact: copy.text.workspaceArtifactMeaning,
    decision: copy.text.workspaceDecisionMeaning,
  }[kind];
  if (meaning) host.append(el("p", meaning, "workspace-meaning field-help"));

  const work = section("work", kind === "work_item" ? copy.text.workspaceWork : copy.text.description,
    copy.text.workspaceNoInstructions);
  let hasWork = false;
  const descriptionShown = Boolean(view.header.description
    && !(kind === "artifact" && view.header.description === record.uri));
  if (descriptionShown) {
    work.node.append(el("p", view.header.description, "detail-description"));
    hasWork = true;
  }
  if (kind === "work_item" && view.work) {
    for (const [key, title] of Object.entries({ instructions: copy.text.instructions,
      constraints: copy.text.constraints, deliverables: copy.text.deliverables, scope_hints: copy.text.scopeHints })) {
      texts(work.node, title, view.work[key]);
      hasWork ||= Boolean(view.work[key]?.length);
    }
    if (view.work.context_refs?.length) {
      work.node.append(el("h4", copy.text.contextReferences));
      for (const ref of view.work.context_refs) work.node.append(reference(ref));
      hasWork = true;
    }
  }
  if (!hasWork && kind === "work_item") work.fallback();
  if (kind !== "work_item" && !hasWork) work.node.remove();
  if (kind === "work_item") localAction(work.node, "set_work_item_execution_spec");
  if (view.coverage?.work?.status === "partial" && onFullRecord)
    work.node.append(actionButton(copy.text.workspaceReadFullRecord, () => onFullRecord(view), "read_full_record"));
  if (kind === "work_item") work.finish();

  const people = section("people", ["work_item", "objective", "outcome"].includes(kind)
    ? copy.text.workspacePeople : copy.text.workspaceAuthorship, copy.text.workspaceNoPeople);
  for (const person of view.people || []) {
    const item = el("div", undefined, "workspace-person");
    item.append(el("small", roleCopy(person.role)),
      el("strong", person.actor_ref?.display_name || person.actor_ref?.id || person.principal_id || copy.text.workspaceUnknown));
    if (person.authority_state) item.append(el("span", authorityCopy(person.authority_state)));
    if (person.source) item.append(el("small", display(person.source), "field-help"));
    people.node.append(item);
  }
  if (!view.people?.length) people.fallback();
  people.node.append(el("p", copy.text.workspaceNoPresence, "field-help"));
  people.finish();
  if (kind === "criterion") people.node.remove();

  const attention = section("attention", copy.text.workspaceAttention, copy.text.workspaceNoAttention);
  let hasAttention = false;
  if (view.attention && kind !== "criterion") {
    const current = view.attention;
    texts(attention.node, copy.text.workspaceAttention,
      (current.obligations || []).map((item) => item.description || display(item.status)));
    texts(attention.node, copy.text.workspaceReasonBlocked,
      (current.operational_state?.readiness_reasons || []).map(reasonCopy));
    texts(attention.node, copy.text.changesRequested,
      (current.correction_findings || []).map((item) => item.description || item.summary || item.message).filter(Boolean));
    hasAttention = Boolean(current.obligations?.length || current.operational_state?.readiness_reasons?.length || current.correction_findings?.length);
  }
  const primary = actions[0];
  if (primary && primary.availability !== "available") {
    texts(attention.node, copy.text.workspaceBlockedAction, (primary.reason_codes || []).map(reasonCopy));
    for (const ref of primary.related_refs || []) attention.node.append(reference(ref));
    hasAttention = true;
  }
  if (primary?.transport === "signed_host") {
    attention.node.append(el("p", copy.text.workspaceSignedHelp, "field-help"));
    hasAttention = true;
  }
  if (!hasAttention && kind === "work_item") attention.fallback();
  if (hasAttention || kind === "work_item") attention.finish(); else attention.node.remove();

  if (kind === "outcome" && view.subject.kind === "entity" && view.coverage?.required_objectives) {
    const required = section("required_objectives", copy.text.workspaceRequiredObjectives, copy.text.workspaceNoRequiredObjectives);
    required.node.append(el("p", copy.text.workspaceRequiredObjectivesHelp, "field-help"));
    for (const objective of view.required_objectives || []) {
      const row = el("p", undefined, "workspace-path");
      row.append(reference(objective.ref, objective.title), el("span", display(objective.lifecycle)));
      required.node.append(row);
    }
    if (!view.required_objectives?.length) required.fallback();
    if (view.coverage.required_objectives.total_count !== undefined)
      required.node.append(el("small", message("workspaceRequiredObjectiveCount", view.required_objectives?.length || 0,
        view.coverage.required_objectives.total_count)));
    required.finish();
  }
  if (view.coverage?.criteria?.status !== "not_applicable") {
    const criteria = section("criteria", copy.text.workspaceCriteria, copy.text.workspaceNoCriteria);
    for (const item of view.criteria || []) {
      const definition = item.definition;
      if (!definition || !scoped(definition.owner_ref, subject)) continue;
      const row = el("article", undefined, "workspace-criterion");
      row.dataset.criterionId = definition.id;
      if (kind === "criterion") row.append(el("h4", copy.text.workspaceCurrentDefinition));
      else row.append(actionButton(definition.title, () => onCriterion?.(definition, definition.owner_ref), "open_criterion", "quiet workspace-criterion-title"));
      const badges = el("p", undefined, "workspace-criterion-state");
      badges.append(el("span", definition.required ? copy.text.workspaceRequired : copy.text.workspaceOptional),
        el("span", display(definition.verification_mode)));
      if (definition.status === "retired") badges.append(el("span", copy.text.workspaceCriterionRetired));
      let assessment = copy.text.workspaceUnassessed;
      if (item.assessment) assessment = item.assessment_applicable
        ? display(item.assessment.result) : copy.text.workspaceAssessmentOutdated;
      badges.append(el("strong", assessment));
      row.append(badges);
      if (definition.description) row.append(el("p", definition.description));
      if (item.evidence_linked) row.append(el("p", copy.text.workspaceLinkedEvidence, "field-help"));
      if (item.assessment?.rationale) row.append(el("p", item.assessment.rationale, "workspace-assessment-rationale"));
      if (item.assessment?.actor_ref || item.assessment?.principal_id)
        row.append(el("small", `${copy.text.workspaceAssessedBy}: ${item.assessment.actor_ref?.id || item.assessment.principal_id}`));
      if (item.assessment?.submission_id && onSubject) row.append(actionButton(copy.text.workspaceReviewedSubmission,
        () => onSubject({ kind: "execution_record", ref: definition.owner_ref,
          record_kind: "submission", record_id: item.assessment.submission_id }), "open_submission"));
      row.append(el("small", message("workspaceDefinitionRevision", definition.criterion_revision)));
      criteria.node.append(row);
    }
    if (!view.criteria?.length) criteria.fallback();
    localAction(criteria.node, "add_criterion");
    if (view.subject.kind === "criterion") localAction(criteria.node, "register_evidence_with_links", copy.text.workspaceAddEvidence);
    if (view.subject.kind === "criterion" && onFullRecord)
      criteria.node.append(actionButton(copy.text.workspaceCriterionHistory,
        () => onFullRecord(view), "read_criterion_history"));
    criteria.finish();
  }

  if (kind === "roadmap") {
    const plan = section("roadmap", copy.text.roadmap, "");
    plan.node.append(el("p", copy.text.entityContextRoleRoadmap, "field-help"));
    const roadmap = view.roadmap;
    const draft = roadmap?.draft || record.draft;
    if (draft) {
      const item = el("article", undefined, "workspace-plan-record");
      item.append(el("h4", copy.text.draft), el("p", display(draft.lifecycle)));
      if (draft.reason) item.append(el("p", draft.reason));
      if (draft.lifecycle === "open" && onPlan) item.append(actionButton(copy.text.workspaceViewDraft,
        () => onPlan({ roadmap_id: subject.id, draft: true }), "view_roadmap_draft"));
      plan.node.append(item);
    }
    const revisions = view.subject.kind === "roadmap_revision"
      ? [roadmap?.selected_revision || record] : roadmap?.revisions || record.revisions || [];
    for (const revision of revisions) {
      const item = el("article", undefined, "workspace-plan-record");
      item.append(el("h4", message("workspacePublishedRevision", revision.revision_number)));
      if (revision.reason) item.append(el("p", revision.reason));
      if (revision.published_at) item.append(el("small", date(revision.published_at)));
      if (onPlan) item.append(actionButton(copy.text.workspaceViewPlan,
        () => onPlan({ roadmap_id: subject.id, revision_number: revision.revision_number }), "view_roadmap_revision"));
      if (onSubject && view.subject.kind !== "roadmap_revision")
        item.append(actionButton(copy.text.workspaceInspectRevision,
          () => onSubject({ kind: "roadmap_revision", ref: subject, revision_number: revision.revision_number }), "inspect_roadmap_revision"));
      for (const node of revision.nodes || []) {
        const line = el("p", undefined, "workspace-plan-node");
        line.append(el("small", display(node.node_type)), el("span", node.title));
        if (node.target_ref) line.append(reference(node.target_ref, node.published_reference_snapshot?.title));
        item.append(line);
      }
      plan.node.append(item);
    }
    if (!draft && !revisions.length) plan.node.append(el("p", copy.text.workspaceNoPlanContent, "field-help"));
    if (roadmap) plan.node.append(el("p", roadmap.is_active
      ? message("workspaceActiveRevision", roadmap.active_revision_number)
      : copy.text.workspacePlanNotActive, "workspace-plan-state"));
    plan.node.append(el("p", copy.text.workspacePlanActivationMeaning, "field-help"));
    if (onFullRecord) plan.node.append(actionButton(copy.text.workspaceReadFullRecord,
      () => onFullRecord(view), "read_full_record"));
    plan.finish();
  }

  if (view.subject.kind === "execution_record") {
    const execution = section("execution", copy.text.workspaceExecutionRecord, "");
    const fields = {
      status: copy.text.workspaceRecordedStatus, effective_status: copy.text.workspaceAuthorityCurrent,
      holder_principal_id: copy.text.workspaceRecordedExecutor,
      acquired_at: copy.text.workspaceAcquiredAt, expires_at: copy.text.workspaceExpiresAt,
      submitted_at: copy.text.workspaceSubmittedAt, summary: copy.text.description,
    };
    for (const [key, label] of Object.entries(fields)) if (record[key]) {
      const row = el("p", undefined, "workspace-record-fact");
      row.append(el("small", label), el("span", key.endsWith("_at") ? date(record[key])
        : key.includes("status") ? display(record[key]) : record[key]));
      execution.node.append(row);
    }
    execution.node.append(el("p", copy.text.workspaceDeliveryPending, "field-help"));
    if (onSubject) execution.node.append(actionButton(copy.text.workspaceReturnToTask,
      () => onSubject({ kind: "entity", ref: subject }), "open_reference"));
  }

  const delivery = section("delivery", kind === "criterion" ? copy.text.workspaceOwnerDelivery : copy.text.workspaceLatestDelivery, copy.text.workspaceNoDelivery);
  if (view.delivery) {
    delivery.node.append(el("p", display(view.delivery.status)),
      el("p", copy.text.workspaceDeliveryPending, "field-help"));
    if (view.delivery.summary) delivery.node.append(el("p", view.delivery.summary));
  } else if (kind === "work_item") delivery.fallback();
  else delivery.node.remove();
  if (kind === "work_item") delivery.finish();

  if (record.conclusion) {
    const conclusion = record.conclusion;
    const current = section("conclusion", copy.text.currentConclusion, "");
    const card = el("article", undefined, "conclusion-card workspace-conclusion");
    card.append(el("strong", display(conclusion.lifecycle_result || view.header.lifecycle)),
      el("p", conclusion.reason),
      el("small", conclusion.actor_ref?.display_name || conclusion.actor_ref?.id || conclusion.principal_id),
      el("small", date(conclusion.concluded_at)));
    if (conclusion.assessments?.length) {
      card.append(el("h4", copy.text.workspaceConclusionAssessments));
      for (const assessment of conclusion.assessments) {
        const owner = assessment.owner_ref || conclusion.owner_ref || subject;
        const snapshot = el("div", undefined, "workspace-conclusion-assessment");
        snapshot.append(el("span", display(assessment.result)),
          el("small", message("workspaceDefinitionRevision", assessment.criterion_revision)),
          actionButton(copy.text.workspaceOpenCriterion,
            () => onCriterion?.({ id: assessment.criterion_id, owner_ref: owner }, owner), "open_criterion"));
        card.append(snapshot);
      }
    }
    if (conclusion.obligations?.required_objective_ids?.length) {
      card.append(el("h4", copy.text.workspaceRequiredObjectives));
      for (const id of conclusion.obligations.required_objective_ids)
        card.append(reference({ ...subject, kind: "objective", id }));
    }
    if (record.result_summary) card.append(el("p", record.result_summary));
    current.node.append(card);
    if (onFullRecord) current.node.append(actionButton(copy.text.workspacePriorConclusions,
      () => onFullRecord(view), "read_full_record"));
  }

  const relations = section("relationships", copy.text.workspaceRelations, copy.text.workspaceNoRelations);
  for (const edge of view.relationships?.items || []) {
    if (!scoped(edge.source_ref, subject) || !scoped(edge.target_ref, subject)) continue;
    const row = el("div", undefined, "workspace-relationship");
    const label = {
      ownership: copy.text.workspaceRelationOwnership, hierarchy: copy.text.workspaceRelationOwnership,
      depends_on: edge.strength === "advisory" ? copy.text.workspaceRelationAdvisory : copy.text.workspaceRelationDependency,
      advisory: copy.text.workspaceRelationAdvisory, affected: copy.text.workspaceRelationAffected,
      blocks: copy.text.workspaceRelationBlocked, causes: copy.text.workspaceRelationCause,
      supports: copy.text.workspaceRelationSupports, contradicts: copy.text.workspaceRelationContradicts,
      context: copy.text.workspaceRelationContext, produces: copy.text.workspaceRelationProduces,
      relates_to: copy.text.workspaceRelationRelated, derived_from: copy.text.workspaceRelationDerived,
    }[edge.kind] || display(edge.kind);
    const membership = edge.kind === "ownership" || edge.kind === "hierarchy";
    row.append(reference(membership ? edge.target_ref : edge.source_ref),
      el("span", label, "workspace-relationship-label"), reference(membership ? edge.source_ref : edge.target_ref));
    if (edge.strength) row.append(el("small", display(edge.strength), "field-help"));
    if (edge.kind === "depends_on" && !edge.strength)
      row.append(el("small", copy.text.workspaceRelationStrengthUnknown, "field-help"));
    if (edge.rationale) row.append(el("p", edge.rationale));
    if (["relation", "evidence_link"].includes(edge.source_of_truth)) {
      const ref = { ...subject, kind: edge.source_of_truth, id: edge.id };
      row.append(actionButton(copy.text.workspaceInspectRelation, () => onReference?.(ref), "open_reference"));
    }
    if (edge.criterion_id) row.append(actionButton(copy.text.workspaceOpenCriterion,
      () => onCriterion?.({ id: edge.criterion_id, owner_ref: edge.target_ref }, edge.target_ref), "open_criterion"));
    relations.node.append(row);
  }
  if (!view.relationships?.items?.length) relations.fallback();
  relations.finish();

  const activity = section("activity", copy.text.workspaceRecentActivity, copy.text.workspaceNoActivity);
  const timeline = view.activity;
  for (const entry of timeline?.items || []) {
    const item = el("article", undefined, "workspace-activity-entry");
    item.dataset.activityId = entry.id;
    const origin = entry.origin === "entity_note"
      ? entry.record_type === "observation" ? copy.text.workspaceObservation : copy.text.workspaceDocumentaryUpdate
      : entry.origin === "work_activity" ? copy.text.workspaceContractActivity : copy.text.workspaceDomainEvent;
    item.append(el("small", origin, "workspace-activity-origin"));
    if (entry.origin === "domain_event" && entry.event_type)
      item.append(el("strong", display(entry.event_type.replaceAll(".", "_"))));
    const repeatsCurrentDescription = entry.origin === "domain_event" && descriptionShown
      && sameRef(entry.subject_ref, subject) && entry.description === view.header.description;
    if (entry.description && !repeatsCurrentDescription) item.append(el("p", entry.description));
    if (entry.principal_id || entry.actor_ref?.id)
      item.append(el("small", entry.actor_ref?.display_name || entry.actor_ref?.id || entry.principal_id, "workspace-activity-author"));
    item.append(el("small", `${copy.text.workspaceRecordedAt}: ${date(entry.recorded_at)}`));
    if (entry.occurred_at) item.append(el("small", `${copy.text.workspaceOccurredAt}: ${date(entry.occurred_at)}`));
    if (entry.descendant_origin || !sameRef(entry.subject_ref, subject)) {
      item.append(el("small", copy.text.workspaceFromRelatedItem), reference(entry.subject_ref));
    }
    if (entry.supersedes_note_id) item.append(el("p", copy.text.workspaceCorrectsUpdate, "field-help"));
    if (entry.origin === "entity_note" && onNote) {
      item.append(actionButton(copy.text.workspaceReadFullUpdate, () => onNote(entry), "read_entity_note"));
      if (principalId && principalId === entry.principal_id && capabilities.has("append_entity_note"))
        item.append(actionButton(copy.text.workspaceCorrectUpdate, () => onNote(entry, { mode: "correct" }), "correct_entity_note"));
    }
    for (const id of entry.artifact_refs || []) item.append(reference({ ...subject, kind: "artifact", id }));
    for (const id of entry.evidence_refs || []) item.append(reference({ ...subject, kind: "evidence", id }));
    activity.node.append(item);
  }
  if (!timeline?.items?.length) activity.fallback();
  if (timeline?.has_newer_entries) activity.node.append(el("p", copy.text.workspaceNewActivity, "field-help"));
  if (timeline?.next_cursor && onTimelineMore) activity.node.append(actionButton(copy.text.workspaceMoreActivity,
    () => onTimelineMore(timeline), "load_more_activity"));
  if (timeline) activity.node.append(el("small", message("workspaceHistoryBoundary", timeline.history_through_revision)),
    el("p", copy.text.workspaceFrozenHistory, "field-help"));
  localAction(activity.node, "append_entity_note", copy.text.workspaceAddUpdate);
  activity.finish();

  if (kind === "evidence" && view.observation) {
    const observation = view.observation;
    const source = section("observation", copy.text.workspaceObservationSource, "");
    if (observation.status === "resolved") {
      source.node.append(el("p", observation.preview || ""),
        el("small", `${copy.text.workspaceAuthor}: ${observation.actor_ref?.display_name || observation.actor_ref?.id || observation.principal_id}`),
        el("small", `${copy.text.workspaceRecordedAt}: ${date(observation.recorded_at)}`));
      if (observation.occurred_at) source.node.append(el("small", `${copy.text.workspaceOccurredAt}: ${date(observation.occurred_at)}`));
      if (observation.subject_ref) source.node.append(reference(observation.subject_ref));
      if (observation.note_id && onNote) source.node.append(actionButton(copy.text.workspaceReadFullUpdate,
        () => onNote({ source: { id: observation.note_id }, subject_ref: observation.subject_ref,
          criterion_id: observation.criterion_id }), "read_entity_note"));
      source.node.append(el("p", copy.text.workspaceObservationSourceHelp, "field-help"));
    } else source.node.append(el("p", copy.text.workspaceObservationSourceUnavailable, "field-help"));
  }
  if (["evidence", "artifact", "issue", "blocker", "decision", "relation", "evidence_link"].includes(kind)) {
    const facts = section("record", copy.text.workspaceSource, "");
    const fields = {
      severity: copy.text.severity, captured_at: copy.text.workspaceObservedAt,
      registered_at: copy.text.workspaceRecordedAt, resolution_summary: copy.text.workspaceResolution,
      release_condition: copy.text.workspaceReleaseCondition, proposal: copy.text.proposal,
      chosen_alternative: copy.text.chosenAlternative, rationale: copy.text.rationale,
      reason: copy.text.reason, strength: copy.text.workspaceDependencyStrength,
      uri: "URL",
    };
    for (const [key, label] of Object.entries(fields)) {
      if (!record[key]) continue;
      if (["proposal", "rationale"].includes(key) && descriptionShown
        && record[key] === view.header.description) continue;
      const value = key.endsWith("_at") ? date(record[key]) : record[key];
      const line = el("p", undefined, "workspace-record-fact");
      line.append(el("small", label), el("span", value));
      if (key === "uri") {
        try {
          const url = new URL(value);
          if (["http:", "https:"].includes(url.protocol)) {
            const link = el("a", value);
            link.href = url.href;
            link.target = "_blank";
            link.rel = "noopener noreferrer";
            line.lastChild.replaceWith(link);
          }
        } catch { /* Keep authored non-URL references as inert text. */ }
      }
      facts.node.append(line);
    }
    if (record.source_ref?.kind) facts.node.append(reference(record.source_ref));
    else if (record.source_ref) {
      const source = el("p", undefined, "workspace-record-fact");
      source.append(el("small", copy.text.workspaceSource),
        el("span", record.source_ref.display_name || record.source_ref.provider || copy.text.workspaceUnknown),
        el("span", record.source_ref.uri || record.source_ref.id || ""));
      facts.node.append(source);
    }
    if (record.target_ref) facts.node.append(reference(record.target_ref));
    if (record.evidence_id) facts.node.append(reference({ ...subject, kind: "evidence", id: record.evidence_id }));
    if (record.criterion_id && record.target_ref)
      facts.node.append(actionButton(copy.text.workspaceOpenCriterion,
        () => onCriterion?.({ id: record.criterion_id, owner_ref: record.target_ref }, record.target_ref), "open_criterion"));
    if (record.blocked_ref) facts.node.append(reference(record.blocked_ref));
    if (record.cause_ref) facts.node.append(reference(record.cause_ref));
    if (facts.node.children.length === 1) facts.node.remove();
    else host.insertBefore(facts.node, people.node);
  }
  const audit = details(copy.text.workspaceAudit);
  audit.classList.add("workspace-audit");
  audit.append(el("p", message("workspaceSnapshot", view.outcome_revision)));
  const payload = el("pre");
  payload.textContent = JSON.stringify({ subject: view.subject, entity_version: view.entity_version,
    evaluated_at: view.evaluated_at, coverage: view.coverage }, null, 2);
  audit.append(payload);
  if (onFullRecord) audit.append(actionButton(copy.text.workspaceReadFullRecord, () => onFullRecord(view), "read_full_record"));
  host.append(audit);
  return { actions };
}

// This focused rendering consumes the authorized canonical owner returned by
// the existing full-record read. Historical records never become current gates.
export function renderCriterionHistory(host, { owner, criterionId, onReference } = {}) {
  const section = el("section", undefined, "detail-section workspace-criterion-history");
  section.append(el("h3", copy.text.workspaceCriterionHistory),
    el("p", copy.text.workspaceCriterionHistoryMeaning, "field-help"));
  const definitions = (owner?.criteria?.definition_revisions || [])
    .filter((value) => value.criterion_id === criterionId);
  const assessments = (owner?.criteria?.assessments || [])
    .filter((value) => value.criterion_id === criterionId);
  for (const definition of definitions) {
    const row = el("article", undefined, "workspace-criterion");
    row.append(el("strong", definition.title),
      el("small", message("workspaceDefinitionRevision", definition.criterion_revision)),
      el("span", display(definition.status)), el("span", display(definition.verification_mode)));
    if (definition.description) row.append(el("p", definition.description));
    section.append(row);
  }
  for (const assessment of assessments) {
    const row = el("article", undefined, "workspace-criterion");
    row.append(el("strong", display(assessment.result)),
      el("small", message("workspaceDefinitionRevision", assessment.criterion_revision)),
      el("p", assessment.rationale),
      el("small", assessment.actor_ref?.id || assessment.principal_id || ""));
    if (assessment.assessed_at) row.append(el("small", date(assessment.assessed_at)));
    if (onReference && assessment.owner_ref)
      for (const id of assessment.evidence_ids || []) row.append(actionButton(copy.text.evidence,
        () => onReference({ ...assessment.owner_ref, kind: "evidence", id }), "open_reference"));
    section.append(row);
  }
  if (!definitions.length && !assessments.length)
    section.append(el("p", copy.text.workspaceNoCriterionHistory, "field-help"));
  host.append(section);
  return section;
}
