import { message } from "./en-US.js";
import { contractViews } from "./contracts-view.js";
import { request as networkRequest } from "./api-client.js";
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
import { button as actionButton } from "./ui.js";
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
  view: "summary",
  area: "overview",
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
};
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
      state.permissions = [];
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
function notice(message, error = false) {
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
async function discover(append = false) {
  const generation = state.generation;
  const q = new URLSearchParams({ limit: "25", text: $("text").value });
  if ($("lifecycle").value) q.set("lifecycle", $("lifecycle").value);
  if (!$("archived").checked) q.set("archived", "false");
  if (append && state.next) q.set("cursor", state.next);
  const page = await request(`${base()}?${q}`);
  if (generation !== state.generation) return;
  if (!append) $("outcomes").replaceChildren();
  for (const item of page.items) {
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
    button.append(el("span", undefined, "nav-dot"), navCopy);
    button.onclick = () => openOutcome(item).catch(report);
    $("outcomes").append(button);
  }
  if (!page.items.length && !append)
    $("outcomes").append(el("p", copy.text.noMatchingOutcomes));
  state.next = page.next_cursor;
  $("more-outcomes").hidden = !state.next;
}
$("more-outcomes").onclick = () => discover(true).catch(report);
async function openOutcome(outcome) {
  state.generation++;
  document.querySelector(".sidebar").classList.add("navigation-collapsed");
  $("navigation-toggle").setAttribute("aria-expanded", "false");
  state.entities.clear();
  state.loaded.clear();
  state.boardLoaded.clear();
  state.outcome = outcome;
  state.selected = outcome;
  state.criterion = null;
  state.view = "summary";
  state.area = "overview";
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
  const generation = state.generation;
  const path = outcomeBase();
  const [snapshot, live, protocol] = await Promise.all([
    request(`${path}/continuity?limit=25`),
    request(path),
    request(`${api}/namespaces/${state.namespace}/work-protocol`),
  ]);
  if (generation !== state.generation) return;
  if (
    live.outcome_revision !== undefined &&
    live.outcome_revision !== snapshot.outcome_revision
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
  if (generation !== state.generation) return;
  renderMetrics();
  renderTabs();
  renderSummary();
  applyView();
  await renderSection();
  notice(
    snapshot.truncated
      ? copy.text.partialViewMoreItemsAreAvailableUseLoadMore
      : copy.text.sharedStateRefreshed,
  );
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
  $("area-tabs").replaceChildren();
  for (const [area, definition] of Object.entries(areas)) {
    const b = actionButton(definition.label, () => navigateArea(area), "quiet");
    b.dataset.area = area;
    b.setAttribute("aria-pressed", String(state.area === area));
    b.classList.toggle("selected", state.area === area);
    $("area-tabs").append(b);
  }
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
function writeRoute(item = null, replace = false) {
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
    if (route.item) await showEntity(route.item);
    else $("detail-dialog").close();
  } finally {
    state.restoringRoute = false;
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
  state.area = area;
  state.section = areas[area].sections[0] || "work_items";
  state.view = area === "overview" ? "summary" : "list";
  $("item-search").value = "";
  $("item-priority").value = "";
  $("work-status").value = state.section;
  renderTabs();
  applyView();
  writeRoute();
  renderSection().catch(report);
}
function applyView() {
  $("summary-view").hidden = state.view !== "summary";
  $("items-view").hidden = state.view === "summary";
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
  const board = state.view === "board";
  $("tabs").hidden = board || state.area === "work";
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
}
function navigateSection(section) {
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
  decisions: "create_decision",
  roadmaps: "create_roadmap",
  ready_work: "create_work_item",
  blocked_work: "create_work_item",
  in_progress_work: "create_work_item",
};
function sectionCommand() {
  return createForSection[state.section] || "";
}
function createItem(command) {
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
  const suggestion = c.blocked_work
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
        if (suggestion.section) navigateSection(suggestion.section);
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
    active = c.in_progress_work || 0,
    blocked = (c.blocked_work || 0) + (c.attention_needed_work || 0),
    total = c.work_items || 0;
  const parts = [
    {
      title: copy.text.toDoPlanned,
      count: Math.max(
        0,
        total - done - active - blocked - (c.cancelled_work || 0),
      ),
      color: "#d6d2c8",
      tone: "",
    },
    {
      title: copy.text.inProgress,
      count: active,
      color: "#e97335",
      tone: "orange",
    },
    {
      title: copy.text.blockedNeedsAttention,
      count: blocked,
      color: "#c75858",
      tone: "red",
    },
    { title: copy.text.done, count: done, color: "#339d71", tone: "green" },
    {
      title: copy.text.cancelled,
      count: c.cancelled_work || 0,
      color: "#8e82b3",
      tone: "violet",
    },
  ];
  const body = el("div", undefined, "summary-stats");
  body.append(donut(parts, total));
  const legend = el("div", undefined, "legend");
  for (const part of parts) {
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
  const items = [
    ...(s.sections.ready_work || []),
    ...(s.sections.in_progress_work || []),
    ...(s.sections.blocked_work || []),
  ].slice(0, 3);
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
  return (
    item.state ||
    item.readiness ||
    operationalSections
      .flatMap((key) => state.snapshot.sections[key] || [])
      .find((x) => x.work_item.ref.id === e.ref?.id)?.state
  );
}
function matches(item) {
  const e = item.work_item || item.current || item;
  const query = $("item-search").value.trim().toLocaleLowerCase("en-US");
  return (
    (!query ||
      `${e.title || ""} ${item.plan_label || ""} ${shortId(e.ref?.id)} ${display(e.lifecycle || "")}`
        .toLocaleLowerCase("en-US")
        .includes(query)) &&
    (!$("item-priority").value || e.priority === $("item-priority").value)
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
  button.onclick = () => showEntity(entity.ref).catch(report);
  const main = el("span", undefined, "item-main");
  const symbol = el("span", undefined, `entity-symbol ${entity.ref.kind}`);
  symbol.append(icon(entity.ref.kind));
  const itemCopy = el("span");
  itemCopy.append(
    el("span", entity.title || kindName(entity.ref.kind), "item-title"),
    el(
      "span",
      `${shortId(entity.ref.id)} · ${kindName(entity.ref.kind)}${item.plan_label ? ` · ${item.plan_label}` : ""}`,
      "item-subtitle",
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
      entity.priority
        ? `${entity.priority === "low" ? "↓" : entity.priority === "normal" ? "—" : "↑"} ${display(entity.priority)}`
        : "—",
      `item-priority ${entity.priority || ""}`,
    );
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
  body.append(
    Object.assign(el("h3", eventName(item.event_type)), {
      title: item.event_type,
    }),
    el(
      "p",
      `${item.actor_ref?.id || item.principal_id} · ${kindName(item.actor_ref?.kind || "human")} · Revision ${item.outcome_revision}`,
    ),
    referenceButton(item.entity_ref),
    el("time", date(item.recorded_at)),
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
  box.append(
    icon(sectionIcons[state.section] || "work_item"),
    el("h3", title),
    el("p", copy.text.activityAppearsHereAsWorkProgresses),
  );
  if (command && state.permissions.includes(inventory[command]?.permission)) {
    const b = el("button", actionName(command), "quiet");
    b.onclick = () => createItem(command);
    box.append(b);
  }
  return box;
}
async function sectionData(section, cursor = "", append = false) {
  const cached = state.loaded.get(section);
  if (!cursor && cached) return cached;
  let items, next;
  if (section === "timeline") {
    const page = await request(
      `${outcomeBase()}/timeline?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    items = page.items;
    next = page.next_cursor;
  } else if (section === "graph") {
    const page = await request(
      `${outcomeBase()}/graph?depth=3&limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    items = page.edges;
    next = page.next_cursor;
  } else if (section === "roadmaps") {
    const page = await request(`${outcomeBase()}/roadmaps`);
    items = (page.items || []).map((x) => ({
      ref: {
        namespace_id: state.namespace,
        outcome_id: state.outcome.id,
        kind: "roadmap",
        id: x.id,
      },
      title: x.title,
      version: x.version,
      lifecycle: x.lifecycle,
    }));
  } else if (!cursor) {
    items = state.snapshot.sections[section] || [];
    next = state.snapshot.section_cursors[section];
  } else {
    const page = await request(
      `${outcomeBase()}/continuity/${section}?limit=25&cursor=${encodeURIComponent(cursor)}`,
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
  if (results.some((r) => r.outcome_revision !== page.outcome_revision)) {
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
  if (!state.snapshot || state.view === "summary") return;
  if (state.view === "board") {
    state.renderId++;
    return renderBoard();
  }
  const section = state.section,
    generation = state.generation,
    run = ++state.renderId;
  const globalWork = section === "work_items";
  let data;
  try {
    data = globalWork
      ? await searchedTasks(cursor, append)
      : await sectionData(section, cursor, append);
  } catch (error) {
    if (generation === state.generation && run === state.renderId) throw error;
    return;
  }
  if (generation !== state.generation || run !== state.renderId) return;
  if (section === "objectives") {
    const path = outcomeBase(),
      revision = state.snapshot.outcome_revision;
    const missing = data.items.filter((x) => !x._hierarchy_loaded);
    const hydrated = await Promise.all(
      missing.map(async (x) => {
        const r = await request(`${path}/objectives/${x.ref.id}`);
        if (r.outcome_revision !== revision)
          throw Error(
            copy.text.stateChangedWhileLoadingRefreshForAConsistentView,
          );
        return { ...r.value, ref: x.ref, _hierarchy_loaded: true };
      }),
    );
    if (generation !== state.generation || run !== state.renderId) return;
    const map = new Map(hydrated.map((x) => [x.ref.id, x]));
    data.items = data.items.map((x) => map.get(x.ref.id) || x);
  }
  state.loaded.set(section, data);
  const content = $("content");
  content.replaceChildren();
  const heading = el("div", undefined, "content-heading");
  const shown = globalWork ? data.items : data.items.filter(matches);
  heading.append(
    el("h2", human(section)),
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
        data.items.length
          ? copy.text.noItemsMatchTheFilters
          : copy.text.noRecordsInThisSectionYet,
        data.items.length ? "" : sectionCommand(),
      ),
    );
  const list = el("div", undefined, "collection");
  let rows = 0;
  for (const item of shown) {
    const entity = item.work_item || item.current || item;
    if (entity.ref) {
      if (!rows) {
        const head = el("div", undefined, "list-head");
        head.append(
          el("span", copy.text.item),
          el("span", copy.text.lifecycle),
          el("span", copy.text.priority),
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
        row.append(
          referenceButton({
            namespace_id: state.namespace,
            outcome_id: state.outcome.id,
            kind: "objective",
            id: entity.parent_objective_id,
          }),
        );
        row.title = parent
          ? message("withinObjective", parent.title)
          : copy.text.parentObjectiveOutsidePage;
      }
      list.append(row);
      rows++;
    } else if (section === "graph") {
      const ref = item.from || item.source_ref,
        other = item.to || item.target_ref;
      const row = el("div", undefined, "relation-card");
      if (ref && other)
        row.append(
          referenceButton(ref),
          el("span", "→", "relation-arrow"),
          referenceButton(other),
          badge(item.kind || item.type || item.relation_type || "depends_on"),
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
          const generation = state.generation;
          more.disabled = true;
          try {
            const page = await request(
              `${outcomeBase()}/continuity/${section}?limit=25&cursor=${encodeURIComponent(cursor)}`,
            );
            if (generation !== state.generation) return;
            const previous =
              state.boardLoaded.get(section)?.items ||
              state.snapshot.sections[section] ||
              [];
            state.boardLoaded.set(section, {
              items: [...previous, ...page.items],
              next: page.next_cursor || "",
            });
            if (state.view === "board") renderBoard();
          } catch (e) {
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
const plurals = {
  objective: "objectives",
  work_item: "work-items",
  issue: "issues",
  blocker: "blockers",
  decision: "decisions",
  evidence: "evidence",
  artifact: "artifacts",
  roadmap: "roadmaps",
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
  for (const action of effectiveActions(kind, entity)) {
    const destination = action.advanced ? advanced : container;
    destination.append(
      actionButton(
        action.label,
        () => {
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
async function showEntity(ref) {
  const detailGeneration = ++state.detailGeneration;
  const generation = state.generation;
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
  let operationalState;
  if (ref.kind === "work_item") {
    const projection = await request(`${path}/operational-state`);
    if (
      generation !== state.generation ||
      detailGeneration !== state.detailGeneration
    )
      return;
    if (result.outcome_revision !== projection.outcome_revision)
      throw Error(copy.text.thisTaskChangedWhileLoadingRefreshToReviewA);
    operationalState = projection.state;
  }
  if (generation !== state.generation) return;
  if (detailGeneration !== state.detailGeneration) return;
  const entity = {
    ...(result.value || result),
    ...(operationalState ? { _operational_state: operationalState } : {}),
  };
  state.contractView = null;
  state.submission = null;
  state.selected = { ...entity, _kind: ref.kind };
  state.criterion = null;
  state.entities.set(`${ref.kind}/${ref.id}`, {
    ref,
    title: entity.title || entity.name || entity.description,
    lifecycle: entity.lifecycle,
  });
  $("detail-kind").textContent = `${kindName(ref.kind)} / ${shortId(ref.id)}`;
  $("detail-title").textContent =
    entity.title || entity.name || entity.description || human(ref.kind);
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
        message(
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
    entity.result_summary;
  if (description) detail.append(el("p", description, "detail-description"));
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
    "resolution_summary",
    "parent_objective_id",
    "verification_mode",
    "uri",
    "source_ref",
    "objective_id",
    "producer_ref",
    "assignee_refs",
  ]) {
    if (entity[key] === undefined || entity[key] === null || entity[key] === "")
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
        el("summary", message("draftVersion3d05", entity.draft.draft_version)),
        planNodes(entity.draft.nodes),
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
  if (!$("detail-dialog").open) $("detail-dialog").showModal();
  writeRoute(ref);
}
$("close-detail").onclick = () => $("detail-dialog").close();
$("detail-dialog").addEventListener("close", () => writeRoute());
$("entity-actions").onclick = () => openCommands();
$("actions").onclick = () => {
  state.selected = { ...state.outcome, _kind: "outcome" };
  state.criterion = null;
  openCommands();
};
$("create-outcome").onclick = () => {
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
  state.humanForm?.dispose();
  state.humanForm = null;
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
  const wrap = el("div");
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
    const node = el("fieldset");
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
$("close-command").onclick = () => $("command-dialog").close();
$("command-dialog").addEventListener("close", () => state.humanForm?.dispose());
function projectCommand(name, value) {
  // The technical catalog validates transport compatibility; it does not generate human fields.
  const allowed =
    state.catalog.find((x) => x.name === name)?.schema.properties || {};
  return Object.fromEntries(
    Object.entries(value).filter(([key]) => key in allowed),
  );
}
function openCommands(name) {
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
    $("command-dialog").showModal();
    return;
  }
  if (!name) {
    const menu = $("action-options");
    menu.replaceChildren();
    const advanced = el("details");
    advanced.append(el("summary", copy.text.advanced));
    for (const action of effectiveActions())
      (action.advanced ? advanced : menu).append(
        actionButton(
          action.label,
          () => {
            $("action-dialog").close();
            openCommands(action.name);
          },
          "quiet",
        ),
      );
    if (advanced.children.length > 1) menu.append(advanced);
    if (!menu.children.length)
      menu.append(
        el("p", copy.text.noChangesAreAvailableForYourCurrentPermissionsAnd),
      );
    $("action-dialog").showModal();
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
  state.humanForm?.dispose();
  state.technical = false;
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
    entity: state.selected || {},
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
  $("detail-dialog").close();
  $("command-dialog").showModal();
}
$("close-actions").onclick = () => $("action-dialog").close();
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
    $("command-dialog").close();
    state.humanForm?.dispose();
    $("fields").inert = false;
    if (result.value?.desired_state !== undefined) {
      state.outcome = result.value;
      state.selected = { ...result.value, _kind: "outcome" };
      if (pending.name === "create_outcome") {
        state.generation++;
        state.view = "summary";
        state.area = "overview";
      }
    }
    const newSection =
      pending.name === "report_issue_with_blocker"
        ? "issues"
        : Object.entries(createForSection).find(
            ([, command]) => command === pending.name,
          )?.[0];
    if (newSection) {
      state.section = newSection;
      state.area = areaFor(newSection);
      state.view = "list";
      $("item-search").value = "";
      $("item-priority").value = "";
    }
    await discover();
    if (state.outcome) await refresh();
    writeRoute();
    if (!state.technical && pending.name === "create_work_item")
      await showEntity({
        ...pending.scope,
        kind: "work_item",
        id: result.value.id,
      });
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
  const root = $("conflict-review");
  const discard = () => {
    $("command-dialog").close();
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
          const entity = state.selected,
            kind = entity?._kind || "outcome",
            path =
              kind === "outcome"
                ? outcomeBase()
                : `${outcomeBase()}/${plurals[kind]}/${entity.id}`;
          const result = await request(path);
          if (!sameScope(pending.scope, currentScope())) return;
          const latest = result.value || result,
            local = JSON.parse(pending.body).command;
          root.replaceChildren(
            el("h3", copy.text.reviewLatestStateAndYourLocalDraft),
          );
          const columns = el("div", undefined, "conflict-columns");
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
