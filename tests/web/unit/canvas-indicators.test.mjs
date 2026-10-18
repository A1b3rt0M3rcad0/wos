import { test } from "node:test";
import assert from "node:assert/strict";
import { canvasNodeIndicators } from "../../../packages/wos-api/web/assets/workspace-canvas.js";

const item = (key, kind, extra = {}) => ({ key, kind, ...extra });
const edge = (id, sourceKey, targetKey, kind, sourceOfTruth, extra = {}) => ({
  key: `graph:${sourceOfTruth}:${id}`,
  sourceKey,
  targetKey,
  kind,
  sourceOfTruth,
  ...extra,
});
const nodes = () => [
  item("root", "outcome", { lifecycle: "active" }),
  item("objective", "objective", { parentKey: "root", lifecycle: "planned" }),
  item("task", "work_item", { parentKey: "objective", displayState: "todo" }),
  item("other", "work_item"),
  item("issue", "issue", { lifecycle: "resolved" }),
  item("blocker", "blocker", { lifecycle: "active" }),
  item("evidence", "evidence", { lifecycle: "registered" }),
];

test("only actual outgoing operational dependencies contribute; strength stays explicit", () => {
  const model = {
    nodes: nodes(),
    edges: [
      edge("hard", "task", "other", "depends_on", "relation", { mode: "hard" }),
      edge("advisory", "task", "objective", "depends_on", "relation", { mode: "advisory" }),
      edge("unknown", "task", "other", "depends_on", "relation"),
      edge("incoming", "other", "task", "depends_on", "relation", { mode: "hard" }),
      edge("parent", "objective", "task", "ownership", "work_item"),
      edge("order", "task", "other", "plan_sequence", "roadmap_revision"),
      edge("fake", "task", "other", "depends_on", "roadmap_revision"),
    ],
  };
  const result = canvasNodeIndicators(model);
  assert.deepEqual(result.get("task").dependencies, {
    total: 3, hard: 1, advisory: 1, unknown: 1,
  });
  assert.equal(result.get("other").dependencies.total, 1);
  assert.equal(result.get("objective").dependencies.total, 0);
  assert.equal(result.get("task").label, "Loaded · 3 deps");
  assert.match(result.get("task").description, /1 strength not loaded/);
  assert.match(result.get("task").description, /Readiness and acceptance remain explicit server state/);
  assert.equal(model.nodes.find((node) => node.key === "task").displayState, "todo");
});

test("canonical references deduplicate overlapping loaded edges without hiding unknown or conflicting detail", () => {
  const original = edge("one", "task", "other", "depends_on", "relation");
  const alias = {
    ...original,
    key: "selected-context:one",
    ref: { kind: "relation", id: "one" },
    mode: "hard",
  };
  const conflicting = edge("two", "task", "other", "depends_on", "relation", { mode: "hard" });
  const edges = [
    original, original, alias, alias,
    conflicting,
    { ...conflicting, mode: "advisory" },
    conflicting,
  ];
  for (const ordered of [edges, [...edges].reverse()])
    assert.deepEqual(canvasNodeIndicators({ nodes: nodes(), edges: ordered }).get("task").dependencies, {
      total: 2, hard: 1, advisory: 0, unknown: 1,
    });
});

test("Blocker impact is direct and typed; Issue association, cause and hierarchy do not manufacture blocking", () => {
  const model = {
    nodes: nodes(),
    edges: [
      edge("impact", "blocker", "objective", "blocks", "blocker"),
      edge("same-impact", "blocker", "objective", "blocks", "blocker"),
      edge("cause", "issue", "blocker", "causes", "blocker"),
      edge("related", "issue", "task", "relates_to", "relation"),
      edge("incorrect-source", "issue", "task", "blocks", "blocker"),
      edge("incorrect-truth", "blocker", "task", "blocks", "relation"),
      edge("hierarchy", "objective", "task", "ownership", "work_item"),
    ],
  };
  const result = canvasNodeIndicators(model);
  assert.equal(result.get("objective").blockers, 1);
  assert.equal(result.get("task").blockers, 0);
  assert.equal(result.get("issue").blockers, 0);
  assert.equal(result.get("blocker").blockers, 0);
  assert.match(result.get("objective").description, /direct Blocker impact link; inherited blocking is not inferred/);
  assert.equal(result.get("task").label, "");
});

test("EvidenceLink stances remain per link and never imply criterion verification", () => {
  const support = edge("support", "evidence", "task", "supports", "evidence_link", {
    criterionId: "criterion-a",
  });
  const model = {
    nodes: nodes(),
    edges: [
      support,
      { ...support, key: "selected-link", ref: { kind: "evidence_link", id: "support" } },
      edge("contradiction", "evidence", "task", "contradicts", "evidence_link"),
      edge("context", "evidence", "objective", "context", "evidence_link"),
      edge("unknown", "evidence", "objective", "unloaded_stance", "evidence_link"),
      edge("relation", "evidence", "task", "supports", "relation"),
    ],
  };
  const result = canvasNodeIndicators(model);
  assert.deepEqual(result.get("evidence").evidence, {
    total: 4, supports: 1, contradicts: 1, context: 1, unknown: 1,
  });
  assert.deepEqual(result.get("task").evidence, {
    total: 2, supports: 1, contradicts: 1, context: 0, unknown: 0,
  });
  assert.match(result.get("evidence").description, /outgoing EvidenceLinks/);
  assert.match(result.get("objective").description, /1 stance not loaded/);
  assert.match(result.get("task").description, /incoming EvidenceLinks/);
  assert.match(result.get("task").description, /Evidence is not criterion verification/);
  assert.equal(result.get("task").label, "Loaded · 2 evidence links");
  assert.equal(support.criterionId, "criterion-a");
  assert.ok(!Object.hasOwn(result.get("task"), "verified"));
  assert.ok(!Object.hasOwn(result.get("task"), "ready"));
});

test("unloaded records are not invented and bounded maps disclose the loaded-only scope", () => {
  const model = {
    nodes: [...nodes(), item("placeholder", "objective", { contextMissing: true })],
    edges: [
      edge("absent", "task", "absent", "depends_on", "relation"),
      edge("known-ref", "task", "placeholder", "depends_on", "relation"),
      edge("absent-evidence", "missing-evidence", "task", "supports", "evidence_link"),
    ],
    partial: true,
    boundary: { counts: { relations: 99, evidence_links: 100 } },
  };
  const result = canvasNodeIndicators(model);
  assert.equal(result.has("absent"), false);
  assert.equal(result.get("task").dependencies.total, 1);
  assert.equal(result.get("task").evidence.total, 0);
  assert.equal(result.get("task").limited, true);
  assert.match(result.get("task").description, /Loaded relationships only; this map is partial or query-limited/);
  assert.equal(result.get("root").description, "");
  assert.equal(canvasNodeIndicators({ ...model, partial: false, boundary: { scopeLimited: true } }).get("task").limited, true);
});

test("relationship indexing leaves frozen server facts unchanged and iterates edges once", () => {
  const freeze = (value) => {
    if (value && typeof value === "object") {
      for (const entry of Object.values(value)) freeze(entry);
      Object.freeze(value);
    }
    return value;
  };
  const model = freeze({
    nodes: nodes(),
    edges: [
      edge("dep", "task", "other", "depends_on", "relation"),
      edge("block", "blocker", "task", "blocks", "blocker"),
      edge("proof", "evidence", "task", "supports", "evidence_link"),
    ],
  });
  const before = structuredClone(model);
  let traversals = 0;
  const result = canvasNodeIndicators({
    ...model,
    edges: {
      *[Symbol.iterator]() {
        traversals++;
        yield* model.edges;
      },
    },
  });
  assert.equal(traversals, 1);
  assert.deepEqual(model, before);
  assert.equal(result.get("task").label, "Loaded · 1 dep · 1 blocker · 1 evidence link");
});
