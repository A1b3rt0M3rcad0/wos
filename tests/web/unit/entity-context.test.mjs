import { test } from "node:test";
import assert from "node:assert/strict";
import { buildEntityContext } from "../../../packages/wos-api/web/assets/workspace-entity-context.js";

const scope = { namespace_id: "workspace-a", outcome_id: "result-a" };
const ref = (kind, id = kind) => ({ ...scope, kind, id });
const key = (value) => `${value.kind}:${value.id}`;
const node = (kind, id = kind, extra = {}) => ({
  key: key(ref(kind, id)), ref: ref(kind, id), kind,
  title: `${kind} ${id}`, ...extra,
});
const edge = (id, source, target, kind, sourceOfTruth, extra = {}) => ({
  key: `graph:${sourceOfTruth}:${id}`, sourceKey: key(source), targetKey: key(target),
  kind, sourceOfTruth, ...extra,
});
const base = () => ({
  scopeKey: `${scope.namespace_id}/${scope.outcome_id}`,
  nodes: [node("outcome", scope.outcome_id), node("objective"), node("work_item"),
    node("work_item", "prerequisite"), node("issue"), node("blocker"),
    node("evidence"), node("artifact"), node("decision")],
  edges: [],
});
const build = (kind, entity = {}, model = base()) =>
  buildEntityContext({ ref: ref(kind, kind === "outcome" ? scope.outcome_id : kind), entity, model });
const group = (context, id) => context.groups.find((value) => value.id === id)?.items || [];
const freeze = (value) => {
  if (value && typeof value === "object") {
    for (const child of Object.values(value)) freeze(child);
    Object.freeze(value);
  }
  return value;
};

test("every Board entity communicates its actual role without conflating completion, evidence or authority", () => {
  for (const kind of ["outcome", "objective", "work_item", "issue", "blocker", "evidence", "artifact", "decision", "roadmap", "relation", "evidence_link"])
    assert.ok(build(kind).role?.length > 30, kind);
  assert.match(build("work_item").role, /does not automatically achieve/);
  assert.match(build("evidence").role, /does not verify a criterion/);
  assert.match(build("issue").role, /separate Blocker/);
  assert.match(build("roadmap").role, /do not create operational dependencies/);
  assert.match(build("decision").role, /do not grant execution authority/);
  assert.equal(build("outcome").scope, null);
});

test("ownership is recorded explicitly; hierarchy, plan grouping and dependencies do not reparent work", () => {
  const model = base();
  model.edges.push(
    edge("objective", ref("outcome", scope.outcome_id), ref("objective"), "hierarchy", "objective"),
    edge("task", ref("objective"), ref("work_item"), "ownership", "work_item"),
    edge("fake", ref("objective"), ref("work_item", "prerequisite"), "ownership", "roadmap_revision"),
    edge("dep", ref("work_item"), ref("work_item", "prerequisite"), "depends_on", "relation", { mode: "hard" }),
  );
  const objective = build("objective", { parent_objective_id: "parent-not-loaded" }, model);
  assert.equal(objective.scope.ref.id, scope.outcome_id);
  assert.equal(objective.placement.target.ref.id, "parent-not-loaded");
  assert.equal(objective.placement.target.detailsLoaded, false);
  assert.deepEqual(group(objective, "PlanWork").map((item) => item.ref), [ref("work_item")]);
  const task = build("work_item", { objective_id: "objective" }, model);
  assert.equal(task.placement.target.title, "objective objective");
  assert.equal(group(task, "Dependencies")[0].qualifier, "Hard prerequisite");
  assert.equal(build("work_item", {}, model).placement.label, "Directly in this Outcome");
});

test("dependency direction, unknown strength and distinct records remain visible without readiness inference", () => {
  const model = base();
  const prerequisite = ref("work_item", "prerequisite");
  model.edges.push(
    edge("hard", ref("work_item"), prerequisite, "depends_on", "relation", { mode: "hard", ref: ref("relation", "hard") }),
    edge("advisory", ref("work_item"), prerequisite, "depends_on", "relation", { mode: "advisory", ref: ref("relation", "advisory") }),
    edge("unknown", ref("work_item"), prerequisite, "depends_on", "relation", { ref: ref("relation", "unknown") }),
    edge("invalid", ref("evidence"), ref("work_item"), "depends_on", "relation"),
    edge("plan", ref("work_item"), prerequisite, "depends_on", "roadmap_revision"),
  );
  const task = build("work_item", {}, model);
  assert.deepEqual(group(task, "Dependencies").map((item) => item.qualifier),
    ["Hard prerequisite", "Advisory dependency", "Strength not loaded"]);
  const incoming = buildEntityContext({ ref: prerequisite, entity: {}, model });
  assert.ok(group(incoming, "Dependencies").every((item) => item.label === "Required by"));
  for (const forbidden of ["ready", "is_blocked", "accepted", "verified", "total"])
    assert.equal(Object.hasOwn(task, forbidden), false);
});

test("EvidenceLink stance and criterion belong to each individual link, never to the Evidence or assessment", () => {
  const model = base();
  model.edges.push(
    edge("support", ref("evidence"), ref("work_item"), "supports", "evidence_link", {
      ref: ref("evidence_link", "support"), criterionId: "criterion-a",
    }),
    edge("contradiction", ref("evidence"), ref("objective"), "contradicts", "evidence_link", {
      ref: ref("evidence_link", "contradiction"), criterionId: "criterion-b",
    }),
    edge("unknown", ref("evidence"), ref("work_item"), "not_loaded", "evidence_link"),
    edge("fake", ref("evidence"), ref("work_item"), "supports", "relation"),
  );
  const evidence = build("evidence", {}, model);
  assert.deepEqual(group(evidence, "Evidence").map(({ label, criterionId }) => ({ label, criterionId })), [
    { label: "Supports", criterionId: "criterion-a" },
    { label: "Contradicts", criterionId: "criterion-b" },
    { label: "Stance not loaded", criterionId: "" },
  ]);
  assert.equal(group(build("work_item", {}, model), "Evidence")[0].label, "Supporting Evidence");
  assert.match(evidence.coverage, /not proof of absence, readiness or acceptance/);
});

test("Issue affected references and Blocker causes never silently turn into inherited blocking", () => {
  const model = base();
  model.nodes.find((value) => value.kind === "issue").sourceSummary = {
    lifecycle: "resolved", affected_refs: [ref("work_item")],
  };
  model.edges.push(
    edge("cause", ref("issue"), ref("blocker"), "causes", "blocker"),
    edge("impact", ref("blocker"), ref("objective"), "blocks", "blocker"),
    edge("not-blocking", ref("issue"), ref("work_item"), "blocks", "blocker"),
    edge("ownership", ref("objective"), ref("work_item"), "ownership", "work_item"),
  );
  const issue = build("issue", { lifecycle: "resolved", affected_refs: [ref("work_item")] }, model);
  assert.deepEqual(group(issue, "Problems").map((item) => item.label), [
    "Affected item · not a blocking statement", "Cause of Blocker",
  ]);
  const task = build("work_item", {}, model);
  assert.deepEqual(group(task, "Problems").map((item) => item.label), [
    "Issue affecting this item · not a blocking statement",
  ]);
  assert.equal(group(build("objective", {}, model), "Problems")[0].label, "Direct Blocker");
  const resolved = build("blocker", { lifecycle: "resolved", blocked_ref: ref("objective"), cause_ref: ref("issue") }, model);
  assert.deepEqual(group(resolved, "Problems").map((item) => item.label), ["Recorded blocked target", "Recorded cause"]);
});

test("documentary links and endpoint records remain distinct from operational structure", () => {
  const model = base();
  model.edges.push(
    edge("artifact", ref("evidence"), ref("artifact"), "artifact_source", "evidence"),
    edge("derived", ref("artifact"), ref("work_item"), "derived_from", "relation", { ref: ref("relation", "derived") }),
  );
  assert.equal(group(build("artifact", {}, model), "Documents")[0].label, "Observed by Evidence");
  assert.equal(group(build("evidence", { artifact_id: "artifact" }, model), "Documents").length, 1);
  const relationship = build("evidence_link", {
    evidence_id: "evidence", target_ref: ref("objective"), criterion_id: "criterion-b", stance: "contradicts",
  }, model);
  assert.deepEqual(group(relationship, "Links").map((item) => item.ref.kind), ["evidence", "objective"]);
  assert.equal(group(relationship, "Links")[1].criterionId, "criterion-b");
  assert.equal(group(relationship, "Links")[1].qualifier, "Contradicts");
  const directOnly = buildEntityContext({ ref: ref("evidence_link"), entity: {
    evidence_id: "evidence", target_ref: ref("objective"), stance: "supports",
  } });
  assert.equal(group(directOnly, "Links")[1].qualifier, "Supports");
  assert.match(directOnly.coverage, /Board connection index is not included here/);
});

test("foreign scopes, missing notes and unrelated model data cannot manufacture navigable connections", () => {
  const model = base();
  const foreign = { ...ref("objective"), namespace_id: "another-workspace" };
  model.nodes.push({ key: "foreign", ref: foreign, title: "Private title" });
  model.edges.push(edge("foreign", ref("work_item"), foreign, "depends_on", "relation"));
  const issue = build("issue", { affected_refs: [foreign, ref("work_item", "not-loaded")] }, model);
  assert.equal(group(issue, "Problems").length, 1);
  assert.equal(group(issue, "Problems")[0].ref.id, "not-loaded");
  assert.equal(group(issue, "Problems")[0].title, "");
  assert.equal(group(issue, "Problems")[0].provenance, "record");
  const stale = build("work_item", {}, { ...model, scopeKey: "another-workspace/result-a" });
  assert.equal(stale.groups.length, 0);
  assert.equal(stale.scope.title, "");
  assert.equal(buildEntityContext({ ref: ref("issue"), entity: { scope: { ...scope, namespace_id: "foreign" } }, model }), null);
  assert.equal(buildEntityContext({ ref: ref("issue"), entity: { id: "wrong-entity" }, model }), null);
});

test("loaded coverage, direct references and pure indexing preserve frozen server facts", () => {
  const model = freeze({ ...base(), partial: true });
  const entity = freeze({ scope, objective_id: "unloaded-objective" });
  const before = structuredClone({ model, entity });
  const context = build("work_item", entity, model);
  assert.equal(context.partial, true);
  assert.equal(context.placement.target.ref.id, "unloaded-objective");
  assert.equal(context.placement.target.detailsLoaded, false);
  context.placement.target.ref.id = "local-change";
  assert.deepEqual({ model, entity }, before);
  assert.equal(build("work_item", {}, { ...base(), boundary: { scopeLimited: true } }).partial, true);
});

test("Task placement can complement authoritative focal context without duplicating loaded relationship sections", () => {
  const model = base();
  model.edges.push(edge("dep", ref("work_item"), ref("objective"), "depends_on", "relation", { mode: "hard" }));
  const context = buildEntityContext({ ref: ref("work_item"), entity: { objective_id: "objective" }, model, placementOnly: true });
  assert.match(context.role, /does not automatically achieve/);
  assert.equal(context.placement.target.ref.kind, "objective");
  assert.equal(context.placementOnly, true);
  assert.equal(context.groups.length, 0);
});
