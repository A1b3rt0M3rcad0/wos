import {copy} from "./en-US.js";
// Presentation only. Lifecycle, readiness, proofs and permissions come from the API.
const ns = "http://www.w3.org/2000/svg";
export const node = (tag, text, className) => {
  const n = document.createElement(tag);
  if (text !== undefined) n.textContent = text;
  if (className) n.className = className;
  return n;
};
const paths = {
  summary: "M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z",
  list: "M8 6h13 M8 12h13 M8 18h13 M3 6h.01 M3 12h.01 M3 18h.01",
  board: "M3 4h5v16H3z M10 4h5v16h-5z M17 4h4v16h-4z",
  work_item: "M8 3h8v3H8z M6 5H4v16h16V5h-2 M8 12l2 2 5-5",
  objective: "M21 12a9 9 0 1 1-9-9 M12 7v5l4-4 M16 3h5v5",
  outcome: "M4 20L20 4 M11 4h9v9",
  roadmap: "M3 5h5v5H3z M16 14h5v5h-5z M8 7h10v7 M5 10v9h11",
  issue: "M12 3 2 21h20z M12 9v5 M12 17h.01",
  blocker: "M6 6h12v12H6z M8 8l8 8 M16 8l-8 8",
  evidence: "M8 3h8l4 4v14H4V3z M15 3v5h5 M8 12h8 M8 16h5",
  artifact: "M3 7h7l2 3h9v10H3z M3 7V4h7l2 3",
  decision: "M6 3v18 M6 7h10l-3-3 M16 7l-3 3 M6 16h10",
  timeline: "M12 3a9 9 0 1 1-9 9 M3 3v6h6 M12 7v5l3 2",
  graph:
    "M5 4a2 2 0 1 0 0 .01 M19 4a2 2 0 1 0 0 .01 M12 18a2 2 0 1 0 0 .01 M7 5h10 M6 6l5 10 M18 6l-5 10",
  search: "M10 3a7 7 0 1 0 0 14 7 7 0 0 0 0-14 M15 15l6 6",
  check: "M5 12l4 4L19 6",
  alert: "M12 3 2 21h20z M12 9v5 M12 17h.01",
  user: "M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M4 21v-2a8 8 0 0 1 16 0v2",
  clock: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 7v5l3 2",
  plus: "M12 5v14 M5 12h14",
};
export function icon(name) {
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("class", "icon");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("fill", "none");
  svg.setAttribute("stroke", "currentColor");
  svg.setAttribute("stroke-width", "1.6");
  svg.setAttribute("stroke-linecap", "round");
  svg.setAttribute("stroke-linejoin", "round");
  const p = document.createElementNS(ns, "path");
  p.setAttribute("d", paths[name] || paths.work_item);
  svg.append(p);
  return svg;
}
export const names = {
  draft: copy.text.draft,
  planned: copy.text.planned,
  archived: copy.text.archived,
  wont_fix: copy.text.wonTFix,
  duplicate: copy.text.duplicate,
  active: copy.text.active,
  achieved: copy.text.achievedf6c90c,
  failed: copy.text.failed,
  abandoned: copy.text.abandoned,
  backlog: "Backlog",
  todo: copy.text.toDo,
  in_progress: copy.text.inProgress,
  done: copy.text.done,
  cancelled: copy.text.cancelled,
  ready: copy.text.ready,
  blocked: copy.text.blocked,
  waiting_scope: copy.text.waitingForActivation,
  waiting_dependencies: copy.text.waitingForDependencies,
  scheduled: copy.text.scheduled,
  attention_needed: copy.text.needsAttention,
  open: copy.text.open,
  resolved: copy.text.resolved,
  investigating: copy.text.investigating,
  recorded: copy.text.registered,
  retracted: copy.text.retracted,
  superseded: copy.text.superseded,
  valid: copy.text.validf51622,
  published: copy.text.published,
  accepted: copy.text.accepted,
  proposed: copy.text.proposed,
  high: copy.text.high,
  critical: copy.text.critical,
  normal: "Normal",
  low: copy.text.low,
  major: copy.text.high,
  minor: copy.text.low,
  informational: copy.text.informational,
  met: copy.text.met,
  not_met: copy.text.notMet,
  inconclusive: copy.text.inconclusive,
  waived: copy.text.waivedNotVerified,
  attestation: copy.text.attestation,
  evidence_review: copy.text.evidenceReview,
  external_evaluation: copy.text.externalEvaluation,
  test_result: copy.text.testResult,
  inspection: copy.text.inspection,
  measurement: copy.text.measurement,
  source: copy.text.source,
  supports: copy.text.supports,
  contradicts: copy.text.contradicts,
  neutral: copy.text.neutral,
  direct: copy.text.direct,
  independent_review: copy.text.independentReview,
  retired: copy.text.retracted,
  revoked: copy.text.revoked,
  delivered: copy.text.executorSubmissionAccepted,
  changes_requested: copy.text.changesRequested,
  in_review: copy.text.inReview,
  subtree: copy.text.includesDescendants,
  human: copy.text.human,
  agent: copy.text.agent,
  service: copy.text.service,
  automation: copy.text.automation,
  external_system: copy.text.externalSystem,
  reference: copy.text.relatedItem,
  phase: copy.text.phase,
  milestone: copy.text.milestone,
  none: copy.text.unassigned,
  expired: copy.text.expired,
  hard_dependency_unsatisfied: copy.text.dependencyNotCompleted,
  not_before: copy.text.scheduledStartHasNotArrived,
  lease_expired: copy.text.reservationExpired,
  active_lease: copy.text.activeReservation,
  outcome_archived: copy.text.outcomeArchived,
  objective_terminal: copy.text.objectiveIsTerminal,
  depends_on: copy.text.dependsOn,
  lifecycle_not_ready: copy.text.itemIsNotReadyForExecution,
  outcome_not_active: copy.text.outcomeIsNotActive,
  objective_not_active: copy.text.objectiveIsNotActive,
  evidence_retracted: copy.text.evidenceRetracted,
  assessment_not_met: copy.text.criterionNotMet,
  assessment_inconclusive: copy.text.inconclusiveAssessment,
  criterion_revision_changed: copy.text.criterionRevised,
};
export const display = (value) =>
  names[value] || String(value ?? "").replaceAll("_", " ");
export const kindName = (kind) =>
  ({
    outcome: copy.text.outcome,
    objective: copy.text.objective,
    work_item: copy.text.task4bc74b,
    issue: copy.text.issueKind,
    blocker: copy.text.blockerKind,
    roadmap: copy.text.roadmap,
    evidence: copy.text.evidence,
    artifact: copy.text.artifact,
    decision: copy.text.decision640ae4,
  })[kind] || display(kind);
export const tone = (value) =>
  [
    "done",
    "achieved",
    "met",
    "ready",
    "valid",
    "resolved",
    "accepted",
  ].includes(value)
    ? "green"
    : [
          "blocked",
          "failed",
          "not_met",
          "retracted",
          "critical",
          "attention_needed",
        ].includes(value)
      ? "red"
      : ["active", "in_progress", "high", "scheduled"].includes(value)
        ? "orange"
        : ["published", "proposed", "draft", "waived"].includes(value)
          ? "violet"
          : "";
export const badge = (value) =>
  node("span", display(value), `badge ${tone(value)}`);
export const shortId = (id) => (id ? `WOS-${id.slice(-6).toUpperCase()}` : "");
export function date(value) {
  if (!value) return "—";
  const d = new Date(value);
  return Number.isNaN(d.valueOf())
    ? String(value)
    : new Intl.DateTimeFormat("en-US", {
        dateStyle: "medium",
        timeStyle: "short",
 timeZone: "UTC",
      }).format(d)+" UTC";
}

// Free text and identifiers retain the caller's content, even if they happen to
// equal a wire enum. Only explicitly typed presentation fields receive labels.
export function formatValue(value, key = "") {
  if (key.endsWith("_at") || key.startsWith("planned_")) return date(value);
  if (typeof value === "boolean") return value ? copy.text.yes : copy.text.no;
  return ["lifecycle", "status", "priority", "readiness", "verification_mode",
    "evidence_type", "result", "stance", "severity", "propagation", "node_type", "kind"]
    .includes(key) ? display(value) : String(value);
}
export function donut(parts, total) {
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("viewBox", "0 0 150 150");
  svg.setAttribute("class", "status-chart");
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", `Task distribution: ${total} items`);
  let offset = 0;
  const circle = (color, length) => {
    const c = document.createElementNS(ns, "circle");
    for (const [k, v] of Object.entries({
      cx: 75,
      cy: 75,
      r: 57,
      fill: "none",
      stroke: color,
      "stroke-width": 13,
      "stroke-dasharray": `${length} ${358.14 - length}`,
      "stroke-dashoffset": -offset,
      transform: "rotate(-90 75 75)",
    }))
      c.setAttribute(k, String(v));
    svg.append(c);
    offset += length;
  };
  circle("#eeece5", 358.14);
  offset = 0;
  for (const p of parts)
    if (total && p.count) circle(p.color, (358.14 * p.count) / total);
  for (const [y, text, cls] of [
    [74, String(total), "chart-total"],
    [94, copy.text.taskItems, "chart-caption"],
  ]) {
    const t = document.createElementNS(ns, "text");
    t.setAttribute("x", "75");
    t.setAttribute("y", String(y));
    t.setAttribute("class", cls);
    t.textContent = text;
    svg.append(t);
  }
  return svg;
}
