import { test } from "node:test";
import assert from "node:assert/strict";
import {
  canvasKey,
  canvasRevision,
  createCanvasData,
} from "../../../packages/wos-api/web/assets/canvas-data.js";

const scope = {
  namespace_id: "01a11992-0000-7000-8000-000000000001",
  outcome_id: "01a11992-0000-7000-8000-000000000002",
};
const id = (n) => `01a11992-0000-7000-8000-${String(n).padStart(12, "0")}`;
const ref = (kind, n) => ({
  ...scope,
  kind,
  id: kind === "outcome" ? scope.outcome_id : id(n),
});
const summary = (kind, n, extra = {}) => ({
  ref: ref(kind, n),
  title: `${kind} ${n}`,
  version: 1,
  lifecycle: kind === "work_item" ? "todo" : "active",
  ...extra,
});
const root = summary("outcome", 2),
  objective = summary("objective", 3),
  work = summary("work_item", 4);
const structure = () => ({
  outcome_revision: 42,
  nodes: [root, objective, work],
  edges: [
    {
      id: `objective:${objective.ref.id}`,
      source_ref: root.ref,
      target_ref: objective.ref,
      kind: "hierarchy",
      source_of_truth: "objective",
    },
    {
      id: `work:${work.ref.id}`,
      source_ref: objective.ref,
      target_ref: work.ref,
      kind: "ownership",
      source_of_truth: "work_item",
    },
  ],
  evaluated_at: "2026-10-16T12:00:00Z",
  depth: 3,
  truncated: false,
});
const snapshot = (extra = {}) => ({
  outcome: root,
  outcome_revision: 42,
  evaluated_at: "2026-10-16T12:00:00Z",
  consistency: "transactional",
  sections: {
    objectives: [objective],
    work_items: [work],
    relations: [],
    evidence_links: [],
  },
  counts: { objectives: 1, work_items: 1, relations: 0, evidence_links: 0 },
  omitted: {},
  section_cursors: {},
  ...extra,
});
const deferred = () => {
  let resolve, reject;
  const promise = new Promise((a, b) => {
    resolve = a;
    reject = b;
  });
  return { promise, resolve, reject };
};

test("the adapter preserves actual hierarchy and dependency strengths without inventing ownership", async () => {
  const other = summary("work_item", 5),
    data = snapshot();
  data.sections.work_items.push(other);
  data.counts.work_items = 2;
  data.sections.relations = [
    {
      id: id(20),
      scope,
      source_ref: work.ref,
      target_ref: other.ref,
      lifecycle: "active",
      relation_type: "depends_on",
      strength: "advisory",
      version: 9,
    },
  ];
  data.counts.relations = 1;
  const source = structure();
  const loader = createCanvasData({ scope, request: async () => source });
  const model = await loader.load(data);
  assert.equal(
    model.nodes.find((n) => n.key === canvasKey(objective.ref)).parentKey,
    canvasKey(root.ref),
  );
  assert.equal(
    model.nodes.find((n) => n.key === canvasKey(work.ref)).parentKey,
    canvasKey(objective.ref),
  );
  assert.equal(
    model.nodes.find((n) => n.key === canvasKey(other.ref)).parentKey,
    undefined,
  );
  const dependency = model.edges.find((edge) => edge.kind === "depends_on");
  assert.equal(dependency.mode, "advisory");
  assert.equal(dependency.ref.kind, "relation");
  assert.equal(dependency.version, 9);
  assert.ok(
    model.partial,
    "an unlocated loaded Task must not look structurally complete",
  );
  assert.ok(Object.isFrozen(model.nodes[0].ref));
  assert.equal(
    model.nodes[0].canCreate,
    undefined,
    "presentation data must not invent permissions",
  );
  source.nodes[0].title = "changed after response";
  assert.notEqual(model.nodes[0].title, source.nodes[0].title);
});

test("unknown dependency strength and evidence stance remain explicit, separate from assessment", async () => {
  const source = structure(),
    evidence = summary("evidence", 6),
    data = snapshot();
  source.edges.push({
    id: id(21),
    source_ref: work.ref,
    target_ref: objective.ref,
    kind: "depends_on",
    source_of_truth: "relation",
  });
  data.sections.evidence = [evidence];
  data.sections.evidence_links = [
    {
      id: id(22),
      scope,
      lifecycle: "active",
      evidence_id: evidence.ref.id,
      target_ref: work.ref,
      stance: "contradicts",
      criterion_id: id(23),
      version: 2,
    },
  ];
  const loader = createCanvasData({ scope, request: async () => source });
  const model = await loader.load(data);
  assert.equal(
    model.edges.find((e) => e.kind === "depends_on").mode,
    "unknown",
  );
  const link = model.edges.find((e) => e.sourceOfTruth === "evidence_link");
  assert.equal(link.kind, "contradicts");
  assert.equal(link.criterionId, id(23));
  assert.ok(!model.nodes.some((n) => n.lifecycle === "met"));
});

test("the initial bounded traversal includes actual Blocker impact and cause without inferring an Issue impact", async () => {
  const data = snapshot(),
    source = structure(),
    calls = [];
  const blocker = summary("blocker", 24),
    issue = summary("issue", 25);
  data.sections.blockers = [blocker];
  data.sections.issues = [issue];
  source.nodes.push(blocker, issue);
  const impacts = [
    {
      id: `blocker:${blocker.ref.id}:blocks`,
      source_ref: blocker.ref,
      target_ref: work.ref,
      kind: "blocks",
      source_of_truth: "blocker",
    },
    {
      id: `blocker:${blocker.ref.id}:causes`,
      source_ref: issue.ref,
      target_ref: blocker.ref,
      kind: "causes",
      source_of_truth: "blocker",
    },
  ];
  const loader = createCanvasData({
    scope,
    request: async (path) => {
      calls.push(path);
      const params = new URL(path, "http://canvas.test").searchParams;
      const kinds = new Set(params.get("kinds").split(","));
      assert.equal(params.get("depth"), "3");
      assert.equal(params.get("limit"), "100");
      return {
        ...source,
        edges: [
          ...source.edges,
          ...impacts.filter((edge) => kinds.has(edge.kind)),
        ],
      };
    },
  });
  const model = await loader.load(data);
  assert.equal(
    calls.length,
    1,
    "impact loading shares the existing graph request",
  );
  assert.deepEqual(
    model.edges
      .filter((edge) => ["blocks", "causes"].includes(edge.kind))
      .map((edge) => ({
        kind: edge.kind,
        truth: edge.sourceOfTruth,
        from: edge.sourceKey,
        to: edge.targetKey,
      })),
    [
      {
        kind: "blocks",
        truth: "blocker",
        from: canvasKey(blocker.ref),
        to: canvasKey(work.ref),
      },
      {
        kind: "causes",
        truth: "blocker",
        from: canvasKey(issue.ref),
        to: canvasKey(blocker.ref),
      },
    ],
  );
  assert.equal(
    model.nodes.find((node) => node.key === canvasKey(issue.ref)).parentKey,
    undefined,
  );
  assert.equal(
    model.nodes.find((node) => node.key === canvasKey(work.ref)).parentKey,
    canvasKey(objective.ref),
  );
});

test("one gesture loads one bounded page and alternates graph and disconnected metadata sources", async () => {
  const urls = [],
    data = snapshot(),
    extra = summary("objective", 9);
  data.section_cursors.objectives = "more-objectives";
  data.counts.objectives = 2;
  data.omitted.objectives = 1;
  const first = structure();
  first.next_cursor = "graph-2";
  first.truncated = true;
  const loader = createCanvasData({
    scope,
    limit: 100,
    request: async (path, options) => {
      urls.push(path);
      assert.equal(options.cache, "no-store");
      if (path.includes("/continuity/objectives"))
        return { items: [extra], count: 2, omitted: 0, outcome_revision: "42" };
      if (new URL(path, "http://local").searchParams.has("cursor"))
        return { ...structure(), next_cursor: "graph-3", truncated: true };
      return first;
    },
  });
  await loader.load(data);
  assert.equal(urls.length, 1);
  await loader.more();
  assert.equal(urls.length, 2);
  const model = await loader.more();
  assert.equal(urls.length, 3);
  assert.ok(urls[1].includes("/graph?"));
  assert.ok(urls[2].includes("/continuity/objectives?"));
  assert.ok(model.nodes.some((n) => n.key === canvasKey(extra.ref)));
  assert.equal(model.boundary.omitted.objectives, 0);
  assert.equal(model.boundary.scopeLimited, true);
  assert.equal(model.boundary.depth, 3);
  assert.ok(model.hasMore);
});

test("new revisions and foreign scopes cannot silently mix into a loaded model", async () => {
  let next = structure();
  next.next_cursor = "page-two";
  const loader = createCanvasData({ scope, request: async () => next });
  const original = await loader.load(snapshot());
  next = { ...structure(), outcome_revision: 43 };
  await assert.rejects(loader.more(), { code: "precondition_failed" });
  assert.strictEqual(loader.current(), original);
  next = structure();
  next.nodes[1] = {
    ...objective,
    ref: { ...objective.ref, outcome_id: id(99) },
  };
  await assert.rejects(loader.expand(work.ref), { code: "invalid_response" });
  assert.strictEqual(loader.current(), original);
  assert.throws(() => loader.expand({ ...work.ref, namespace_id: id(99) }), {
    code: "invalid_response",
  });
});

test("late reads and late errors from a disposed or replaced Outcome are ignored", async () => {
  const response = deferred();
  let current = true;
  const loader = createCanvasData({
    scope,
    isCurrent: () => current,
    request: () => response.promise,
  });
  const loading = loader.load(snapshot());
  current = false;
  response.reject(Error("obsolete network failure"));
  assert.equal(await loading, null);
  assert.equal(loader.current(), null);
  const later = deferred();
  const disposed = createCanvasData({ scope, request: () => later.promise });
  const pending = disposed.load(snapshot());
  disposed.dispose();
  later.resolve(structure());
  assert.equal(await pending, null);
  assert.equal(disposed.current(), null);
});

test("contract summary lease expiry is unknown until explicit authority inspection; legacy expiry remains visible", async () => {
  const task = summary("work_item", 4, { lifecycle: "in_progress" }),
    data = snapshot();
  data.sections.work_items = [task];
  data.sections.attention_needed_work = [
    {
      work_item: task,
      state: {
        display_state: "attention_needed",
        is_blocked: false,
        readiness_reasons: ["lease_expired"],
      },
    },
  ];
  const graph = structure();
  graph.nodes[2] = task;
  const signed = await createCanvasData({
    scope,
    protocolMode: "signed_contracts_v2",
    request: async () => graph,
  }).load(data);
  const legacy = await createCanvasData({
    scope,
    protocolMode: "legacy",
    request: async () => graph,
  }).load(data);
  assert.equal(
    signed.nodes.find((n) => n.kind === "work_item").displayState,
    "in_progress",
  );
  assert.equal(
    signed.nodes.find((n) => n.kind === "work_item").authorityUnknown,
    true,
  );
  assert.equal(
    legacy.nodes.find((n) => n.kind === "work_item").displayState,
    "attention_needed",
  );
  assert.equal(
    legacy.nodes.find((n) => n.kind === "work_item").authorityUnknown,
    undefined,
  );
});

test("published and draft planning are distinct overlays and never become execution dependencies", async () => {
  const plan = {
    revision_number: 2,
    nodes: [
      {
        node_key: "phase",
        node_type: "phase",
        title: "Preparation",
        position: 0,
      },
      {
        node_key: "reference",
        node_type: "reference",
        title: "Implement",
        parent_node_key: "phase",
        target_ref: work.ref,
        position: 1,
      },
      {
        node_key: "milestone",
        node_type: "milestone",
        title: "Verify",
        position: 2,
      },
    ],
    after_links: [{ node_key: "milestone", after_node_key: "reference" }],
  };
  let requests = 0;
  const loader = createCanvasData({
    scope,
    request: async (url) => {
      requests++;
      return url.includes("/revisions/")
        ? { value: plan, outcome_revision: 42 }
        : structure();
    },
  });
  await loader.load(snapshot());
  const published = await loader.roadmap({
    roadmap_id: id(30),
    revision_number: 2,
  });
  assert.equal(
    published.edges.filter((e) => e.kind === "plan_sequence").length,
    1,
  );
  assert.equal(
    published.edges.filter((e) => e.kind === "depends_on").length,
    0,
  );
  assert.equal(
    published.nodes.find((n) => n.kind === "work_item").parentKey,
    canvasKey(objective.ref),
  );
  const phase = published.nodes.find((n) => n.kind === "phase");
  assert.equal(phase.ref, null);
  assert.equal(phase.roadmapRef.kind, "roadmap");
  const draft = {
    ...plan,
    draft_version: 7,
    lifecycle: "open",
    nodes: [
      {
        node_key: "new",
        node_type: "phase",
        title: "New draft phase",
        position: 0,
      },
    ],
  };
  const drafted = await loader.roadmap({
    roadmap_id: id(30),
    draft,
    outcome_revision: "42",
  });
  assert.equal(
    requests,
    2,
    "Draft data already loaded in a consistent detail response needs no extra GET",
  );
  assert.ok(!drafted.nodes.some((n) => n.title === "Preparation"));
  assert.ok(
    drafted.nodes.some(
      (n) => n.title === "New draft phase" && n.lifecycle === "draft",
    ),
  );
  assert.equal(
    drafted.nodes.find((n) => n.kind === "work_item").parentKey,
    canvasKey(objective.ref),
  );
  await assert.rejects(
    loader.roadmap({ roadmap_id: id(30), draft, outcome_revision: "43" }),
    { code: "precondition_failed" },
  );
});

test("authorized search and expansion use bounded distinct queries without silently merging newer state", async () => {
  const calls = [],
    found = summary("work_item", 50);
  const loader = createCanvasData({
    scope,
    request: async (path) => {
      calls.push(path);
      if (path.includes("/references?"))
        return {
          items: [found],
          next_cursor: "search-next",
          outcome_revision: 42,
        };
      if (path.includes("root_id"))
        return { ...structure(), nodes: [found], edges: [], depth: 1 };
      return structure();
    },
  });
  await loader.load(snapshot());
  const result = await loader.search("independent task");
  assert.equal(result.items[0].key, canvasKey(found.ref));
  assert.equal(result.next_cursor, "search-next");
  assert.ok(
    !loader.current().nodes.some((n) => n.key === canvasKey(found.ref)),
    "search results do not pretend graph context was loaded",
  );
  const expanded = await loader.expand(found.ref);
  assert.ok(expanded.nodes.some((n) => n.key === canvasKey(found.ref)));
  assert.ok(calls.at(-1).includes("depth=1"));
  assert.ok(
    !calls.at(-1).includes("kinds="),
    "context expansion includes actual dependency/documentary edges",
  );
});

test("revision comparison preserves exact strings and rejects already-rounded numeric responses", () => {
  assert.equal(canvasRevision("18446744073709551615"), "18446744073709551615");
  assert.equal(canvasRevision(42), "42");
  for (const value of [
    9007199254740992,
    "18446744073709551616",
    "01",
    -1,
    undefined,
  ])
    assert.throws(() => canvasRevision(value));
});

test("the adapter reuses the configured local API prefix", async () => {
  let requested;
  const loader = createCanvasData({
    scope,
    apiPrefix: "/custom/v1/",
    request: async (path) => {
      requested = path;
      return structure();
    },
  });
  await loader.load(snapshot());
  assert.ok(requested.startsWith("/custom/v1/namespaces/"));
  assert.throws(() =>
    createCanvasData({ scope, apiPrefix: "//external.test", request() {} }),
  );
});

test("selected current context upgrades unknown edges and authority without inheriting stale detail", async () => {
  const source = structure();
  source.edges.push({
    id: id(80),
    source_ref: work.ref,
    target_ref: objective.ref,
    kind: "depends_on",
    source_of_truth: "relation",
  });
  const loader = createCanvasData({
    scope,
    protocolMode: "signed_contracts_v2",
    request: async () => source,
  });
  const initial = await loader.load(snapshot());
  assert.equal(
    initial.edges.find((e) => e.kind === "depends_on").mode,
    "unknown",
  );
  const context = {
    outcome_revision: "42",
    outcome: root,
    evaluated_at: "2026-10-16T12:00:00Z",
    work_item: {
      id: work.ref.id,
      scope,
      title: "Current selected Task",
      version: 5,
      lifecycle: "in_progress",
      objective_id: objective.ref.id,
    },
    objective: {
      id: objective.ref.id,
      scope,
      title: "Current objective",
      lifecycle: "active",
      version: 2,
    },
    dependencies: [
      {
        id: id(80),
        scope,
        source_ref: work.ref,
        target_ref: objective.ref,
        relation_type: "depends_on",
        strength: "hard",
        lifecycle: "active",
        version: 4,
      },
    ],
    evidence_links: [],
  };
  const attention = {
    outcome_revision: 42,
    work_item_id: work.ref.id,
    status: "awaiting_review",
    awaiting_review: true,
    operational_state: { ref: work.ref, display_state: "attention_needed" },
  };
  const updated = loader.context(context, { attention });
  assert.equal(updated.edges.find((e) => e.kind === "depends_on").mode, "hard");
  const selected = updated.nodes.find((n) => n.kind === "work_item");
  assert.equal(selected.displayState, "awaiting_review");
  assert.equal(selected.authorityKnown, true);
  assert.equal(selected.version, 5);
  assert.equal(selected.parentKey, canvasKey(objective.ref));
  assert.equal(
    selected.sourceSummary.criteria,
    undefined,
    "canvas metadata never retains full proof/history bodies",
  );
  assert.throws(() => loader.context({ ...context, outcome_revision: "43" }), {
    code: "precondition_failed",
  });
  assert.strictEqual(loader.current(), updated);
  assert.throws(
    () =>
      loader.context(context, {
        attention: { ...attention, work_item_id: id(90) },
      }),
    { code: "invalid_response" },
  );
  assert.strictEqual(loader.current(), updated);
});

test(
  "a replacement snapshot is not held behind obsolete queued network work",
  { timeout: 1000 },
  async () => {
    const old = deferred();
    let calls = 0;
    const loader = createCanvasData({
      scope,
      request: async () => {
        calls++;
        if (calls === 2) return old.promise;
        return {
          ...structure(),
          outcome_revision: calls >= 3 ? 43 : 42,
          next_cursor: calls === 4 ? "" : "more",
        };
      },
    });
    await loader.load(snapshot());
    const obsolete = loader.more();
    await Promise.resolve();
    await loader.load(snapshot({ outcome_revision: 43 }));
    const latest = await loader.more();
    assert.equal(latest.revision, "43");
    old.reject(Error("obsolete queued failure"));
    assert.equal(await obsolete, null);
  },
);
