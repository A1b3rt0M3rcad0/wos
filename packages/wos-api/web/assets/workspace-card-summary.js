import { copy, message } from "./en-US.js";
import { display } from "./presentation.js";

export const WORK_MAP_KINDS = new Set(["outcome", "objective", "work_item"]);
export const MAX_CONTEXT_RECORDS = 20;

// Follow only loaded documentary connections. Do not traverse through another
// operational item or manufacture ownership from a contextual relationship.
export function canvasRecordExplorer(model) {
  const nodes = new Map((model.nodes || []).map((item) => [item.key, item]));
  const adjacency = new Map();
  for (const edge of model.edges || []) {
    if (!nodes.has(edge.sourceKey) || !nodes.has(edge.targetKey)) continue;
    for (const [from, to] of [[edge.sourceKey, edge.targetKey], [edge.targetKey, edge.sourceKey]]) {
      if (WORK_MAP_KINDS.has(nodes.get(to).kind)) continue;
      if (!adjacency.has(from)) adjacency.set(from, []);
      adjacency.get(from).push(to);
    }
  }
  const cached = new Map();
  return (key) => {
    if (cached.has(key)) return cached.get(key);
    const result = new Set(), queue = [...(adjacency.get(key) || [])];
    for (let index = 0; index < queue.length; index++) {
      const current = queue[index];
      if (result.has(current)) continue;
      result.add(current);
      queue.push(...(adjacency.get(current) || []).filter((entry) => !result.has(entry)));
    }
    cached.set(key, result);
    return result;
  };
}

export function canvasLinkedRecords(model, key) { return canvasRecordExplorer(model)(key); }

// A view projection only: the complete loaded model remains available for
// search, relationship inspection, coverage and the explicit records layer.
export function canvasWorkLayer(model, { layer = "work", recordRoots = new Set(), selectedKey, selectedEdgeKey, recordsFor = canvasRecordExplorer(model) } = {}) {
  if (layer === "records") return { model, hiddenRecordCount: 0 };
  const visible = new Set((model.nodes || []).filter((item) => WORK_MAP_KINDS.has(item.kind)).map((item) => item.key));
  for (const key of recordRoots) {
    visible.add(key);
    for (const record of [...recordsFor(key)].slice(0, MAX_CONTEXT_RECORDS)) visible.add(record);
  }
  if (selectedKey) visible.add(selectedKey);
  const edge = (model.edges || []).find((entry) => entry.key === selectedEdgeKey);
  if (edge) {
    visible.add(edge.sourceKey);
    visible.add(edge.targetKey);
  }
  const nodes = (model.nodes || []).filter((item) => visible.has(item.key));
  const edges = (model.edges || []).filter((entry) => visible.has(entry.sourceKey) && visible.has(entry.targetKey));
  return { model: { ...model, nodes, edges }, hiddenRecordCount: (model.nodes || []).length - nodes.length };
}

function actorName(person) {
  return person?.actor_ref?.id || person?.principal_id || "";
}

// Summarize server facts rather than running readiness or acceptance rules in
// the browser. In particular, assignees never become authorized executors.
export function canvasCardPresentation(item) {
  const summary = item.cardSummary;
  const result = {
    state: item.displayState || item.lifecycle || "",
    lifecycle: item.lifecycle || "",
    person: "",
    summary: item.summary || (WORK_MAP_KINDS.has(item.kind) ? copy.text.boardCriteriaNotLoaded : ""),
    description: summary?.description || "",
    required: "",
    authorityUnknown: Boolean(item.authorityUnknown),
  };
  if (!summary) return result;
  if (item.kind === "objective")
    result.required = summary.required_for_outcome ? copy.text.boardRequiredOutcome : copy.text.boardOptionalOutcome;
  const people = summary.participants || [];
  const authorized = summary.authority_state === "active"
    ? people.filter((person) => person.role === "authorized_executor" && person.authority_state === "active" && actorName(person))
    : [];
  const declared = people.filter((person) => ["declared_assignee", "declared_owner"].includes(person.role) && actorName(person));
  const chosen = authorized.length ? authorized : declared;
  if (chosen.length) {
    const key = authorized.length ? "boardAuthorizedExecutor" : chosen[0].role === "declared_assignee" ? "boardDeclaredAssignee" : "boardDeclaredOwner";
    result.person = message(key, actorName(chosen[0]));
    if (chosen.length > 1) result.person += ` · ${message("boardMorePeople", chosen.length - 1)}`;
  }
  if (summary.criteria) {
    const { active, met, waived } = summary.criteria;
    if ([active, met, waived].every((value) => Number.isSafeInteger(value) && value >= 0)) {
      result.summary = active ? message("boardCriteriaCounts", met, active) : copy.text.boardCriteriaMissing;
      if (waived) result.summary += ` · ${message("boardCriteriaWaived", waived)}`;
    }
  }
  if (summary.attention_reason)
    result.summary = message("boardCurrentAttention", display(summary.attention_reason));
  return result;
}
