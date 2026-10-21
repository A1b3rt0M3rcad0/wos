import { copy } from "./en-US.js";
export const areas = Object.freeze({
  canvas: { label: copy.text.canvas, sections: [] },
  overview: { label: copy.text.overview, sections: [] },
  plan: {
    label: copy.text.plan,
    sections: [
      "objectives",
      "roadmaps",
      "active_roadmaps",
      "active_plan_references",
    ],
  },
  work: { label: copy.text.work, sections: ["work_items"] },
  issues: { label: copy.text.issues, sections: ["issues", "blockers"] },
  evidence: {
    label: copy.text.evidence,
    sections: ["evidence", "artifacts", "decisions"],
  },
  activity: {
    label: copy.text.activity,
    sections: ["timeline", "conclusion_contestations", "graph"],
  },
});
export const workSections = [
  "work_items",
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
const uuid =
  /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const kinds = new Set([
  "outcome",
  "objective",
  "work_item",
  "issue",
  "blocker",
  "evidence",
  "artifact",
  "decision",
  "roadmap",
  "relation",
  "evidence_link",
]);
export function areaFor(section) {
  return workSections.includes(section)
    ? "work"
    : Object.keys(areas).find((a) => areas[a].sections.includes(section)) ||
        "overview";
}
export function readRoute(search) {
  const p = new URLSearchParams(search.replace(/^[?#]/, "")),
    namespace = p.get("workspace"),
    outcome = p.get("outcome");
  if (!namespace && !outcome) return null;
  if (!uuid.test(namespace || "") || !uuid.test(outcome || ""))
    throw Error(copy.text.thisWorkspaceLinkHasInvalidIdentifiers);
  const area = p.get("area") || "canvas";
  if (!areas[area]) throw Error(copy.text.thisWorkspaceLinkHasAnUnknownArea);
  let section = p.get("section") || areas[area].sections[0] || "work_items";
  if (!["canvas", "overview"].includes(area) && areaFor(section) !== area)
    throw Error(copy.text.theLinkedSectionDoesNotBelongToThisArea);
  const view =
    area === "canvas"
      ? "canvas"
      : area === "overview"
        ? "summary"
        : area === "work" && p.get("view") === "board"
          ? "board"
          : "list";
  let item = null;
  if (p.has("item")) {
    const [kind, id, ...extra] = p.get("item").split("/");
    if (extra.length || !kinds.has(kind) || !uuid.test(id || ""))
      throw Error(copy.text.thisWorkspaceLinkHasAnInvalidItem);
    item = { kind, id, namespace_id: namespace, outcome_id: outcome };
  }
  let subject;
  if (p.has("subject")) {
    if (!item) throw Error(copy.text.thisWorkspaceLinkHasAnInvalidItem);
    const kind = p.get("subject");
    if (kind === "criterion" && ["outcome", "objective", "work_item"].includes(item.kind) && uuid.test(p.get("criterion") || ""))
      subject = { kind, ref: item, criterion_id: p.get("criterion") };
    else if (kind === "roadmap_revision" && item.kind === "roadmap" && /^[1-9][0-9]{0,19}$/.test(p.get("revision") || "") && BigInt(p.get("revision")) <= 18446744073709551615n)
      subject = { kind, ref: item, revision_number: p.get("revision") };
    else if (kind === "execution_record" && item.kind === "work_item" && ["contract", "submission", "review_case"].includes(p.get("record_kind")) && uuid.test(p.get("record") || ""))
      subject = { kind, ref: item, record_kind: p.get("record_kind"), record_id: p.get("record") };
    else throw Error(copy.text.thisWorkspaceLinkHasAnInvalidItem);
  }
  return { namespace, outcome, area, view, section, item, ...(subject ? { subject } : {}) };
}
export function routeQuery({
  namespace,
  outcome,
  area = "canvas",
  view = "canvas",
  section = "work_items",
  item,
  subject,
}) {
  if (!namespace || !outcome) return "";
  const p = new URLSearchParams({ workspace: namespace, outcome, area });
  if (!["canvas", "overview"].includes(area)) p.set("section", section);
  if (area === "work") p.set("view", view);
  if (item) p.set("item", `${item.kind}/${item.id}`);
  if (item && subject && subject.kind !== "entity" && subject.ref?.kind === item.kind && subject.ref?.id === item.id &&
      subject.ref.namespace_id === namespace && subject.ref.outcome_id === outcome) {
    p.set("subject", subject.kind);
    if (subject.criterion_id) p.set("criterion", subject.criterion_id);
    if (subject.revision_number) p.set("revision", subject.revision_number);
    if (subject.record_kind) p.set("record_kind", subject.record_kind);
    if (subject.record_id) p.set("record", subject.record_id);
  }
  return "?" + p;
}
