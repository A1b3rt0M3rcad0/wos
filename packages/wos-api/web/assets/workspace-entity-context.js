import { copy, message } from "./en-US.js";
import { el, button } from "./ui.js";
import { kindName, shortId } from "./presentation.js";

const kinds = new Set([
  "outcome", "objective", "work_item", "issue", "blocker", "evidence",
  "artifact", "decision", "roadmap", "relation", "evidence_link",
]);
const roles = {
  outcome: "Outcome", objective: "Objective", work_item: "Task", issue: "Issue",
  blocker: "Blocker", evidence: "Evidence", artifact: "Artifact", decision: "Decision",
  roadmap: "Roadmap", relation: "Relation", evidence_link: "EvidenceLink",
};
const text = (suffix) => copy.text[`entityContext${suffix}`];
const keyOf = (ref) => `${ref.kind}:${ref.id}`;

// A communication index, not a second domain projection. Only the current
// entity's persisted references and typed edges in the loaded Board are used.
// No requests, commands, readiness calculations, totals or acceptance inference.
export function buildEntityContext({ ref, entity = {}, model, placementOnly = false } = {}) {
  if (!ref?.id || !kinds.has(ref.kind) || !ref.namespace_id || !ref.outcome_id)
    return null;
  const inScope = (value) => value && kinds.has(value.kind) &&
    typeof value.id === "string" && value.id &&
    value.namespace_id === ref.namespace_id && value.outcome_id === ref.outcome_id;
  if (entity.id && entity.id !== ref.id) return null;
  if (entity.scope && (entity.scope.namespace_id !== ref.namespace_id ||
    entity.scope.outcome_id !== ref.outcome_id)) return null;
  const nodes = new Map((model?.nodes || [])
    .filter((node) => inScope(node.ref) && node.key === keyOf(node.ref))
    .map((node) => [node.key, node]));
  const matchingModel = !model?.scopeKey ||
    model.scopeKey === `${ref.namespace_id}/${ref.outcome_id}`;
  const hasModel = Boolean(model && matchingModel);
  if (!matchingModel) nodes.clear();
  const references = new Map([...nodes.values()].map((node) => [keyOf(node.ref), node]));
  const canonical = (value) => inScope(value) ? { ...value } : null;
  const reference = (kind, id) => id ? canonical({ ...ref, kind, id }) : null;
  const descriptor = (value) => {
    const target = canonical(value);
    if (!target) return null;
    const node = references.get(keyOf(target));
    return {
      ref: target,
      title: node?.contextMissing ? "" : node?.title || "",
      lifecycle: node?.lifecycle || "",
      detailsLoaded: Boolean(node && !node.contextMissing),
    };
  };
  const key = keyOf(ref), groups = new Map(), seen = new Set();
  const group = (id) => {
    if (!groups.has(id)) groups.set(id, { id, title: text(id), items: [] });
    return groups.get(id);
  };
  function add(groupId, label, target, { via, qualifier = "", criterionId = "", provenance = "loaded" } = {}) {
    const item = descriptor(target);
    if (!item) return;
    // Distinct EvidenceLinks to the same target/criterion are distinct records.
    const identity = [groupId, label, keyOf(item.ref), via?.kind, via?.id,
      qualifier, criterionId].join("|");
    if (seen.has(identity)) return;
    seen.add(identity);
    group(groupId).items.push({ ...item, label, qualifier, criterionId,
      via: canonical(via), provenance });
  }
  const scope = descriptor(reference("outcome", ref.outcome_id));
  let placement = null;
  if (ref.kind === "objective" || ref.kind === "work_item") {
    const id = ref.kind === "objective" ? entity.parent_objective_id : entity.objective_id;
    placement = {
      label: id
        ? text(ref.kind === "objective" ? "Parent" : "Objective")
        : ref.kind === "objective" ? text("Direct") : text("Unassigned"),
      target: id ? descriptor(reference("objective", id)) : null,
    };
  }
  const result = () => ({
    role: text(`Role${roles[ref.kind]}`),
    scope: ref.kind === "outcome" ? null : scope,
    placement,
    placementOnly,
    groups: [...groups.values()],
    partial: hasModel && Boolean(model.partial || model.boundary?.scopeLimited),
    coverage: text(hasModel ? "Coverage" : "DirectCoverage"),
    hasModel,
  });
  // Tasks retain their authoritative focal context for dependencies, Evidence,
  // blocking, attention and current authority. Do not duplicate those sections
  // with a less complete loaded-map index when only placement is requested.
  if (placementOnly) return result();

  // Direct fields are authoritative even when their endpoints are not among
  // the loaded notes. They never become invented edges or inferred state.
  const direct = { provenance: "record" };
  if (ref.kind === "issue") {
    for (const target of entity.affected_refs || [])
      add("Problems", text("Affects"), target, direct);
    if (entity.duplicate_of_issue_id)
      add("Problems", text("Duplicate"), reference("issue", entity.duplicate_of_issue_id), direct);
  }
  if (ref.kind === "blocker") {
    add("Problems", text(entity.lifecycle === "active" ? "Blocks" : "RecordedTarget"), entity.blocked_ref, direct);
    add("Problems", text("Cause"), entity.cause_ref, direct);
  }
  if (ref.kind === "evidence" && entity.artifact_id)
    add("Documents", text("ArtifactSource"), reference("artifact", entity.artifact_id), direct);
  if (ref.kind === "decision" && entity.supersedes_decision_id)
    add("Documents", text("Supersedes"), reference("decision", entity.supersedes_decision_id), direct);
  if (ref.kind === "relation") {
    add("Links", text("Source"), entity.source_ref, direct);
    add("Links", text("Target"), entity.target_ref, direct);
  }
  if (ref.kind === "evidence_link") {
    add("Links", text("EvidenceSource"), reference("evidence", entity.evidence_id), direct);
    const stance = { supports: "Supports", contradicts: "Contradicts", context: "ProvidesContext" };
    add("Links", text("Target"), entity.target_ref, { ...direct,
      qualifier: text(stance[entity.stance] || "StanceUnknown"),
      criterionId: entity.criterion_id || "",
    });
  }

  for (const edge of matchingModel ? model?.edges || [] : []) {
    const source = nodes.get(edge.sourceKey), target = nodes.get(edge.targetKey);
    if (!source || !target) continue;
    const outgoing = keyOf(source.ref) === key;
    if (!outgoing && keyOf(target.ref) !== key) continue;
    const other = outgoing ? target.ref : source.ref;
    const via = edge.ref;
    if (edge.kind === "hierarchy" && edge.sourceOfTruth === "objective" &&
      ["outcome", "objective"].includes(source.ref.kind) && target.ref.kind === "objective") {
      if (outgoing) add("PlanWork", text("ChildObjective"), other);
    } else if (edge.kind === "ownership" && edge.sourceOfTruth === "work_item" &&
      ["outcome", "objective"].includes(source.ref.kind) && target.ref.kind === "work_item") {
      if (outgoing) add("PlanWork", text("Task"), other);
    } else if (edge.kind === "depends_on" && edge.sourceOfTruth === "relation" &&
      ["objective", "work_item"].includes(source.ref.kind) &&
      ["objective", "work_item"].includes(target.ref.kind)) {
      add("Dependencies", text(outgoing ? "DependsOn" : "RequiredBy"), other, {
        via,
        qualifier: text(edge.mode === "hard" ? "Hard" : edge.mode === "advisory" ? "Advisory" : "StrengthUnknown"),
      });
    } else if (edge.sourceOfTruth === "evidence_link" && source.ref.kind === "evidence") {
      const stances = outgoing
        ? { supports: "Supports", contradicts: "Contradicts", context: "ProvidesContext" }
        : { supports: "SupportingEvidence", contradicts: "ContradictingEvidence", context: "ContextEvidence" };
      add("Evidence", text(stances[edge.kind] || "StanceUnknown"), other, {
        via, criterionId: edge.criterionId || "",
      });
    } else if (edge.kind === "blocks" && edge.sourceOfTruth === "blocker" &&
      source.ref.kind === "blocker") {
      // Active Blocker edges express direct impact only. Inherited effects
      // belong to the authoritative focal blocking response, not this index.
      if (!outgoing) add("Problems", text("BlockedBy"), other);
      else if (ref.kind !== "blocker" || !entity.blocked_ref)
        add("Problems", text("Blocks"), other);
    } else if (edge.kind === "causes" && edge.sourceOfTruth === "blocker" &&
      target.ref.kind === "blocker") {
      if (outgoing) add("Problems", text("CauseOf"), other);
      else if (!entity.cause_ref) add("Problems", text("Cause"), other);
    } else if (edge.kind === "artifact_source" && edge.sourceOfTruth === "evidence" &&
      source.ref.kind === "evidence" && target.ref.kind === "artifact") {
      if (!outgoing || !entity.artifact_id)
        add("Documents", text(outgoing ? "ArtifactSource" : "ObservedBy"), other);
    } else if (edge.kind === "supersedes" && edge.sourceOfTruth === "decision" &&
      source.ref.kind === "decision" && target.ref.kind === "decision") {
      if (!outgoing || !entity.supersedes_decision_id)
        add("Documents", text(outgoing ? "Supersedes" : "SupersededBy"), other);
    } else if (edge.sourceOfTruth === "relation" && edge.kind === "relates_to") {
      add("Documents", text("Related"), other, { via });
    } else if (edge.sourceOfTruth === "relation" && edge.kind === "derived_from") {
      add("Documents", text(outgoing ? "DerivedFrom" : "SourceFor"), other, { via });
    }
  }
  // A loaded Issue may include its affected_refs even without a graph edge.
  // Do not infer such links when a compact summary omits those fields.
  for (const node of nodes.values()) {
    if (node.ref.kind !== "issue" || keyOf(node.ref) === key) continue;
    if ((node.sourceSummary?.affected_refs || []).some((value) =>
      inScope(value) && keyOf(value) === key))
      add("Problems", text("AffectedByIssue"), node.ref);
  }
  return result();
}

export function renderEntityContext(host, { onReference, referenceTitle, ...options } = {}) {
  const context = buildEntityContext(options);
  if (!context) return null;
  const root = el("section", undefined, "detail-section entity-context");
  root.append(el("h3", text("Title")), el("p", context.role, "canvas-context-text"));
  function reference(value) {
    const label = value.title || referenceTitle?.(value.ref) || `${kindName(value.ref.kind)} ${shortId(value.ref.id)}`;
    if (typeof onReference !== "function")
      return el("span", label, "canvas-context-reference");
    const control = button(label, () => onReference({ ...value.ref }), "quiet canvas-context-reference");
    control.dataset.entityContextRef = `${value.ref.namespace_id}/${value.ref.outcome_id}/${value.ref.kind}/${value.ref.id}`;
    return control;
  }
  if (context.scope) {
    const line = el("p", `${text("Scope")}: `, "entity-context-placement");
    line.append(reference(context.scope));
    root.append(line);
  }
  if (context.placement) {
    const line = el("p", context.placement.label, "entity-context-placement");
    if (context.placement.target) {
      line.append(document.createTextNode(": "), reference(context.placement.target));
    }
    root.append(line);
  }
  function row(item) {
    const line = el("li", undefined, "canvas-context-card");
    line.append(el("small", item.label, "field-help"), reference(item));
    if (item.qualifier) line.append(el("small", item.qualifier, "field-help"));
    if (item.criterionId)
      line.append(el("small", message("entityContextCriterion", shortId(item.criterionId)), "field-help"));
    if (item.via)
      line.append(reference({ ref: item.via, title: text("InspectLink") }));
    return line;
  }
  for (const group of context.groups) {
    const section = el("section", undefined, "canvas-context-section");
    section.append(el("h4", group.title));
    const list = el("ul", undefined, "canvas-context-list");
    for (const item of group.items.slice(0, 4)) list.append(row(item));
    section.append(list);
    if (group.items.length > 4) {
      const more = el("details", undefined, "entity-context-more");
      more.dataset.entityContextSection = group.id;
      more.append(el("summary", message("entityContextMore", group.items.length - 4)));
      const remaining = el("ul", undefined, "canvas-context-list");
      for (const item of group.items.slice(4)) remaining.append(row(item));
      more.append(remaining);
      section.append(more);
    }
    root.append(section);
  }
  if (context.placementOnly) {
    host.append(root);
    return root;
  }
  if (!context.groups.length)
    root.append(el("p", text(context.hasModel ? "NoConnections" : "NoDirectConnections"), "field-help"));
  const coverage = el("details", undefined, "entity-context-coverage");
  coverage.dataset.entityContextSection = "coverage";
  coverage.append(el("summary", text("CoverageTitle")),
    el("p", context.coverage, "field-help"));
  if (context.partial) coverage.append(el("p", text("Partial"), "field-help"));
  root.append(coverage);
  host.append(root);
  return root;
}
