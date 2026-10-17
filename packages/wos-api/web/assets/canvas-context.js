import { el, button } from "./ui.js";
import { badge, date, display, kindName, shortId } from "./presentation.js";

// A bounded view of authoritative focal reads. This module performs no reads,
// mutations, readiness calculations, or interpretation of execution authority.
export function renderCanvasContext(
  host,
  { entity = {}, context, attention, onReference, onHistory } = {},
) {
  const root = el("section", undefined, "detail-section canvas-task-context");
  root.append(el("h3", "Task context"));
  const work = context?.work_item || entity;
  const scope = work.scope || entity.scope || {};
  const localOmissions = new Map();
  const itemLimit = 10;
  const textLimit = 1200;

  function text(value, label = "Text") {
    const characters = Array.from(String(value ?? ""));
    if (characters.length <= textLimit) return characters.join("");
    localOmissions.set(label, (localOmissions.get(label) || 0) + characters.length - textLimit);
    return `${characters.slice(0, textLimit).join("")}…`;
  }
  function paragraph(parent, value, label) {
    if (value !== undefined && value !== null && value !== "")
      parent.append(el("p", text(value, label), "canvas-context-text"));
  }
  function section(label) {
    const node = el("section", undefined, "canvas-context-section");
    node.append(el("h4", label));
    root.append(node);
    return node;
  }
  function list(parent, values, label, render) {
    if (!Array.isArray(values)) return;
    const node = el("ul", undefined, "canvas-context-list");
    for (const value of values.slice(0, itemLimit)) {
      const item = el("li", undefined, "canvas-context-card");
      render(item, value);
      node.append(item);
    }
    if (node.children.length) parent.append(node);
    if (values.length > itemLimit)
      parent.append(el("small", `${values.length - itemLimit} additional ${label} not shown in this summary.`, "canvas-context-omissions field-help"));
  }
  function recordRef(record, kind) {
    if (record?.ref) return record.ref;
    return record?.id ? { ...(record.scope || scope), kind, id: record.id } : null;
  }
  function reference(ref, title) {
    if (!ref?.id || !ref?.kind) return el("span", "Reference not included.");
    const label = title ? text(title, "Reference titles") : `${kindName(ref.kind)} ${shortId(ref.id)}`;
    return typeof onReference === "function"
      ? button(label, () => onReference({ ...ref }), "quiet canvas-context-reference")
      : el("span", label, "canvas-context-reference");
  }
  function snapshot(parent, value, label) {
    if (!value) return;
    const parts = [];
    if (value.outcome_revision !== undefined)
      parts.push(
        typeof value.outcome_revision === "number" && !Number.isSafeInteger(value.outcome_revision)
          ? "Exact revision unavailable in this response"
          : `Revision ${String(value.outcome_revision)}`,
      );
    if (value.evaluated_at) parts.push(`evaluated ${date(value.evaluated_at)}`);
    if (parts.length) parent.append(el("small", `${label}: ${parts.join(" · ")}`, "field-help"));
  }

  snapshot(root, context, "Context snapshot");
  if (context?.outcome || context?.objective) {
    const placement = section("Outcome and Objective");
    if (context.outcome) {
      const row = el("p", "Outcome: ");
      row.append(reference(recordRef(context.outcome, "outcome"), context.outcome.title));
      placement.append(row);
    }
    if (context.objective) {
      const row = el("p", "Objective: ");
      row.append(reference(recordRef(context.objective, "objective"), context.objective.title));
      placement.append(row);
    }
  }

  const spec = work.execution_spec;
  if (spec) {
    for (const [key, label] of [
      ["instructions", "Instructions"],
      ["constraints", "Constraints"],
      ["deliverables", "Deliverables"],
      ["scope_hints", "Scope hints"],
    ]) {
      if (!spec[key]?.length) continue;
      list(section(label), spec[key], label.toLowerCase(), (item, value) => paragraph(item, value, label));
    }
    if (spec.context_refs?.length)
      list(section("Execution context references"), spec.context_refs, "context references", (item, ref) => item.append(reference(ref)));
  }

  if (attention) {
    const current = section("Current attention");
    if (attention.status) current.append(badge(attention.status));
    snapshot(current, attention, "Attention snapshot");
    if (attention.awaiting_review === true)
      paragraph(current, "Awaiting independent review. Accepted executor material does not by itself complete the Task.");
    if (attention.evidence_search_complete === false)
      paragraph(current, "The evidence search is incomplete. Missing evidence cannot be inferred from this response.");
    list(current, attention.obligations, "obligations", (item, obligation) => {
      paragraph(item, obligation.description, "Obligation descriptions");
      if (obligation.status) item.append(badge(obligation.status));
      if (obligation.assessment_result) item.append(badge(obligation.assessment_result));
      if (obligation.criterion_revision !== undefined)
        item.append(el("small", `Criterion revision ${String(obligation.criterion_revision)}`, "field-help"));
    });
    if (attention.correction_findings?.length)
      list(section("Correction findings"), attention.correction_findings, "correction findings", (item, finding) => {
        paragraph(item, finding.description, "Correction descriptions");
        paragraph(item, finding.requirement_ref, "Requirement references");
      });
  }
  const operational = attention?.operational_state || context?.operational_state;
  if (operational?.readiness_reasons?.length)
    list(section("Readiness reasons"), operational.readiness_reasons, "readiness reasons", (item, reason) => paragraph(item, display(reason)));

  if (context?.dependencies?.length) {
    list(section("Dependencies and relations"), context.dependencies, "relations", (item, relation) => {
      const strength = relation.strength === "hard"
        ? "Hard prerequisite"
        : relation.strength === "advisory"
          ? "Advisory dependency"
          : `Dependency type: ${relation.strength ? display(relation.strength) : "not included"}`;
      item.append(el("strong", relation.relation_type === "depends_on" ? strength : display(relation.relation_type)));
      const target = el("p", relation.relation_type === "depends_on" ? "Depends on: " : "Target: ");
      target.append(reference(relation.target_ref));
      item.append(target);
      paragraph(item, relation.reason, "Dependency reasons");
      if (relation.lifecycle) item.append(badge(relation.lifecycle));
      if (relation.id) item.append(reference(recordRef(relation, "relation"), "Inspect relation"));
    });
  }
  if (context?.blocking) {
    const blocking = section("Blocking impact");
    if (typeof context.blocking.is_blocked === "boolean")
      paragraph(blocking, `Blocked: ${context.blocking.is_blocked ? "Yes" : "No"}`);
    list(blocking, context.blocking.active_blockers, "Blockers", (item, applied) => {
      const blocker = applied.blocker;
      if (!blocker) return;
      item.append(reference(recordRef(blocker, "blocker"), `Blocker ${shortId(blocker.id)}`));
      paragraph(item, blocker.description, "Blocker descriptions");
      if (blocker.lifecycle) item.append(badge(blocker.lifecycle));
      if (blocker.cause_ref) {
        const cause = el("p", "Cause: ");
        cause.append(reference(blocker.cause_ref));
        item.append(cause);
      }
      if (blocker.external_cause) paragraph(item, blocker.external_cause.description, "External cause descriptions");
      if (applied.inherited === true) {
        const inherited = el("p", "Inherited blocking impact from: ");
        inherited.append(reference(applied.inherited_from));
        item.append(inherited);
      }
    });
  }
  if (context?.evidence_links?.length) {
    const evidence = section("Evidence relationships");
    paragraph(evidence, "A relationship describes how evidence relates to its target; it is not an assessment or acceptance decision.");
    list(evidence, context.evidence_links, "evidence relationships", (item, link) => {
      const stance = { supports: "Supports", contradicts: "Contradicts", context: "Context" }[link.stance] || `Stance: ${link.stance ? display(link.stance) : "not included"}`;
      item.append(el("strong", stance), reference(recordRef({ id: link.evidence_id, scope: link.scope }, "evidence")));
      const target = el("p", "Target: ");
      target.append(reference(link.target_ref));
      item.append(target);
      paragraph(item, link.rationale, "Evidence rationales");
      if (link.lifecycle) item.append(badge(link.lifecycle));
    });
  }
  if (context?.current_decisions?.length)
    list(section("Current decisions"), context.current_decisions, "decisions", (item, decision) => {
      item.append(reference(recordRef(decision, "decision"), decision.title));
      if (decision.lifecycle) item.append(badge(decision.lifecycle));
      paragraph(item, decision.chosen_alternative, "Chosen alternatives");
      paragraph(item, decision.rationale, "Decision rationales");
    });

  const completeness = section("Context coverage");
  if (!context) paragraph(completeness, "Focal context was not included. Missing related records cannot be inferred from this summary.");
  for (const [value, label] of [[context, "Context"], [attention, "Attention"]]) {
    if (!value) continue;
    if (value.truncated) paragraph(completeness, `${label} is partial. Only returned records are shown.`);
    for (const [key, count] of Object.entries(value.omitted || {}))
      if (/^[1-9]\d*$/.test(String(count)))
        paragraph(completeness, `${label}: ${String(count)} ${display(key)} omitted by the server.`);
  }
  for (const [label, count] of localOmissions)
    paragraph(completeness, `${label}: ${count} characters not shown in this summary.`);
  paragraph(completeness, "This focal snapshot does not include the complete historical record.");
  if (typeof onHistory === "function" && work.id)
    completeness.append(button("Load task history", () => onHistory(recordRef(work, "work_item")), "quiet"));
  host.replaceChildren(root);
  return root;
}
