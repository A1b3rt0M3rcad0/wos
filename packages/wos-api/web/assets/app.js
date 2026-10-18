import { message } from "./en-US.js";
import { renderEditorial, sectionIllustration } from "./editorial.js";
import { contractViews } from "./contracts-view.js";
import { request as networkRequest } from "./api-client.js";
import { createCanvasData, canvasRevision } from "./canvas-data.js";
import { mountWorkspaceCanvas } from "./workspace-canvas.js";
import { createWorkspaceInspector } from "./workspace-inspector.js";
import { renderCanvasContext } from "./canvas-context.js";
import { buildEntityContext, renderEntityContext } from "./workspace-entity-context.js";
import { renderActionMenu } from "./workspace-action-menu.js";
import {
  areas,
  areaFor,
  readRoute,
  routeQuery,
  workSections,
} from "./navigation.js";
import { humanActions, availableActions, permissionLabels } from "./actions.js";
import { humanForm } from "./human-forms.js";
import { createIntent, sameScope, reconcileDraft } from "./intents.js";
import { button as actionButton, conflictComparison } from "./ui.js";
import { copy, initializeCopy } from "./en-US.js";
initializeCopy(document);
import {
  icon,
  display,
  kindName,
  tone,
  badge,
  shortId,
  date,
  formatValue,
  donut,
} from "./presentation.js";
const $ = (id) => document.getElementById(id);
const state = {
  namespace: "",
  permissions: [],
  principal: "",
  actorKey: "",
  humanForm: null,
  formScope: null,
  formGeneration: 0,
  submitting: false,
  technical: false,
  protocol: null,
  outcome: null,
  snapshot: null,
  section: "work_items",
  view: "canvas",
  workFilterReset: false,
  area: "canvas",
  restoringRoute: false,
  generation: 0,
  renderId: 0,
  loaded: new Map(),
  boardLoaded: new Map(),
  selected: null,
  entities: new Map(),
  criterion: null,
  contractView: null,
  submission: null,
  detailGeneration: 0,
  catalog: [],
  next: "",
  pending: null,
  refreshGeneration: 0,
  canvasEpoch: 0,
  canvasData: null,
  canvasView: null,
  canvasModel: null,
  canvasSelected: null,
  entityContextRefresh: null,
  canvasPlan: null,
  formTarget: null,
  formOriginCanvas: false,
  formEpoch: 0,
};
let outcomeSearchTimer;
let outcomeSearchController;
let outcomeSearchEpoch = 0;
let outcomeSearchLoading = false;
let outcomeSearchQuery = "";
const canvasInspector = createWorkspaceInspector({
  host: $("canvas-details"),
  dialogs: {
    detail: $("detail-dialog"),
    command: $("command-dialog"),
    action: $("action-dialog"),
  },
  onClose: (surface, { nextSurface, reason }) => {
    if (
      surface === "form" &&
      (state.pending || state.submitting) &&
      reason !== "reset"
    ) {
      queueMicrotask(presentForm);
      notice(
        copy.text.resolveThePreviousUncertainResponseWithRetryTheSame,
        true,
      );
      return;
    }
    if (surface === "form" && !state.pending) {
      state.formEpoch++;
      state.humanForm?.dispose();
      if (
        nextSurface === "detail" &&
        state.canvasSelected &&
        reason !== "reset"
      )
        showEntity(state.canvasSelected, { preserveCamera: true }).catch(
          report,
        );
    }
    if (!nextSurface && surface !== "form") {
      state.detailGeneration++;
      const selected = state.canvasSelected;
      state.canvasSelected = null;
      if (state.canvasModel)
        state.canvasView?.update(canvasPresentation(), { selectedKey: null });
      if (selected && reason !== "reset") {
        const key = `${selected.kind}:${selected.id}`;
        state.canvasView?.focus(key);
      }
      writeRoute(null);
    }
  },
});
function canvasActive() {
  return state.area === "canvas" && Boolean(state.outcome);
}
function canvasNavigationLocked() {
  return (
    (state.formOriginCanvas && (state.submitting || Boolean(state.pending))) ||
    (canvasInspector.enabled && canvasInspector.isVisible("form"))
  );
}
function allowCanvasNavigation() {
  if (!canvasNavigationLocked()) return true;
  notice(
    state.pending
      ? copy.text.resolveThePreviousUncertainResponseWithRetryTheSame
      : "Finish or close the current change before selecting another item.",
    true,
  );
  return false;
}
function presentForm() {
  closeWorkspaceDisclosures();
  if (canvasInspector.enabled && !state.technical) {
    canvasInspector.showForm();
    revealCanvasInspector();
  } else if (!$("command-dialog").open) $("command-dialog").showModal();
}
function revealCanvasInspector() {
  closeWorkspaceDisclosures();
  const host = $("canvas-details");
  // The inspector stays in the workspace. Its own body scrolls; opening it
  // must not move the document or replace focus inside an existing form.
  if (!host.hidden && !host.contains(document.activeElement))
    host
      .querySelector(".canvas-inspector-slot:not([hidden])")
      ?.focus({ preventScroll: true });
}

function closeWorkspaceDisclosures({ except = null } = {}) {
  for (const id of ["outcome-views", "outcome-tools", "canvas-context"]) {
    const disclosure = $(id);
    if (disclosure && disclosure !== except) {
      const returnFocus =
        disclosure.open && disclosure.contains(document.activeElement);
      disclosure.open = false;
      if (returnFocus)
        disclosure.querySelector("summary")?.focus({ preventScroll: true });
    }
  }
}

document.addEventListener("click", (event) => {
  for (const id of ["outcome-views", "outcome-tools", "canvas-context"]) {
    const disclosure = $(id);
    if (disclosure?.open && !disclosure.contains(event.target))
      disclosure.open = false;
  }
});
document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape" || event.defaultPrevented || event.isComposing)
    return;
  const disclosure = event.target.closest?.(
    ".outcome-heading details[open]",
  );
  if (
    !disclosure ||
    event.target.closest?.(
      'select, [role="listbox"], [role="combobox"][aria-expanded="true"], dialog[open]',
    )
  )
    return;
  disclosure.open = false;
  disclosure.querySelector("summary")?.focus({ preventScroll: true });
  event.preventDefault();
});
$("outcome-tools").addEventListener("click", (event) => {
  const button = event.target.closest?.(".outcome-tool-options button");
  if (!button) return;
  $("outcome-tools").open = false;
  if (document.activeElement === button)
    $("outcome-tools").querySelector("summary").focus({ preventScroll: true });
});
function closeForm() {
  if (canvasInspector.enabled) canvasInspector.close("form");
  if ($("command-dialog").open) $("command-dialog").close();
}
function closeDetail({ clearSelection = true } = {}) {
  state.detailGeneration++;
  if (canvasInspector.enabled) canvasInspector.close("detail");
  if ($("detail-dialog").open) $("detail-dialog").close();
  if (clearSelection) state.canvasSelected = null;
}
function closeActions() {
  if (canvasInspector.enabled) canvasInspector.close("actions");
  if ($("action-dialog").open) $("action-dialog").close();
}
function resetCanvas() {
  state.canvasEpoch++;
  state.canvasData?.dispose();
  state.canvasData = null;
  state.canvasView?.dispose();
  state.canvasView = null;
  state.canvasModel = null;
  state.canvasSelected = null;
  state.entityContextRefresh = null;
  state.canvasPlan = null;
  state.canvasPlanLabel = "";
  canvasInspector.closeAll();
  canvasInspector.setEnabled(false);
}
const labels = {
  contract_id: copy.text.contract,
  execution_id: copy.text.execution,
  spec_digest: copy.text.specificationIntegrity,
  expected_contract_version: copy.text.contractVersion,
  expected_work_item_version: copy.text.taskVersion,
  expected_lease_version: copy.text.leaseVersion,
  submission_id: copy.text.reviewedSubmission,
  supersedes_submission_id: copy.text.previousSubmission,
  authority: copy.text.executionAuthority,
  checkpoint: copy.text.materialProgress,
  material: copy.text.submittedMaterial,
  blocked_ref: copy.text.blockedItem,
  cause_ref: copy.text.blockingCause,
  resolution_summary: copy.text.resolutionSummary,
  severity: copy.text.severity,
  propagation: copy.text.propagation,
  captured_at: copy.text.observationDate,
  source_ref: copy.text.source,
  source_version: copy.text.sourceVersion,
  evidence_id: copy.text.evidence,
  artifact_id: copy.text.artifact,
  base_revision_number: copy.text.baseRevision,
  active_plan_references: copy.text.activePlanTasks,
  required_criteria_met: copy.text.criteriaMet,
  required_criteria_waived: copy.text.criteriaWaivedNotVerified,
  work_completion: copy.text.taskProgress,
  objective_completion: copy.text.objectivesAchieved,
  required_objectives_completion: copy.text.requiredObjectivesAchieved,
  work_items: copy.text.tasks,
  objectives: copy.text.objectives,
  ready_work: copy.text.ready,
  blocked_work: copy.text.blocked,
  in_progress_work: copy.text.inProgress,
  scheduled_work: copy.text.scheduled,
  attention_needed_work: copy.text.needsAttention,
  issues: copy.text.issues,
  blockers: copy.text.blockers,
  decisions: copy.text.decisions,
  evidence: copy.text.evidence,
  artifacts: copy.text.artifacts,
  active_roadmaps: copy.text.activeRoadmaps,
  conclusion_contestations: copy.text.contestedConclusions,
  timeline: copy.text.activity,
  roadmaps: copy.text.allRoadmaps,
  graph: copy.text.relations,
  title: copy.text.title,
  description: copy.text.description,
  desired_state: copy.text.desiredOutcome,
  reason: copy.text.reason,
  rationale: copy.text.rationale,
  result_summary: copy.text.taskResult,
  required: copy.text.required,
  priority: copy.text.priority,
  lifecycle: copy.text.lifecycle,
  verification_mode: copy.text.verificationMethod,
  expected_version: copy.text.observedVersion,
  expected_roadmap_version: copy.text.roadmapVersion,
  expected_draft_version: copy.text.draftVersion,
  criterion_revision: copy.text.criterionRevision,
  ttl_seconds: copy.text.reservationDurationSeconds,
  owner: copy.text.assessedItem,
  scope: copy.text.workspace,
  namespace_id: "Namespace",
  outcome_id: copy.text.outcome,
  objective_id: copy.text.objective,
  work_item_id: copy.text.task4bc74b,
  claim_id: copy.text.reservation,
  fencing_token: copy.text.fencingProtection,
  criterion_id: copy.text.successCriterion,
  evidence_ids: copy.text.evidenceUsed,
  evaluator_ref: copy.text.externalEvaluator,
  target_ref: copy.text.relatedItem,
  source_ref: copy.text.relationSource,
  reference_ref: copy.text.roadmapReference,
  parent_objective_id: copy.text.parentObjective,
  nodes: copy.text.roadmapItems,
  after: copy.text.planOrdering,
  parent_node_key: copy.text.parentGroup,
  node_key: copy.text.nodeKey,
  planned_start: copy.text.plannedStart,
  planned_end: copy.text.plannedEnd,
  metadata: copy.text.additionalInformation,
  revision_number: copy.text.publishedRevision,
  kind: copy.text.type,
  id: copy.text.identifier,
  provider: copy.text.provider,
  result: copy.text.assessment,
  stance: copy.text.evidenceStance,
  artifact_type: copy.text.artifactType,
  evidence_type: copy.text.evidenceType,
  producer_ref: copy.text.producedBy,
  name: copy.text.name,
  uri: copy.text.url,
  chosen_alternative: copy.text.chosenAlternative,
  alternatives: copy.text.alternatives,
  proposal: copy.text.proposal,
  required_for_outcome: copy.text.requiredForOutcome,
};
const choices = {
  severity: ["major", "critical", "minor", "informational"],
  propagation: ["direct", "subtree"],
  evidence_type: [
    "test_result",
    "inspection",
    "measurement",
    "attestation",
    "source",
    "external_evaluation",
  ],
  priority: ["normal", "high", "critical", "low"],
  verification_mode: ["attestation", "evidence_review", "external_evaluation"],
  result: ["met", "not_met", "inconclusive", "waived"],
  stance: ["supports", "contradicts", "neutral"],
  lifecycle: ["todo", "backlog"],
  scope_kind: ["outcome", "objective"],
  node_type: ["reference", "phase", "milestone"],
  kind: ["human", "agent", "service", "automation", "external_system"],
};
const human = (s) =>
  labels[s] ||
  {
    work_item: copy.text.task4bc74b,
    objective: copy.text.objective,
    outcome: copy.text.outcome,
    roadmap: copy.text.roadmap,
    created_at: copy.text.created,
    updated_at: copy.text.updated,
    archived_at: copy.text.archived,
    version: copy.text.version,
    principal_id: copy.text.principal,
    actor_ref: copy.text.actor,
    concluded_at: copy.text.concluded,
    owner_ref: copy.text.item,
    entity_ref: copy.text.item,
    recorded_at: copy.text.recorded,
    node_type: copy.text.itemType,
    content_hash: copy.text.contentHash,
    checksum: copy.text.checksum,
    obligations: copy.text.requirements,
    assessments: copy.text.assessments,
    criteria: copy.text.successCriteria,
    external_context: copy.text.externalContext,
    assignee_refs: copy.text.assignees,
    not_before: copy.text.scheduledStart,
    reference: copy.text.reference,
  }[s] ||
  display(s);
const verbs = {
  acquire: copy.text.acquire,
  resume: copy.text.resume,
  sync: copy.text.saveProgressFor,
  submit: copy.text.submit,
  finalize: copy.text.finalize,
  revoke: copy.text.revoke,
  create: copy.text.create,
  register: copy.text.record,
  add: copy.text.add,
  update: copy.text.edit,
  activate: copy.text.activate,
  start: copy.text.start,
  claim: copy.text.reserve,
  renew: copy.text.renewReservationFor,
  reclaim: copy.text.reclaimExpiredReservationFor,
  release: copy.text.releaseReservationFor,
  complete: copy.text.complete,
  achieve: copy.text.markAsAchieved,
  reopen: copy.text.reopen,
  archive: copy.text.archive,
  unarchive: copy.text.unarchive,
  record: copy.text.record,
  publish: copy.text.publish,
  replace: copy.text.edit,
  resolve: copy.text.resolve,
  retract: copy.text.retract,
  cancel: copy.text.cancel,
  remove: copy.text.remove,
  retire: copy.text.retire,
  supersede: copy.text.supersede,
  investigate: copy.text.investigate,
  open: copy.text.open,
  set: copy.text.set,
  clear: copy.text.clear,
  delete: copy.text.delete,
  redeliver: copy.text.redeliver,
};
const actionSubjects = {
  work_contract: copy.text.contractcc8321,
  next_work_contract: copy.text.nextContract,
  work_result: copy.text.submission,
  criterion_assessment: copy.text.criterionAssessment,
  criterion: copy.text.successCriterion048852,
  criterion_definition: copy.text.criterionDefinition,
  roadmap_draft: copy.text.roadmapDraft,
  roadmap_revision: copy.text.roadmapRevision,
  work_item: copy.text.task,
  outcome: copy.text.outcome25a63e,
  objective: copy.text.objectived454be,
  evidence_link: copy.text.evidenceLink,
  evidence: copy.text.evidenceee8250,
  artifact: copy.text.artifactc7c5c1,
  decision: copy.text.decision,
  issue: copy.text.issue,
  blocker: copy.text.blocker,
  roadmap: copy.text.roadmap3de158,
  relation: copy.text.relation,
  work_item_assignees: copy.text.taskAssignees,
  trigger: copy.text.integrationTrigger,
};
const actionName = (name) => {
  const [verb, ...subject] = name.split("_");
  return `${verbs[verb] || display(verb)} ${actionSubjects[subject.join("_")] || display(subject.join("_"))}`;
};
async function request(path, options = {}) {
  const generation = state.generation;
  try {
    return await networkRequest(path, options);
  } catch (error) {
    if (generation !== state.generation) {
      error.discarded = true;
      error.name = "AbortError";
    } else if ([401, 403].includes(error.status) && state.namespace) {
      invalidateOutcomeSearch();
      $("outcomes").replaceChildren();
      state.generation++;
      state.detailGeneration++;
      state.refreshGeneration++;
      state.permissions = [];
      state.snapshot = null;
      resetCanvas();
      state.entities.clear();
      state.loaded.clear();
      state.boardLoaded.clear();
      $("developer-tools").hidden = true;
      $("administration").hidden = true;
      $("actions").disabled = true;
      $("outcome").hidden = true;
      $("detail-dialog").close();
      $("detail-content").replaceChildren();
      $("content").replaceChildren();
    }
    throw error;
  }
}
const { api_prefix: api } = await request("/app/config");
const inventory = await request("/app/command-exposure.json");
const currentScope = () => ({
  namespace_id: state.namespace,
  outcome_id: state.outcome?.id,
});
async function refreshPermissions() {
  const namespace = state.namespace,
    generation = state.generation,
    scope = currentScope();
  const value = await request(
    `${api}/namespaces/${namespace}/effective-permissions${scope.outcome_id ? `?outcome_id=${scope.outcome_id}` : ""}`,
  );
  if (generation !== state.generation || namespace !== state.namespace) return;
  state.permissions = value.permissions;
  $("actions").disabled = false;
  $("developer-tools").hidden = !state.permissions.includes("namespace:admin");
  $("administration").hidden = !state.permissions.includes("namespace:admin");
  for (const id of ["create-outcome", "empty-create"]) {
    $(id).disabled = !state.permissions.includes("outcome:write");
  }
}
function placeNotice(onBoard = canvasActive() && !$("outcome").hidden && !$("canvas-view").hidden) {
  const parent = onBoard ? $("canvas-feedback") : $("main-content");
  if (onBoard && !$("notice").classList.contains("error") &&
    [copy.text.partialViewMoreItemsAreAvailableUseLoadMore, copy.text.sharedStateRefreshed]
      .includes($("notice").textContent))
    $("notice").textContent = "";
  if ($("notice").parentElement !== parent)
    parent.insertBefore($("notice"), parent.firstChild);
}
function notice(message, error = false) {
  // A failed Outcome/workspace transition can leave the previous Board hidden.
  // Announce its error in the visible main workspace rather than that subtree.
  placeNotice();
  $("notice").textContent = message;
  $("notice").classList.toggle("error", error);
}
function report(error) {
  if (error.discarded || error.name === "AbortError") return;
  notice(
    error.code === "version_conflict" || error.code === "precondition_failed"
      ? copy.text.stateChangedRefreshAndReviewYourIntentBeforeTryingAgain
      : error.message,
    true,
  );
}
const base = () => `${api}/namespaces/${state.namespace}/outcomes`;
const outcomeBase = () => `${base()}/${state.outcome.id}`;
// The map is a bounded presentation of persisted state. All changes go through
// the same permission registry, forms and command intents as the list views.
function canvasPresentation() {
  if (!state.canvasModel) return null;
  return {
    ...state.canvasModel,
    nodes: state.canvasModel.nodes.map((item) => {
      if (!item.ref) return item;
      const actions = effectiveActions(item.kind, {
        ...item.sourceSummary,
        _kind: item.kind,
      });
      return {
        ...item,
        canCreate: [
          ...(actions.some((a) => a.name === "create_objective")
            ? ["objective"]
            : []),
          ...(actions.some((a) => a.name === "create_work_item")
            ? ["work_item"]
            : []),
        ],
        canConnect: actions.some((a) => a.name === "add_dependency"),
      };
    }),
  };
}
function renderCanvasSummary() {
  const root = $("canvas-summary");
  root.replaceChildren();
  const purpose = el("div", undefined, "canvas-purpose");
  purpose.append(
    el("h2", state.outcome.title, "canvas-context-title"),
    el("p", "Desired outcome", "eyebrow"),
    el(
      "p",
      state.outcome.desired_state ||
        state.outcome.description ||
        state.outcome.title,
    ),
  );
  const actions = el("div", undefined, "canvas-summary-actions");
  for (const command of [
    "create_objective",
    "create_work_item",
    "create_issue",
  ])
    if (
      effectiveActions("outcome", state.outcome).some((a) => a.name === command)
    )
      actions.append(
        actionButton(
          humanActions[command].label,
          () => createItem(command),
          "quiet",
        ),
      );
  actions.append(
    actionButton(
      copy.text.reviewOutcome,
      () => showEntity(state.snapshot.outcome.ref).catch(report),
      "quiet",
    ),
  );
  purpose.append(actions);
  const metrics = el("dl", undefined, "canvas-summary-metrics");
  const progress = state.snapshot.progress?.work_completion;
  const count = state.snapshot.counts || {};
  $("canvas-counts").textContent = [
    [count.objectives ?? 0, "Objective"],
    [count.work_items ?? 0, "Task"],
    [count.issues ?? 0, "Issue"],
  ]
    .map(([value, label]) => `${value} ${label}${value === 1 ? "" : "s"}`)
    .join(" · ");
  const provenance = el("p", undefined, "canvas-context-provenance");
  provenance.textContent = `${$("context-name").textContent} · ${$("revision").textContent}`;
  for (const [label, value] of [
    [
      "Task progress",
      progress?.denominator
        ? `${Math.round((progress.numerator / progress.denominator) * 100)}%`
        : "—",
    ],
    ["Objectives", count.objectives ?? 0],
    ["Ready", count.ready_work ?? 0],
    ["Blocked", count.blocked_work ?? 0],
    ["signed_contracts_v2", "contracts_v1", "draining_to_signed_v2"].includes(
      state.protocol?.phase,
    )
      ? ["Tasks", count.work_items ?? 0]
      : ["Needs attention", count.attention_needed_work ?? 0],
  ]) {
    const group = el("div");
    group.append(el("dt", label), el("dd", String(value)));
    metrics.append(group);
  }
  root.append(purpose, metrics, provenance);
}
function renderCanvasPlanning() {
  const root = $("canvas-planning");
  root.replaceChildren(el("span", "Planning / optional"));
  root.append(
    el(
      "p",
      state.canvasPlanLabel ||
        "Work directly with Objectives and Tasks, or choose a Roadmap. Planning order does not create dependencies.",
    ),
  );
  root.append(
    actionButton("Browse Roadmaps", () => navigateArea("plan"), "quiet"),
  );
  if (
    effectiveActions("outcome", state.outcome).some(
      (a) => a.name === "create_roadmap",
    )
  )
    root.append(
      actionButton(
        humanActions.create_roadmap.label,
        () => createItem("create_roadmap"),
        "quiet",
      ),
    );
  if (state.canvasPlan)
    root.append(
      actionButton(
        "Hide plan",
        () => {
          if (!allowCanvasNavigation()) return;
          state.canvasPlan = null;
          state.canvasPlanLabel = "";
          loadCanvas({ force: true }).catch(report);
        },
        "quiet",
      ),
    );
  for (const plan of state.canvasModel?.plans || []) {
    const slot = plan.slot;
    root.append(
      actionButton(
        `Active plan · revision ${slot.revision_number}`,
        () =>
          viewCanvasPlan({
            roadmap_id: slot.roadmap_id,
            revision_number: slot.revision_number,
          }),
        "quiet",
      ),
    );
  }
}
function acceptCanvas(model) {
  if (
    !model ||
    !canvasActive() ||
    !state.snapshot ||
    model.scopeKey !== `${state.namespace}/${state.outcome.id}` ||
    model.revision !== canvasRevision(state.snapshot.outcome_revision)
  )
    return;
  state.canvasModel = model;
  state.entityContextRefresh?.();
  const selectedKey = state.canvasSelected
    ? `${state.canvasSelected.kind}:${state.canvasSelected.id}`
    : null;
  if (state.canvasView)
    state.canvasView.update(canvasPresentation(), { selectedKey });
  renderCanvasPlanning();
}
async function canvasCreate(kind, parentRef) {
  if (!allowCanvasNavigation()) return;
  const entity = await showEntity(parentRef);
  if (
    !entity ||
    state.canvasSelected?.id !== parentRef.id ||
    !allowCanvasNavigation()
  )
    return;
  openCommands(kind === "objective" ? "create_objective" : "create_work_item");
}
async function loadCanvas({ force = false } = {}) {
  if (!canvasActive() || !state.snapshot) return;
  renderCanvasSummary();
  if (
    !force &&
    state.canvasData &&
    state.canvasModel?.revision ===
      canvasRevision(state.snapshot.outcome_revision)
  ) {
    renderCanvasPlanning();
    return;
  }
  state.canvasData?.dispose();
  const epoch = ++state.canvasEpoch,
    generation = state.generation,
    refreshGeneration = state.refreshGeneration,
    scope = currentScope();
  const data = createCanvasData({
    request,
    scope,
    limit: 100,
    protocolMode: state.protocol?.phase,
    apiPrefix: api,
    isCurrent: () =>
      epoch === state.canvasEpoch &&
      generation === state.generation &&
      refreshGeneration === state.refreshGeneration &&
      sameScope(scope, currentScope()),
  });
  state.canvasData = data;
  let model = await data.load(state.snapshot);
  if (!model || state.canvasData !== data || !canvasActive()) return;
  if (state.canvasPlan) {
    const selected = state.canvasPlan;
    if (selected.draft) {
      const current = await request(
        `${outcomeBase()}/roadmaps/${selected.roadmap_id}`,
      );
      if (
        state.canvasData !== data ||
        state.canvasPlan !== selected ||
        !canvasActive()
      )
        return;
      if (canvasRevision(current.outcome_revision) !== model.revision)
        throw Error(
          copy.text.stateChangedWhileLoadingRefreshForAConsistentView,
        );
      const draft = current.value?.draft;
      if (draft?.lifecycle === "open") {
        model = await data.roadmap({
          roadmap_id: selected.roadmap_id,
          draft,
          outcome_revision: current.outcome_revision,
        });
        state.canvasPlanLabel = `Draft · version ${draft.draft_version} · not published or activated`;
      } else {
        state.canvasPlan = null;
        state.canvasPlanLabel =
          "The Draft is closed. Choose a published revision to view its plan.";
      }
    } else model = await data.roadmap(selected);
    if (
      !model ||
      state.canvasData !== data ||
      (state.canvasPlan && state.canvasPlan !== selected) ||
      !canvasActive()
    )
      return;
  } else state.canvasPlanLabel = "";
  state.canvasModel = model;
  if (!state.canvasView)
    state.canvasView = mountWorkspaceCanvas($("operational-canvas"), {
      inspectorHost: $("canvas-details"),
      model: canvasPresentation(),
      selectedKey: state.canvasSelected
        ? `${state.canvasSelected.kind}:${state.canvasSelected.id}`
        : null,
      onSelect: (ref, item) => {
        if (!allowCanvasNavigation()) return;
        const target = ref || item?.roadmapRef;
        if (target) return showEntity(target);
      },
      onSelectEdge: (edge) => edge.ref && showEntity(edge.ref),
      onCreate: canvasCreate,
      onConnect: async ({ sourceRef, targetRef }) => {
        if (!allowCanvasNavigation()) return;
        const entity = await showEntity(sourceRef);
        if (
          !entity ||
          state.canvasSelected?.id !== sourceRef.id ||
          !allowCanvasNavigation()
        )
          return;
        if (targetRef)
          state.selected = { ...entity, _canvasDependencyTarget: targetRef };
        openCommands("add_dependency");
      },
      onLoadMore: async () => acceptCanvas(await state.canvasData?.more()),
      onExpand: async (item, expanded) => {
        if (expanded && item.ref)
          acceptCanvas(await state.canvasData?.expand(item.ref));
      },
      onSearch: async (query, { cursor = "" } = {}) =>
        state.canvasData?.search(query, {
          cursor,
          kinds: [
            "objective",
            "work_item",
            "issue",
            "blocker",
            "evidence",
            "artifact",
            "decision",
            "roadmap",
          ],
        }),
    });
  else
    state.canvasView.update(canvasPresentation(), {
      selectedKey: state.canvasSelected
        ? `${state.canvasSelected.kind}:${state.canvasSelected.id}`
        : null,
    });
  renderCanvasPlanning();
}
async function loadCanvasHistory(host, ref, detailGeneration) {
  const section = el("section", undefined, "detail-section");
  section.append(el("h3", "Recorded task activity"));
  const list = el("ol"),
    status = el("p", undefined, "field-help");
  let cursor = "",
    busy = false,
    through;
  const more = actionButton("Load more activity", load, "quiet");
  section.append(status, list, more);
  host.querySelector(".canvas-task-history")?.remove();
  section.classList.add("canvas-task-history");
  host.append(section);
  async function load() {
    if (busy) return;
    busy = true;
    more.disabled = true;
    try {
      const query = new URLSearchParams({
        work_item_id: ref.id,
        limit: "25",
        descending: "true",
      });
      if (cursor) query.set("cursor", cursor);
      const page = await request(`${outcomeBase()}/work-activity?${query}`);
      if (detailGeneration !== state.detailGeneration || !section.isConnected)
        return;
      const revision = canvasRevision(page.history_through_revision);
      if (through && through !== revision)
        throw Error(
          "Activity changed its historical boundary. Reload this Task.",
        );
      through = revision;
      for (const item of page.items || []) {
        const row = el("li");
        row.append(
          el("strong", display(item.type)),
          el("p", item.description),
          el(
            "small",
            `${date(item.recorded_at)} · ${item.principal_id || item.actor_ref?.id || "Recorded participant"}`,
          ),
        );
        list.append(row);
      }
      cursor = page.next_cursor || "";
      more.hidden = !cursor;
      status.textContent = `Recorded activity through revision ${through}. ${cursor ? "More records are available." : list.children.length ? "All records in this traversal are shown." : "No recorded activity in this traversal."} Activity is documentary and does not approve a result.`;
    } catch (error) {
      if (detailGeneration !== state.detailGeneration || !section.isConnected)
        return;
      report(error);
      status.textContent = error.message;
    } finally {
      busy = false;
      more.disabled = false;
    }
  }
  await load();
}
async function viewCanvasPlan(value) {
  if (!allowCanvasNavigation()) return;
  const selection = value.draft
    ? { roadmap_id: value.roadmap_id, draft: true }
    : { roadmap_id: value.roadmap_id, revision_number: value.revision_number };
  state.canvasPlan = selection;
  try {
    const model = await state.canvasData.roadmap(value);
    if (!model || state.canvasPlan !== selection) return;
    state.canvasPlanLabel = value.draft
      ? `Draft · version ${value.draft.draft_version} · not published or activated`
      : `Published revision ${value.revision_number} · activation is a separate action`;
    acceptCanvas(model);
    if (!canvasNavigationLocked()) {
      state.canvasView?.fit(true);
      const focal =
        model.nodes.find(
          (item) =>
            item.kind === "phase" && item.roadmapRef?.id === value.roadmap_id,
        ) ||
        model.nodes.find(
          (item) => !item.ref && item.roadmapRef?.id === value.roadmap_id,
        );
      if (focal) state.canvasView?.focus(focal.key);
      if (canvasInspector.isVisible("detail")) canvasInspector.showDetails();
    }
  } catch (error) {
    if (state.canvasPlan === selection) report(error);
  }
}

const {
  contractSection,
  commandAvailableInUI,
  refreshNamespaceProtocol,
  renderProtocolBanner,
  bindProtocolEvents,
} = contractViews({
  state,
  api,
  request,
  inventory,
  $,
  el,
  copy,
  date,
  display,
  badge,
  readable,
  referenceButton,
  outcomeBase,
  notice,
  report,
  openCommands,
  shortId,
});
bindProtocolEvents();

function el(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = text;
  if (className) node.className = className;
  return node;
}
async function connect(namespace) {
  invalidateOutcomeSearch();
  $("outcomes").replaceChildren();
  resetCanvas();
  state.generation++;
  state.detailGeneration++;
  state.namespace = namespace;
  state.permissions = [];
  state.principal = "";
  state.actorKey = "";
  state.protocol = null;
  $("signed-protocol-info").hidden = true;
  $("connect").hidden = true;
  $("workspace").hidden = false;
  state.catalog = (await request(`${api}/commands`)).commands;
  await refreshNamespaceProtocol();
  await refreshPermissions();
  const identity = await request(`${api}/identity`).catch(() => ({}));
  state.principal = identity.principal_id || "";
  state.actorKey = JSON.stringify([
    identity.actor_ref?.kind,
    identity.actor_ref?.provider,
    identity.actor_ref?.id,
  ]);
  await discover();
  $("context-name").textContent =
    $("namespace").selectedOptions[0]?.textContent || copy.text.workspace;
}
async function namespaces() {
  const result = await request(`${api}/namespaces`);
  $("namespace").replaceChildren();
  for (const n of result.items) {
    const option = el("option", n.name);
    option.value = n.id;
    $("namespace").append(option);
  }
  if (result.items.length) {
    const route = readRoute(location.search);
    const target = route
      ? result.items.find((n) => n.id === route.namespace)
      : result.items[0];
    if (!target)
      throw Error(copy.text.theLinkedWorkspaceIsNotAvailableToThisAccount);
    $("namespace").value = target.id;
    await connect(target.id);
    if (route) await restoreRoute(route);
  }
}
$("login").onsubmit = async (event) => {
  event.preventDefault();
  try {
    await request("/session", {
      method: "POST",
      body: JSON.stringify({ token: $("token").value }),
    });
    $("token").value = "";
    await namespaces();
  } catch (e) {
    $("token").setCustomValidity(e.message);
    $("token").reportValidity();
    $("token").oninput = () => $("token").setCustomValidity("");
  }
};
$("local").onsubmit = async (event) => {
  event.preventDefault();
  try {
    const id = $("local-ns").value.trim();
    const option = el("option", id);
    option.value = id;
    $("namespace").replaceChildren(option);
    await connect(id);
  } catch (e) {
    report(e);
    $("connect").hidden = false;
  }
};
$("logout").onclick = async () => {
  await request("/session", { method: "DELETE" }).catch(() => {});
  location.reload();
};
$("namespace").onchange = () => {
  if (!allowCanvasNavigation()) {
    $("namespace").value = state.namespace;
    return;
  }
  state.outcome = null;
  state.snapshot = null;
  $("workspace-loading").hidden = true;
  state.entities.clear();
  state.selected = null;
  $("outcome").hidden = true;
  $("empty").hidden = false;
  connect($("namespace").value).catch(report);
};
$("search").onsubmit = (event) => {
  event.preventDefault();
  discover().catch(report);
};
$("text").oninput = () => {
  invalidateOutcomeSearch();
  $("outcomes").setAttribute("aria-busy", "true");
  $("outcome-search-status").textContent = copy.text.findingOutcomes;
  outcomeSearchTimer = setTimeout(() => discover().catch(report), 250);
};
for (const id of ["lifecycle", "archived"])
  $(id).onchange = () => discover().catch(report);
function invalidateOutcomeSearch() {
  clearTimeout(outcomeSearchTimer);
  outcomeSearchController?.abort();
  outcomeSearchController = null;
  outcomeSearchEpoch++;
  outcomeSearchLoading = false;
  state.next = "";
  $("more-outcomes").hidden = true;
  $("more-outcomes").disabled = false;
  $("outcomes").setAttribute("aria-busy", "false");
  $("outcome-search-status").textContent = "";
}
function outcomeSearchParams() {
  const query = new URLSearchParams({ limit: "25", text: $("text").value });
  if ($("lifecycle").value) query.set("lifecycle", $("lifecycle").value);
  if (!$("archived").checked) query.set("archived", "false");
  return query;
}
async function discover(append = false) {
  const q = outcomeSearchParams();
  const namespace = state.namespace;
  const signature = `${namespace}?${q}`;
  const cursor = state.next;
  if (
    append &&
    (outcomeSearchLoading || !cursor || signature !== outcomeSearchQuery)
  )
    return;
  if (!append) invalidateOutcomeSearch();
  else clearTimeout(outcomeSearchTimer);
  const epoch = ++outcomeSearchEpoch;
  const generation = state.generation;
  const controller = new AbortController();
  outcomeSearchController = controller;
  outcomeSearchLoading = true;
  $("outcomes").setAttribute("aria-busy", "true");
  $("more-outcomes").disabled = true;
  $("outcome-search-status").textContent = copy.text.findingOutcomes;
  if (append) q.set("cursor", cursor);
  try {
    const page = await request(`${base()}?${q}`, { signal: controller.signal });
    if (
      epoch !== outcomeSearchEpoch ||
      generation !== state.generation ||
      namespace !== state.namespace ||
      signature !== `${state.namespace}?${outcomeSearchParams()}` ||
      (append && cursor !== state.next)
    )
      return;
    if (!append) $("outcomes").replaceChildren();
    const loaded = new Set(
      [...$("outcomes").children].map((child) => child.dataset.outcomeId),
    );
    for (const item of page.items) {
      if (loaded.has(item.id)) continue;
      loaded.add(item.id);
      const button = el("button", undefined, "item");
      button.dataset.outcomeId = item.id;
      button.classList.toggle("selected", state.outcome?.id === item.id);
      button.setAttribute("aria-pressed", String(state.outcome?.id === item.id));
      const navCopy = el("span", undefined, "nav-item-text");
      navCopy.append(
        el("strong", item.title),
        el(
          "small",
          `${display(item.lifecycle)}${item.archived_at ? copy.text.archivedc089d2 : ""}`,
        ),
      );
      const navIndex = el("span", undefined, "nav-dot");
      navIndex.setAttribute("aria-hidden", "true");
      button.append(navIndex, navCopy);
      button.onclick = () => openOutcome(item).catch(report);
      $("outcomes").append(button);
    }
    if (!page.items.length && !append)
      $("outcomes").append(el("p", copy.text.noMatchingOutcomes));
    state.next = page.next_cursor || "";
    outcomeSearchQuery = signature;
    $("more-outcomes").hidden = !state.next;
    $("outcome-search-status").textContent = message(
      "outcomesShown",
      $("outcomes").querySelectorAll("button[data-outcome-id]").length,
    );
  } catch (error) {
    if (error.discarded || error.name === "AbortError") return;
    // A current-scope access denial must still clear private state through
    // request(), even if the user has already typed a newer search query.
    if ([401, 403].includes(error.status)) throw error;
    if (epoch !== outcomeSearchEpoch || generation !== state.generation) return;
    $("outcome-search-status").textContent = error.message;
    throw error;
  } finally {
    if (epoch === outcomeSearchEpoch) {
      outcomeSearchLoading = false;
      outcomeSearchController = null;
      $("outcomes").setAttribute("aria-busy", "false");
      $("more-outcomes").disabled = false;
    }
  }
}
$("more-outcomes").onclick = () => discover(true).catch(report);
async function openOutcome(outcome) {
  if (!allowCanvasNavigation()) return;
  const sidebar = document.querySelector(".sidebar");
  const openedFromSidebar = sidebar.contains(document.activeElement);
  resetCanvas();
  state.generation++;
  document.querySelector(".sidebar").classList.add("navigation-collapsed");
  $("navigation-toggle").setAttribute("aria-expanded", "false");
  state.entities.clear();
  state.loaded.clear();
  state.boardLoaded.clear();
  state.outcome = outcome;
  state.selected = outcome;
  state.criterion = null;
  state.view = "canvas";
  state.area = "canvas";
  state.snapshot = null;
  $("item-search").value = "";
  $("item-priority").value = "";
  $("detail-dialog").close();
  $("command-dialog").close();
  for (const button of $("outcomes").children) {
    button.classList.toggle(
      "selected",
      button.dataset.outcomeId === outcome.id,
    );
    button.setAttribute(
      "aria-pressed",
      String(button.dataset.outcomeId === outcome.id),
    );
  }
  $("empty").hidden = true;
  $("outcome").hidden = true;
  $("workspace-loading").hidden = false;
  const generation = state.generation;
  try {
    await refresh();
    writeRoute();
    if (
      generation === state.generation &&
      canvasActive() &&
      openedFromSidebar &&
      (document.activeElement === document.body ||
        sidebar.contains(document.activeElement))
    )
      $("area-tabs")
        .querySelector('[data-area="canvas"]')
        ?.focus({ preventScroll: true });
  } catch (error) {
    if (generation !== state.generation) return;
    $("empty").hidden = false;
    throw error;
  } finally {
    if (generation === state.generation) $("workspace-loading").hidden = true;
  }
}
async function refresh() {
  if (!state.outcome) return discover();
  state.detailGeneration++;
  const generation = state.generation;
  const refreshGeneration = ++state.refreshGeneration;
  try {
    const path = outcomeBase();
    const [snapshot, live, protocol] = await Promise.all([
      request(`${path}/continuity?limit=25`),
      request(path),
      request(`${api}/namespaces/${state.namespace}/work-protocol`),
    ]);
    if (
      generation !== state.generation ||
      refreshGeneration !== state.refreshGeneration
    )
      return;
    if (
      live.outcome_revision !== undefined &&
      canvasRevision(live.outcome_revision) !==
        canvasRevision(snapshot.outcome_revision)
    ) {
      const changed = new Error(
        copy.text.stateChangedWhileLoadingRefreshForAConsistentView,
      );
      changed.code = "precondition_failed";
      throw changed;
    }
    state.protocol = protocol;
    renderProtocolBanner();
    state.snapshot = snapshot;
    state.outcome = live.value || live;
    state.loaded.clear();
    state.boardLoaded.clear();
    state.entities.clear();
    for (const section of Object.values(snapshot.sections))
      for (const item of section) {
        const e = item?.ref ? item : item?.work_item || item?.current;
        if (e?.ref) state.entities.set(`${e.ref.kind}/${e.ref.id}`, e);
      }
    $("empty").hidden = true;
    $("outcome").hidden = false;
    $("title").textContent = snapshot.outcome.title;
    $("title").title = snapshot.outcome.title;
    $("state").textContent = display(snapshot.outcome.lifecycle);
    $("state").dataset.lifecycle = snapshot.outcome.lifecycle;
    $("state").className = `badge ${tone(snapshot.outcome.lifecycle)}`;
    $("outcome-key").textContent = shortId(snapshot.outcome.ref.id);
    $("revision").textContent = message(
      "revision0c81",
      snapshot.outcome_revision,
      date(snapshot.evaluated_at),
    );
    await refreshPermissions();
    if (
      generation !== state.generation ||
      refreshGeneration !== state.refreshGeneration
    )
      return;
    renderMetrics();
    renderTabs();
    renderSummary();
    applyView();
    await renderSection();
    if (canvasActive()) {
      await loadCanvas({ force: true });
      if (state.canvasSelected && !canvasNavigationLocked())
        await showEntity(state.canvasSelected, { preserveCamera: true });
    }
    if (
      generation !== state.generation ||
      refreshGeneration !== state.refreshGeneration
    )
      return;
    notice(
      // The Board footer announces loaded counts, revision and partial-map
      // coverage. Avoid repeating routine refresh feedback in another row.
      canvasActive()
        ? ""
        : snapshot.truncated
          ? copy.text.partialViewMoreItemsAreAvailableUseLoadMore
          : copy.text.sharedStateRefreshed,
    );
  } catch (error) {
    if (
      generation !== state.generation ||
      refreshGeneration !== state.refreshGeneration
    ) {
      error.discarded = true;
      error.name = "AbortError";
    }
    throw error;
  }
}
$("copy-workspace-link").onclick = async () => {
  try {
    await navigator.clipboard.writeText(
      location.origin +
        location.pathname +
        routeQuery({
          namespace: state.namespace,
          outcome: state.outcome.id,
          area: state.area,
          section: state.section,
          view: state.view,
        }),
    );
    notice(copy.text.workspaceLinkCopied);
  } catch (error) {
    report(error);
  }
};
$("refresh").onclick = () => refresh().catch(report);
$("navigation-toggle").onclick = () => {
  const sidebar = document.querySelector(".sidebar");
  sidebar.classList.toggle("navigation-collapsed");
  $("navigation-toggle").setAttribute(
    "aria-expanded",
    String(!sidebar.classList.contains("navigation-collapsed")),
  );
};
const operationalSections = [
  "backlog_work",
  "waiting_scope_work",
  "waiting_dependencies_work",
  "scheduled_work",
  "ready_work",
  "in_progress_work",
  "blocked_work",
  "attention_needed_work",
  "done_work",
  "cancelled_work",
];
const sectionIcons = {
  work_items: "work_item",
  objectives: "objective",
  issues: "issue",
  blockers: "blocker",
  evidence: "evidence",
  artifacts: "artifact",
  decisions: "decision",
  roadmaps: "roadmap",
  active_roadmaps: "roadmap",
  active_plan_references: "roadmap",
  timeline: "timeline",
  graph: "graph",
  conclusion_contestations: "alert",
};
function renderTabs() {
  const contractWork = usesWorkContracts();
  const terminalSections = ["work_items", "done_work", "cancelled_work"];
  for (const option of $("work-status").options) {
    option.disabled = contractWork && !terminalSections.includes(option.value);
    option.hidden = option.disabled;
  }
  state.workFilterReset =
    contractWork &&
    state.area === "work" &&
    !terminalSections.includes(state.section);
  if (state.workFilterReset) {
    state.section = "work_items";
    writeRoute(undefined, true);
  }
  $("work-status").value = state.section;
  const navigation = $("area-tabs");
  const hadNavigationFocus = navigation.contains(document.activeElement);
  navigation.replaceChildren();
  const views = el("details", undefined, "outcome-views");
  views.id = "outcome-views";
  views.name = "outcome-header";
  const viewLabel = el("summary", copy.text.workspaceViews);
  if (state.area !== "canvas") {
    viewLabel.append(
      el("span", ` · ${areas[state.area].label}`, "outcome-current-view"),
    );
  }
  const viewOptions = el("div", undefined, "outcome-view-options");
  views.append(viewLabel, viewOptions);
  for (const [area, definition] of Object.entries(areas)) {
    const b = actionButton(
      definition.label,
      () => {
        navigateArea(area);
        if (area !== "canvas") {
          const currentViews = $("outcome-views");
          currentViews.open = false;
          currentViews.querySelector("summary").focus({ preventScroll: true });
        }
      },
      "quiet",
    );
    b.dataset.area = area;
    b.setAttribute("aria-pressed", String(state.area === area));
    b.classList.toggle("selected", state.area === area);
    if (state.area === area) b.setAttribute("aria-current", "page");
    (area === "canvas" ? navigation : viewOptions).append(b);
  }
  navigation.append(views);
  if (hadNavigationFocus)
    (state.area === "canvas"
      ? navigation.querySelector('[data-area="canvas"]')
      : viewLabel
    ).focus({ preventScroll: true });
  $("tabs").replaceChildren();
  for (const section of areas[state.area].sections) {
    const b = actionButton(
      human(section),
      () => navigateSection(section),
      "quiet",
    );
    b.dataset.section = section;
    b.setAttribute("aria-pressed", String(section === state.section));
    b.classList.toggle("selected", section === state.section);
    $("tabs").append(b);
  }
}
function writeRoute(item = undefined, replace = false) {
  if (item === undefined) item = canvasActive() ? state.canvasSelected : null;
  if (state.restoringRoute || !state.outcome) return;
  const query = routeQuery({
    namespace: state.namespace,
    outcome: state.outcome.id,
    area: state.area,
    section: state.section,
    view: state.view,
    item,
  });
  const target = location.pathname + query;
  if (target === location.pathname + location.search) return;
  history[replace ? "replaceState" : "pushState"]({}, "", target);
}
async function restoreRoute(route) {
  if (!route) return;
  if (!allowCanvasNavigation()) {
    writeRoute(undefined, true);
    return;
  }
  state.restoringRoute = true;
  try {
    if (state.namespace !== route.namespace) {
      const option = [...$("namespace").options].find(
        (n) => n.value === route.namespace,
      );
      if (!option)
        throw Error(copy.text.theLinkedWorkspaceIsNotAvailableToThisAccount);
      $("namespace").value = route.namespace;
      state.outcome = null;
      state.snapshot = null;
      await connect(route.namespace);
    }
    if (state.outcome?.id !== route.outcome) {
      const value = await request(`${base()}/${route.outcome}`);
      await openOutcome(value.value || value);
    }
    state.area = route.area;
    state.section = route.section;
    state.view = route.view;
    renderTabs();
    applyView();
    await renderSection();
    if (canvasActive() && !state.canvasModel) await loadCanvas();
    if (route.item) await showEntity(route.item);
    else closeDetail();
  } finally {
    state.restoringRoute = false;
    if (state.area === "work" && state.section !== route.section)
      writeRoute(route.item, true);
  }
}
window.addEventListener("popstate", () => {
  try {
    const route = readRoute(location.search);
    if (route) restoreRoute(route).catch(report);
  } catch (error) {
    report(error);
  }
});
function navigateArea(area) {
  if (!allowCanvasNavigation()) return;
  closeDetail();
  state.area = area;
  state.section = areas[area].sections[0] || "work_items";
  state.view =
    area === "canvas" ? "canvas" : area === "overview" ? "summary" : "list";
  $("item-search").value = "";
  $("item-priority").value = "";
  $("work-status").value = state.section;
  renderTabs();
  applyView();
  writeRoute();
  (canvasActive() ? loadCanvas() : renderSection()).catch(report);
}
function applyView() {
  const isCanvas = canvasActive();
  // Keep the original live region and its announcements. On the Board it is
  // ordinary workspace feedback, outside the map and inspector, never a toast.
  placeNotice(isCanvas);
  const refresh = $("refresh");
  const activeHeaderControl = document.activeElement;
  const leavingOutcomeTools =
    !isCanvas &&
    activeHeaderControl !== refresh &&
    ($("outcome-tools").contains(activeHeaderControl) ||
      $("canvas-context").contains(activeHeaderControl) ||
      activeHeaderControl === $("actions"));
  const hadSidebarFocus = document
    .querySelector(".sidebar")
    .contains(document.activeElement);
  if (isCanvas) $("outcome").dataset.editorialArea = "canvas";
  else renderEditorial(state.area);
  $("area-intro").hidden = isCanvas;
  $("canvas-view").hidden = !isCanvas;
  $("canvas-context").hidden = !isCanvas;
  $("canvas-counts").hidden = !isCanvas;
  if (!isCanvas) $("canvas-context").open = false;
  $("outcome-tools").hidden = !isCanvas;
  $("actions").hidden = !isCanvas;
  if (!isCanvas) $("outcome-tools").open = false;
  // A view can refresh its current snapshot without exposing Outcome-wide
  // operations or changing routes. Reuse the same button and its handler.
  const refreshParent = isCanvas
    ? $("view-outcome").parentElement
    : $("outcome-tools").parentElement;
  if (refresh.parentElement !== refreshParent) {
    refreshParent.insertBefore(
      refresh,
      isCanvas ? $("view-outcome") : $("outcome-tools"),
    );
    if (activeHeaderControl === refresh)
      (isCanvas && !$("outcome-tools").open
        ? $("outcome-tools").querySelector("summary")
        : refresh
      ).focus({ preventScroll: true });
  }
  refresh.dataset.copy = isCanvas ? "refresh" : "refreshView";
  refresh.dataset.copyTitle = isCanvas ? "refreshState" : "refreshView";
  refresh.textContent = copy.text[refresh.dataset.copy];
  refresh.title = copy.text[refresh.dataset.copyTitle];
  canvasInspector.setEnabled(isCanvas);
  $("summary-view").hidden = state.view !== "summary";
  $("items-view").hidden = state.view === "summary" || isCanvas;
  for (const button of $("view-tabs").children) {
    button.classList.toggle("selected", button.dataset.view === state.view);
    button.setAttribute(
      "aria-pressed",
      String(button.dataset.view === state.view),
    );
  }
  $("view-tabs").hidden = state.area !== "work";
  $("work-status-label").hidden =
    state.area !== "work" || state.view === "board";
  $("item-priority").closest("label").hidden = !collectionHasPriority();
  if (!collectionHasPriority()) $("item-priority").value = "";
  $("content").dataset.section = state.section;
  const board = state.view === "board";
  $("tabs").hidden = isCanvas || board || state.area === "work";
  $("add-item").textContent =
    board ||
    ["work_items", "ready_work", "blocked_work", "in_progress_work"].includes(
      state.section,
    )
      ? copy.text.newTask
      : `+ ${actionName(sectionCommand())}`;
  $("add-item").hidden =
    (!board && !sectionCommand()) ||
    !state.permissions.includes(
      inventory[board ? "create_work_item" : sectionCommand()]?.permission,
    );
  if (isCanvas && hadSidebarFocus)
    $("area-tabs")
      .querySelector('[data-area="canvas"]')
      ?.focus({ preventScroll: true });
  else if (leavingOutcomeTools)
    $("outcome-views").querySelector("summary").focus({ preventScroll: true });
}
function navigateSection(section) {
  if (!allowCanvasNavigation()) return;
  closeDetail();
  state.section = section;
  state.area = areaFor(section);
  state.view = "list";
  $("work-status").value = section;
  renderTabs();
  $("item-search").value = "";
  $("item-priority").value = "";
  applyView();
  writeRoute();
  renderSection().catch(report);
}
for (const button of $("view-tabs").children) {
  button.prepend(icon(button.dataset.view));
  button.onclick = () => {
    if (!allowCanvasNavigation()) return;
    state.area = "work";
    state.view = button.dataset.view;
    renderTabs();
    writeRoute();
    applyView();
    renderSection().catch(report);
  };
}
let searchTimer;
$("item-search").oninput = () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => renderSection().catch(report), 180);
};
$("work-status").onchange = () => navigateSection($("work-status").value);
$("item-priority").onchange = () => renderSection().catch(report);
const createForSection = {
  work_items: "create_work_item",
  objectives: "create_objective",
  issues: "create_issue",
  blockers: "create_blocker",
  evidence: "register_evidence",
  artifacts: "register_artifact",
  decisions: "propose_decision",
  roadmaps: "create_roadmap",
  ready_work: "create_work_item",
  blocked_work: "create_work_item",
  in_progress_work: "create_work_item",
};
function sectionCommand() {
  return createForSection[state.section] || "";
}
function createItem(command) {
  if (!allowCanvasNavigation()) return;
  state.selected = { ...state.outcome, _kind: "outcome" };
  state.criterion = null;
  openCommands(command);
}
$("add-item").onclick = () =>
  createItem(state.view === "board" ? "create_work_item" : sectionCommand());
$("empty-create").onclick = () => {
  state.selected = null;
  openCommands("create_outcome");
};
function renderMetrics() {
  $("metrics").replaceChildren();
  for (const key of [
    "work_completion",
    "objective_completion",
    "required_criteria_met",
    "required_objectives_completion",
    "required_criteria_waived",
  ]) {
    const m = state.snapshot.progress[key];
    if (!m) continue;
    const card = el("div", undefined, "metric");
    const label = el("div", undefined, "metric-label");
    label.append(
      icon(
        key.includes("criteria")
          ? "check"
          : key === "work_completion"
            ? "work_item"
            : "objective",
      ),
      el("span", human(key)),
    );
    const value = el(
      "strong",
      m.value === null
        ? copy.text.notApplicable
        : `${Math.round(m.value * 100)}%`,
      m.value === null ? "not-applicable" : "",
    );
    const progress = el("progress");
    progress.max = m.denominator || 1;
    progress.value = m.numerator;
    progress.setAttribute(
      "aria-label",
      message("of", human(key), m.numerator, m.denominator),
    );
    card.append(
      label,
      value,
      progress,
      el(
        "small",
        message(
          "ofc71c",
          m.numerator,
          m.denominator,
          key.includes("criteria") ? copy.text.criteria : copy.text.items,
        ),
      ),
    );
    const link = actionButton(
      copy.text.reviewContributingItems,
      () =>
        key.includes("criteria")
          ? showEntity(state.snapshot.outcome.ref).catch(report)
          : navigateSection(
              key === "work_completion" ? "work_items" : "objectives",
            ),
      "text-button",
    );
    card.append(link);
    $("metrics").append(card);
  }
}
function summaryCard(title, section) {
  const card = el("section", undefined, "summary-card");
  const heading = el("div", undefined, "card-heading");
  heading.append(el("h2", title));
  if (section) {
    const link = el("button", copy.text.viewAll, "text-button");
    link.onclick = () => navigateSection(section);
    heading.append(link);
  }
  card.append(heading);
  return card;
}
function renderSummary() {
  const root = $("summary-content");
  root.replaceChildren();
  const s = state.snapshot,
    c = s.counts;
  const purpose = summaryCard(copy.text.desiredOutcome);
  purpose.classList.add("summary-purpose-card");
  purpose.append(
    el(
      "p",
      state.outcome.desired_state ||
        state.outcome.description ||
        state.outcome.title,
      "summary-purpose",
    ),
  );
  const quick = el("div", undefined, "summary-quick");
  for (const [label, command] of [
    [copy.text.addObjective, "create_objective"],
    [copy.text.newTask3e9922, "create_work_item"],
    [copy.text.registerEvidence, "register_evidence"],
  ]) {
    if (!state.permissions.includes(inventory[command]?.permission)) continue;
    const b = el("button", label, "quiet");
    b.onclick = () => createItem(command);
    quick.append(b);
  }
  purpose.append(quick);
  const review = actionButton(
    copy.text.reviewOutcome,
    () => showEntity(state.snapshot.outcome.ref).catch(report),
    "quiet",
  );
  review.id = "review-outcome";
  quick.append(review);
  root.append(purpose);
  if (state.outcome.lifecycle === "draft") {
    const setup = summaryCard(copy.text.prepareYourOutcome);
    setup.classList.add("setup-checklist");
    setup.append(
      el("p", copy.text.thisDraftIsSavedReviewTheseStepsBeforeExplicitly),
    );
    const rows = [
      [
        copy.text.defineSuccessCriteria,
        (state.outcome.criteria?.items || []).some((c) => !c.retired_at),
        () => {
          state.selected = { ...state.outcome, _kind: "outcome" };
          openCommands("add_criterion");
        },
        "planning:write",
      ],
      [
        copy.text.addAnObjective,
        c.objectives > 0,
        () => createItem("create_objective"),
        "planning:write",
      ],
      [
        copy.text.addATaskOptional,
        c.work_items > 0,
        () => createItem("create_work_item"),
        "work:write",
      ],
      [
        copy.text.createARoadmapOptional,
        c.roadmaps > 0 || c.active_roadmaps > 0,
        () => createItem("create_roadmap"),
        "planning:write",
      ],
    ];
    for (const [label, done, run, permission] of rows) {
      const row = el("div", undefined, "checklist-row");
      row.append(el("span", done ? "✓" : "○"), el("span", label));
      if (state.permissions.includes(permission))
        row.append(
          actionButton(
            done ? copy.text.review : copy.text.setUp,
            done
              ? () =>
                  label.includes("criteria")
                    ? showEntity(state.snapshot.outcome.ref)
                    : navigateSection(
                        label.includes(copy.text.objective)
                          ? "objectives"
                          : label.includes(copy.text.task4bc74b)
                            ? "work_items"
                            : "roadmaps",
                      )
              : run,
            "quiet",
          ),
        );
      setup.append(row);
    }
    if (state.permissions.includes("outcome:write"))
      setup.append(
        actionButton(copy.text.activateOutcome, () => {
          state.selected = { ...state.outcome, _kind: "outcome" };
          openCommands("activate_outcome");
        }),
      );
    root.append(setup);
  }
  const next = summaryCard(copy.text.suggestedNextStep);
  const suggestion = usesWorkContracts()
    ? {
        text: copy.text.reviewCurrentWork,
        reason: copy.text.workspaceContractAuthority,
        section: "work_items",
      }
    : c.work_items &&
        (c.done_work || 0) + (c.cancelled_work || 0) === c.work_items
      ? {
          text: copy.text.workspaceReviewCriteria,
          reason:
            copy.text
              .taskCompletionDoesNotCertifyTheOutcomeReviewRequirementsAndR37f94561,
          inspect: true,
        }
      : c.blocked_work
        ? {
            text: copy.text.reviewBlockingImpact,
            reason: copy.text.blockedTasksArePresentInTheCurrentState,
            section: "blockers",
          }
        : !(state.outcome.criteria?.items || []).length
          ? {
              text: copy.text.defineSuccessCriteria,
              reason: copy.text.noSuccessCriteriaAreRecordedYet,
              command: "add_criterion",
            }
          : !c.work_items
            ? {
                text: copy.text.addATask,
                reason: copy.text.noTasksAreRecordedYet,
                command: "create_work_item",
              }
            : {
                text: copy.text.reviewCurrentWork,
                reason: copy.text.tasksAreAvailableForExplicitCoordination,
                section: "work_items",
              };
  next.append(
    el("p", suggestion.reason),
    el("small", copy.text.aPresentationHintFromPersistedStateNoPlanningOr),
    actionButton(
      suggestion.text,
      () => {
        if (suggestion.inspect)
          showEntity(state.snapshot.outcome.ref).catch(report);
        else if (suggestion.section) navigateSection(suggestion.section);
        else {
          state.selected = { ...state.outcome, _kind: "outcome" };
          openCommands(suggestion.command);
        }
      },
      "quiet",
    ),
  );
  root.append(next);

  const status = summaryCard(copy.text.taskOverview, "work_items");
  const done = c.done_work || 0,
    active = usesWorkContracts() ? 0 : c.in_progress_work || 0,
    blocked = usesWorkContracts()
      ? 0
      : (c.blocked_work || 0) + (c.attention_needed_work || 0),
    total = c.work_items || 0;
  const parts = [
    {
      title: usesWorkContracts()
        ? copy.text.workspaceOpenWork
        : copy.text.toDoPlanned,
      count: Math.max(
        0,
        total - done - active - blocked - (c.cancelled_work || 0),
      ),
      color: "var(--wos-border)",
      tone: "",
    },
    {
      title: copy.text.inProgress,
      count: active,
      color: "var(--wos-warning)",
      tone: "orange",
    },
    {
      title: copy.text.blockedNeedsAttention,
      count: blocked,
      color: "var(--wos-error)",
      tone: "red",
    },
    {
      title: copy.text.done,
      count: done,
      color: "var(--wos-success)",
      tone: "green",
    },
    {
      title: copy.text.cancelled,
      count: c.cancelled_work || 0,
      color: "var(--wos-muted)",
      tone: "violet",
    },
  ];
  const shownParts = usesWorkContracts()
    ? parts.filter((_, index) => ![1, 2].includes(index))
    : parts;
  const body = el("div", undefined, "summary-stats");
  body.append(donut(shownParts, total));
  const legend = el("div", undefined, "legend");
  for (const part of shownParts) {
    const row = el("div", undefined, "legend-row");
    row.append(
      el("span", undefined, `legend-dot ${part.tone}`),
      el("span", part.title),
      el("strong", String(part.count)),
    );
    legend.append(row);
  }
  body.append(legend);
  status.append(body);
  if (usesWorkContracts())
    status.append(
      el("p", copy.text.workspaceContractAuthority, "section-note"),
    );
  root.append(status);
  const proof = summaryCard(copy.text.continueWithContext);
  const counts = el("div", undefined, "summary-counts");
  for (const [key, title] of [
    ["objectives", copy.text.objectives],
    ["evidence", copy.text.evidence],
    ["roadmaps", copy.text.roadmaps],
  ]) {
    const b = el("button");
    b.append(
      el(
        "strong",
        key === "roadmaps"
          ? String(c.active_roadmaps || 0)
          : String(c[key] || 0),
      ),
      el("span", key === "roadmaps" ? copy.text.activeRoadmaps : title),
    );
    b.onclick = () => navigateSection(key);
    counts.append(b);
  }
  proof.append(
    counts,
    el(
      "p",
      copy.text
        .taskCompletionDoesNotCertifyTheOutcomeReviewRequirementsAndR37f94561,
    ),
  );
  root.append(proof);
  const recent = summaryCard(copy.text.currentTasks, "work_items");
  recent.classList.add("summary-recent");
  const items = (
    usesWorkContracts()
      ? [...(s.sections.work_items || [])]
      : [
          ...(s.sections.ready_work || []),
          ...(s.sections.in_progress_work || []),
          ...(s.sections.blocked_work || []),
        ]
  ).slice(0, 3);
  if (!items.length) items.push(...(s.sections.work_items || []).slice(0, 3));
  for (const item of items) recent.append(itemView(item));
  if (!items.length)
    recent.append(el("p", copy.text.createATaskToOrganizeTheNextStep));
  root.append(recent);
  if (blocked || c.conclusion_contestations) {
    const alert = summaryCard(
      copy.text.itemsNeedingAttention,
      c.conclusion_contestations ? "conclusion_contestations" : "blocked_work",
    );
    alert.classList.add("summary-alert");
    alert.append(
      el(
        "p",
        message(
          "blockedItemsOrReservationsToReviewContestedConclusionsRecorded",
          blocked,
          c.conclusion_contestations || 0,
        ),
      ),
    );
    root.append(alert);
  }
}
function titleOf(ref) {
  if (!ref) return copy.text.item;
  if (ref.kind === "outcome" && ref.id === state.outcome?.id)
    return state.outcome.title;
  return (
    state.entities.get(`${ref.kind}/${ref.id}`)?.title ||
    `${kindName(ref.kind)} · ${shortId(ref.id)}`
  );
}
function referenceButton(ref) {
  const b = el("button", titleOf(ref), "text-button");
  b.onclick = () => showEntity(ref).catch(report);
  return b;
}
function operational(item) {
  const e = item.work_item || item.current || item;
  if (usesWorkContracts() && e.ref?.kind === "work_item")
    return { display_state: e.lifecycle, readiness_reasons: [] };
  return (
    item.state ||
    item.operational_state ||
    item.readiness ||
    operationalSections
      .flatMap((key) => state.snapshot.sections[key] || [])
      .find((x) => x.work_item.ref.id === e.ref?.id)?.state
  );
}
function usesWorkContracts() {
  return [
    "contracts_v1",
    "signed_contracts_v2",
    "draining_to_signed_v2",
  ].includes(state.protocol?.phase);
}
function collectionHasPriority() {
  return state.area === "work" || state.section === "objectives";
}
function matches(item) {
  const e = item.work_item || item.current || item;
  const query = $("item-search").value.trim().toLocaleLowerCase("en-US");
  const refs = [
    e.ref,
    item.entity_ref,
    item.owner_ref,
    item.from,
    item.to,
    item.source_ref,
    item.target_ref,
    item.slot && { kind: "roadmap", id: item.slot.roadmap_id },
  ];
  const searchable = [
    e.title,
    e.description,
    item.plan_label,
    item.slot && copy.text.active,
    item.slot &&
      message(
        "revisionPublishedItems",
        item.slot.revision_number,
        item.node_count,
      ),
    display(e.lifecycle || ""),
    item.event_type && eventName(item.event_type),
    item.event_type,
    item.actor_ref?.id,
    item.actor_ref?.kind,
    item.principal_id,
    item.outcome_revision,
    item.recorded_at,
    item.kind,
    item.type,
    item.relation_type,
    ...refs
      .filter(Boolean)
      .flatMap((ref) => [
        ref.id,
        shortId(ref.id),
        kindName(ref.kind),
        titleOf(ref),
      ]),
  ]
    .filter((value) => value !== undefined && value !== null)
    .join(" ")
    .toLocaleLowerCase("en-US");
  return (
    (!query || searchable.includes(query)) &&
    (!collectionHasPriority() ||
      !$("item-priority").value ||
      e.priority === $("item-priority").value)
  );
}
function itemView(item, mode = "card") {
  if (item.slot) {
    const card = el("div", undefined, "item");
    const ref = {
      namespace_id: state.namespace,
      outcome_id: state.outcome.id,
      kind: "roadmap",
      id: item.slot.roadmap_id,
    };
    card.append(
      referenceButton(ref),
      el(
        "small",
        message(
          "revisionPublishedItems",
          item.slot.revision_number,
          item.node_count,
        ),
      ),
      badge("active"),
    );
    return card;
  }
  if (item.entity_ref && item.event_type) return timelineView(item);
  if (item.owner_ref && item.kind) {
    const card = el("div", undefined, "item");
    card.append(el("h3", display(item.kind)), referenceButton(item.owner_ref));
    const evidence = item.evidence_id || item.evidence_ref?.id;
    if (evidence)
      card.append(
        referenceButton({
          namespace_id: state.namespace,
          outcome_id: state.outcome.id,
          kind: "evidence",
          id: evidence,
        }),
      );
    card.append(
      el(
        "p",
        copy.text
          .theConclusionRemainsInHistoryReviewTheNewObservationBeforeRe45f90af8,
      ),
    );
    return card;
  }
  if (item.target_ref && item.missing) {
    const card = el("div", undefined, "item");
    card.append(
      el(
        "h3",
        item.plan_label ||
          item.published_reference_title ||
          kindName(item.target_ref.kind),
      ),
      el("p", copy.text.workspaceMissingPlanReference),
      el("small", shortId(item.target_ref.id)),
      referenceButton({
        ...item.target_ref,
        kind: "roadmap",
        id: item.roadmap_id,
      }),
    );
    return card;
  }
  const entity = item.work_item || item.current || item;
  if (!entity.ref) {
    const card = el("div", undefined, "item");
    card.append(readable(item));
    return card;
  }
  state.entities.set(`${entity.ref.kind}/${entity.ref.id}`, entity);
  const status = operational(item);
  const button = el("button", undefined, mode === "list" ? "list-row" : "item");
  button.dataset.entityId = entity.ref.id;
  button.dataset.entityKind = entity.ref.kind;
  button.onclick = () => showEntity(entity.ref).catch(report);
  const main = el("span", undefined, "item-main");
  const symbol = el("span", undefined, `entity-symbol ${entity.ref.kind}`);
  symbol.append(icon(entity.ref.kind));
  const itemCopy = el("span");
  itemCopy.append(
    el("span", entity.title || kindName(entity.ref.kind), "item-title"),
    el(
      "span",
      `${mode === "list" && !collectionHasPriority() ? "" : shortId(entity.ref.id) + " · "}${kindName(entity.ref.kind)}${item.plan_label ? ` · ${message("workspaceHistoricalPlanLabel", item.plan_label, item.revision_number)}` : ""}`,
      "item-subtitle",
    ),
  );
  if (
    item.published_reference_title &&
    item.published_reference_title !== entity.title
  )
    itemCopy.append(
      el(
        "span",
        message("workspacePublishedTitle", item.published_reference_title),
        "item-parent",
      ),
    );
  main.append(symbol, itemCopy);
  button.append(main);
  const lifecycle = status?.display_state || entity.lifecycle;
  const stateCell = el("span", undefined, "item-state");
  stateCell.append(badge(lifecycle));
  if (mode === "list") {
    const priority = el(
      "span",
      !collectionHasPriority()
        ? shortId(entity.ref.id)
        : entity.priority
          ? `${entity.priority === "low" ? "↓" : entity.priority === "normal" ? "—" : "↑"} ${display(entity.priority)}`
          : "—",
      collectionHasPriority()
        ? `item-priority ${entity.priority || ""}`
        : "item-reference",
    );
    if (!collectionHasPriority()) priority.title = entity.ref.id;
    button.append(stateCell, priority);
  } else {
    const footer = el("span", undefined, "board-card-footer");
    footer.append(
      stateCell,
      el(
        "span",
        entity.priority
          ? `${entity.priority === "low" ? "↓" : entity.priority === "normal" ? "—" : "↑"} ${display(entity.priority)}`
          : "",
        `item-priority ${entity.priority || ""}`,
      ),
    );
    button.append(footer);
    if (
      status?.readiness_reasons?.length &&
      [
        "blocked",
        "attention_needed",
        "waiting_dependencies",
        "scheduled",
      ].includes(status.display_state)
    )
      button.append(
        el(
          "small",
          status.readiness_reasons.map(display).join(copy.text.message),
          "board-reason",
        ),
      );
    if (item.plan_label)
      button.append(
        el(
          "small",
          message("roadmapRevision592f", item.plan_label, item.revision_number),
        ),
      );
  }
  return button;
}
function timelineView(item) {
  const row = el("div", undefined, "timeline-entry");
  const marker = el("span", undefined, "timeline-marker");
  marker.append(icon(item.actor_ref?.kind === "agent" ? "work_item" : "user"));
  const body = el("div");
  body.className = "timeline-body";
  const recorded = el("time", date(item.recorded_at));
  recorded.dateTime = item.recorded_at;
  body.append(
    Object.assign(el("h3", eventName(item.event_type)), {
      title: item.event_type,
    }),
    el(
      "p",
      `${item.actor_ref?.id || item.principal_id} · ${kindName(item.actor_ref?.kind || "human")} · Revision ${item.outcome_revision}`,
    ),
    referenceButton(item.entity_ref),
    recorded,
  );
  row.append(marker, body);
  return row;
}
function eventName(type) {
  const subjects = {
    work_item: copy.text.task4bc74b,
    objective: copy.text.objective,
    outcome: copy.text.outcome,
    roadmap: copy.text.roadmap,
    evidence: copy.text.evidence,
    issue: copy.text.issueKind,
    blocker: copy.text.blockerKind,
    decision: copy.text.decision640ae4,
    artifact: copy.text.artifact,
    criterion: copy.text.successCriterion,
    relation: copy.text.relation136748,
  };
  const actions = {
    created: "created",
    updated: "updated",
    completed: copy.text.completed,
    claimed: "reserved",
    achieved: copy.text.achieved,
    activated: "activated",
    registered: "recorded",
    retracted: "retracted",
    resolved: "resolved",
    recorded: "recorded",
    published: "published",
    reopened: "reopened",
    archived: "archived",
    assessment_recorded: copy.text.assessmentRecorded,
    draft_opened: copy.text.draftOpened,
    draft_replaced: copy.text.draftEdited,
    draft_published: copy.text.revisionPublished,
  };
  const normalized = type
    .replace(/([a-z])([A-Z])/g, "$1_$2")
    .toLowerCase()
    .replace(/(^wos[._]|[._]v\d+$)/g, "")
    .replaceAll(".", "_");
  const subject = Object.keys(subjects).find((s) =>
    normalized.startsWith(s + "_"),
  );
  if (subject) {
    const action = normalized.slice(subject.length + 1);
    return `${subjects[subject]} · ${actions[action] || display(action)}`;
  }
  return display(normalized);
}
function emptySection(title, command) {
  if (title === copy.text.noRecordsInThisSectionYet)
    title =
      {
        work_items: copy.text.noTasksYet,
        objectives: copy.text.noObjectivesYet,
        roadmaps: copy.text.noRoadmapsYet,
        evidence: copy.text.noEvidenceYet,
        issues: copy.text.noIssuesYet,
        blockers: copy.text.noBlockersYet,
      }[state.section] || title;
  const box = el("div", undefined, "empty-section");
  const filtered = title === copy.text.noItemsMatchTheFilters;
  box.append(
    sectionIllustration(state.area),
    el("h3", title),
    el("p", filtered ? copy.text.workspaceFilterEmpty : sectionGuidance()),
  );
  if (filtered)
    box.append(
      actionButton(
        copy.text.workspaceFilterReset,
        () => {
          $("item-search").value = "";
          $("item-priority").value = "";
          renderSection().catch(report);
        },
        "quiet",
      ),
    );
  if (command && state.permissions.includes(inventory[command]?.permission)) {
    const b = el("button", actionName(command), "quiet");
    b.onclick = () => createItem(command);
    box.append(b);
  }
  return box;
}
function sectionGuidance() {
  if (state.area === "work" && state.section !== "work_items")
    return copy.text.taskCompletionDoesNotCertifyTheOutcomeReviewRequirementsAndR37f94561;
  const keys = {
    work_items: "workspaceNoTasks",
    objectives: "workspaceNoObjectives",
    roadmaps: "workspaceNoRoadmaps",
    evidence: "workspaceNoEvidence",
    artifacts: "workspaceNoArtifacts",
    decisions: "workspaceNoDecisions",
    issues: "workspaceNoIssues",
    blockers: "workspaceNoBlockers",
    timeline: "workspaceNoActivity",
    graph: "workspaceNoRelations",
    active_roadmaps: "workspaceNoRoadmaps",
    active_plan_references: "workspaceNoPlanReferences",
    conclusion_contestations: "workspaceNoContestations",
  };
  return (
    copy.text[keys[state.section]] ||
    copy.text.activityAppearsHereAsWorkProgresses
  );
}
function collectionStatus(error) {
  const box = el("div", undefined, "collection-status");
  box.dataset.renderState = error ? "error" : "loading";
  box.setAttribute("role", error ? "alert" : "status");
  box.append(
    el("p", human(state.section), "eyebrow"),
    el(
      "h3",
      error
        ? copy.text.workspaceCollectionError
        : copy.text.workspaceCollectionLoading,
    ),
  );
  if (error) {
    box.append(el("p", error.message));
    box.append(
      actionButton(
        copy.text.workspaceReloadView,
        () => renderSection().catch(report),
        "quiet",
      ),
    );
  }
  return box;
}
async function sectionData(section, cursor = "", append = false) {
  const cached = state.loaded.get(section);
  if (!cursor && cached) return cached;
  const revision = state.snapshot.outcome_revision;
  const consistent = (page) => {
    if (canvasRevision(page.outcome_revision) !== canvasRevision(revision))
      throw Error(copy.text.stateChangedWhileLoadingRefreshForAConsistentView);
    return page;
  };
  let items, next;
  if (section === "timeline") {
    const page = consistent(
      await request(
        `${outcomeBase()}/timeline?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
      ),
    );
    items = page.items;
    next = page.next_cursor;
  } else if (section === "graph") {
    const page = consistent(
      await request(
        `${outcomeBase()}/graph?depth=3&limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
      ),
    );
    items = page.edges;
    next = page.next_cursor;
  } else if (section === "roadmaps") {
    const page = await request(
      `${outcomeBase()}/references?kind=roadmap&limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    consistent(page);
    items = page.items || [];
    next = page.next_cursor;
  } else if (!cursor) {
    items = state.snapshot.sections[section] || [];
    next = state.snapshot.section_cursors[section];
  } else {
    const page = consistent(
      await request(
        `${outcomeBase()}/continuity/${section}?limit=25&cursor=${encodeURIComponent(cursor)}`,
      ),
    );
    items = page.items;
    next = page.next_cursor;
  }
  return {
    items: append ? [...(cached?.items || []), ...(items || [])] : items || [],
    next,
  };
}
async function searchedTasks(cursor = "", append = false) {
  const scope = outcomeBase(),
    generation = state.generation;
  const q = new URLSearchParams({
    kind: "work_item",
    limit: "25",
    query: $("item-search").value,
  });
  if ($("item-priority").value) q.set("priority", $("item-priority").value);
  if (cursor) q.set("cursor", cursor);
  const page = await request(`${scope}/references?${q}`);
  const results = await Promise.all(
    page.items.map((item) => request(`${scope}/work-items/${item.ref.id}`)),
  );
  if (generation !== state.generation) return { items: [] };
  if (
    results.some(
      (r) =>
        canvasRevision(r.outcome_revision) !==
        canvasRevision(page.outcome_revision),
    )
  ) {
    const e = Error(
      copy.text.stateChangedWhileLoadingSearchResultsRefreshTheSearch,
    );
    e.code = "precondition_failed";
    throw e;
  }
  const items = results.map((r, i) => ({ ...r.value, ref: page.items[i].ref }));
  return {
    items: append
      ? [...(state.loaded.get("work_items")?.items || []), ...items]
      : items,
    next: page.next_cursor,
  };
}
async function renderSection(cursor = "", append = false) {
  // Explicit navigation/filter/pagination consumes the current input. A queued
  // search must not later discard the page the user just loaded.
  clearTimeout(searchTimer);
  if (!state.snapshot || state.view === "summary" || canvasActive()) return;
  if (state.view === "board") {
    state.renderId++;
    return renderBoard();
  }
  const section = state.section,
    generation = state.generation,
    snapshot = state.snapshot,
    run = ++state.renderId;
  const current = () =>
    generation === state.generation &&
    snapshot === state.snapshot &&
    run === state.renderId;
  const content = $("content");
  content.setAttribute("aria-busy", "true");
  if (!append) content.replaceChildren(collectionStatus());
  const globalWork = section === "work_items";
  let data;
  try {
    data = globalWork
      ? await searchedTasks(cursor, append)
      : await sectionData(section, cursor, append);
  } catch (error) {
    if (current()) {
      content.setAttribute("aria-busy", "false");
      content.replaceChildren(collectionStatus(error));
      throw error;
    }
    if ([401, 403].includes(error.status) && !error.discarded) throw error;
    return;
  }
  if (!current()) return;
  if (section === "objectives") {
    const path = outcomeBase(),
      revision = state.snapshot.outcome_revision;
    const missing = data.items.filter((x) => !x._hierarchy_loaded);
    let hydrated;
    try {
      hydrated = await Promise.all(
        missing.map(async (x) => {
          const r = await request(`${path}/objectives/${x.ref.id}`);
          if (canvasRevision(r.outcome_revision) !== canvasRevision(revision))
            throw Error(
              copy.text.stateChangedWhileLoadingRefreshForAConsistentView,
            );
          return { ...r.value, ref: x.ref, _hierarchy_loaded: true };
        }),
      );
    } catch (error) {
      if (!current()) {
        if ([401, 403].includes(error.status) && !error.discarded) throw error;
        return;
      }
      content.setAttribute("aria-busy", "false");
      content.replaceChildren(collectionStatus(error));
      throw error;
    }
    if (!current()) return;
    const map = new Map(hydrated.map((x) => [x.ref.id, x]));
    data.items = data.items.map((x) => map.get(x.ref.id) || x);
  }
  state.loaded.set(section, data);
  content.setAttribute("aria-busy", "false");
  content.replaceChildren();
  const heading = el("div", undefined, "content-heading");
  const headingCopy = el("div", undefined, "collection-heading-copy");
  const shown = globalWork ? data.items : data.items.filter(matches);
  headingCopy.append(
    el("p", copy.text.workspaceCollectionContext, "eyebrow"),
    el("h2", human(section)),
  );
  heading.append(
    headingCopy,
    el(
      "small",
      message(
        "itemsShown",
        shown.length,
        state.snapshot.counts[section] !== undefined
          ? ` · ${state.snapshot.counts[section]} total`
          : "",
      ),
    ),
  );
  content.append(heading);
  content.append(el("p", sectionGuidance(), "collection-guidance"));
  if (state.workFilterReset)
    content.append(
      el("p", copy.text.workspaceContractFilterReset, "section-note"),
    );
  if (globalWork)
    content.append(
      el(
        "p",
        copy.text.searchAndPriorityFilterCoverEveryAuthorizedTaskPages,
        "section-note",
      ),
    );
  for (const b of $("tabs").children) {
    b.classList.toggle("selected", b.dataset.section === section);
    b.setAttribute("aria-pressed", String(b.dataset.section === section));
  }
  if (!globalWork)
    content.append(
      el("p", copy.text.filterAppliesToLoadedItemsInThisView, "section-note"),
    );
  if (!shown.length)
    content.append(
      emptySection(
        data.items.length || $("item-search").value || $("item-priority").value
          ? copy.text.noItemsMatchTheFilters
          : copy.text.noRecordsInThisSectionYet,
        data.items.length || $("item-search").value || $("item-priority").value
          ? ""
          : sectionCommand(),
      ),
    );
  const list = el("div", undefined, "collection");
  let rows = 0;
  let activityDate, activityGroup;
  for (const item of shown) {
    const entity = item.work_item || item.current || item;
    if (entity.ref) {
      if (!rows) {
        const head = el("div", undefined, "list-head");
        head.append(
          el("span", copy.text.item),
          el("span", copy.text.lifecycle),
          el(
            "span",
            collectionHasPriority() ? copy.text.priority : copy.text.reference,
          ),
        );
        list.append(head);
        content.append(list);
      }
      const row = itemView(item, "list");
      if (section === "objectives" && entity.parent_objective_id) {
        const parent = data.items.find(
          (x) => x.ref.id === entity.parent_objective_id,
        );
        row.classList.add("objective-child");
        row
          .querySelector(".item-main > span:last-child")
          .append(
            el(
              "span",
              parent
                ? message("withinObjective", parent.title)
                : copy.text.parentObjectiveOutsidePage,
              "item-parent",
            ),
          );
        row.title = parent
          ? message("withinObjective", parent.title)
          : copy.text.parentObjectiveOutsidePage;
      }
      list.append(row);
      rows++;
    } else if (section === "timeline" && item.event_type) {
      const day = String(item.recorded_at || "").slice(0, 10);
      if (!activityGroup || activityDate !== day) {
        activityDate = day;
        activityGroup = el("section", undefined, "activity-day");
        activityGroup.append(
          el("h3", day || copy.text.workspaceActivityDay, "activity-day-label"),
        );
        content.append(activityGroup);
      }
      activityGroup.append(timelineView(item));
    } else if (section === "graph") {
      const ref = item.from || item.source_ref,
        other = item.to || item.target_ref;
      const row = el("div", undefined, "relation-card");
      if (ref && other)
        row.append(
          referenceButton(ref),
          el("span", "→", "relation-arrow"),
          referenceButton(other),
          badge(
            item.kind ||
              item.type ||
              item.relation_type ||
              copy.text.relation136748,
          ),
        );
      else row.append(readable(item));
      content.append(row);
    } else content.append(itemView(item));
  }
  if (data.next) {
    const more = el("button", copy.text.loadMore, copy.text.quietMoreButton);
    more.dataset.more = "true";
    more.onclick = () => renderSection(data.next, true).catch(report);
    content.append(more);
  }
}
const boardGroups = [
  {
    title: copy.text.planned,
    tone: "violet",
    sections: [
      "backlog_work",
      "waiting_scope_work",
      "waiting_dependencies_work",
      "scheduled_work",
    ],
  },
  { title: copy.text.ready, tone: "", sections: ["ready_work"] },
  {
    title: copy.text.inProgress,
    tone: "orange",
    sections: ["in_progress_work"],
  },
  {
    title: copy.text.blockers,
    tone: "red",
    sections: ["blocked_work", "attention_needed_work"],
  },
  {
    title: copy.text.done,
    tone: "green",
    sections: ["done_work"],
  },
];
function renderBoard() {
  if (usesWorkContracts()) return renderContractBoard();
  const content = $("content");
  content.replaceChildren(
    el(
      "p",
      copy.text
        .theBoardShowsCurrentReadinessAndReservationsOpenATaskToChooseAnAction,
      "board-caption",
    ),
  );
  const board = el("div", undefined, "board");
  const groups = [...boardGroups];
  if (state.snapshot.counts.cancelled_work)
    groups.push({
      title: copy.text.cancelled,
      tone: "violet",
      sections: ["cancelled_work"],
    });
  board.classList.toggle("board-six", groups.length === 6);
  for (const group of groups) {
    const column = el("section", undefined, "board-column");
    column.setAttribute("aria-label", group.title);
    const heading = el("div", undefined, "column-heading");
    const count = group.sections.reduce(
      (n, s) => n + (state.snapshot.counts[s] || 0),
      0,
    );
    heading.append(
      el("span", undefined, `legend-dot ${group.tone}`),
      el("strong", group.title),
      el("span", String(count), "count"),
    );
    column.append(heading);
    const items = group.sections
      .flatMap(
        (s) =>
          state.boardLoaded.get(s)?.items || state.snapshot.sections[s] || [],
      )
      .filter(matches);
    for (const item of items) column.append(itemView(item));
    if (!items.length)
      column.append(
        el(
          "p",
          count ? copy.text.noLoadedItemsMatchTheFilters : copy.text.noItems,
          "board-empty",
        ),
      );
    for (const section of group.sections) {
      const cursor =
        state.boardLoaded.get(section)?.next ??
        (state.boardLoaded.has(section)
          ? ""
          : state.snapshot.section_cursors[section]);
      if (cursor) {
        const more = el(
          "button",
          message("loadMore1c89", display(section.replace("_work", ""))),
          copy.text.quietMoreButton,
        );
        more.onclick = async () => {
          const generation = state.generation,
            snapshot = state.snapshot;
          more.disabled = true;
          try {
            const page = await request(
              `${outcomeBase()}/continuity/${section}?limit=25&cursor=${encodeURIComponent(cursor)}`,
            );
            if (generation !== state.generation || snapshot !== state.snapshot)
              return;
            if (
              canvasRevision(page.outcome_revision) !==
              canvasRevision(snapshot.outcome_revision)
            )
              throw Error(
                copy.text.stateChangedWhileLoadingRefreshForAConsistentView,
              );
            const previous =
              state.boardLoaded.get(section)?.items ||
              state.snapshot.sections[section] ||
              [];
            state.boardLoaded.set(section, {
              items: [
                ...new Map(
                  [...previous, ...page.items].map((item) => [
                    (item.work_item || item.current || item).ref.id,
                    item,
                  ]),
                ).values(),
              ],
              next: page.next_cursor || "",
            });
            if (state.view === "board") renderBoard();
          } catch (e) {
            if (
              (generation === state.generation &&
                snapshot === state.snapshot) ||
              [401, 403].includes(e.status)
            )
              report(e);
            more.disabled = false;
          }
        };
        column.append(more);
      }
    }
    board.append(column);
  }
  content.append(board);
  if (
    state.snapshot.truncated ||
    $("item-search").value ||
    $("item-priority").value
  )
    content.append(
      el(
        "p",
        copy.text.countsCoverTheFullStateCardsAndFiltersIncludeLoadedItems,
        "section-note",
      ),
    );
}
function renderContractBoard() {
  const content = $("content"),
    snapshot = state.snapshot;
  content.setAttribute("aria-busy", "false");
  content.replaceChildren(
    el("p", copy.text.workspaceLoadedTaskBoard, "board-caption"),
  );
  const page = state.boardLoaded.get("work_items");
  const loaded = page?.items || snapshot.sections.work_items || [];
  const board = el("div", undefined, "board contract-board");
  const groups = [
    ["backlog", copy.text.backlog],
    ["todo", display("todo")],
    ["in_progress", copy.text.inProgress],
    ["done", copy.text.done],
    ["cancelled", copy.text.cancelled],
  ];
  for (const [lifecycle, label] of groups) {
    const column = el("section", undefined, "board-column");
    column.setAttribute("aria-label", label);
    const records = loaded
      .filter(
        (entry) =>
          (entry.work_item || entry.current || entry).lifecycle === lifecycle,
      )
      .filter(matches);
    const heading = el("div", undefined, "column-heading");
    heading.append(
      el("strong", label),
      el("span", String(records.length), "count"),
    );
    column.append(heading);
    for (const item of records) column.append(itemView(item));
    if (!records.length)
      column.append(
        el("p", copy.text.noLoadedItemsMatchTheFilters, "board-empty"),
      );
    board.append(column);
  }
  content.append(
    board,
    el(
      "p",
      message(
        "itemsShown",
        loaded.filter(matches).length,
        ` · ${snapshot.counts.work_items || 0} total`,
      ),
      "section-note",
    ),
  );
  const cursor = page ? page.next : snapshot.section_cursors.work_items;
  if (cursor) {
    const more = actionButton(
      copy.text.loadMore,
      async () => {
        const generation = state.generation;
        more.disabled = true;
        try {
          const next = await request(
            `${outcomeBase()}/continuity/work_items?limit=25&cursor=${encodeURIComponent(cursor)}`,
          );
          if (generation !== state.generation || snapshot !== state.snapshot)
            return;
          if (
            canvasRevision(next.outcome_revision) !==
            canvasRevision(snapshot.outcome_revision)
          )
            throw Error(
              copy.text.stateChangedWhileLoadingRefreshForAConsistentView,
            );
          const unique = new Map(
            [...loaded, ...next.items].map((item) => [
              (item.work_item || item.current || item).ref.id,
              item,
            ]),
          );
          state.boardLoaded.set("work_items", {
            items: [...unique.values()],
            next: next.next_cursor || "",
          });
          if (state.area === "work" && state.view === "board") renderBoard();
        } catch (error) {
          if (
            (generation === state.generation && snapshot === state.snapshot) ||
            [401, 403].includes(error.status)
          )
            report(error);
        } finally {
          more.disabled = false;
        }
      },
      "quiet more-button",
    );
    more.dataset.more = "true";
    content.append(more);
  }
}
const plurals = {
  objective: "objectives",
  work_item: "work-items",
  issue: "issues",
  blocker: "blockers",
  decision: "decisions",
  evidence: "evidence",
  artifact: "artifacts",
  roadmap: "roadmaps",
  relation: "relations",
  evidence_link: "evidence-links",
};
function readable(value, key = "", depth = 0, authored = false) {
  authored =
    authored ||
    ["external_context", "metadata", "execution_context"].includes(key);
  if (value === null || value === undefined) return el("span", "—");
  if (typeof value !== "object")
    return el("span", authored ? String(value) : formatValue(value, key));
  if (depth > 4)
    return el("span", copy.text.openTechnicalDetailsForMoreInformation);
  if (value.kind && value.id && value.namespace_id)
    return referenceButton(value);
  if (Array.isArray(value)) {
    const list = el("div");
    if (!value.length) list.append(el("span", copy.text.noRecords));
    for (const v of value) {
      const row = el("div", undefined, "readable-row");
      row.append(readable(v, key, depth + 1, authored));
      list.append(row);
    }
    return list;
  }
  const list = el("dl", undefined, "detail-properties");
  for (const [k, v] of Object.entries(value)) {
    if (
      v === null ||
      v === undefined ||
      [
        "namespace_id",
        "outcome_id",
        "id",
        "scope",
        "content_hash",
        "command_id",
        "event_id",
        "checksum",
      ].includes(k)
    )
      continue;
    list.append(el("dt", authored ? k : human(k)));
    const dd = el("dd");
    dd.append(readable(v, k, depth + 1, authored));
    list.append(dd);
  }
  return list;
}
function conclusionView(conclusion) {
  const card = el("div", undefined, "conclusion-card");
  const status = conclusion.lifecycle_result || "achieved";
  card.append(
    badge(status),
    el("p", conclusion.reason),
    el(
      "small",
      `${conclusion.actor_ref?.id || conclusion.principal_id} · ${date(conclusion.concluded_at)}`,
    ),
  );
  if (conclusion.assessments?.length)
    card.append(
      el(
        "p",
        message(
          "assessmentsUsedInTheConclusion",
          conclusion.assessments.length,
        ),
      ),
    );
  if (conclusion.obligations?.required_objective_ids?.length)
    card.append(
      el(
        "p",
        message(
          "requiredObjectivesVerified",
          conclusion.obligations.required_objective_ids.length,
        ),
      ),
    );
  return card;
}
function planNodes(nodes) {
  const box = el("div");
  for (const [i, n] of [...(nodes || [])]
    .sort((a, b) => a.position - b.position)
    .entries()) {
    const row = el(
      "div",
      undefined,
      `plan-node${n.parent_node_key ? copy.text.planNodeChild : ""}`,
    );
    const body = el("div");
    body.append(
      el("strong", n.title),
      el(
        "small",
        `${display(n.node_type)}${n.parent_node_key ? ` · ${(nodes || []).find((parent) => parent.node_key === n.parent_node_key)?.title || n.parent_node_key}` : ""}`,
      ),
    );
    if (n.target_ref) body.append(referenceButton(n.target_ref));
    if (n.planned_start || n.planned_end)
      body.append(
        el("small", `${date(n.planned_start)} → ${date(n.planned_end)}`),
      );
    row.append(
      el("span", String(i + 1).padStart(2, "0"), "plan-node-number"),
      body,
    );
    box.append(row);
  }
  if (!nodes?.length) box.append(el("p", copy.text.thisRoadmapHasNoItemsYet));
  return box;
}
function effectiveActions(
  context = state.criterion ? "criterion" : state.selected?._kind || "outcome",
  entity = state.selected || state.outcome || {},
) {
  return availableActions({
    context,
    entity,
    criterion: state.criterion,
    protocol: state.protocol,
    permissions: state.permissions,
    principal: state.principal,
    inventory,
  });
}
function quickActions(kind, entity) {
  const container = el("div", undefined, "quick-actions");
  const advanced = el("details");
  advanced.append(el("summary", copy.text.advanced));
  for (const action of effectiveActions(kind, { ...entity, _kind: kind })) {
    const destination = action.advanced ? advanced : container;
    destination.append(
      actionButton(
        action.label,
        () => {
          if (!allowCanvasNavigation()) return;
          state.selected = { ...entity, _kind: kind };
          openCommands(action.name);
        },
        "quiet",
      ),
    );
  }
  if (advanced.children.length > 1) container.append(advanced);
  return container;
}
async function showEntity(
  ref,
  { internal = false, preserveCamera = false } = {},
) {
  if (!internal && !allowCanvasNavigation()) return;
  const detailGeneration = ++state.detailGeneration;
  state.entityContextRefresh = null;
  const generation = state.generation;
  const isCanvas = canvasActive();
  try {
    if (
      ref.namespace_id !== state.namespace ||
      ref.outcome_id !== state.outcome.id
    ) {
      notice(copy.text.openTheCorrespondingOutcomeToViewThisItem, true);
      return;
    }
    const path =
      ref.kind === "outcome"
        ? outcomeBase()
        : `${outcomeBase()}/${plurals[ref.kind]}/${ref.id}`;
    const result = await request(path);
    if (
      generation !== state.generation ||
      detailGeneration !== state.detailGeneration
    )
      return;
    let operationalState, context, attention;
    if (ref.kind === "work_item") {
      const needsAttention = isCanvas || usesWorkContracts();
      const projection = needsAttention
        ? await request(`${path}/attention`)
        : await request(`${path}/operational-state`);
      if (needsAttention) attention = projection;
      if (isCanvas) {
        context = await request(`${path}/context`);
        if (
          generation !== state.generation ||
          detailGeneration !== state.detailGeneration
        )
          return;
        if (
          canvasRevision(context.outcome_revision) !==
          canvasRevision(result.outcome_revision)
        )
          throw Error(copy.text.thisTaskChangedWhileLoadingRefreshToReviewA);
      }
      if (
        generation !== state.generation ||
        detailGeneration !== state.detailGeneration
      )
        return;
      if (
        canvasRevision(result.outcome_revision) !==
        canvasRevision(projection.outcome_revision)
      )
        throw Error(copy.text.thisTaskChangedWhileLoadingRefreshToReviewA);
      operationalState = needsAttention
        ? attention.operational_state
        : projection.state;
    }
    if (generation !== state.generation) return;
    if (detailGeneration !== state.detailGeneration) return;
    if (
      isCanvas &&
      state.canvasModel &&
      canvasRevision(result.outcome_revision) !== state.canvasModel.revision
    )
      throw Error(copy.text.stateChangedWhileLoadingRefreshForAConsistentView);
    state.detailRevision = result.outcome_revision;
    const entity = {
      ...(result.value || result),
      ...(operationalState ? { _operational_state: operationalState } : {}),
    };
    const focusWasInDetails =
      isCanvas &&
      canvasInspector.isVisible("detail") &&
      $("canvas-details").contains(document.activeElement);
    state.contractView = null;
    state.submission = null;
    state.selected = { ...entity, _kind: ref.kind };
    if (isCanvas) state.canvasSelected = ref;
    state.criterion = null;
    state.entities.set(`${ref.kind}/${ref.id}`, {
      ref,
      title: entity.title || entity.name || entity.description,
      lifecycle: entity.lifecycle,
    });
    $("detail-kind").textContent = `${kindName(ref.kind)} / ${shortId(ref.id)}`;
    $("detail-title").textContent =
      entity.title || entity.name || kindName(ref.kind);
    const detail = $("detail-content");
    detail.replaceChildren();
    const status = el("div", undefined, "detail-status");
    if (entity.lifecycle) status.append(badge(entity.lifecycle));
    if (entity.priority)
      status.append(
        el("span", message("prioritye9f4", display(entity.priority)), "badge"),
      );
    if (entity.archived_at)
      status.append(el("span", copy.text.archived, "badge"));
    detail.append(status);
    if (entity._operational_state) {
      const readiness = el("section", undefined, "detail-section");
      readiness.append(
        el("h3", copy.text.operationalReadiness),
        badge(entity._operational_state.display_state),
        el(
          "p",
          usesWorkContracts()
            ? copy.text.workspaceContractAuthority
            : message(
                "reservationAuthorityEvaluated",
                display(entity._operational_state.lease_status),
                date(entity._operational_state.evaluated_at),
              ),
        ),
      );
      if (entity._operational_state.readiness_reasons?.length)
        readiness.append(
          el(
            "p",
            entity._operational_state.readiness_reasons
              .map(display)
              .join(copy.text.message),
          ),
        );
      detail.append(readiness);
    }
    const description =
      entity.desired_state ||
      entity.description ||
      entity.proposal ||
      entity.result_summary ||
      entity.reason ||
      entity.rationale;
    if (description) detail.append(el("p", description, "detail-description"));
    const relationshipHost = el("div");
    detail.append(relationshipHost);
    let relationshipContext, relationshipSnapshot;
    function refreshRelationships() {
      if (generation !== state.generation || detailGeneration !== state.detailGeneration ||
        !relationshipHost.isConnected) return;
      const options = {
        ref, entity, model: isCanvas ? state.canvasModel : undefined,
        placementOnly: ref.kind === "work_item" && isCanvas,
        // Reuse names from the current authorized snapshot outside the Board
        // too; labels neither add relationships nor trigger extra reads.
        referenceTitle: titleOf,
        onReference: (target) => showEntity(target).catch(report),
      };
      const snapshot = JSON.stringify(buildEntityContext(options));
      if (snapshot === relationshipSnapshot) return;
      relationshipSnapshot = snapshot;
      const active = relationshipHost.contains(document.activeElement) ? document.activeElement : null;
      const activeReference = active?.dataset.entityContextRef;
      const activeSection = active?.tagName === "SUMMARY" ? active.parentElement.dataset.entityContextSection : null;
      const openSections = new Set(Array.from(relationshipHost.querySelectorAll("details[open]"))
        .map((element) => element.dataset.entityContextSection));
      relationshipHost.replaceChildren();
      relationshipContext = renderEntityContext(relationshipHost, options);
      for (const disclosure of relationshipHost.querySelectorAll("details"))
        disclosure.open = openSections.has(disclosure.dataset.entityContextSection);
      const restored = activeReference
        ? Array.from(relationshipHost.querySelectorAll("[data-entity-context-ref]"))
          .find((element) => element.dataset.entityContextRef === activeReference)
        : activeSection
          ? Array.from(relationshipHost.querySelectorAll("details"))
            .find((element) => element.dataset.entityContextSection === activeSection)?.querySelector("summary")
          : null;
      restored?.focus({ preventScroll: true });
    }
    refreshRelationships();
    if (isCanvas) state.entityContextRefresh = refreshRelationships;
    if (isCanvas && ref.kind === "work_item") {
      const contextHost = el("div");
      detail.append(contextHost);
      renderCanvasContext(contextHost, {
        entity,
        context,
        attention,
        skipPlacement: Boolean(relationshipContext),
        onReference: (target) => showEntity(target).catch(report),
        onHistory: (target) =>
          loadCanvasHistory(contextHost, target, detailGeneration),
      });
      const model = state.canvasData?.context(context, { attention });
      if (model) acceptCanvas(model);
    }
    const props = el("dl", undefined, "detail-properties");
    for (const key of [
      "created_at",
      "updated_at",
      "captured_at",
      "required_for_outcome",
      "not_before",
      "severity",
      "blocked_ref",
      "cause_ref",
      "external_cause",
      "propagation",
      "resolution_summary",
      "parent_objective_id",
      "verification_mode",
      "uri",
      "source_ref",
      "objective_id",
      "producer_ref",
      "assignee_refs",
      "target_ref",
      "relation_type",
      "strength",
      "evidence_id",
      "stance",
      "criterion_id",
    ]) {
      if (
        relationshipContext &&
        ["parent_objective_id", "objective_id", "blocked_ref", "cause_ref",
          "evidence_id", "target_ref"].includes(key)
      ) continue;
      if (
        entity[key] === undefined ||
        entity[key] === null ||
        entity[key] === ""
      )
        continue;
      const dd = el("dd");
      if (["objective_id", "parent_objective_id"].includes(key))
        dd.append(
          referenceButton({ ...ref, kind: "objective", id: entity[key] }),
        );
      else if (key === "uri") {
        try {
          const u = new URL(entity[key]);
          if (["https:", "http:"].includes(u.protocol)) {
            const a = el("a", entity[key]);
            a.href = u.href;
            a.target = "_blank";
            a.rel = copy.text.noopenerNoreferrer;
            dd.append(a);
          } else dd.append(el("span", entity[key]));
        } catch {
          dd.append(el("span", entity[key]));
        }
      } else dd.append(readable(entity[key], key));
      props.append(el("dt", human(key)), dd);
    }
    detail.append(props);
    if (entity.current_lease) {
      const lease = el("section", undefined, "detail-section");
      lease.append(
        el("h3", copy.text.executionReservation),
        el(
          "p",
          message(
            "expires",
            entity.current_lease.actor_ref?.id ||
              entity.current_lease.principal_id,
            date(entity.current_lease.expires_at),
          ),
        ),
      );
      detail.append(lease);
    }
    if (ref.kind === "work_item" && entity.contracts_enabled) {
      await contractSection(detail, entity, detailGeneration);
      if (detailGeneration !== state.detailGeneration) return;
    }
    const criteria = entity.criteria?.items || [];
    if (criteria.length) {
      const group = el("section", undefined, "detail-section");
      group.append(el("h3", copy.text.successCriteria));
      for (const criterion of criteria) {
        const button = el("button", undefined, copy.text.itemCriterionItem);
        button.append(
          el("strong", criterion.title),
          el(
            "small",
            message(
              "revision51f7",
              criterion.required ? copy.text.required : copy.text.optional,
              display(criterion.verification_mode),
              criterion.criterion_revision,
            ),
          ),
        );
        const assessment = entity.criteria.current_assessments?.[criterion.id];
        if (assessment) button.append(badge(assessment.result));
        button.append(el("small", copy.text.reviewAssessment));
        button.onclick = () => {
          if (!allowCanvasNavigation()) return;
          state.criterion = criterion;
          openCommands("record_criterion_assessment");
        };
        group.append(button);
      }
      detail.append(group);
    }
    if (entity.conclusion) {
      const group = el("section", undefined, "detail-section");
      group.append(
        el("h3", copy.text.currentConclusion),
        conclusionView(entity.conclusion),
      );
      detail.append(group);
    }
    if (entity.conclusion_history?.length) {
      const history = el("details");
      history.append(el("summary", copy.text.previousConclusions));
      for (const c of entity.conclusion_history)
        history.append(conclusionView(c));
      detail.append(history);
    }
    if (ref.kind === "roadmap") {
      if (entity.draft) {
        const draft = el("details");
        draft.open = true;
        draft.append(
          el(
            "summary",
            message("draftVersion3d05", entity.draft.draft_version),
          ),
          planNodes(entity.draft.nodes),
        );
        if (isCanvas && entity.draft.lifecycle === "open")
          draft.append(
            actionButton(
              "View draft on Outcome Board",
              () =>
                viewCanvasPlan({
                  roadmap_id: entity.id,
                  draft: entity.draft,
                  outcome_revision: result.outcome_revision,
                }),
              "quiet",
            ),
          );
        detail.append(draft);
      }
      for (const revision of entity.revisions || []) {
        const history = el("details");
        history.open = revision === entity.revisions.at(-1);
        history.append(
          el(
            "summary",
            message(
              "revision0c81",
              revision.revision_number,
              revision.reason || copy.text.publishedRoadmap,
            ),
          ),
          planNodes(revision.nodes),
        );
        if (isCanvas)
          history.append(
            actionButton(
              `View revision ${revision.revision_number} on Outcome Board`,
              () =>
                viewCanvasPlan({
                  roadmap_id: entity.id,
                  revision_number: revision.revision_number,
                }),
              "quiet",
            ),
          );
        detail.append(history);
      }
    }
    const tech = el("details", undefined, "technical");
    tech.append(
      el("summary", copy.text.technicalDetailsAndAudit),
      el("pre", JSON.stringify(entity, null, 2)),
    );
    detail.append(tech);
    $("detail-actions").querySelector(".quick-actions")?.remove();
    $("detail-actions").prepend(quickActions(ref.kind, entity));
    if (isCanvas) {
      if (
        !preserveCamera ||
        !canvasInspector.isVisible("detail") ||
        (focusWasInDetails && document.activeElement === document.body)
      )
        canvasInspector.showDetails();
      if (
        [
          "outcome",
          "objective",
          "work_item",
          "issue",
          "blocker",
          "evidence",
          "artifact",
          "decision",
        ].includes(ref.kind)
      ) {
        const expansionFocus = document.activeElement;
        const model = await state.canvasData?.expand(ref);
        if (
          detailGeneration !== state.detailGeneration ||
          generation !== state.generation
        )
          return;
        if (model) acceptCanvas(model);
        // A bounded read may finish after someone has started using the
        // inspector or search. Its result must not take their focus back.
        if (document.activeElement !== expansionFocus && document.activeElement !== document.body) {
          state.canvasView?.update(canvasPresentation(), {
            selectedKey: `${ref.kind}:${ref.id}`,
          });
          writeRoute(ref);
          return state.selected;
        }
      }
      state.canvasView?.update(canvasPresentation(), {
        selectedKey: `${ref.kind}:${ref.id}`,
      });
      if (!preserveCamera) state.canvasView?.focus(`${ref.kind}:${ref.id}`);
      if (!preserveCamera || !canvasInspector.isVisible("detail"))
        canvasInspector.showDetails();
      if (!preserveCamera) revealCanvasInspector();
    } else if (!$("detail-dialog").open) $("detail-dialog").showModal();
    writeRoute(ref);
    return state.selected;
  } catch (error) {
    if (
      generation !== state.generation ||
      detailGeneration !== state.detailGeneration
    ) {
      error.discarded = true;
      error.name = "AbortError";
    }
    throw error;
  }
}
$("close-detail").onclick = () => closeDetail();
$("detail-dialog").addEventListener("close", () => {
  if (!canvasActive()) writeRoute(null);
});
$("entity-actions").onclick = () => openCommands();
$("actions").onclick = () => {
  if (!allowCanvasNavigation()) return;
  openCommands(undefined, { kind: "outcome", entity: state.outcome });
};
$("create-outcome").onclick = () => {
  if (!allowCanvasNavigation()) return;
  state.selected = null;
  openCommands("create_outcome");
};
function defaults() {
  const e = state.selected || {},
    kind = e._kind || "outcome";
  const scope = {
    namespace_id: state.namespace,
    outcome_id: state.outcome?.id,
  };
  const ref = { ...scope, kind, id: e.id || state.outcome?.id };
  return {
    title: e.title,
    description: e.description,
    desired_state: e.desired_state,
    nodes: e.draft?.nodes,
    after_links: e.draft?.after_links,
    ...scope,
    scope,
    owner: ref,
    target_ref: ref,
    source_ref: ref,
    expected_version: e.version || state.outcome?.version,
    expected_roadmap_version: e.version,
    expected_draft_version: e.draft?.draft_version,
    plan_scope: { kind: "outcome", id: state.outcome?.id },
    scope_kind: "outcome",
    scope_id: state.outcome?.id,
    priority: "normal",
    lifecycle: "todo",
    ttl_seconds: 900,
    producer_ref: { kind: "human", provider: "wos-web", id: "browser" },
    ...Object.fromEntries(
      [
        ["work_item", "work_item_id"],
        ["objective", "objective_id"],
        ["roadmap", "roadmap_id"],
        ["evidence", "evidence_id"],
        ["artifact", "artifact_id"],
        ["decision", "decision_id"],
        ["issue", "issue_id"],
        ["blocker", "blocker_id"],
      ].map(([k, f]) => [f, kind === k ? e.id : undefined]),
    ),
    contract_id: state.contractView?.contract.id,
    expected_contract_version: state.contractView?.contract.version,
    expected_work_item_version: e.version,
    expected_lease_version: state.contractView?.contract.lease_version,
    authority: state.contractView
      ? {
          execution_id: state.contractView.contract.execution_id,
          fencing_token: state.contractView.contract.fencing_token,
          spec_digest: state.contractView.contract.spec_digest,
        }
      : undefined,
    submission_id:
      state.submission?.id || state.contractView?.contract.latest_submission_id,
    claim_id: e.current_lease?.claim_id,
    fencing_token: e.current_lease?.fencing_token,
    criterion_id: state.criterion?.id,
    criterion_revision: state.criterion?.criterion_revision,
    verification_mode: "attestation",
    result: "met",
  };
}
function openTechnicalCommands(name) {
  if (state.pending) {
    openCommands();
    return;
  }
  if (!state.permissions.includes("namespace:admin")) {
    notice(
      copy.text.developerToolsRequireWorkspaceAdministrationPermission,
      true,
    );
    return;
  }
  if (!allowCanvasNavigation()) return;
  if (canvasActive()) navigateArea("overview");
  state.humanForm?.dispose();
  state.humanForm = null;
  state.formOriginCanvas = false;
  state.formEpoch++;
  state.formTarget = structuredClone(state.selected);
  state.technical = true;
  state.formScope = currentScope();
  state.formGeneration = state.generation;
  $("command-select").closest("label").hidden = false;
  $("conflict-review").replaceChildren();
  if (name && !commandAvailableInUI(name)) {
    notice(
      copy.text
        .thisFlowRequiresAnAuthorizedProfileAndProtectedSignatureUseW6de2babd,
      true,
    );
    return;
  }
  $("submit-command").disabled = false;
  $("detail-dialog").close();
  state.pending = null;
  $("retry-command").hidden = true;
  $("command-error").textContent = "";
  $("command-select").replaceChildren();
  const relevant = el("optgroup");
  relevant.label = copy.text.actionsForThisItem;
  const other = el("optgroup");
  other.label = copy.text.otherActions;
  const kind = state.selected?._kind || "outcome";
  for (const d of state.catalog) {
    if (!commandAvailableInUI(d.name)) continue;
    if (
      state.selected?.contracts_enabled &&
      [
        "claim_work_item",
        "release_work_item",
        "renew_work_item_lease",
        "reclaim_work_item",
        "complete_work_item",
        "administrative_complete_work_item",
      ].includes(d.name)
    )
      continue;
    const option = el("option", actionName(d.name));
    option.value = d.name;
    (d.name.endsWith(`_${kind}`) ||
    (state.criterion && d.name.includes("criterion"))
      ? relevant
      : other
    ).append(option);
  }
  $("command-select").append(relevant, other);
  if (name) $("command-select").value = name;
  else {
    const kind = state.selected?._kind || "outcome";
    const preferred = `update_${kind}`;
    if (state.catalog.some((x) => x.name === preferred))
      $("command-select").value = preferred;
  }
  buildForm();
  $("command-dialog").showModal();
}
function field(name, schema, value, required = false) {
  const wrap = el("div", undefined, "technical-field");
  wrap.dataset.fieldKind = schema.type || "";
  let getter;
  if (
    value !== undefined &&
    value !== null &&
    ([
      "namespace_id",
      "outcome_id",
      "expected_version",
      "expected_roadmap_version",
      "expected_draft_version",
      "claim_id",
      "fencing_token",
    ].includes(name) ||
      (["criterion_id", "criterion_revision"].includes(name) &&
        state.criterion))
  ) {
    return { node: wrap, get: () => value };
  }

  const entityKind = {
    objective_id: "objective",
    parent_objective_id: "objective",
    work_item_id: "work_item",
    evidence_id: "evidence",
    artifact_id: "artifact",
    issue_id: "issue",
    blocker_id: "blocker",
    decision_id: "decision",
    roadmap_id: "roadmap",
  }[name];
  if (
    entityKind &&
    value === state.selected?.id &&
    entityKind === state.selected?._kind &&
    name !== "parent_objective_id"
  )
    return { node: wrap, get: () => value };
  if (name === "permissions") {
    const node = el("fieldset", undefined, "permission-options");
    node.append(el("legend", copy.text.permissions));
    const controls = Object.keys(permissionLabels).map((permission) => {
      const label = el("label", undefined, "check-label"),
        input = el("input");
      input.type = "checkbox";
      input.checked = (value || []).includes(permission);
      label.append(
        input,
        document.createTextNode(permissionLabels[permission]),
      );
      node.append(label);
      return { permission, input };
    });
    return {
      node,
      get: () =>
        controls.filter((x) => x.input.checked).map((x) => x.permission),
    };
  }
  if (entityKind) {
    const label = el("label", human(name)),
      select = el("select");
    select.append(
      Object.assign(el("option", copy.text.chooseAnItem), { value: "" }),
    );
    const items = new Map(state.entities);
    for (const section of Object.values(state.snapshot?.sections || {})) {
      for (const entry of section) {
        const item = entry?.ref ? entry : entry?.work_item;
        if (item?.ref) items.set(`${item.ref.kind}/${item.ref.id}`, item);
      }
    }
    for (const item of items.values()) {
      if (item.ref.kind !== entityKind) continue;
      const option = el("option", item.title || human(entityKind));
      option.value = item.ref.id;
      select.append(option);
    }
    if (value && !Array.from(select.options).some((o) => o.value === value)) {
      const option = el("option", state.selected?.title || human(entityKind));
      option.value = value;
      select.append(option);
    }
    select.value = value || "";
    select.required = required;
    label.append(select);
    wrap.append(label);
    return { node: wrap, get: () => select.value || undefined };
  }

  if (
    schema.properties?.namespace_id &&
    schema.properties?.outcome_id &&
    schema.properties?.kind &&
    schema.properties?.id
  ) {
    const label = el("label", human(name));
    const select = el("select");
    const blank = el("option", copy.text.chooseAnItem);
    blank.value = "";
    select.append(blank);
    const items = new Map(state.entities);
    if (state.snapshot) {
      const root = state.snapshot.outcome;
      items.set(`${root.ref.kind}/${root.ref.id}`, root);
      for (const values of Object.values(state.snapshot.sections)) {
        for (const item of values) {
          const entity = item?.ref ? item : item?.work_item;
          if (entity?.ref)
            items.set(`${entity.ref.kind}/${entity.ref.id}`, entity);
        }
      }
    }
    for (const [key, item] of items) {
      const option = el(
        "option",
        `${item.title || kindName(item.ref.kind)} · ${kindName(item.ref.kind)}`,
      );
      option.value = key;
      select.append(option);
    }
    if (value?.id) select.value = `${value.kind}/${value.id}`;
    select.required = required;
    label.append(select);
    wrap.append(label);
    getter = () => (select.value ? items.get(select.value).ref : undefined);
    return { node: wrap, get: getter };
  }
  if (name === "scope" && value?.namespace_id && value?.outcome_id) {
    return { node: wrap, get: () => value };
  }

  if (
    schema.type === "object" &&
    schema.properties &&
    !required &&
    value === undefined
  ) {
    const label = el("label");
    const enabled = el("input");
    enabled.type = "checkbox";
    label.append(enabled, document.createTextNode(` Include ${human(name)}`));
    const child = field(name, schema, undefined, true);
    const group = el("fieldset");
    group.append(child.node);
    group.hidden = true;
    group.disabled = true;
    enabled.onchange = () => {
      group.hidden = !enabled.checked;
      group.disabled = !enabled.checked;
    };
    wrap.append(label, group);
    return {
      node: wrap,
      get: () => (enabled.checked ? child.get() : undefined),
    };
  }

  if (!schema.type) {
    const label = el("label", `${human(name)} (JSON)`);
    const input = el("textarea");
    input.value = value === undefined ? "" : JSON.stringify(value);
    input.placeholder = copy.text.exampleValueTrueItemOrKeyValue;
    label.append(input);
    wrap.append(label);
    getter = () => (input.value.trim() ? JSON.parse(input.value) : undefined);
  } else if (schema.type === "object" && !schema.properties) {
    const label = el("label", message("contextPairsInJSON", human(name)));
    const input = el("textarea");
    input.value = value === undefined ? "{}" : JSON.stringify(value, null, 2);
    input.placeholder = copy.text.productExampleUserIdPerson42;
    label.append(input);
    wrap.append(label);
    getter = () => (input.value.trim() ? JSON.parse(input.value) : undefined);
  } else if (schema.type === "object") {
    const group = el("fieldset");
    group.append(el("legend", human(name)));
    const extra = el("details", undefined, "command-extra");
    extra.append(el("summary", copy.text.additionalInformationAndProvenance));
    const secondary = el("div");
    extra.append(secondary);
    const fields = {};
    for (const [key, child] of Object.entries(schema.properties || {})) {
      if (
        [
          "reference_snapshot",
          "published_reference_snapshot",
          "criterion_snapshots",
        ].includes(key)
      )
        continue;
      const rendered = field(
        key,
        child,
        value?.[key],
        schema.required?.includes(key),
      );
      const optionalExtra =
        [
          "metadata",
          "external_context",
          "execution_context",
          "source_version",
          "checksum",
          "producer_ref",
          "assignee_refs",
          "actor_ref",
          "not_before",
          "criterion_refs",
        ].includes(key) && !schema.required?.includes(key);
      (optionalExtra ? secondary : group).append(rendered.node);
      fields[key] = rendered.get;
    }
    if (secondary.children.length) group.append(extra);
    wrap.append(group);
    getter = () => {
      const result = {};
      for (const [key, get] of Object.entries(fields)) {
        const v = get();
        if (v !== undefined) result[key] = v;
      }
      return Object.keys(result).length ? result : undefined;
    };
  } else if (schema.type === "array") {
    const group = el("fieldset");
    group.append(el("legend", human(name)));
    const rows = el("div");
    const getters = [];
    const add = el("button", copy.text.addItem, "quiet");
    add.type = "button";
    function append(v) {
      const row = el("div");
      const child = field(
        name === "evidence_ids" ? "evidence_id" : "item",
        schema.items || {},
        v,
        true,
      );
      const remove = el("button", copy.text.remove, "quiet");
      remove.type = "button";
      const item = { get: child.get, active: true };
      getters.push(item);
      remove.onclick = () => {
        item.active = false;
        row.remove();
      };
      row.append(child.node, remove);
      rows.append(row);
    }
    for (const item of value || []) append(item);
    add.onclick = () => append();
    group.append(rows, add);
    wrap.append(group);
    getter = () =>
      getters
        .filter((x) => x.active)
        .map((x) => x.get())
        .filter((x) => x !== undefined);
  } else {
    const label = el("label", human(name));
    let input;
    const options = schema.enum || choices[name];
    if (options) {
      input = el("select");
      if (!required) {
        const blank = el("option", copy.text.notProvided);
        blank.value = "";
        input.append(blank);
      }
      for (const choice of options) {
        const option = el("option", display(choice));
        option.value = choice;
        input.append(option);
      }
    } else {
      input = el(
        [
          "description",
          "reason",
          "rationale",
          "result_summary",
          "desired_state",
          "proposal",
        ].includes(name)
          ? "textarea"
          : "input",
      );
      if (input.tagName === "INPUT")
        input.type =
          schema.type === "boolean"
            ? "checkbox"
            : schema.type === "integer" || schema.type === "number"
              ? "number"
              : schema.format === "date-time"
                ? "datetime-local"
                : "text";
      if (schema.pattern) input.pattern = schema.pattern;
      if (schema.type === "integer") {
        input.step = "1";
        input.min = "0";
      }
    }
    if (
      name.startsWith("expected_") ||
      ["namespace_id", "outcome_id"].includes(name)
    )
      input.readOnly = true;
    if (name === "position" && value === undefined) value = 0;
    if (name === "node_type" && value === undefined) value = "reference";
    if (input.type === "checkbox") input.checked = Boolean(value);
    else if (value !== undefined && value !== null)
      input.value =
        schema.format === "date-time"
          ? new Date(
              new Date(value).getTime() -
                new Date(value).getTimezoneOffset() * 60000,
            )
              .toISOString()
              .slice(0, 16)
          : String(value);
    if (required && input.type !== "checkbox") input.required = true;
    label.append(input);
    wrap.append(label);
    getter = () => {
      if (input.type === "checkbox") return input.checked;
      if (input.value === "") return undefined;
      if (schema.type === "integer" || schema.type === "number")
        return Number(input.value);
      if (schema.format === "date-time")
        return new Date(input.value).toISOString();
      return input.value;
    };
  }
  return { node: wrap, get: getter };
}
let formGetter;
function buildForm() {
  $("command-form").dataset.mode = "technical";
  const descriptor = state.catalog.find(
    (x) => x.name === $("command-select").value,
  );
  if (!descriptor) return;
  $("command-title").textContent = actionName(descriptor.name);
  $("command-help").textContent =
    descriptor.name === "record_criterion_assessment"
      ? copy.text
          .reviewEvidenceAndRecordAnAssessmentEvidenceAloneDoesNotVerifyTheCriterion
      : descriptor.name === "claim_work_item"
        ? copy.text
            .reserveWorkBeforeExecutingATimeLimitedReservationPreventsCon7f9d6428
        : copy.text.theChangeIsSharedWithAllParticipantsReviewBeforeConfirming;
  const context = el(
    "div",
    state.criterion
      ? message(
          "successCriterionRevision",
          state.criterion.title,
          state.criterion.criterion_revision,
        )
      : state.selected
        ? `${kindName(state.selected._kind || "outcome")}: ${state.selected.title || state.selected.name || state.selected.description || state.outcome?.title || ""}${state.selected.version ? message("versiona6e5", state.selected.version) : ""}`
        : copy.text.newOutcomeInThisWorkspace,
    "field-context",
  );
  const form = field(
    copy.text.information,
    descriptor.schema,
    defaults(),
    true,
  );
  $("fields").replaceChildren(context, form.node);
  formGetter = form.get;
  $("command-error").textContent = "";
  state.pending = null;
  $("retry-command").hidden = true;
}

$("command-select").onchange = buildForm;
$("close-command").onclick = () => {
  if (state.submitting || (state.pending && state.formOriginCanvas)) {
    notice(copy.text.resolveThePreviousUncertainResponseWithRetryTheSame, true);
    return;
  }
  closeForm();
};
$("command-dialog").addEventListener("close", () => {
  state.formEpoch++;
  state.humanForm?.dispose();
});
$("command-dialog").addEventListener("cancel", (event) => {
  if (state.submitting || (state.pending && state.formOriginCanvas))
    event.preventDefault();
});
function projectCommand(name, value) {
  // The technical catalog validates transport compatibility; it does not generate human fields.
  const allowed =
    state.catalog.find((x) => x.name === name)?.schema.properties || {};
  return Object.fromEntries(
    Object.entries(value).filter(([key]) => key in allowed),
  );
}
function openCommands(name, menuTarget) {
  if (state.submitting) {
    notice(copy.text.waitForTheCurrentChangeToFinishBeforeStarting, true);
    return;
  }
  if (!state.principal) {
    notice(copy.text.waitForTheAuthenticatedWorkspaceIdentityToLoad, true);
    return;
  }
  if (state.pending) {
    notice(copy.text.resolveThePreviousUncertainResponseWithRetryTheSame, true);
    presentForm();
    return;
  }
  if (canvasInspector.isVisible("form")) {
    allowCanvasNavigation();
    return;
  }
  if (!name) {
    const kind = menuTarget?.kind || (state.criterion ? "criterion" : state.selected?._kind || "outcome");
    const entity = menuTarget?.entity || state.selected || state.outcome;
    renderActionMenu($("action-options"), {
      actions: effectiveActions(kind, entity),
      kind,
      entity: menuTarget?.entity || state.criterion || entity,
      onAction: (action) => {
        closeActions();
        if (menuTarget) {
          state.selected = { ...entity, _kind: kind };
          state.criterion = null;
        }
        openCommands(action.name);
      },
    });
    if (canvasInspector.enabled) canvasInspector.showActions();
    else $("action-dialog").showModal();
    return;
  }
  const context =
    name === "create_outcome"
      ? "workspace"
      : state.criterion
        ? "criterion"
        : state.selected?._kind || "outcome";
  if (!effectiveActions(context).some((a) => a.name === name)) {
    notice(copy.text.thisActionIsUnavailableInTheCurrentStatePermission, true);
    return;
  }
  if (!commandAvailableInUI(name)) return;
  state.detailGeneration++;
  state.formEpoch++;
  state.formTarget = structuredClone(state.selected);
  state.formOriginCanvas = canvasActive();
  state.humanForm?.dispose();
  state.technical = false;
  $("command-form").dataset.mode = "human";
  state.formScope = currentScope();
  state.formGeneration = state.generation;
  $("command-select").replaceChildren(
    Object.assign(el("option", humanActions[name].label), { value: name }),
  );
  $("command-select").closest("label").hidden = true;
  $("command-title").textContent = humanActions[name].label;
  $("command-help").textContent =
    copy.text.reviewYourIntentBeforeSavingTheServerChecksPermissions;
  $("submit-command").textContent =
    name === "create_outcome"
      ? copy.text.saveDraft
      : name === "replace_roadmap_draft"
        ? copy.text.saveDraft
        : name === "publish_roadmap_draft"
          ? copy.text.publishRevision
          : copy.text.saveChanges;
  state.humanForm = humanForm(name, {
    scope: state.formScope,
    entity: state.formTarget || {},
    criterion: state.criterion,
    request: (path, options) => request(`${api}${path}`, options),
    permissions: state.permissions,
    protocol: state.protocol,
    submission: state.submission,
    contractView: state.contractView,
  });
  formGetter = state.humanForm.get;
  $("fields").replaceChildren(
    ...(state.selected?.title
      ? [
          el(
            "p",
            `${kindName(state.selected._kind || "outcome")}: ${state.selected.title}`,
            "field-context",
          ),
        ]
      : []),
    state.humanForm.node,
  );
  $("command-error").textContent = "";
  $("conflict-review").replaceChildren();
  $("retry-command").hidden = true;
  $("submit-command").disabled = false;
  if (!canvasInspector.enabled) $("detail-dialog").close();
  presentForm();
}
$("close-actions").onclick = closeActions;
$("developer-tools").onclick = () => {
  if (state.pending) {
    openCommands();
    return;
  }
  openTechnicalCommands();
};
async function submit(retry = false, reconciled) {
  if (state.submitting) return;
  state.submitting = true;
  $("submit-command").disabled = true;
  $("retry-command").disabled = true;
  let committed = false,
    dispatched = false;
  const dispatchGeneration = state.generation;
  try {
    if (!retry) {
      const scope = state.formScope,
        generation = state.formGeneration;
      const command = reconciled || (await formGetter());
      if (generation !== state.generation || !sameScope(scope, currentScope()))
        throw Error(
          copy.text.workspaceContextChangedReopenTheFormInTheIntended,
        );
      state.pending = createIntent(
        $("command-select").value,
        projectCommand($("command-select").value, command),
        scope,
        undefined,
        { principal_id: state.principal, actor_key: state.actorKey },
      );
    }
    const pending = state.pending;
    if (!pending) return;
    if (retry && sameScope(pending.scope, currentScope()))
      state.formGeneration = state.generation;
    if (!sameScope(pending.scope, currentScope()))
      throw Error(
        copy.text.returnToTheOriginalWorkspaceAndOutcomeBeforeRetrying,
      );
    if (
      pending.identity.principal_id !== state.principal ||
      pending.identity.actor_key !== state.actorKey
    )
      throw Error(
        copy.text.thisIntentBelongsToTheOriginalAuthenticatedIdentityReconnect,
      );
    await refreshNamespaceProtocol();
    await refreshPermissions();
    if (
      !sameScope(pending.scope, currentScope()) ||
      dispatchGeneration !== state.generation
    )
      throw Error(copy.text.workspaceContextChangedBeforeDispatch);
    if (
      !commandAvailableInUI(pending.name) ||
      !state.permissions.includes(inventory[pending.name]?.permission)
    )
      throw Error(copy.text.theProtocolOrPermissionChangedThisIntentCannotBe);
    dispatched = true;
    const result = await request(`${api}/commands/${pending.name}`, {
      method: "POST",
      headers: { "Idempotency-Key": pending.key },
      body: pending.body,
    });
    committed = true;
    state.pending = null;
    if (
      !sameScope(pending.scope, currentScope()) ||
      state.formGeneration !== state.generation
    )
      return;
    const canvasOrigin = state.formOriginCanvas;
    if (canvasOrigin) {
      canvasInspector.closeAll();
    } else $("command-dialog").close();
    state.humanForm?.dispose();
    $("fields").inert = false;
    if (result.value?.desired_state !== undefined) {
      state.outcome = result.value;
      state.selected = { ...result.value, _kind: "outcome" };
      if (pending.name === "create_outcome") {
        state.generation++;
        resetCanvas();
        state.view = "canvas";
        state.area = "canvas";
      }
    }
    const newSection =
      pending.name === "report_issue_with_blocker"
        ? "issues"
        : Object.entries(createForSection).find(
            ([, command]) => command === pending.name,
          )?.[0];
    if (newSection && !canvasOrigin) {
      state.section = newSection;
      state.area = areaFor(newSection);
      state.view = "list";
      $("item-search").value = "";
      $("item-priority").value = "";
    }
    await discover();
    if (state.outcome) await refresh();
    writeRoute();
    const createdKind = {
      create_objective: "objective",
      create_work_item: "work_item",
      create_issue: "issue",
      register_evidence: "evidence",
      register_artifact: "artifact",
      create_roadmap: "roadmap",
      create_decision: "decision",
    }[pending.name];
    const selectedKind = createdKind || state.formTarget?._kind;
    const selectedID = createdKind ? result.value?.id : state.formTarget?.id;
    if (canvasOrigin && selectedKind && selectedID) {
      await showEntity(
        { ...currentScope(), kind: selectedKind, id: selectedID },
        { internal: true },
      );
    } else if (!state.technical && pending.name === "create_work_item")
      await showEntity(
        {
          ...pending.scope,
          kind: "work_item",
          id: result.value.id,
        },
        { internal: true },
      );
    notice(
      pending.name === "create_outcome"
        ? copy.text.draftSavedContinueSetupWithCriteriaObjectivesAndTasks
        : result.result_omitted
          ? copy.text.changeRecordedCheckTheRefreshedState
          : copy.text.changeSaved,
    );
  } catch (e) {
    $("command-error").className = "danger";
    $("command-error").textContent = committed
      ? copy.text.savedButTheRefreshedViewCouldNotBeLoaded
      : e.code === "version_conflict"
        ? copy.text.aNewerVersionExistsReviewChangesBeforeSaving
        : e.message;
    if (canvasInspector.isVisible("form")) {
      $("command-error").tabIndex = -1;
      $("command-error").focus({ preventScroll: true });
    }
    if (committed) {
      report(e);
      return;
    }
    if (
      state.pending &&
      (retry || (dispatched && (e.status === undefined || e.status >= 500)))
    ) {
      $("retry-command").hidden = false;
      $("fields").inert = true;
      $("command-error").textContent +=
        copy.text.theResponseMayHaveBeenLostAfterCommitRetryTheSameIntentToRecoverTheResult;
    } else {
      const pending = state.pending;
      state.pending = null;
      $("fields").inert = false;
      if (
        pending &&
        (e.code === "version_conflict" || e.code === "precondition_failed")
      )
        offerConflict(pending);
    }
  } finally {
    state.submitting = false;
    $("submit-command").disabled = !$("retry-command").hidden;
    $("retry-command").disabled = false;
  }
}
const rebaseEdits = new Set([
  "update_outcome",
  "update_objective",
  "update_work_item",
  "update_issue",
  "update_blocker_description",
  "replace_roadmap_draft",
]);
function offerConflict(pending) {
  const epoch = state.formEpoch,
    target = structuredClone(state.formTarget),
    generation = state.generation;
  const current = () =>
    epoch === state.formEpoch &&
    generation === state.generation &&
    sameScope(pending.scope, currentScope()) &&
    $("command-select").value === pending.name;
  const root = $("conflict-review");
  const discard = () => {
    closeForm();
    refresh().catch(report);
  };
  root.replaceChildren(
    el("h3", copy.text.thisItemChangedWhileYouWereEditing),
    el("p", copy.text.yourLocalDraftIsRetainedLoadingTheLatestState),
  );
  root.append(actionButton(copy.text.discardLocalEdits, discard, "quiet"));
  if (!rebaseEdits.has(pending.name)) {
    root.append(
      el("p", copy.text.thisTransitionOrAssessmentNeedsAFreshStateDefinition),
    );
    return;
  }
  root.append(
    actionButton(
      copy.text.compareWithLatestState,
      async () => {
        try {
          if (!current()) return;
          const entity = target,
            kind = entity?._kind || "outcome",
            path =
              kind === "outcome"
                ? outcomeBase()
                : `${outcomeBase()}/${plurals[kind]}/${entity.id}`;
          const result = await request(path);
          if (!current()) return;
          const latest = result.value || result,
            local = JSON.parse(pending.body).command;
          root.replaceChildren(
            el("h3", copy.text.reviewLatestStateAndYourLocalDraft),
          );
          const columns = conflictComparison();
          for (const [label, data] of [
            [copy.text.latestSavedState, latest],
            [copy.text.yourLocalChanges, local],
          ]) {
            const box = el("section");
            box.append(el("h4", label), readable(data));
            columns.append(box);
          }
          root.append(
            columns,
            actionButton(
              copy.text.useLatestVersionAndSaveMyEdits,
              () => {
                if (!current()) return;
                const next = reconcileDraft(local, latest);
                state.formScope = pending.scope;
                state.formGeneration = state.generation;
                submit(false, next).catch(report);
              },
              "quiet",
            ),
          );
        } catch (error) {
          report(error);
        }
      },
      "quiet",
    ),
  );
}
$("command-form").onsubmit = (event) => {
  event.preventDefault();
  submit().catch(report);
};
$("retry-command").onclick = () => submit(true).catch(report);
namespaces().catch(() => {});

let administration, adminGetter, adminPending;
$("administration").onclick = async () => {
  const generation = state.generation,
    namespace = state.namespace;
  try {
    const result = await request(
      `${api}/namespaces/${state.namespace}/administration`,
    );
    if (generation !== state.generation || namespace !== state.namespace)
      return;
    administration = result;
    $("admin-version").textContent = message(
      "administrationVersion",
      administration.namespace_version,
    );
    $("admin-snapshot").replaceChildren(readable(administration));
    $("admin-error").textContent = "";
    $("admin-secret").hidden = true;
    buildAdmin();
    $("admin-dialog").showModal();
  } catch (e) {
    report(e);
  }
};
function buildAdmin() {
  adminPending = null;
  $("retry-admin").hidden = true;
  $("submit-admin").disabled = false;
  const op = $("admin-operation").value;
  const props = {};
  if (op === "create_namespace") props.namespace_name = { type: "string" };
  if (op === "set_grant") {
    props.principal_id = { type: "string" };
    props.permissions = { type: "array", items: { type: "string" } };
  }
  if (op === "issue_credential") {
    props.principal_id = { type: "string" };
    props.actor_ref = {
      type: "object",
      required: ["kind", "provider", "id"],
      properties: {
        kind: { type: "string" },
        provider: { type: "string" },
        id: { type: "string" },
      },
    };
    props.expires_at = { type: "string", format: "date-time" };
  }
  if (op === "revoke_credential") props.credential_id = { type: "string" };
  const rendered = field(
    copy.text.workspaceSettings,
    { type: "object", properties: props, required: Object.keys(props) },
    {},
  );
  $("admin-fields").replaceChildren(rendered.node);
  adminGetter = rendered.get;
}
$("admin-operation").onchange = buildAdmin;
$("close-admin").onclick = () => {
  $("admin-dialog").close();
  $("admin-token").value = "";
};
async function submitAdmin(retry = false) {
  if (!retry)
    adminPending = {
      key: crypto.randomUUID(),
      intent: {
        namespace_id: state.namespace,
        expected_namespace_version: administration.namespace_version,
        operation: $("admin-operation").value,
        ...adminGetter(),
      },
    };
  if (!adminPending) return;
  $("submit-admin").disabled = true;
  $("retry-admin").hidden = true;
  try {
    const response = await request(`${api}/security/commands`, {
      method: "POST",
      headers: { "Idempotency-Key": adminPending.key },
      body: JSON.stringify(adminPending.intent),
    });
    adminPending = null;
    administration = await request(
      `${api}/namespaces/${state.namespace}/administration`,
    );
    $("admin-version").textContent = message(
      "administrationVersion",
      administration.namespace_version,
    );
    $("admin-snapshot").replaceChildren(readable(administration));
    $("admin-error").textContent = response.result?.token_omitted
      ? copy.text
          .thePreviousIssuanceWasConfirmedTheCredentialCannotBeRecovere660cace4
      : copy.text.changeRecorded;
    if (response.token) {
      $("admin-token").value = response.token;
      $("admin-secret").hidden = false;
    }
  } catch (e) {
    $("admin-error").textContent = e.message;
    if (e.status === undefined || e.status >= 500) {
      $("retry-admin").hidden = false;
      $("admin-error").textContent +=
        copy.text.theChangeMayHaveCommittedRetryTheSameIntentToRetrieveItsReceipt;
    } else
      $("admin-error").textContent +=
        copy.text.closeAndReopenToReviewCurrentStateBeforeCreatingANewIntent;
  } finally {
    $("submit-admin").disabled = !$("retry-admin").hidden;
  }
}
$("admin-form").onsubmit = (event) => {
  event.preventDefault();
  submitAdmin().catch(report);
};
$("retry-admin").onclick = () => submitAdmin(true).catch(report);

$("view-outcome").onclick = () =>
  showEntity(state.snapshot.outcome.ref).catch(report);
