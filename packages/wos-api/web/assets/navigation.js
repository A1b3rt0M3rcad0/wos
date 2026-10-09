import { copy } from "./en-US.js";
export const areas = Object.freeze({
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
  const area = p.get("area") || "overview";
  if (!areas[area]) throw Error(copy.text.thisWorkspaceLinkHasAnUnknownArea);
  let section = p.get("section") || areas[area].sections[0] || "work_items";
  if (area !== "overview" && areaFor(section) !== area)
    throw Error(copy.text.theLinkedSectionDoesNotBelongToThisArea);
  const view =
    area === "overview"
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
  return { namespace, outcome, area, view, section, item };
}
export function routeQuery({
  namespace,
  outcome,
  area = "overview",
  view = "summary",
  section = "work_items",
  item,
}) {
  if (!namespace || !outcome) return "";
  const p = new URLSearchParams({ workspace: namespace, outcome, area });
  if (area !== "overview") p.set("section", section);
  if (area === "work") p.set("view", view);
  if (item) p.set("item", `${item.kind}/${item.id}`);
  return "?" + p;
}
