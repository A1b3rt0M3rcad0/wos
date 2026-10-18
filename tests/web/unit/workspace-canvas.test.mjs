import { test } from "node:test";
import assert from "node:assert/strict";
import {
  layoutCanvas,
  canvasNodeSize,
  canvasVisibleEdges,
  canvasEdgeLabel,
  zoomCanvasCamera,
} from "../../../packages/wos-api/web/assets/workspace-canvas.js";

const item = (key, kind, parentKey) => ({ key, kind, title: key, parentKey });
const edge = (key, sourceKey, targetKey, kind, mode) => ({
  key,
  sourceKey,
  targetKey,
  kind,
  mode,
});
const fixture = () => ({
  rootKey: "outcome:root",
  nodes: [
    item("outcome:root", "outcome"),
    item("objective:a", "objective", "outcome:root"),
    item("objective:b", "objective", "objective:a"),
    item("work_item:one", "work_item", "objective:b"),
    item("work_item:two", "work_item", "outcome:root"),
    item("issue:problem", "issue"),
  ],
  edges: [
    edge("hierarchy:a", "outcome:root", "objective:a", "hierarchy"),
    edge("hierarchy:b", "objective:a", "objective:b", "hierarchy"),
    edge("work:one", "objective:b", "work_item:one", "ownership"),
    edge("dependency", "work_item:two", "work_item:one", "depends_on", "hard"),
    edge("issue", "issue:problem", "work_item:one", "relates_to"),
  ],
});

function overlaps(a, b) {
  return (
    a.x < b.x + b.width &&
    a.x + a.width > b.x &&
    a.y < b.y + b.height &&
    a.y + a.height > b.y
  );
}

test("layout follows real parents, keeps disconnected records and ignores dependencies as ownership", () => {
  const model = fixture(),
    layout = layoutCanvas(model);
  assert.equal(layout.nodes.length, model.nodes.length);
  assert.equal(layout.parents.get("work_item:one"), "objective:b");
  assert.equal(layout.parents.has("issue:problem"), false);
  assert.equal(layout.parents.get("work_item:two"), "outcome:root");
  assert.ok(
    layout.byKey.get("work_item:one").x > layout.byKey.get("objective:b").x,
  );
  assert.equal(layout.edges.find((e) => e.key === "dependency").mode, "hard");
  assert.deepEqual(
    layoutCanvas({ ...model, nodes: [...model.nodes].reverse() }).nodes,
    layout.nodes,
  );
  for (let i = 0; i < layout.nodes.length; i++)
    for (let j = i + 1; j < layout.nodes.length; j++)
      assert.equal(
        overlaps(layout.nodes[i], layout.nodes[j]),
        false,
        `${layout.nodes[i].key}/${layout.nodes[j].key}`,
      );
});

test("collapse hides structural descendants without hiding dependency peers or inventing edges", () => {
  const model = fixture(),
    layout = layoutCanvas(model, { collapsed: new Set(["objective:a"]) });
  assert.equal(layout.hiddenCount, 2);
  assert.ok(!layout.byKey.has("objective:b"));
  assert.ok(!layout.byKey.has("work_item:one"));
  assert.ok(layout.byKey.has("work_item:two"));
  assert.ok(layout.byKey.has("issue:problem"));
  assert.equal(layout.byKey.get("objective:a").descendantCount, 2);
  assert.ok(layout.edges.every((e) => model.edges.includes(e)));
});

test("new data avoids existing positions and leaves the input model untouched", () => {
  const model = fixture(),
    before = structuredClone(model),
    initial = layoutCanvas(model);
  const positions = new Map(
    initial.nodes.map((n) => [n.key, { x: n.x, y: n.y }]),
  );
  const extended = {
    ...model,
    nodes: [...model.nodes, item("work_item:new", "work_item", "objective:b")],
  };
  const updated = layoutCanvas(extended, { positions });
  for (const previous of initial.nodes) {
    const next = updated.byKey.get(previous.key);
    assert.deepEqual({ x: next.x, y: next.y }, positions.get(previous.key));
    assert.equal(overlaps(next, updated.byKey.get("work_item:new")), false);
  }
  assert.deepEqual(model, before);
});

test("local arrangements keep signed coordinates and bounds follow cards in every direction", () => {
  const model = fixture(),
    before = structuredClone(model),
    initial = layoutCanvas(model);
  const translate = (x, y) =>
    new Map(initial.nodes.map((entry) => [
      entry.key, { x: entry.x + x, y: entry.y + y },
    ]));
  for (const [dx, dy] of [[-5000, -4000], [5000, 4000], [-5000, 4000]]) {
    const positions = translate(dx, dy),
      moved = layoutCanvas(model, { positions });
    for (const entry of moved.nodes) {
      assert.deepEqual({ x: entry.x, y: entry.y }, positions.get(entry.key));
      assert.ok(entry.x >= moved.bounds.x);
      assert.ok(entry.y >= moved.bounds.y);
      assert.ok(entry.x + entry.width <= moved.bounds.x + moved.bounds.width);
      assert.ok(entry.y + entry.height <= moved.bounds.y + moved.bounds.height);
    }
    assert.equal(moved.bounds.x, initial.bounds.x + dx);
    assert.equal(moved.bounds.y, initial.bounds.y + dy);
    assert.ok(Math.abs(moved.width - initial.width) < 1e-9);
    assert.ok(Math.abs(moved.height - initial.height) < 1e-9);
    // Refreshes and newly loaded records must retain an intentional local
    // position, rather than snapping a card back to a fabricated origin wall.
    const refreshed = layoutCanvas({
      ...model,
      nodes: [...model.nodes, item("work_item:later", "work_item", "objective:a")],
    }, { positions });
    for (const entry of moved.nodes) {
      const next = refreshed.byKey.get(entry.key);
      assert.deepEqual({ x: next.x, y: next.y }, positions.get(entry.key));
    }
    assert.deepEqual(moved.edges, initial.edges);
  }
  assert.deepEqual(model, before);
});

test("invalid local coordinates fall back to a finite layout without changing source data", () => {
  const model = fixture();
  const layout = layoutCanvas(model, { positions: new Map([
    ["outcome:root", { x: Infinity, y: 0 }],
    ["objective:a", { x: 0, y: NaN }],
  ]) });
  assert.ok(Object.values(layout.bounds).every(Number.isFinite));
  assert.ok(layout.nodes.every((entry) => Number.isFinite(entry.x) && Number.isFinite(entry.y)));
});

test("hundreds of mixed paper cards fit a deterministic non-overlapping layout", () => {
  const nodes = [item("root", "outcome")];
  for (let group = 0; group < 12; group++) {
    nodes.push(item(`objective:${group}`, "objective", "root"));
    for (let n = 0; n < 24; n++)
      nodes.push(item(`task:${group}:${n}`, "work_item", `objective:${group}`));
  }
  const layout = layoutCanvas({ rootKey: "root", nodes, edges: [] });
  assert.equal(layout.nodes.length, 301);
  assert.ok(
    layout.nodes.every(
      (n) =>
        n.x >= 0 &&
        n.y >= 0 &&
        n.x + n.width <= layout.width &&
        n.y + n.height <= layout.height,
    ),
  );
  for (let i = 0; i < layout.nodes.length; i++)
    for (let j = i + 1; j < layout.nodes.length; j++)
      assert.equal(overlaps(layout.nodes[i], layout.nodes[j]), false);
  assert.equal(
    canvasNodeSize("objective").width,
    canvasNodeSize("objective").height,
  );
  assert.ok(
    Math.abs(
      canvasNodeSize("work_item").width / canvasNodeSize("work_item").height -
        1000 / 730,
    ) < 0.0001,
  );
});

test("planning order and reference edges never change operational parents", () => {
  const model = fixture();
  model.nodes.push(
    item("phase", "phase"),
    item("milestone", "milestone", "phase"),
  );
  model.edges.push(
    edge("plan-ref", "phase", "work_item:two", "plan_reference"),
    edge("plan-seq", "work_item:two", "work_item:one", "plan_sequence"),
  );
  const layout = layoutCanvas(model);
  assert.equal(layout.parents.get("work_item:two"), "outcome:root");
  assert.equal(layout.parents.get("work_item:one"), "objective:b");
  assert.equal(layout.parents.get("milestone"), "phase");
  assert.ok(layout.edges.some((e) => e.kind === "plan_sequence"));
});

test("malformed visual cycles terminate without mutating real edges or keys", () => {
  const nodes = [item("a", "objective", "b"), item("b", "objective", "a")];
  const edges = [edge("real", "a", "b", "hierarchy")];
  const layout = layoutCanvas({ nodes, edges });
  assert.equal(layout.nodes.length, 2);
  assert.deepEqual(layout.edges, edges);
  assert.throws(
    () => layoutCanvas({ nodes: [nodes[0], nodes[0]], edges }),
    /duplicate/,
  );
});

test("connection filters preserve selected documentary context and distinct unknown dependency strength", () => {
  const edges = [
    edge("structure", "a", "b", "hierarchy"),
    edge("dependency", "b", "c", "depends_on", "advisory"),
    edge("proof", "e", "c", "supports"),
    edge("plan", "p", "c", "plan_sequence"),
  ];
  assert.deepEqual(
    canvasVisibleEdges(edges, null).map((e) => e.key),
    ["structure"],
  );
  assert.deepEqual(
    canvasVisibleEdges(edges, "e").map((e) => e.key),
    ["structure", "proof"],
  );
  assert.equal(canvasVisibleEdges(edges, null, "all").length, 4);
  assert.deepEqual(
    canvasVisibleEdges(edges, null, "planning").map((e) => e.key),
    ["plan"],
  );
  assert.match(
    canvasEdgeLabel(edge("unknown", "a", "b", "depends_on")),
    /strength not loaded/,
  );
  assert.match(canvasEdgeLabel(edges[1]), /advisory/);
  assert.match(canvasEdgeLabel(edges[3]), /Plan order/);
});

test("camera zoom preserves the anchored point and clamps independently of layout and responsive scale", () => {
  const camera = { x: 40, y: -20, zoom: 1 },
    anchor = { x: 300, y: 200 };
  const zoomed = zoomCanvasCamera(camera, 1.5, anchor);
  assert.equal(
    (anchor.x - zoomed.x) / zoomed.zoom,
    (anchor.x - camera.x) / camera.zoom,
  );
  assert.equal(
    (anchor.y - zoomed.y) / zoomed.zoom,
    (anchor.y - camera.y) / camera.zoom,
  );
  assert.equal(zoomCanvasCamera(camera, 100, anchor).zoom, 2.5);
  assert.equal(zoomCanvasCamera(camera, 0.001, anchor).zoom, 0.25);
  assert.deepEqual(camera, { x: 40, y: -20, zoom: 1 });
});
