import { node, badge, kindName, display } from "./presentation.js";
import { copy, message } from "./en-US.js";
import { canvasWorkLayer, canvasRecordExplorer, canvasCardPresentation, MAX_CONTEXT_RECORDS } from "./workspace-card-summary.js";

// Coordinates describe presentation only. They never imply ownership,
// dependencies, readiness, execution authority, or a server-side mutation.
const SVG = "http://www.w3.org/2000/svg";
const STRUCTURE = new Set(["hierarchy", "ownership", "plan_hierarchy"]);
const OPERATIONAL = new Set([
  "hierarchy",
  "ownership",
  "depends_on",
  "blocks",
  "causes",
]);
const PLANNING = new Set(["plan_hierarchy", "plan_reference", "plan_sequence"]);
const MIN_ZOOM = 0.25;
const MAX_ZOOM = 2.5;
let sequence = 0;

// The supplied paper is the card itself. Keep its original geometry and put
// live HTML inside the source safe area; paper style never grants authority.
const NOTE_PAPERS = {
  outcome: ["wide", "pinned", "surface"],
  objective: ["square", "tabbed", "workspace"],
  work_item: ["wide", "curled", "surface"],
  issue: ["wide", "taped", "workspace"],
  blocker: ["wide", "torn", "workspace"],
  evidence: ["wide", "notebook", "surface"],
  artifact: ["wide", "stacked", "surface"],
  decision: ["wide", "dogear", "workspace"],
  phase: ["wide", "rounded", "workspace"],
  milestone: ["wide", "pinned", "surface"],
  plan_reference: ["wide", "classic", "workspace"],
};
const DEFAULT_PAPER = ["wide", "rounded", "workspace"];
const paperFor = (kind) =>
  Object.hasOwn(NOTE_PAPERS, kind) ? NOTE_PAPERS[kind] : DEFAULT_PAPER;
function paperSource(kind, selected) {
  const [format, style, tone] = paperFor(kind);
  return new URL(
    `./postits/svg/${format}/${style}/${selected ? "selected" : tone}.svg`,
    import.meta.url,
  ).href;
}

export function canvasNodeSize(kind) {
  const width =
    kind === "outcome"
      ? 336
      : kind === "objective" ||
          ["phase", "milestone", "plan_reference"].includes(kind)
        ? 252
        : kind === "work_item" ? 304 : 284;
  return {
    width,
    height: paperFor(kind)[0] === "square" ? width : (width * 730) / 1000,
  };
}

const compare = (a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0);
const clamp = (value, min, max) => Math.min(max, Math.max(min, value));
const unit = (value) => `calc(${value} * var(--wos-unit, 1px))`;

// These are counts of loaded, typed links, not a second readiness or proof
// projection. Index once for the whole render, including collapsed records;
// paper position and visual grouping never contribute operational facts.
export function canvasNodeIndicators(model) {
  const nodes = new Map((model.nodes || []).map((item) => [item.key, item]));
  const indexed = new Map(
    [...nodes.keys()].map((key) => [
      key,
      {
        dependencies: new Map(),
        blockers: new Set(),
        evidence: new Map(),
      },
    ]),
  );
  const identity = (edge) => {
    const truth = edge.sourceOfTruth;
    if (edge.ref?.kind === truth && edge.ref.id)
      return `${truth}:${edge.ref.id}`;
    const prefix = `graph:${truth}:`;
    if (edge.key?.startsWith(prefix)) return edge.key.slice(6);
    return `${truth}:${edge.key || `${edge.sourceKey}:${edge.targetKey}`}`;
  };
  const remember = (entries, key, value) => {
    if (!entries.has(key)) entries.set(key, new Set());
    if (value !== "unknown") entries.get(key).add(value);
  };
  const countClasses = (entries, classes) => {
    const counts = { total: entries.size, ...classes, unknown: 0 };
    for (const values of entries.values())
      counts[values.size === 1 ? [...values][0] : "unknown"]++;
    return counts;
  };
  const operational = new Set(["objective", "work_item"]);
  const blockedKinds = new Set(["outcome", ...operational]);
  const evidenceKinds = new Set([...blockedKinds, "issue", "decision"]);
  for (const edge of model.edges || []) {
    const source = nodes.get(edge.sourceKey),
      target = nodes.get(edge.targetKey);
    // An unloaded endpoint is not a loaded relationship in this model.
    // A contextMissing placeholder still carries its real typed reference.
    if (!source || !target) continue;
    if (
      edge.kind === "depends_on" &&
      edge.sourceOfTruth === "relation" &&
      operational.has(source.kind) &&
      operational.has(target.kind)
    )
      remember(
        indexed.get(source.key).dependencies,
        identity(edge),
        ["hard", "advisory"].includes(edge.mode) ? edge.mode : "unknown",
      );
    else if (
      edge.kind === "blocks" &&
      edge.sourceOfTruth === "blocker" &&
      source.kind === "blocker" &&
      blockedKinds.has(target.kind)
    )
      // A Blocker owns exactly one direct target. Its cause edge and any
      // associated Issue cannot add an extra or inherited impact here.
      indexed.get(target.key).blockers.add(source.key);
    else if (
      edge.sourceOfTruth === "evidence_link" &&
      source.kind === "evidence" &&
      evidenceKinds.has(target.kind)
    ) {
      const stance = ["supports", "contradicts", "context"].includes(edge.kind)
        ? edge.kind
        : "unknown";
      const key = identity(edge);
      remember(indexed.get(source.key).evidence, key, stance);
      remember(indexed.get(target.key).evidence, key, stance);
    }
  }
  const limited = Boolean(model.partial || model.boundary?.scopeLimited);
  const result = new Map();
  for (const [key, links] of indexed) {
    const dependencies = countClasses(links.dependencies, {
      hard: 0,
      advisory: 0,
    });
    const evidence = countClasses(links.evidence, {
      supports: 0,
      contradicts: 0,
      context: 0,
    });
    const blockers = links.blockers.size;
    const labels = [],
      details = [];
    if (dependencies.total) {
      labels.push(
        `${dependencies.total} ${dependencies.total === 1 ? "dep" : "deps"}`,
      );
      details.push(
        `${dependencies.total} outgoing dependencies: ${dependencies.hard} hard, ${dependencies.advisory} advisory, ${dependencies.unknown} strength not loaded`,
      );
    }
    if (blockers) {
      labels.push(`${blockers} ${blockers === 1 ? "blocker" : "blockers"}`);
      details.push(
        `${blockers} direct Blocker impact ${blockers === 1 ? "link" : "links"}; inherited blocking is not inferred`,
      );
    }
    if (evidence.total) {
      labels.push(
        `${evidence.total} evidence ${evidence.total === 1 ? "link" : "links"}`,
      );
      details.push(
        `${evidence.total} ${nodes.get(key).kind === "evidence" ? "outgoing" : "incoming"} EvidenceLinks: ${evidence.supports} supporting, ${evidence.contradicts} contradicting, ${evidence.context} context, ${evidence.unknown} stance not loaded; Evidence is not criterion verification`,
      );
    }
    result.set(key, {
      dependencies,
      blockers,
      evidence,
      limited,
      label: labels.length ? `Loaded · ${labels.join(" · ")}` : "",
      description: details.length
        ? `Loaded relationships only${limited ? "; this map is partial or query-limited" : ""}. ${details.join(". ")}. Readiness and acceptance remain explicit server state.`
        : "",
    });
  }
  return result;
}

// A deterministic forest layout. Only the adapter's explicit structural parent
// is considered: operational dependencies and Roadmap order cannot reparent a
// card. Existing local positions win during a refresh; new cards avoid them.
export function layoutCanvas(
  model,
  { collapsed = new Set(), positions = new Map() } = {},
) {
  const byKey = new Map();
  for (const item of model.nodes || []) {
    if (
      !item ||
      typeof item.key !== "string" ||
      !item.key ||
      byKey.has(item.key)
    )
      throw Error("The work map contains an invalid or duplicate item key.");
    byKey.set(item.key, { ...item, ...canvasNodeSize(item.kind) });
  }
  const sorted = [...byKey.values()].sort(compare);
  const parents = new Map();
  for (const item of sorted)
    if (item.parentKey !== item.key && byKey.has(item.parentKey))
      parents.set(item.key, item.parentKey);
  // Malformed presentation data must not cause an infinite traversal. Breaking
  // a visual cycle does not remove or rewrite its real typed API edge.
  for (const item of sorted) {
    const path = [],
      seen = new Set();
    let key = item.key;
    while (parents.has(key)) {
      if (seen.has(key)) {
        const cycle = path.slice(path.indexOf(key)).sort();
        parents.delete(cycle[0]);
        break;
      }
      seen.add(key);
      path.push(key);
      key = parents.get(key);
    }
  }
  const children = new Map(sorted.map((item) => [item.key, []]));
  for (const item of sorted)
    if (parents.has(item.key)) children.get(parents.get(item.key)).push(item);
  const roots = sorted.filter((item) => !parents.has(item.key));
  roots.sort((a, b) =>
    a.key === model.rootKey ? -1 : b.key === model.rootKey ? 1 : compare(a, b),
  );
  const descendants = new Map();
  function count(key) {
    if (descendants.has(key)) return descendants.get(key);
    const value = children
      .get(key)
      .reduce((total, item) => total + 1 + count(item.key), 0);
    descendants.set(key, value);
    return value;
  }
  const visible = [],
    heights = new Map(),
    columns = [];
  function measure(item, depth) {
    const entry = {
      ...item,
      depth,
      childCount: children.get(item.key).length,
      descendantCount: count(item.key),
      collapsed: collapsed.has(item.key),
    };
    visible.push(entry);
    columns[depth] = Math.max(columns[depth] || 0, entry.width);
    const rows = collapsed.has(item.key) ? [] : children.get(item.key);
    const height = Math.max(
      entry.height,
      rows.reduce(
        (total, child, index) =>
          total + measure(child, depth + 1) + (index ? 32 : 0),
        0,
      ),
    );
    heights.set(item.key, height);
    return height;
  }
  roots.forEach((item) => measure(item, 0));
  const columnX = [48];
  for (let i = 1; i < columns.length; i++)
    columnX[i] = columnX[i - 1] + columns[i - 1] + 88;
  const ideal = new Map();
  function place(item, top, depth) {
    ideal.set(item.key, {
      x: columnX[depth],
      y: top + (heights.get(item.key) - item.height) / 2,
    });
    if (collapsed.has(item.key)) return;
    const rows = children.get(item.key);
    const total = rows.reduce(
      (sum, child, index) => sum + heights.get(child.key) + (index ? 32 : 0),
      0,
    );
    let y = top + (heights.get(item.key) - total) / 2;
    for (const child of rows) {
      place(child, y, depth + 1);
      y += heights.get(child.key) + 32;
    }
  }
  let top = 48;
  for (const root of roots) {
    place(root, top, 0);
    top += heights.get(root.key) + 72;
  }
  const occupied = [];
  const placed = new Map();
  for (const item of visible) {
    const saved = positions.get(item.key);
    if (saved && Number.isFinite(saved.x) && Number.isFinite(saved.y)) {
      const entry = {
        ...item,
        x: saved.x,
        y: saved.y,
      };
      placed.set(item.key, entry);
      occupied.push(entry);
    }
  }
  for (const item of visible) {
    if (placed.has(item.key)) continue;
    const entry = { ...item, ...ideal.get(item.key) };
    let collision;
    do {
      collision = occupied.find(
        (other) =>
          entry.x < other.x + other.width + 16 &&
          entry.x + entry.width + 16 > other.x &&
          entry.y < other.y + other.height + 16 &&
          entry.y + entry.height + 16 > other.y,
      );
      if (collision) entry.y = collision.y + collision.height + 32;
    } while (collision);
    placed.set(item.key, entry);
    occupied.push(entry);
  }
  const nodes = visible.map((item) => placed.get(item.key));
  // The map has no top/left wall. Bounds describe only its loaded cards and
  // their reading margin; they do not constrain a user's local arrangement.
  const x = nodes.length ? Math.min(...nodes.map((item) => item.x - 48)) : 0,
    y = nodes.length ? Math.min(...nodes.map((item) => item.y - 48)) : 0;
  const width =
      Math.max(600, ...nodes.map((item) => item.x + item.width + 112 - x)),
    height =
      Math.max(400, ...nodes.map((item) => item.y + item.height + 48 - y));
  return {
    nodes,
    byKey: placed,
    parents,
    children,
    edges: (model.edges || []).filter(
      (edge) => placed.has(edge.sourceKey) && placed.has(edge.targetKey),
    ),
    hiddenCount: sorted.length - nodes.length,
    bounds: { x, y, width, height },
    width,
    height,
  };
}

export function zoomCanvasCamera(
  camera,
  factor,
  anchor,
  { min = MIN_ZOOM, max = MAX_ZOOM } = {},
) {
  const zoom = clamp(camera.zoom * factor, min, max);
  const ratio = zoom / camera.zoom;
  return {
    zoom,
    x: anchor.x - (anchor.x - camera.x) * ratio,
    y: anchor.y - (anchor.y - camera.y) * ratio,
  };
}

export function canvasVisibleEdges(edges, selectedKey, filter = "context") {
  return edges.filter(
    (edge) =>
      edge.sourceKey === selectedKey ||
      edge.targetKey === selectedKey ||
      filter === "all" ||
      (filter === "planning"
        ? PLANNING.has(edge.kind)
        : filter === "context"
          ? STRUCTURE.has(edge.kind)
          : OPERATIONAL.has(edge.kind)),
  );
}

export function canvasEdgeLabel(edge) {
  if (edge.kind === "depends_on")
    return `Depends on · ${edge.mode === "hard" ? "hard" : edge.mode === "advisory" ? "advisory" : "strength not loaded"}`;
  if (edge.label) return edge.label;
  return (
    {
      hierarchy: "Objective hierarchy",
      ownership: "Task association",
      blocks: "Blocking impact",
      causes: "Blocking cause",
      supports: "Supporting evidence",
      contradicts: "Contradicting evidence",
      context: "Context evidence",
      plan_hierarchy: "Plan grouping",
      plan_reference: "Plan reference",
      plan_sequence: "Plan order",
    }[edge.kind] || display(edge.kind)
  );
}

export function mountWorkspaceCanvas(host, options) {
  const id = `work-map-${++sequence}`;
  const abort = new AbortController();
  const root = node("section", undefined, "workspace-canvas");
  root.setAttribute("aria-label", "Outcome Board");
  const toolbar = node("div", undefined, "canvas-toolbar");
  const switcher = node("div", undefined, "canvas-view-switch");
  switcher.setAttribute("aria-label", "Outcome Board view");
  const cameraControls = node("div", undefined, "canvas-camera-controls");
  const zoomOutput = node("output", "100%", "canvas-zoom");
  zoomOutput.setAttribute("aria-label", "Outcome Board zoom");
  const searchWrap = node("div", undefined, "canvas-search");
  const search = node("input");
  search.type = "search";
  search.placeholder = "Find an item…";
  search.setAttribute("aria-label", "Search Outcome Board");
  search.setAttribute("role", "combobox");
  search.setAttribute("aria-autocomplete", "list");
  search.setAttribute("aria-expanded", "false");
  const searchResults = node("div", undefined, "canvas-search-results");
  const searchList = node("div", undefined, "canvas-search-list");
  searchList.id = id + "-search";
  searchList.setAttribute("role", "listbox");
  searchList.setAttribute("aria-label", "Matching items");
  const searchMore = control(
    "Load more search results",
    () => loadMoreSearch(),
    "quiet canvas-search-more",
  );
  const searchStatus = node("p", undefined, "canvas-search-status");
  searchStatus.setAttribute("role", "status");
  searchResults.append(searchList, searchMore, searchStatus);
  searchResults.hidden = true;
  search.setAttribute("aria-controls", searchList.id);
  searchWrap.append(search, searchResults);
  const filterWrap = node("label", "Connections", "canvas-link-filter");
  const edgeFilter = node("select");
  edgeFilter.setAttribute("aria-label", "Connections");
  for (const [value, label] of [
    ["operational", "Operational links"],
    ["context", "Structure and selection"],
    ["planning", "Planning links"],
    ["all", "All loaded links"],
  ]) {
    const option = node("option", label);
    option.value = value;
    edgeFilter.append(option);
  }
  filterWrap.append(edgeFilter);
  edgeFilter.value = "context";
  const layerWrap = node("label", copy.text.boardLayers, "canvas-layer-filter");
  const layerSelect = node("select");
  layerSelect.setAttribute("aria-label", copy.text.boardLayers);
  for (const [value, label] of [["work", copy.text.boardWorkMap], ["records", copy.text.boardAllRecords]]) {
    const option = node("option", label);
    option.value = value;
    layerSelect.append(option);
  }
  layerWrap.append(layerSelect);
  const viewport = node("div", undefined, "canvas-viewport");
  viewport.tabIndex = 0;
  viewport.setAttribute("role", "region");
  viewport.setAttribute(
    "aria-label",
    "Outcome Board map. Drag cards to arrange them; drag the background or scroll to pan. Use Control or Command with scroll to zoom, arrow keys to navigate, plus and minus to zoom.",
  );
  const world = node("div", undefined, "canvas-world");
  const edges = document.createElementNS(SVG, "svg");
  edges.classList.add("canvas-edges");
  edges.setAttribute("role", "group");
  edges.setAttribute("aria-label", "Loaded connections");
  world.append(edges);
  viewport.append(world);
  const map = node("div", undefined, "canvas-map");
  map.append(viewport, cameraControls);
  const list = node("ul", undefined, "canvas-list");
  list.setAttribute("aria-label", "Outcome Board list");
  const stage = node("div", undefined, "canvas-stage");
  stage.append(map, list);
  const inspectorHost = options.inspectorHost;
  const inspectorParent = inspectorHost?.parentNode;
  const inspectorNext = inspectorHost?.nextSibling;
  if (inspectorHost) stage.append(inspectorHost);
  const footer = node("div", undefined, "canvas-footer");
  const status = node("p", undefined, "canvas-status");
  status.setAttribute("role", "status");
  const legend = node("details", undefined, "canvas-legend");
  const legendItems = node("div", undefined, "canvas-legend-items");
  for (const [kind, label, mode] of [
    ["hierarchy", "Hierarchy / association"],
    ["depends_on", "Hard dependency", "hard"],
    ["depends_on", "Advisory dependency", "advisory"],
    ["depends_on", "Dependency strength not loaded", "unknown"],
    ["blocks", "Blocking impact"],
    ["supports", "Evidence stance"],
    ["plan_sequence", "Planning · not a dependency"],
  ]) {
    const row = node("span"),
      sample = node("i", undefined, "canvas-legend-line");
    sample.dataset.kind = kind;
    if (mode) sample.dataset.mode = mode;
    row.append(sample, document.createTextNode(label));
    legendItems.append(row);
  }
  legend.append(node("summary", "Connection key"), legendItems);
  footer.append(status, legend);
  root.append(toolbar, stage, footer);
  host.replaceChildren(root);

  let model,
    layout,
    selectedKey = options.selectedKey || null,
    selectedEdgeKey = null,
    connectionDescription = "";
  let collapsed = new Set(),
    positions = new Map(),
    manualPositions = new Map(),
    opened = new Set(),
    nodeButtons = new Map();
  let layer = options.initialLayer === "records" ? "records" : "work",
    recordRoots = new Set(), recordsFor = () => new Set(), hiddenRecordCount = 0;
  layerSelect.value = layer;
  let camera = { x: 24, y: 24, zoom: 1 },
    gesture = null,
    queued = null;
  let disposed = false,
    first = true,
    cameraInitialized = false,
    lastError = "",
    ignoreClickUntil = 0;
  let searchTimer = null,
    searchAbort = null,
    searchVersion = 0,
    candidates = [],
    activeCandidate = -1;
  let searchLocal = [],
    searchRemote = [],
    searchCursor = "",
    searchShown = 12,
    searchLoading = false,
    searchQuery = "",
    searchIntent = false;
  let userView = null;
  const compact = window.matchMedia("(max-width: 700px)");
  const view = () => userView || (compact.matches ? "list" : "map");
  function control(label, action, className = "quiet") {
    const button = node("button", label, className);
    button.type = "button";
    button.onclick = action;
    return button;
  }
  const mapButton = control("Map view", () => changeView("map"));
  const listButton = control("List view", () => changeView("list"));
  switcher.append(mapButton, listButton);
  function cameraControl(label, paths, action) {
    const button = control(label, action, "quiet canvas-camera-button");
    button.setAttribute("aria-label", label);
    button.title = label;
    const icon = document.createElementNS(SVG, "svg");
    icon.setAttribute("viewBox", "0 0 20 20");
    icon.setAttribute("aria-hidden", "true");
    icon.setAttribute("fill", "none");
    icon.setAttribute("stroke", "currentColor");
    icon.setAttribute("stroke-width", "1.5");
    icon.setAttribute("stroke-linecap", "round");
    icon.setAttribute("stroke-linejoin", "round");
    for (const d of paths) {
      const path = document.createElementNS(SVG, "path");
      path.setAttribute("d", d);
      icon.append(path);
    }
    button.replaceChildren(icon);
    return button;
  }
  cameraControls.setAttribute("role", "group");
  cameraControls.setAttribute("aria-label", "Board camera controls");
  cameraControls.append(
    cameraControl("Zoom in", ["M10 4v12M4 10h12"], () => zoom(1.2)),
    cameraControl("Zoom out", ["M4 10h12"], () => zoom(1 / 1.2)),
    zoomOutput,
    cameraControl("Center view", [
      "M10 2v4m0 8v4M2 10h4m8 0h4",
      "M14 10a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z",
    ], () => center()),
    cameraControl("Fit view", ["M7 3H3v4m10-4h4v4M3 13v4h4m10-4v4h-4"], () => fit()),
  );
  const autoLayout = control("Auto-layout", () => {
    positions.clear();
    manualPositions.clear();
    render();
    fit();
  }, "quiet canvas-auto-layout");
  layerSelect.onchange = () => {
    layer = layerSelect.value;
    render();
  };
  toolbar.append(switcher, searchWrap, layerWrap, filterWrap, autoLayout);
  const more = control("Load more map items", () => invoke(options.onLoadMore));
  footer.insertBefore(more, legend);

  function invoke(callback, ...args) {
    if (!callback || disposed) return;
    try {
      Promise.resolve(callback(...args)).catch(report);
    } catch (error) {
      report(error);
    }
  }
  function report(error) {
    if (error?.discarded || error?.name === "AbortError") return;
    if (!disposed) {
      lastError = error?.message || "The work map could not be updated.";
      announce();
    }
  }
  function announce() {
    const prefix = model?.partial ? "Partial map · " : "Loaded context · ";
    status.textContent = `${prefix}${model?.nodes?.length || 0} loaded items · ${model?.edges?.length || 0} loaded connections · revision ${model?.revision ?? "not loaded"}${layout?.hiddenCount ? ` · ${layout.hiddenCount} items inside collapsed groups` : ""}.${hiddenRecordCount ? " " + message("boardRecordsHidden", hiddenRecordCount) : ""}${model?.boundary?.scopeLimited ? " Graph depth and query limits apply." : ""}${root.dataset.detail === "overview" ? " Overview scale: select an item for details, or use List view for full controls." : ""}${connectionDescription ? " " + connectionDescription : ""}${lastError ? " " + lastError : ""}`;
    root.dataset.partial = String(Boolean(model?.partial));
    more.hidden = !model?.hasMore || !options.onLoadMore;
  }
  function uiUnit() {
    return layout?.width ? world.offsetWidth / layout.width || 1 : 1;
  }
  function cameraPaint() {
    world.style.transform = `translate(${camera.x}px, ${camera.y}px) scale(${camera.zoom})`;
    zoomOutput.value =
      zoomOutput.textContent = `${Math.round(camera.zoom * 100)}%`;
    viewport.dataset.scale = String(camera.zoom);
    const detail = uiUnit() * camera.zoom < 0.6 ? "overview" : "detail";
    if (root.dataset.detail !== detail) {
      if (
        detail === "overview" &&
        world.contains(document.activeElement) &&
        document.activeElement?.closest(".canvas-node-actions")
      )
        document.activeElement
          .closest(".canvas-node")
          ?.querySelector(".canvas-node-main")
          ?.focus({ preventScroll: true });
      root.dataset.detail = detail;
      announce();
    }
  }
  function cameraArea() {
    const area = { width: viewport.clientWidth, height: viewport.clientHeight };
    if (inspectorHost && !inspectorHost.hidden) {
      const mapBox = viewport.getBoundingClientRect();
      const panelBox = inspectorHost.getBoundingClientRect();
      const uncovered = panelBox.left - mapBox.left - 12 * uiUnit();
      // A right overlay should not become the center of fit/zoom. On compact
      // screens it covers nearly the whole map; preserve that map's camera.
      if (
        panelBox.left < mapBox.right &&
        uncovered >= 160 && uncovered < area.width &&
        panelBox.bottom > mapBox.top
      )
        area.width = uncovered;
    }
    return area;
  }
  function center() {
    if (!layout) return;
    const area = cameraArea(),
      scale = uiUnit();
    camera.x =
      (area.width - layout.width * scale * camera.zoom) / 2 -
      layout.bounds.x * scale * camera.zoom;
    camera.y =
      (area.height - layout.height * scale * camera.zoom) / 2 -
      layout.bounds.y * scale * camera.zoom;
    cameraPaint();
  }
  function zoom(factor, point) {
    const area = cameraArea();
    point ||= { x: area.width / 2, y: area.height / 2 };
    camera = zoomCanvasCamera(camera, factor, point, {
      min: Math.min(MIN_ZOOM, fitScale(area)),
    });
    cameraPaint();
  }
  function fitScale(area) {
    if (!layout) return MIN_ZOOM;
    const scale = uiUnit();
    return Math.max(0.001,
      Math.min(
        (area.width - 48) / (layout.width * scale),
        (area.height - 48) / (layout.height * scale),
        1,
      ),
    );
  }
  function fit(initial = false) {
    if (!layout || !viewport.clientWidth || !viewport.clientHeight) return;
    const scale = uiUnit();
    const width = layout.width * scale,
      height = layout.height * scale;
    const area = cameraArea();
    const desired = fitScale(area);
    // Initial entry keeps notes readable. An explicit Fit may use overview
    // scale to show the whole loaded map; List/details keep full-size content.
    camera.zoom = initial ? Math.max(desired, 0.75) : desired;
    camera.x =
      Math.max(24, (area.width - width * camera.zoom) / 2) -
      layout.bounds.x * scale * camera.zoom;
    camera.y =
      Math.max(24, (area.height - height * camera.zoom) / 2) -
      layout.bounds.y * scale * camera.zoom;
    if (initial && desired < 0.75) {
      const anchor =
        layout.byKey.get(selectedKey) ||
        layout.byKey.get(model.rootKey) ||
        layout.nodes[0];
      if (anchor) {
        camera.x = 32 - anchor.x * scale * camera.zoom;
        camera.y =
          viewport.clientHeight / 2 -
          (anchor.y + anchor.height / 2) * scale * camera.zoom;
      }
    }
    cameraInitialized = true;
    cameraPaint();
  }
  function changeView(next) {
    userView = next;
    paintView();
    // Compact workspaces begin in List view, which keeps the map viewport at
    // zero size during the first render. If the initial fit ran while hidden,
    // fit as soon as the user opens Map so the first cards are in view.
    if (next === "map" && !cameraInitialized)
      requestAnimationFrame(() => {
        if (!disposed && !viewport.hidden && !cameraInitialized) fit(true);
      });
    if (next === "map" && selectedKey) reveal(selectedKey);
    invoke(options.onViewChange, next);
  }
  function paintView() {
    const mode = view();
    root.dataset.view = mode;
    map.hidden = mode !== "map";
    viewport.hidden = mode !== "map";
    list.hidden = mode !== "list";
    cameraControls.hidden = mode !== "map";
    autoLayout.hidden = mode !== "map";
    filterWrap.hidden = mode !== "map";
    mapButton.setAttribute("aria-pressed", String(mode === "map"));
    listButton.setAttribute("aria-pressed", String(mode === "list"));
    if (mode === "map") cameraPaint();
  }
  function select(item, { focus = false } = {}) {
    selectedKey = item.key;
    selectedEdgeKey = null;
    connectionDescription = "";
    if (!layout?.byKey.has(item.key) && model.nodes.some((entry) => entry.key === item.key)) render();
    paintSelection();
    if (focus) reveal(item.key);
    invoke(options.onSelect, item.ref || null, item);
  }
  function paintSelection() {
    for (const article of world.querySelectorAll("[data-canvas-node]")) {
      const selected = article.dataset.canvasNode === selectedKey;
      article.dataset.selected = String(selected);
      const paper = article.querySelector(".canvas-node-paper");
      const source = paperSource(article.dataset.kind, selected);
      if (paper && paper.src !== source) paper.src = source;
    }
    for (const button of root.querySelectorAll(
      ".canvas-node-main, .canvas-list-main",
    ))
      button.setAttribute(
        "aria-pressed",
        String(button.dataset.key === selectedKey),
      );
    renderEdges();
  }
  function toggle(item) {
    if (collapsed.has(item.key)) collapsed.delete(item.key);
    else collapsed.add(item.key);
    if (collapsed.has(item.key)) opened.delete(item.key);
    else opened.add(item.key);
    positions = new Map(manualPositions);
    render();
    const updated = layout.byKey.get(item.key);
    if (updated) {
      camera.x += (item.x - updated.x) * uiUnit() * camera.zoom;
      camera.y += (item.y - updated.y) * uiUnit() * camera.zoom;
      cameraPaint();
    }
    invoke(options.onExpand, item, !collapsed.has(item.key));
  }
  function reveal(key) {
    if (!model.nodes.some((item) => item.key === key)) {
      const edge = model.edges.find(
        (entry) =>
          entry.key === key ||
          (entry.ref && `${entry.ref.kind}:${entry.ref.id}` === key),
      );
      if (!edge) return false;
      selectedEdgeKey = edge.key;
      reveal(edge.sourceKey);
      reveal(edge.targetKey);
      const source = layout.byKey.get(edge.sourceKey),
        target = layout.byKey.get(edge.targetKey);
      if (source && target) {
        const scale = uiUnit() * camera.zoom,
          area = cameraArea();
        camera.x =
          area.width / 2 -
          ((source.x + source.width / 2 + target.x + target.width / 2) / 2) *
            scale;
        camera.y =
          area.height / 2 -
          ((source.y + source.height / 2 + target.y + target.height / 2) / 2) *
            scale;
        cameraPaint();
        renderEdges();
        [...edges.querySelectorAll("[data-edge-key]")]
          .find((entry) => entry.dataset.edgeKey === edge.key)
          ?.focus({ preventScroll: true });
      }
      return Boolean(source && target);
    }
    let current = key,
      changed = false;
    const seen = new Set();
    const raw = new Map(model.nodes.map((item) => [item.key, item]));
    while (raw.has(current) && !seen.has(current)) {
      seen.add(current);
      const parent = raw.get(current).parentKey;
      if (!parent) break;
      if (collapsed.delete(parent)) changed = true;
      current = parent;
    }
    if (changed) render();
    if (!layout.byKey.has(key) && layer === "work" && raw.has(key)) {
      recordRoots.add(key);
      render();
    }
    const item = layout.byKey.get(key);
    if (!item) return false;
    const scale = uiUnit() * camera.zoom,
      area = cameraArea();
    camera.x = area.width / 2 - (item.x + item.width / 2) * scale;
    camera.y = area.height / 2 - (item.y + item.height / 2) * scale;
    cameraPaint();
    const button =
      view() === "list"
        ? [...list.querySelectorAll(".canvas-list-main")].find(
            (entry) => entry.dataset.key === key,
          )
        : nodeButtons.get(key);
    button?.focus({ preventScroll: true });
    if (view() === "list") button?.scrollIntoView({ block: "nearest" });
    return true;
  }
  function nodeActions(item) {
    const actions = node("div", undefined, "canvas-node-actions");
    if (item.childCount || item.expandable) {
      const label = item.childCount
        ? `${item.collapsed ? "Show" : "Hide"} ${item.childCount}`
        : "Explore";
      const expand = control(label, () => toggle(item), "quiet canvas-expand");
      expand.setAttribute("aria-label", `${item.collapsed || !item.childCount ? "Expand" : "Collapse"} ${item.title}`);
      expand.setAttribute("aria-expanded", String(!item.collapsed));
      actions.append(expand);
    }
    const records = recordsFor(item.key);
    const canCreate = Boolean(item.canCreate?.length && options.onCreate);
    const canDepend = Boolean(item.canConnect && item.ref && options.onConnect && ["objective", "work_item"].includes(item.kind));
    const canExplore = layer === "work" && records.size && ["outcome", "objective", "work_item"].includes(item.kind);
    if (canCreate || canDepend || canExplore) {
      const menu = node("div", undefined, "canvas-node-menu");
      menu.hidden = true;
      const add = control(canCreate ? "Add…" : copy.text.boardMoreActions, () => {
        const opening = menu.hidden;
        closeMenus();
        menu.hidden = !opening;
        add.setAttribute("aria-expanded", String(opening));
        if (opening) menu.querySelector("button")?.focus();
      }, "quiet canvas-add");
      add.setAttribute("aria-label", canCreate ? `Add an item within ${item.title}` : message("boardMoreActionsFor", item.title));
      add.setAttribute("aria-expanded", "false");
      const close = () => { menu.hidden = true; add.setAttribute("aria-expanded", "false"); };
      if (canCreate)
        for (const kind of item.canCreate)
          menu.append(control(kind === "objective" ? "Add objective" : "New task", () => {
            close();
            invoke(options.onCreate, kind, item.ref || null);
          }));
      if (canDepend) {
        const depend = control(copy.text.boardAddDependency, () => {
          close();
          invoke(options.onConnect, { sourceRef: item.ref, targetRef: null });
        }, "quiet canvas-connect");
        depend.setAttribute("aria-label", message("boardAddDependencyFor", item.title));
        menu.append(depend);
      }
      if (canExplore) {
        const expanded = recordRoots.has(item.key);
        const label = expanded ? copy.text.boardHideRecords : copy.text.boardShowRecords;
        const trigger = control(label, () => {
          close();
          if (recordRoots.has(item.key)) recordRoots.delete(item.key);
          else {
            recordRoots.add(item.key);
            arrangeContextRecords(item, records);
          }
          render();
          if (recordRoots.has(item.key) && view() === "map")
            showRecordInViewport([...records][0]);
          const origin = view() === "list"
            ? [...list.querySelectorAll("[data-canvas-list-node]")].find((entry) => entry.dataset.canvasListNode === item.key)
            : nodeButtons.get(item.key)?.closest(".canvas-node");
          origin?.querySelector(".canvas-add")?.focus({ preventScroll: true });
        }, "quiet canvas-records");
        trigger.setAttribute("aria-expanded", String(expanded));
        trigger.setAttribute("aria-label", `${label}: ${item.title}. ${message("boardLinkedRecords", records.size)}`);
        if (records.size > MAX_CONTEXT_RECORDS) trigger.title = message("boardContextPreview", MAX_CONTEXT_RECORDS);
        menu.append(trigger);
      }
      actions.append(add, menu);
    }
    return actions;
  }
  function arrangeContextRecords(source, records) {
    const byKey = new Map(model.nodes.map((item) => [item.key, item]));
    const occupied = [...positions].map(([key, point]) => ({ ...point, ...canvasNodeSize(byKey.get(key)?.kind) }));
    let y = source.y;
    for (const key of [...records].slice(0, MAX_CONTEXT_RECORDS)) {
      if (positions.has(key)) continue;
      const candidate = { x: source.x + source.width + 48, y, ...canvasNodeSize(byKey.get(key)?.kind) };
      let collision;
      do {
        collision = occupied.find((other) => candidate.x < other.x + other.width + 16 && candidate.x + candidate.width + 16 > other.x && candidate.y < other.y + other.height + 16 && candidate.y + candidate.height + 16 > other.y);
        if (collision) candidate.y = collision.y + collision.height + 32;
      } while (collision);
      positions.set(key, { x: candidate.x, y: candidate.y });
      occupied.push(candidate);
      y = candidate.y + candidate.height + 32;
    }
  }
  function showRecordInViewport(key) {
    const record = layout.byKey.get(key);
    if (!record) return;
    const scale = uiUnit() * camera.zoom, area = cameraArea(), margin = 16;
    const left = camera.x + record.x * scale, top = camera.y + record.y * scale;
    const right = left + record.width * scale, bottom = top + record.height * scale;
    if (left < margin) camera.x += margin - left;
    else if (right > area.width - margin) camera.x -= right - (area.width - margin);
    if (top < margin) camera.y += margin - top;
    else if (bottom > area.height - margin) camera.y -= bottom - (area.height - margin);
    cameraPaint();
  }
  function render() {
    if (disposed || !model) return;
    const focused = document.activeElement;
    const focusedCard = focused?.closest?.(
      "[data-canvas-node], [data-canvas-list-node]",
    );
    const focusedKey =
      focusedCard?.dataset.canvasNode || focusedCard?.dataset.canvasListNode;
    const focusedEdge = focused?.closest?.("[data-edge-key]")?.dataset.edgeKey;
    const focusedAction = [
      "canvas-expand",
      "canvas-add",
      "canvas-connect",
      "canvas-records",
    ].find((name) => focused?.classList.contains(name));
    recordsFor = canvasRecordExplorer(model);
    const projection = canvasWorkLayer(model, { layer, recordRoots, selectedKey, selectedEdgeKey, recordsFor });
    hiddenRecordCount = projection.hiddenRecordCount;
    layout = layoutCanvas(projection.model, { collapsed, positions });
    const indicators = canvasNodeIndicators(model);
    for (const item of layout.nodes)
      positions.set(item.key, { x: item.x, y: item.y });
    world.style.width = unit(layout.width);
    world.style.height = unit(layout.height);
    nodeButtons = new Map();
    world.replaceChildren(edges);
    list.replaceChildren();
    for (const item of layout.nodes) {
      const article = node("article", undefined, "canvas-node");
      article.dataset.canvasNode = item.key;
      article.dataset.kind = item.kind;
      article.dataset.paper = paperFor(item.kind)[1];
      article.dataset.cardFacts = String(Boolean(item.cardSummary));
      article.dataset.state = item.displayState || item.lifecycle || "";
      article.dataset.selected = String(item.key === selectedKey);
      article.setAttribute("role", "group");
      article.setAttribute(
        "aria-label",
        `${kindName(item.kind)}: ${item.title}`,
      );
      Object.assign(article.style, {
        left: unit(item.x),
        top: unit(item.y),
        width: unit(item.width),
        height: unit(item.height),
      });
      const paper = node("img", undefined, "canvas-node-paper");
      paper.alt = "";
      paper.setAttribute("aria-hidden", "true");
      paper.draggable = false;
      paper.width = paperFor(item.kind)[0] === "square" ? 800 : 1000;
      paper.height = paperFor(item.kind)[0] === "square" ? 800 : 730;
      paper.src = paperSource(item.kind, item.key === selectedKey);
      const content = node("div", undefined, "canvas-node-content");
      const main = control(
        "",
        (event) => {
          if (!event.detail || performance.now() > ignoreClickUntil)
            select(item);
        },
        "canvas-node-main",
      );
      main.dataset.key = item.key;
      main.setAttribute(
        "aria-label",
        `${kindName(item.kind)}: ${item.title}${item.displayState || item.lifecycle ? ", " + display(item.displayState || item.lifecycle) : ""}${indicators.get(item.key)?.description ? ". " + indicators.get(item.key).description : ""}`,
      );
      main.setAttribute("aria-pressed", String(item.key === selectedKey));
      main.append(
        node("span", kindName(item.kind), "canvas-node-type"),
        node("span", item.title || "Untitled item", "canvas-node-title"),
      );
      const meta = node("span", undefined, "canvas-node-meta");
      const card = canvasCardPresentation(item);
      if (item.displayState || item.lifecycle)
        meta.append(badge(item.displayState || item.lifecycle));
      if (item.priority && !item.cardSummary)
        meta.append(
          node("span", display(item.priority), "canvas-node-priority"),
        );
      main.append(meta);
      if (card.required) meta.append(node("span", card.required, "canvas-node-obligation"));
      if (card.person) main.append(node("span", card.person, "canvas-node-person"));
      if (card.summary) main.append(node("span", card.summary, "canvas-node-summary"));
      else if (card.description) main.append(node("span", card.description, "canvas-node-summary"));
      const indicator = indicators.get(item.key);
      if (indicator?.label) {
        const summary = node("span", indicator.label, "canvas-node-indicators");
        summary.title = indicator.description;
        main.append(summary);
      }
      if (item.cardSummary) main.setAttribute("aria-label", `${main.getAttribute("aria-label")}. ${[card.required, card.person, card.summary, message("boardCanonicalLifecycle", display(card.lifecycle))].filter(Boolean).join(". ")}`);
      main.onkeydown = (event) => nodeKey(event, item);
      content.append(main, nodeActions(item));
      article.append(paper, content);
      world.append(article);
      nodeButtons.set(item.key, main);
      const row = node("li", undefined, "canvas-list-row");
      row.dataset.canvasListNode = item.key;
      row.style.setProperty("--canvas-depth", String(item.depth));
      const listMain = control("", () => select(item), "canvas-list-main");
      listMain.dataset.key = item.key;
      listMain.setAttribute("aria-pressed", String(item.key === selectedKey));
      listMain.append(
        node("span", kindName(item.kind), "canvas-node-type"),
        node("span", item.title || "Untitled item", "canvas-node-title"),
      );
      if (item.displayState || item.lifecycle)
        listMain.append(badge(item.displayState || item.lifecycle));
      if (card.required) listMain.append(node("span", card.required, "canvas-node-obligation"));
      if (card.person) listMain.append(node("span", card.person, "canvas-node-person"));
      if (card.summary) listMain.append(node("span", card.summary, "canvas-node-summary"));
      else if (card.description) listMain.append(node("span", card.description, "canvas-node-summary"));
      if (indicator?.label) {
        const summary = node("span", indicator.label, "canvas-node-indicators");
        summary.title = indicator.description;
        listMain.append(summary);
        listMain.setAttribute(
          "aria-label",
          `${kindName(item.kind)}: ${item.title}${item.displayState || item.lifecycle ? ", " + display(item.displayState || item.lifecycle) : ""}. ${indicator.description}`,
        );
      }
      if (item.cardSummary) listMain.setAttribute("aria-label", `${listMain.getAttribute("aria-label") || `${kindName(item.kind)}: ${item.title}`}. ${[card.required, card.person, card.summary, message("boardCanonicalLifecycle", display(card.lifecycle))].filter(Boolean).join(". ")}`);
      row.append(listMain, nodeActions(item));
      list.append(row);
    }
    renderEdges();
    announce();
    paintView();
    cameraPaint();
    if (focusedKey) {
      const container =
        view() === "list"
          ? [...list.querySelectorAll("[data-canvas-list-node]")].find(
              (entry) => entry.dataset.canvasListNode === focusedKey,
            )
          : nodeButtons.get(focusedKey)?.closest(".canvas-node");
      const target =
        (focusedAction && container?.querySelector(`.${focusedAction}`)) ||
        container?.querySelector(".canvas-node-main, .canvas-list-main");
      target?.focus({ preventScroll: true });
    } else if (focusedEdge) {
      [...edges.querySelectorAll("[data-edge-key]")]
        .find((entry) => entry.dataset.edgeKey === focusedEdge)
        ?.focus({ preventScroll: true });
    }
  }
  function renderEdges() {
    if (!layout) return;
    const focusedEdge =
      document.activeElement?.closest?.("[data-edge-key]")?.dataset.edgeKey;
    edges.replaceChildren();
    edges.setAttribute(
      "viewBox",
      `${layout.bounds.x} ${layout.bounds.y} ${layout.width} ${layout.height}`,
    );
    edges.style.left = unit(layout.bounds.x);
    edges.style.top = unit(layout.bounds.y);
    edges.style.width = unit(layout.width);
    edges.style.height = unit(layout.height);
    const defs = document.createElementNS(SVG, "defs"),
      marker = document.createElementNS(SVG, "marker"),
      arrow = document.createElementNS(SVG, "path");
    marker.id = id + "-arrow";
    for (const [key, value] of Object.entries({
      viewBox: "0 0 10 10",
      refX: "9",
      refY: "5",
      markerWidth: "6",
      markerHeight: "6",
      orient: "auto-start-reverse",
    }))
      marker.setAttribute(key, value);
    arrow.setAttribute("d", "M 0 0 L 10 5 L 0 10 z");
    arrow.setAttribute("fill", "context-stroke");
    marker.append(arrow);
    defs.append(marker);
    edges.append(defs);
    const visibleEdges = canvasVisibleEdges(
      layout.edges,
      selectedKey,
      edgeFilter.value,
    );
    const selectedEdge = layout.edges.find(
      (entry) => entry.key === selectedEdgeKey,
    );
    if (selectedEdge && !visibleEdges.includes(selectedEdge))
      visibleEdges.push(selectedEdge);
    for (const edge of visibleEdges) {
      const source = layout.byKey.get(edge.sourceKey),
        target = layout.byKey.get(edge.targetKey);
      const forward = target.x >= source.x;
      const sx = source.x + (forward ? source.width : 0),
        sy = source.y + source.height / 2;
      const sameColumn = Math.abs(source.x - target.x) < 1;
      const tx =
          target.x + (sameColumn ? target.width : forward ? 0 : target.width),
        ty = target.y + target.height / 2;
      const bend = Math.max(44, Math.abs(tx - sx) / 2);
      const group = document.createElementNS(SVG, "g");
      group.classList.add("canvas-edge");
      group.dataset.canvasEdge = edge.kind;
      group.dataset.edgeKey = edge.key;
      group.dataset.kind = edge.kind;
      group.dataset.mode = group.dataset.strength = edge.mode || "unknown";
      group.dataset.selected = String(
        edge.key === selectedEdgeKey ||
          edge.sourceKey === selectedKey ||
          edge.targetKey === selectedKey,
      );
      group.setAttribute("role", "button");
      group.setAttribute("tabindex", "0");
      const description = edge.kind === "depends_on"
        ? `${source.title} depends on ${target.title}: ${canvasEdgeLabel(edge)}`
        : `${source.title} → ${target.title}: ${canvasEdgeLabel(edge)}`;
      group.setAttribute(
        "aria-label",
        description,
      );
      const activate = () => {
        selectedEdgeKey = edge.key;
        connectionDescription = `${description}.`;
        renderEdges();
        announce();
        if (edge.ref) {
          if (options.onSelectEdge) invoke(options.onSelectEdge, edge);
          else
            invoke(options.onSelect, edge.ref, {
              ...edge,
              kind: edge.ref.kind,
              title: canvasEdgeLabel(edge),
              ref: edge.ref,
              edge: true,
            });
        }
      };
      group.onclick = activate;
      group.onkeydown = (event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          activate();
        }
      };
      const title = document.createElementNS(SVG, "title");
      title.textContent = description;
      const path = document.createElementNS(SVG, "path");
      path.classList.add("canvas-edge-line");
      path.setAttribute(
        "d",
        source.key === target.key
          ? `M ${sx} ${sy - 12} C ${sx + 84} ${sy - 84}, ${sx + 84} ${sy + 84}, ${sx} ${sy + 12}`
          : sameColumn
            ? `M ${sx} ${sy} C ${sx + 72} ${sy}, ${tx + 72} ${ty}, ${tx} ${ty}`
            : `M ${sx} ${sy} C ${sx + (forward ? bend : -bend)} ${sy}, ${tx + (forward ? -bend : bend)} ${ty}, ${tx} ${ty}`,
      );
      path.setAttribute("fill", "none");
      path.setAttribute("marker-end", `url(#${id}-arrow)`);
      group.append(title, path);
      const hit = path.cloneNode();
      hit.classList.remove("canvas-edge-line");
      hit.classList.add("canvas-edge-hit");
      hit.removeAttribute("marker-end");
      group.append(hit);
      if (group.dataset.selected === "true") {
        const label = document.createElementNS(SVG, "text");
        label.classList.add("canvas-edge-label");
        label.textContent = canvasEdgeLabel(edge);
        label.setAttribute("x", String((sx + tx) / 2));
        label.setAttribute("y", String((sy + ty) / 2 - 8));
        label.setAttribute("text-anchor", "middle");
        group.append(label);
      }
      edges.append(group);
    }
    if (focusedEdge)
      [...edges.querySelectorAll("[data-edge-key]")]
        .find((entry) => entry.dataset.edgeKey === focusedEdge)
        ?.focus({ preventScroll: true });
  }
  function nodeKey(event, item) {
    let key;
    if (event.key === "ArrowLeft") key = layout.parents.get(item.key);
    if (event.key === "ArrowRight")
      key = layout.children
        .get(item.key)
        ?.find((child) => layout.byKey.has(child.key))?.key;
    const index = layout.nodes.findIndex((entry) => entry.key === item.key);
    if (event.key === "ArrowDown") key = layout.nodes[index + 1]?.key;
    if (event.key === "ArrowUp") key = layout.nodes[index - 1]?.key;
    if (event.key === "Home") key = model.rootKey || layout.nodes[0]?.key;
    if (key) {
      event.preventDefault();
      event.stopPropagation();
      reveal(key);
    }
  }
  function closeSearch({ preserveIntent = false } = {}) {
    clearTimeout(searchTimer);
    searchAbort?.abort();
    ++searchVersion;
    searchResults.hidden = true;
    search.setAttribute("aria-expanded", "false");
    search.removeAttribute("aria-activedescendant");
    candidates = [];
    activeCandidate = -1;
    if (!preserveIntent) searchIntent = false;
  }
  function closeMenus(restoreFocus = false) {
    for (const menu of root.querySelectorAll(".canvas-node-menu")) {
      const trigger = menu.parentElement.querySelector(".canvas-add");
      if (restoreFocus && menu.contains(document.activeElement))
        trigger?.focus({ preventScroll: true });
      menu.hidden = true;
      trigger?.setAttribute("aria-expanded", "false");
    }
  }
  function allCandidates() {
    return [
      ...new Map(
        [...searchLocal, ...searchRemote].map((item) => [item.key, item]),
      ).values(),
    ];
  }
  function showCandidates() {
    const all = allCandidates();
    candidates = all.slice(0, searchShown);
    searchList.replaceChildren();
    for (const [index, item] of candidates.entries()) {
      const option = control(
        `${kindName(item.kind)} · ${item.title}`,
        () => chooseCandidate(index),
        "canvas-search-option",
      );
      option.setAttribute("role", "option");
      option.tabIndex = -1;
      option.setAttribute("aria-selected", String(index === activeCandidate));
      option.id = `${id}-option-${index}`;
      searchList.append(option);
    }
    searchMore.hidden = searchShown >= all.length && !searchCursor;
    searchMore.disabled = searchLoading;
    searchStatus.textContent = searchLoading
      ? "Searching…"
      : `${candidates.length} matching items shown${searchCursor || searchShown < all.length ? "; more results available" : ""}.`;
    searchResults.hidden = false;
    search.setAttribute("aria-expanded", "true");
  }
  function chooseCandidate(index) {
    const item = candidates[index];
    if (!item) return;
    closeSearch();
    search.value = "";
    select(item, { focus: true });
  }
  async function searchItems() {
    const query = search.value.trim(),
      version = ++searchVersion;
    searchAbort?.abort();
    searchAbort = new AbortController();
    if (!query) {
      closeSearch();
      return;
    }
    searchQuery = query;
    searchShown = 12;
    searchRemote = [];
    searchCursor = "";
    activeCandidate = -1;
    searchLocal = model.nodes.filter((item) =>
      `${item.title} ${kindName(item.kind)}`
        .toLocaleLowerCase("en-US")
        .includes(query.toLocaleLowerCase("en-US")),
    );
    searchLoading = Boolean(options.onSearch);
    showCandidates();
    if (options.onSearch)
      await fetchSearchPage(query, "", version, searchAbort.signal);
  }
  async function fetchSearchPage(query, cursor, version, signal) {
    try {
      const remote = await options.onSearch(query, { signal, cursor });
      if (!disposed && version === searchVersion && !signal.aborted) {
        searchRemote.push(
          ...(Array.isArray(remote) ? remote : remote?.items || []),
        );
        searchCursor = Array.isArray(remote) ? "" : remote?.next_cursor || "";
        searchLoading = false;
        showCandidates();
      }
    } catch (error) {
      if (error?.name !== "AbortError" && version === searchVersion) {
        searchLoading = false;
        showCandidates();
        report(error);
      }
    }
  }
  async function loadMoreSearch() {
    if (searchLoading) return;
    if (searchShown < allCandidates().length) {
      searchShown += 12;
      showCandidates();
      return;
    }
    if (!searchCursor || !options.onSearch) return;
    searchShown += 12;
    searchLoading = true;
    showCandidates();
    await fetchSearchPage(
      searchQuery,
      searchCursor,
      searchVersion,
      searchAbort.signal,
    );
  }
  search.addEventListener(
    "input",
    () => {
      // Invalidate immediately, rather than after the debounce. An earlier
      // query must not replace results while the user is typing a new one.
      closeSearch();
      searchIntent = Boolean(search.value.trim());
      searchTimer = setTimeout(searchItems, 220);
    },
    { signal: abort.signal },
  );
  search.addEventListener(
    "keydown",
    (event) => {
      if (event.key === "Escape") {
        closeSearch();
        return;
      }
      if (["ArrowDown", "ArrowUp"].includes(event.key) && candidates.length) {
        event.preventDefault();
        activeCandidate = clamp(
          activeCandidate + (event.key === "ArrowDown" ? 1 : -1),
          0,
          candidates.length - 1,
        );
        for (const [index, option] of [...searchList.children].entries())
          option.setAttribute(
            "aria-selected",
            String(index === activeCandidate),
          );
        search.setAttribute(
          "aria-activedescendant",
          `${id}-option-${activeCandidate}`,
        );
      }
      if (
        event.key === "Enter" &&
        (activeCandidate >= 0 || candidates.length === 1)
      ) {
        event.preventDefault();
        chooseCandidate(Math.max(0, activeCandidate));
      }
    },
    { signal: abort.signal },
  );
  edgeFilter.addEventListener("change", renderEdges, { signal: abort.signal });
  root.addEventListener(
    "keydown",
    (event) => {
      if (event.key === "Escape") {
        closeSearch();
        closeMenus(true);
      }
    },
    { signal: abort.signal },
  );
  root.addEventListener(
    "click",
    (event) => {
      if (!event.target.closest(".canvas-node-actions")) closeMenus();
      if (!searchWrap.contains(event.target)) closeSearch();
    },
    { signal: abort.signal },
  );
  viewport.addEventListener(
    "keydown",
    (event) => {
      if (event.target !== viewport) return;
      if (event.key === "+" || event.key === "=") {
        event.preventDefault();
        zoom(1.2);
      }
      if (event.key === "-") {
        event.preventDefault();
        zoom(1 / 1.2);
      }
      if (event.key === "0") {
        event.preventDefault();
        fit();
      }
      const movement = {
        ArrowLeft: [48, 0],
        ArrowRight: [-48, 0],
        ArrowUp: [0, 48],
        ArrowDown: [0, -48],
      }[event.key];
      if (movement) {
        event.preventDefault();
        camera.x += movement[0];
        camera.y += movement[1];
        cameraPaint();
      }
    },
    { signal: abort.signal },
  );
  viewport.addEventListener(
    "wheel",
    (event) => {
      // Leave native controls and any scrollable record content alone. The
      // map owns its background/card surface, not the inspector or toolbar.
      if (
        gesture ||
        event.target.closest(
          ".canvas-node-actions, button:not(.canvas-node-main), a[href], input, select, textarea, [contenteditable]:not([contenteditable=false])",
        )
      )
        return;
      for (
        let element = event.target;
        element && element !== viewport;
        element = element.parentElement
      ) {
        const style = getComputedStyle(element);
        if (
          (/auto|scroll/.test(style.overflowY) &&
            element.scrollHeight > element.clientHeight) ||
          (/auto|scroll/.test(style.overflowX) &&
            element.scrollWidth > element.clientWidth)
        )
          return;
      }
      event.preventDefault();
      const deltaUnit =
        event.deltaMode === WheelEvent.DOM_DELTA_LINE
          ? 16
          : event.deltaMode === WheelEvent.DOM_DELTA_PAGE
            ? viewport.clientHeight
            : 1;
      if (event.ctrlKey || event.metaKey) {
        const rect = viewport.getBoundingClientRect();
        zoom(Math.exp(-event.deltaY * deltaUnit * 0.003), {
          x: event.clientX - rect.left,
          y: event.clientY - rect.top,
        });
      } else {
        camera.x -= event.deltaX * deltaUnit;
        camera.y -= event.deltaY * deltaUnit;
        cameraPaint();
      }
    },
    { signal: abort.signal, passive: false },
  );
  viewport.addEventListener(
    "pointerdown",
    (event) => {
      if (
        event.button !== 0 ||
        event.target.closest(".canvas-node-actions, .canvas-edge[role=button]")
      )
        return;
      // A new press is a new intention. Only the trailing click of the
      // preceding drag is suppressed; its timer must not swallow this click.
      ignoreClickUntil = 0;
      const card = event.target.closest("[data-canvas-node]");
      const item = card && layout.byKey.get(card.dataset.canvasNode);
      gesture = {
        pointerId: event.pointerId,
        startX: event.clientX,
        startY: event.clientY,
        camera: { ...camera },
        item,
        position: item && { x: item.x, y: item.y },
        moved: false,
      };
      if (!card) {
        viewport.focus({ preventScroll: true });
        event.preventDefault();
      }
    },
    { signal: abort.signal },
  );
  viewport.addEventListener(
    "pointermove",
    (event) => {
      if (!gesture || gesture.pointerId !== event.pointerId) return;
      const dx = event.clientX - gesture.startX,
        dy = event.clientY - gesture.startY;
      if (!gesture.moved && Math.hypot(dx, dy) < 4) return;
      gesture.moved = true;
      viewport.setPointerCapture(event.pointerId);
      root.dataset.interacting = "true";
      if (gesture.item) {
        const scale = uiUnit() * camera.zoom;
        const position = {
          x: gesture.position.x + dx / scale,
          y: gesture.position.y + dy / scale,
        };
        positions.set(gesture.item.key, position);
        manualPositions.set(gesture.item.key, position);
        Object.assign(gesture.item, position);
        const card = nodeButtons.get(gesture.item.key)?.closest(".canvas-node");
        if (card) {
          card.style.left = unit(position.x);
          card.style.top = unit(position.y);
        }
        renderEdges();
      } else {
        camera.x = gesture.camera.x + dx;
        camera.y = gesture.camera.y + dy;
        cameraPaint();
      }
    },
    { signal: abort.signal },
  );
  function endGesture(event) {
    if (!gesture || gesture.pointerId !== event.pointerId) return;
    const moved = gesture.moved;
    if (moved) ignoreClickUntil = performance.now() + 250;
    if (viewport.hasPointerCapture(event.pointerId))
      viewport.releasePointerCapture(event.pointerId);
    gesture = null;
    root.dataset.interacting = "false";
    if (queued) {
      const pending = queued;
      queued = null;
      update(...pending);
    } else if (moved) render();
  }
  // A pointer can leave before the movement threshold enables capture. Always
  // release that pending gesture, otherwise subsequent model updates queue
  // forever after an ordinary press/release outside the viewport.
  window.addEventListener("pointerup", endGesture, { signal: abort.signal });
  window.addEventListener("pointercancel", endGesture, {
    signal: abort.signal,
  });
  compact.addEventListener("change", paintView, { signal: abort.signal });
  let previousViewport = { width: 0, height: 0, unit: 1 };
  const resize = new ResizeObserver(() => {
    if (disposed) return;
    const current = {
      width: viewport.clientWidth,
      height: viewport.clientHeight,
      unit: uiUnit(),
    };
    const item = layout?.byKey.get(selectedKey);
    if (
      item &&
      previousViewport.width &&
      current.width &&
      (current.width !== previousViewport.width ||
        current.height !== previousViewport.height)
    ) {
      const visibleAt = (size) =>
        camera.x + item.x * size.unit * camera.zoom >= 0 &&
        camera.y + item.y * size.unit * camera.zoom >= 0 &&
        camera.x + (item.x + item.width) * size.unit * camera.zoom <=
          size.width &&
        camera.y + (item.y + item.height) * size.unit * camera.zoom <=
          size.height;
      // Preserve a user's intentional pan. Recenter only a previously visible
      // selection that an opening inspector or viewport resize would cut off.
      if (visibleAt(previousViewport) && !visibleAt(current) && !gesture)
        reveal(item.key);
    }
    previousViewport = current;
    cameraPaint();
  });
  resize.observe(viewport);

  function update(next, selection = {}) {
    if (disposed) return;
    if (gesture) {
      queued = [next, selection];
      return;
    }
    const scopeChanged = !model || model.scopeKey !== next.scopeKey;
    const revisionChanged = model && model.revision !== next.revision;
    const planningOpened = model && !model.nodes.some((entry) => entry.roadmapRef) && next.nodes.some((entry) => entry.roadmapRef);
    if (!scopeChanged && planningOpened) {
      // Loading a published revision or draft is an explicit exploration
      // request. Keep its planning cards available in the records layer.
      layer = "records";
      layerSelect.value = layer;
    }
    const hasSelection = Object.hasOwn(selection, "selectedKey");
    let incomingSelectedKey = selection.selectedKey;
    if (
      !scopeChanged &&
      hasSelection &&
      incomingSelectedKey &&
      !next.nodes.some((item) => item.key === incomingSelectedKey)
    ) {
      const current = next.nodes.find((item) => item.key === selectedKey);
      // A planning card opens its real Roadmap in the inspector. Keep that
      // clicked presentation selected when the Roadmap is not itself a graph
      // node; never manufacture an EntityRef or carry it into another scope.
      if (
        current?.ref === null &&
        current.roadmapRef?.kind === "roadmap" &&
        incomingSelectedKey === `roadmap:${current.roadmapRef.id}`
      )
        incomingSelectedKey = current.key;
    }
    const resumeSearch =
      !scopeChanged &&
      revisionChanged &&
      searchIntent &&
      Boolean(search.value.trim()) &&
      (document.activeElement === search || !searchResults.hidden);
    const selectionChanged =
      hasSelection && selectedKey !== incomingSelectedKey;
    if (scopeChanged) {
      positions = new Map();
      manualPositions = new Map();
      opened = new Set();
      collapsed = new Set();
      recordRoots = new Set();
      layer = options.initialLayer === "records" ? "records" : "work";
      layerSelect.value = layer;
      camera = { x: 24, y: 24, zoom: 1 };
      cameraInitialized = false;
      selectedEdgeKey = null;
      connectionDescription = "";
      first = true;
      search.value = "";
      clearTimeout(searchTimer);
      closeSearch();
      searchAbort?.abort();
      ++searchVersion;
      if ((next.nodes?.length || 0) > 40)
        for (const item of next.nodes || [])
          if (item.kind === "objective" || item.kind === "phase")
            collapsed.add(item.key);
    }
    const knownParents = new Set(next.nodes.map((item) => item.parentKey));
    for (const item of next.nodes)
      if (
        item.expandable &&
        !knownParents.has(item.key) &&
        !opened.has(item.key)
      )
        collapsed.add(item.key);
    if (revisionChanged) closeSearch({ preserveIntent: resumeSearch });
    model = next;
    if (hasSelection) selectedKey = incomingSelectedKey;
    if (selectedKey && !model.nodes.some((item) => item.key === selectedKey)) {
      const edge = model.edges.find(
        (entry) =>
          entry.ref && `${entry.ref.kind}:${entry.ref.id}` === selectedKey,
      );
      selectedEdgeKey = edge?.key || null;
      selectedKey = null;
    }
    if (hasSelection && !incomingSelectedKey) selectedEdgeKey = null;
    if (selectedKey && (scopeChanged || selectionChanged)) {
      const byKey = new Map(model.nodes.map((item) => [item.key, item]));
      let key = selectedKey;
      const seen = new Set();
      while (byKey.has(key) && !seen.has(key)) {
        seen.add(key);
        const parent = byKey.get(key).parentKey;
        if (!parent) break;
        collapsed.delete(parent);
        key = parent;
      }
    }
    render();
    // A coherent refresh may arrive between typing and the debounce, or while
    // a query is running. Preserve that active intent against the new model;
    // Escape, selection and outside clicks deliberately leave it closed.
    if (resumeSearch) searchTimer = setTimeout(searchItems, 0);
    if (first) {
      first = false;
      requestAnimationFrame(() => {
        if (!disposed) fit(true);
      });
    }
  }
  update(options.model, { selectedKey });
  return {
    update,
    focus: reveal,
    fit,
    dispose() {
      disposed = true;
      abort.abort();
      resize.disconnect();
      clearTimeout(searchTimer);
      searchAbort?.abort();
      // The inspector owns persistent native dialog/form nodes. Return its
      // original host before removing this renderer, without cloning state.
      if (inspectorHost && inspectorParent)
        inspectorParent.insertBefore(
          inspectorHost,
          inspectorNext?.parentNode === inspectorParent ? inspectorNext : null,
        );
      root.remove();
    },
  };
}
