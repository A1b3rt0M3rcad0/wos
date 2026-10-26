// Presentation data only. Edges retain their actual source of truth; position,
// hierarchy, planning order and execution authority never substitute for it.
// Compact Blocker summaries omit their endpoints. Include the actual impact
// edges in the same bounded traversal rather than guessing them from Issues.
const initialKinds = "hierarchy,ownership,blocks,causes";
const supportedKinds = new Set([
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
const mapSections = [
  "objectives",
  "work_items",
  "relations",
  "evidence_links",
  "issues",
  "blockers",
  "evidence",
  "artifacts",
  "decisions",
  "active_roadmaps",
  "active_plan_references",
  "ready_work",
  "in_progress_work",
  "blocked_work",
  "scheduled_work",
  "attention_needed_work",
  "waiting_dependencies_work",
  "waiting_scope_work",
  "backlog_work",
  "done_work",
  "cancelled_work",
];
const kindLabels = {
  outcome: "Outcome",
  objective: "Objective",
  work_item: "Task",
  issue: "Issue",
  blocker: "Blocker",
  evidence: "Evidence",
  artifact: "Artifact",
  decision: "Decision",
};
const compare = (a, b) => (a < b ? -1 : a > b ? 1 : 0);
const clone = (value) => structuredClone(value);

export const canvasKey = (ref) => `${ref.kind}:${ref.id}`;

function failure(message, code = "invalid_response") {
  const error = new Error(message);
  error.code = code;
  return error;
}

export function canvasRevision(value) {
  if (typeof value === "number" && Number.isSafeInteger(value) && value >= 0)
    return String(value);
  if (
    typeof value === "string" &&
    /^(0|[1-9][0-9]*)$/.test(value) &&
    BigInt(value) <= 18446744073709551615n
  )
    return value;
  throw failure("The map response does not contain an exact Outcome revision.");
}

function deepFreeze(value) {
  if (value && typeof value === "object" && !Object.isFrozen(value)) {
    for (const child of Object.values(value)) deepFreeze(child);
    Object.freeze(value);
  }
  return value;
}

function checkedRef(ref, scope) {
  if (
    !ref ||
    !supportedKinds.has(ref.kind) ||
    typeof ref.id !== "string" ||
    !ref.id ||
    ref.namespace_id !== scope.namespace_id ||
    ref.outcome_id !== scope.outcome_id
  ) {
    throw failure(
      "The map response contains a reference outside this Outcome.",
    );
  }
  return ref;
}

function checkScope(scope) {
  if (
    !scope ||
    typeof scope.namespace_id !== "string" ||
    !scope.namespace_id ||
    typeof scope.outcome_id !== "string" ||
    !scope.outcome_id
  ) {
    throw failure("Choose a Workspace and Outcome before loading the map.");
  }
}

function assertRevision(response, revision) {
  if (canvasRevision(response?.outcome_revision) !== revision) {
    throw failure(
      "The Outcome changed while loading the map. Refresh to view the current state.",
      "precondition_failed",
    );
  }
}

function itemIdentity(item) {
  if (item?.ref) return canvasKey(item.ref);
  if (item?.work_item?.ref) return canvasKey(item.work_item.ref);
  if (item?.id) return item.id;
  if (item?.slot) return `${item.slot.roadmap_id}:${item.slot.revision_number}`;
  if (item?.roadmap_id) return `${item.roadmap_id}:${item.node_key}`;
  return JSON.stringify(item);
}

function mergeItems(previous, incoming) {
  const items = new Map(previous.map((item) => [itemIdentity(item), item]));
  for (const item of incoming) items.set(itemIdentity(item), clone(item));
  return [...items.values()];
}

function buildModel(
  scope,
  snapshot,
  sections,
  graphs,
  plans,
  protocolMode,
  contexts = new Map(),
  cardSummaries = new Map(),
) {
  const nodes = new Map(),
    edges = new Map(),
    operational = new Map();
  const rootRef = checkedRef(snapshot.outcome.ref, scope),
    revision = canvasRevision(snapshot.outcome_revision);
  const rootKey = canvasKey(rootRef);
  const addNode = (summary, extra = {}) => {
    const ref = checkedRef(summary.ref, scope),
      key = canvasKey(ref),
      previous = nodes.get(key);
    const title =
      summary.title || previous?.title || kindLabels[ref.kind] || ref.kind;
    nodes.set(key, {
      ...previous,
      key,
      ref: clone(ref),
      kind: ref.kind,
      title,
      lifecycle: summary.lifecycle || previous?.lifecycle || "",
      priority: summary.priority || previous?.priority || "",
      version: summary.version ?? previous?.version,
      sourceSummary: { ...(previous?.sourceSummary || {}), ...clone(summary) },
      contextMissing: false,
      expandable: ["outcome", "objective", "work_item"].includes(ref.kind),
      ...extra,
    });
    return key;
  };
  const ensureNode = (ref) => {
    checkedRef(ref, scope);
    const key = canvasKey(ref);
    if (!nodes.has(key))
      addNode(
        {
          ref,
          title: `${kindLabels[ref.kind] || ref.kind} · details not loaded`,
          detail_omitted: true,
        },
        { contextMissing: true },
      );
    return key;
  };
  const addEdge = (key, source, target, kind, truth, extra = {}) => {
    const sourceKey = ensureNode(source),
      targetKey = ensureNode(target);
    edges.set(key, {
      key,
      sourceKey,
      targetKey,
      kind,
      sourceOfTruth: truth,
      ...extra,
    });
  };
  addNode(snapshot.outcome);
  for (const section of sections.values())
    for (const item of section.items) {
      if (item?.ref) addNode(item);
      if (item?.work_item?.ref) {
        addNode(item.work_item);
        if (item.state)
          operational.set(canvasKey(item.work_item.ref), item.state);
      }
      if (item?.current?.ref) addNode(item.current);
    }
  for (const graph of graphs.values()) {
    for (const summary of graph.page.nodes || []) addNode(summary);
    for (const edge of graph.page.edges || []) {
      // Legacy/direct Task edges do not establish a valid Outcome placement.
      // Tasks are owned by Objectives; keep the Task as an unplaced record,
      // but never project the Outcome as its parent.
      if (
        edge.kind === "ownership" &&
        edge.source_of_truth === "work_item" &&
        edge.source_ref?.kind === "outcome" &&
        edge.target_ref?.kind === "work_item"
      ) continue;
      addEdge(
        `graph:${edge.source_of_truth}:${edge.id}`,
        edge.source_ref,
        edge.target_ref,
        edge.kind,
        edge.source_of_truth,
        {
          mode: edge.kind === "depends_on" ? "unknown" : undefined,
          ref: ["relation", "evidence_link"].includes(edge.source_of_truth)
            ? { ...scope, kind: edge.source_of_truth, id: edge.id }
            : undefined,
        },
      );
    }
  }
  for (const context of contexts.values()) {
    if (context.workSummary) addNode(context.workSummary);
    if (context.objectiveSummary) addNode(context.objectiveSummary);
  }
  for (const relation of sections.get("relations")?.items || []) {
    if (relation.lifecycle !== "active") continue;
    if (
      relation.scope?.namespace_id !== scope.namespace_id ||
      relation.scope?.outcome_id !== scope.outcome_id
    )
      throw failure("Relation data belongs to another Outcome.");
    addEdge(
      `graph:relation:${relation.id}`,
      relation.source_ref,
      relation.target_ref,
      relation.relation_type,
      "relation",
      {
        mode:
          relation.relation_type === "depends_on"
            ? relation.strength || "unknown"
            : undefined,
        version: relation.version,
        label:
          relation.relation_type === "depends_on"
            ? `${relation.strength || "Unclassified"} dependency`
            : relation.relation_type.replaceAll("_", " "),
        ref: { ...scope, kind: "relation", id: relation.id },
      },
    );
  }
  for (const link of sections.get("evidence_links")?.items || []) {
    if (link.lifecycle !== "active") continue;
    if (
      link.scope?.namespace_id !== scope.namespace_id ||
      link.scope?.outcome_id !== scope.outcome_id
    )
      throw failure("Evidence link data belongs to another Outcome.");
    addEdge(
      `graph:evidence_link:${link.id}`,
      { ...scope, kind: "evidence", id: link.evidence_id },
      link.target_ref,
      link.stance,
      "evidence_link",
      {
        version: link.version,
        label: link.stance,
        criterionId: link.criterion_id,
        ref: { ...scope, kind: "evidence_link", id: link.id },
      },
    );
  }
  for (const [key, context] of contexts)
    for (const parent of context.parents || []) {
      addEdge(
        `graph:${parent.truth}:${parent.id}`,
        parent.source,
        parent.target,
        parent.kind,
        parent.truth,
      );
    }
  // Only persisted hierarchy/association establishes operational parents.
  for (const edge of edges.values())
    if (
      (edge.kind === "hierarchy" && edge.sourceOfTruth === "objective") ||
      (edge.kind === "ownership" && edge.sourceOfTruth === "work_item")
    )
      nodes.get(edge.targetKey).parentKey = edge.sourceKey;

  for (const [key, state] of operational) {
    const node = nodes.get(key);
    if (!node) continue;
    node.reportedOperationalState = clone(state);
    const contractSummary = [
      "signed_contracts_v2",
      "contracts_v1",
      "draining_to_signed_v2",
    ].includes(protocolMode);
    if (
      contractSummary &&
      node.lifecycle === "in_progress" &&
      state.readiness_reasons?.includes("lease_expired")
    ) {
      node.displayState = state.is_blocked ? "blocked" : node.lifecycle;
      node.authorityUnknown = true;
      node.summary = "Open details to inspect current contract authority.";
    } else node.displayState = state.display_state;
  }
  for (const [key, context] of contexts) {
    const node = nodes.get(key);
    if (!node) continue;
    if (context.attention) {
      const attention = context.attention;
      node.reportedOperationalState = clone(attention.operational_state);
      node.displayState =
        attention.status === "awaiting_review"
          ? "awaiting_review"
          : attention.status === "corrections_requested"
            ? "changes_requested"
            : attention.operational_state.display_state;
      node.authorityUnknown = false;
      node.authorityKnown = true;
      node.awaitingReview = Boolean(attention.awaiting_review);
      node.summary = node.awaitingReview
        ? "Awaiting independent review."
        : undefined;
    }
  }

  // Card facts are a bounded server projection from this exact Outcome
  // revision. They never hydrate inspectors or infer authority from assignees.
  for (const [key, summary] of cardSummaries) {
    const entry = nodes.get(key);
    if (!entry) continue;
    entry.cardSummary = clone(summary);
    entry.title = summary.title || entry.title;
    entry.lifecycle = summary.lifecycle || entry.lifecycle;
    if (summary.display_state && (!entry.reportedOperationalState ||
        ["awaiting_review", "corrections_requested"].includes(summary.display_state)))
      entry.displayState = summary.display_state;
    if (!entry.authorityKnown)
      entry.authorityUnknown = summary.authority_state === "unknown";
  }

  const planHeaders = [];
  for (const header of sections.get("active_roadmaps")?.items || []) {
    const slot = header.slot;
    if (!slot?.roadmap_id) continue;
    planHeaders.push(clone(header));
  }
  for (const reference of sections.get("active_plan_references")?.items || []) {
    ensureNode(reference.target_ref);
  }
  for (const loaded of plans.values()) {
    const plan = loaded.value,
      id = loaded.roadmap_id,
      number = loaded.revision_number;
    const truth = loaded.draft ? "roadmap_draft" : "roadmap_revision";
    const planLabel = loaded.draft
      ? `Draft plan · version ${plan.draft_version}`
      : `Published plan · revision ${number}`;
    const planKey = (key) =>
      `plan:${id}:${loaded.draft ? `draft:${plan.draft_version}` : number}:${key}`;
    const known = new Set((plan.nodes || []).map((node) => node.node_key));
    for (const node of plan.nodes || []) {
      const key = planKey(node.node_key),
        parentKey = known.has(node.parent_node_key)
          ? planKey(node.parent_node_key)
          : undefined;
      nodes.set(key, {
        key,
        ref: null,
        kind:
          node.node_type === "reference" ? "plan_reference" : node.node_type,
        title: node.title,
        lifecycle: loaded.draft ? "draft" : "published",
        parentKey,
        roadmapRef: { ...scope, kind: "roadmap", id },
        revisionNumber: number,
        planSummary: {
          roadmap_id: id,
          revision_number: number,
          draft_version: loaded.draft ? plan.draft_version : undefined,
          node_key: node.node_key,
          node_type: node.node_type,
          title: node.title,
          source_of_truth: truth,
        },
        sourceSummary: clone(node),
        expandable: false,
        summary: planLabel,
      });
      if (parentKey)
        edges.set(`plan-parent:${key}`, {
          key: `plan-parent:${key}`,
          sourceKey: parentKey,
          targetKey: key,
          kind: "plan_hierarchy",
          sourceOfTruth: truth,
          label: "Plan grouping",
        });
      if (node.target_ref) {
        const targetKey = ensureNode(node.target_ref);
        edges.set(`plan-reference:${key}`, {
          key: `plan-reference:${key}`,
          sourceKey: key,
          targetKey,
          kind: "plan_reference",
          sourceOfTruth: truth,
          label: "Plan reference",
        });
      }
    }
    for (const link of plan.after_links || [])
      if (known.has(link.node_key) && known.has(link.after_node_key)) {
        const key = `plan-sequence:${id}:${number}:${link.after_node_key}:${link.node_key}`;
        edges.set(key, {
          key,
          sourceKey: planKey(link.after_node_key),
          targetKey: planKey(link.node_key),
          kind: "plan_sequence",
          sourceOfTruth: truth,
          label: "Planned after · not an execution dependency",
        });
      }
  }

  const cursors = {},
    counts = {},
    omitted = {},
    loadedCounts = {};
  for (const [name, section] of sections) {
    counts[name] = section.count;
    omitted[name] = Math.max(
      0,
      section.count - section.items.length,
      section.omitted || 0,
    );
    if (section.cursor) cursors[`section:${name}`] = section.cursor;
  }
  for (const [key, graph] of graphs)
    if (graph.page.next_cursor)
      cursors[`graph:${key}`] = graph.page.next_cursor;
  for (const node of nodes.values())
    if (node.ref && !node.contextMissing)
      loadedCounts[node.kind] = (loadedCounts[node.kind] || 0) + 1;
  const missingParents = [...nodes.values()].filter(
    (node) => ["objective", "work_item"].includes(node.kind) && !node.parentKey,
  ).length;
  const partial =
    Object.keys(cursors).length > 0 ||
    Object.values(omitted).some((value) => value > 0) ||
    missingParents > 0 ||
    [...nodes.values()].some((node) => node.contextMissing);
  return deepFreeze({
    scopeKey: `${scope.namespace_id}/${scope.outcome_id}`,
    rootKey,
    revision,
    nodes: [...nodes.values()].sort((a, b) =>
      a.key === rootKey ? -1 : b.key === rootKey ? 1 : compare(a.key, b.key),
    ),
    edges: [...edges.values()].sort((a, b) => compare(a.key, b.key)),
    partial,
    hasMore: Object.keys(cursors).length > 0,
    plans: planHeaders,
    boundary: {
      depth: 3,
      scopeLimited: true,
      loadedCounts,
      counts,
      omitted,
      cursors,
      missingParents,
    },
    sources: {
      snapshot: {
        outcomeRevision: revision,
        evaluatedAt: snapshot.evaluated_at,
        consistency: snapshot.consistency,
      },
      graphs: [...graphs.values()].map((graph) => ({
        query: clone(graph.query),
        outcomeRevision: canvasRevision(graph.page.outcome_revision),
        evaluatedAt: graph.page.evaluated_at,
        truncated: Boolean(graph.page.truncated),
        nextCursor: graph.page.next_cursor || "",
      })),
      contexts: [...contexts].map(([key, context]) => ({
        key,
        outcomeRevision: revision,
        evaluatedAt: context.evaluatedAt,
      })),
    },
  });
}

export function createCanvasData({
  request,
  scope,
  isCurrent = () => true,
  limit = 100,
  protocolMode = "legacy",
  apiPrefix = "/api/v1",
}) {
  checkScope(scope);
  if (
    typeof request !== "function" ||
    !Number.isInteger(limit) ||
    limit < 1 ||
    limit > 100
  )
    throw failure(
      "A map request function and page limit between 1 and 100 are required.",
    );
  if (
    typeof apiPrefix !== "string" ||
    !/^\/(?!\/)[^?#\\]*$/.test(apiPrefix) ||
    apiPrefix.split("/").includes("..")
  )
    throw failure("The map API prefix must be a local path.");
  scope = clone(scope);
  const base = `${apiPrefix.replace(/\/+$/, "")}/namespaces/${encodeURIComponent(scope.namespace_id)}/outcomes/${encodeURIComponent(scope.outcome_id)}`;
  let epoch = 0,
    disposed = false,
    snapshot = null,
    sections = new Map(),
    graphs = new Map(),
    plans = new Map(),
    contexts = new Map(),
    cardSummaries = new Map(),
    model = null,
    pending = Promise.resolve(),
    lastMoreWasGraph = false;
  const active = (token) => !disposed && token === epoch && isCurrent();
  const fetchCurrent = async (path, token) => {
    try {
      const value = await request(path, { cache: "no-store" });
      return active(token) ? value : null;
    } catch (error) {
      if (!active(token)) return null;
      throw error;
    }
  };
  const graphPath = (query, cursor = "") => {
    const parameters = new URLSearchParams({
      depth: String(query.depth),
      limit: String(limit),
      direction: "both",
    });
    if (query.kinds) parameters.set("kinds", query.kinds);
    if (query.root) {
      parameters.set("root_kind", query.root.kind);
      parameters.set("root_id", query.root.id);
    }
    if (cursor) parameters.set("cursor", cursor);
    return `${base}/graph?${parameters}`;
  };
  const publish = (
    nextSections = sections,
    nextGraphs = graphs,
    nextPlans = plans,
    nextContexts = contexts,
    nextCards = cardSummaries,
  ) => {
    const next = buildModel(
      scope,
      snapshot,
      nextSections,
      nextGraphs,
      nextPlans,
      protocolMode,
      nextContexts,
      nextCards,
    );
    sections = nextSections;
    graphs = nextGraphs;
    plans = nextPlans;
    contexts = nextContexts;
    cardSummaries = nextCards;
    model = next;
    return next;
  };
  const graphKey = (query) => JSON.stringify(query);
  const enqueue = (operation) => {
    const token = epoch;
    const run = async () =>
      active(token) && snapshot ? operation(token) : null;
    const result = pending.then(run, run);
    pending = result.catch(() => null);
    return result;
  };
  return {
    async load(value) {
      const token = ++epoch;
      if (!active(token)) return null;
      pending = Promise.resolve();
      const root = checkedRef(value?.outcome?.ref, scope);
      if (root.kind !== "outcome" || root.id !== scope.outcome_id)
        throw failure(
          "The map snapshot does not identify its containing Outcome.",
        );
      canvasRevision(value.outcome_revision);
      const nextSnapshot = clone(value),
        nextSections = new Map();
      for (const name of mapSections)
        if (value.sections?.[name] || value.section_cursors?.[name]) {
          nextSections.set(name, {
            items: clone(value.sections?.[name] || []),
            count: value.counts?.[name] ?? value.sections?.[name]?.length ?? 0,
            omitted: value.omitted?.[name] || 0,
            cursor: value.section_cursors?.[name] || "",
          });
        }
      const query = { depth: 3, kinds: initialKinds },
        graph = await fetchCurrent(graphPath(query), token);
      if (!graph) return null;
      assertRevision(graph, canvasRevision(value.outcome_revision));
      const nextGraphs = new Map([
          [graphKey(query), { query, page: clone(graph) }],
        ]),
        nextPlans = new Map();
      const nextModel = buildModel(
        scope,
        nextSnapshot,
        nextSections,
        nextGraphs,
        nextPlans,
        protocolMode,
      );
      snapshot = nextSnapshot;
      sections = nextSections;
      graphs = nextGraphs;
      plans = nextPlans;
      contexts = new Map();
      cardSummaries = new Map();
      model = nextModel;
      lastMoreWasGraph = false;
      return nextModel;
    },
    summaries(page) {
      if (!active(epoch) || !model) return null;
      assertRevision(page, model.revision);
      if (!Array.isArray(page.items) || page.items.length > 100)
        throw failure("Card summaries must be a bounded batch of at most 100 records.");
      const next = new Map(cardSummaries);
      const seen = new Set();
      for (const summary of page.items) {
        const ref = checkedRef(summary.ref, scope), key = canvasKey(ref);
        if (seen.has(key)) throw failure("A card summary batch contains a duplicate record.");
        seen.add(key);
        canvasRevision(summary.version);
        const current = model.nodes.find((entry) => entry.key === key);
        if (!current)
          throw failure("Card summaries may enrich only records already loaded in this map.");
        if (current.version != null && canvasRevision(current.version) !== canvasRevision(summary.version))
          throw failure("Card summaries do not match the loaded entity version.", "precondition_failed");
        next.set(key, clone(summary));
      }
      return publish(sections, graphs, plans, contexts, next);
    },
    more() {
      return enqueue(async (token) => {
        // One bounded request per gesture; rotate visited sources for fair progress.
        const graphEntry = [...graphs].find(
          ([, value]) => value.page.next_cursor,
        );
        const sectionEntry = [...sections].find(([, value]) => value.cursor);
        const graphFirst = graphEntry && (!sectionEntry || !lastMoreWasGraph);
        if (graphFirst || (!sectionEntry && graphEntry)) {
          const [key, current] = graphEntry;
          const page = await fetchCurrent(
            graphPath(current.query, current.page.next_cursor),
            token,
          );
          if (!page) return null;
          assertRevision(page, model.revision);
          const next = new Map(graphs);
          next.delete(key);
          next.set(key, {
            query: current.query,
            page: {
              ...clone(page),
              nodes: mergeItems(current.page.nodes || [], page.nodes || []),
              edges: mergeItems(current.page.edges || [], page.edges || []),
            },
          });
          const updated = publish(sections, next);
          lastMoreWasGraph = true;
          return updated;
        }
        if (sectionEntry) {
          const [name, current] = sectionEntry;
          const parameters = new URLSearchParams({
            limit: String(limit),
            cursor: current.cursor,
          });
          const page = await fetchCurrent(
            `${base}/continuity/${name}?${parameters}`,
            token,
          );
          if (!page) return null;
          assertRevision(page, model.revision);
          const next = new Map(sections);
          next.delete(name);
          next.set(name, {
            items: mergeItems(current.items, page.items || []),
            count: page.count ?? current.count,
            omitted: page.omitted || 0,
            cursor: page.next_cursor || "",
          });
          const updated = publish(next);
          lastMoreWasGraph = false;
          return updated;
        }
        return model;
      });
    },
    expand(ref) {
      checkedRef(ref, scope);
      return enqueue(async (token) => {
        const query = { depth: 1, root: clone(ref) },
          key = graphKey(query);
        if (graphs.has(key)) return model;
        const page = await fetchCurrent(graphPath(query), token);
        if (!page) return null;
        assertRevision(page, model.revision);
        const next = new Map(graphs);
        next.set(key, { query, page: clone(page) });
        return publish(sections, next);
      });
    },
    context(value, { attention } = {}) {
      if (!active(epoch) || !model) return null;
      assertRevision(value, model.revision);
      const work = value.work_item;
      const workRef = checkedRef(
        { ...work?.scope, kind: "work_item", id: work?.id },
        scope,
      );
      if (value.outcome?.ref) checkedRef(value.outcome.ref, scope);
      if (attention) {
        assertRevision(attention, model.revision);
        if (
          attention.work_item_id !== workRef.id ||
          canvasKey(checkedRef(attention.operational_state?.ref, scope)) !==
            canvasKey(workRef)
        )
          throw failure("Attention data identifies another Task.");
      }
      const next = new Map(sections);
      const append = (name, incoming) => {
        const previous = next.get(name) || {
          items: [],
          count: 0,
          omitted: 0,
          cursor: "",
        };
        const items = mergeItems(previous.items, incoming);
        const count = Math.max(previous.count, items.length);
        next.set(name, {
          ...previous,
          items,
          count,
          omitted: Math.max(0, count - items.length),
        });
      };
      const metadata = (entity, ref) => ({
        ref,
        version: entity.version,
        title: entity.title,
        lifecycle: entity.lifecycle,
        priority: entity.priority,
        detail_omitted: true,
      });
      append("work_items", [metadata(work, workRef)]);
      if (value.objective) {
        const objectiveRef = checkedRef(
          {
            ...value.objective.scope,
            kind: "objective",
            id: value.objective.id,
          },
          scope,
        );
        append("objectives", [metadata(value.objective, objectiveRef)]);
      }
      append("relations", value.dependencies || []);
      append("evidence_links", value.evidence_links || []);
      const nextContexts = new Map(contexts);
      const parents = work.objective_id
        ? [{
            id: `work:${work.id}`,
            source: { ...scope, kind: "objective", id: work.objective_id },
            target: workRef,
            kind: "ownership",
            truth: "work_item",
          }]
        : [];
      if (value.objective)
        parents.push({
          id: `objective:${value.objective.id}`,
          source: value.objective.parent_objective_id
            ? {
                ...scope,
                kind: "objective",
                id: value.objective.parent_objective_id,
              }
            : { ...scope, kind: "outcome", id: scope.outcome_id },
          target: { ...scope, kind: "objective", id: value.objective.id },
          kind: "hierarchy",
          truth: "objective",
        });
      nextContexts.set(canvasKey(workRef), {
        parents,
        workSummary: metadata(work, workRef),
        objectiveSummary: value.objective
          ? metadata(value.objective, {
              ...scope,
              kind: "objective",
              id: value.objective.id,
            })
          : undefined,
        attention: attention
          ? clone(attention)
          : contexts.get(canvasKey(workRef))?.attention,
        evaluatedAt: attention?.evaluated_at || value.evaluated_at,
      });
      return publish(next, graphs, plans, nextContexts);
    },
    async search(
      text,
      { cursor = "", kinds = ["objective", "work_item"] } = {},
    ) {
      const token = epoch;
      if (!active(token) || !model) return null;
      if (!kinds.length || kinds.some((kind) => !supportedKinds.has(kind)))
        throw failure("Choose supported entity kinds for map search.");
      const parameters = new URLSearchParams({
        kind: kinds.join(","),
        query: String(text),
        limit: "25",
      });
      if (cursor) parameters.set("cursor", cursor);
      const page = await fetchCurrent(
        `${base}/references?${parameters}`,
        token,
      );
      if (!page) return null;
      assertRevision(page, model.revision);
      for (const item of page.items || []) checkedRef(item.ref, scope);
      return deepFreeze({
        ...clone(page),
        items: (page.items || []).map((item) => ({
          ...clone(item),
          key: canvasKey(item.ref),
          kind: item.ref.kind,
        })),
        stale: false,
      });
    },
    roadmap({ roadmap_id, revision_number, draft, outcome_revision }) {
      if (
        typeof roadmap_id !== "string" ||
        !roadmap_id ||
        (!draft && !/^[1-9][0-9]*$/.test(String(revision_number)))
      )
        throw failure("Choose a published Roadmap revision or an open Draft.");
      return enqueue(async (token) => {
        if (draft) {
          assertRevision({ outcome_revision }, model.revision);
          if (
            draft.lifecycle !== "open" ||
            !/^[1-9][0-9]*$/.test(String(draft.draft_version))
          )
            throw failure("Choose an open Roadmap Draft.");
          return publish(
            sections,
            graphs,
            new Map([
              [
                `${roadmap_id}:draft`,
                { roadmap_id, draft: true, value: clone(draft) },
              ],
            ]),
          );
        }
        const page = await fetchCurrent(
          `${base}/roadmaps/${encodeURIComponent(roadmap_id)}/revisions/${encodeURIComponent(revision_number)}`,
          token,
        );
        if (!page) return null;
        assertRevision(page, model.revision);
        const value = page.value || page;
        if (canvasRevision(value.revision_number) !== String(revision_number))
          throw failure(
            "The published plan response identifies another revision.",
          );
        const next = new Map([
          [
            `${roadmap_id}:${revision_number}`,
            {
              roadmap_id,
              revision_number: String(revision_number),
              value: clone(value),
            },
          ],
        ]);
        return publish(sections, graphs, next);
      });
    },
    current: () => model,
    dispose() {
      disposed = true;
      epoch++;
    },
  };
}
