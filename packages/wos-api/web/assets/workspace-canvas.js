import { node, badge, kindName, display } from "./presentation.js";

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
        : 284;
  return {
    width,
    height: paperFor(kind)[0] === "square" ? width : (width * 730) / 1000,
  };
}

const compare = (a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0);
const clamp = (value, min, max) => Math.min(max, Math.max(min, value));
const unit = (value) => `calc(${value} * var(--wos-unit, 1px))`;

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
        x: Math.max(16, saved.x),
        y: Math.max(16, saved.y),
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
  return {
    nodes,
    byKey: placed,
    parents,
    children,
    edges: (model.edges || []).filter(
      (edge) => placed.has(edge.sourceKey) && placed.has(edge.targetKey),
    ),
    hiddenCount: sorted.length - nodes.length,
    width: Math.max(600, ...nodes.map((item) => item.x + item.width + 112)),
    height: Math.max(400, ...nodes.map((item) => item.y + item.height + 48)),
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
  if (edge.label) return edge.label;
  if (edge.kind === "depends_on")
    return `Depends on · ${edge.mode === "hard" ? "hard" : edge.mode === "advisory" ? "advisory" : "strength not loaded"}`;
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
  root.setAttribute("aria-label", "Outcome work map");
  const toolbar = node("div", undefined, "canvas-toolbar");
  const switcher = node("div", undefined, "canvas-view-switch");
  switcher.setAttribute("aria-label", "Work map view");
  const cameraControls = node("div", undefined, "canvas-camera-controls");
  const zoomOutput = node("output", "100%", "canvas-zoom");
  zoomOutput.setAttribute("aria-label", "Canvas zoom");
  const searchWrap = node("div", undefined, "canvas-search");
  const search = node("input");
  search.type = "search";
  search.placeholder = "Find an item…";
  search.setAttribute("aria-label", "Search canvas");
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
  const viewport = node("div", undefined, "canvas-viewport");
  viewport.tabIndex = 0;
  viewport.setAttribute("role", "region");
  viewport.setAttribute(
    "aria-label",
    "Interactive work map. Drag the background to pan; use arrow keys to move focus, plus and minus to zoom.",
  );
  const world = node("div", undefined, "canvas-world");
  const edges = document.createElementNS(SVG, "svg");
  edges.classList.add("canvas-edges");
  edges.setAttribute("role", "group");
  edges.setAttribute("aria-label", "Loaded connections");
  world.append(edges);
  viewport.append(world);
  const list = node("ul", undefined, "canvas-list");
  list.setAttribute("aria-label", "Work map items");
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
  root.append(toolbar, viewport, list, footer);
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
  let camera = { x: 24, y: 24, zoom: 1 },
    gesture = null,
    queued = null;
  let disposed = false,
    first = true,
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
  cameraControls.append(
    control("Zoom out", () => zoom(1 / 1.2)),
    zoomOutput,
    control("Zoom in", () => zoom(1.2)),
    control("Fit view", () => fit()),
    control("Auto-layout", () => {
      positions.clear();
      manualPositions.clear();
      render();
      fit();
    }),
  );
  toolbar.append(switcher, cameraControls, searchWrap, filterWrap);
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
    status.textContent = `${prefix}${model?.nodes?.length || 0} loaded items · ${model?.edges?.length || 0} loaded connections · revision ${model?.revision ?? "not loaded"}${layout?.hiddenCount ? ` · ${layout.hiddenCount} items inside collapsed groups` : ""}.${model?.boundary?.scopeLimited ? " Graph depth and query limits apply." : ""}${root.dataset.detail === "overview" ? " Overview scale: select an item for details, or use List view for full controls." : ""}${connectionDescription ? " " + connectionDescription : ""}${lastError ? " " + lastError : ""}`;
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
  function zoom(
    factor,
    point = { x: viewport.clientWidth / 2, y: viewport.clientHeight / 2 },
  ) {
    camera = zoomCanvasCamera(camera, factor, point);
    cameraPaint();
  }
  function fit(initial = false) {
    if (!layout || !viewport.clientWidth || !viewport.clientHeight) return;
    const scale = uiUnit();
    const width = layout.width * scale,
      height = layout.height * scale;
    const desired = Math.min(
      (viewport.clientWidth - 48) / width,
      (viewport.clientHeight - 48) / height,
      1,
    );
    camera.zoom = clamp(desired, initial ? 0.75 : MIN_ZOOM, 1);
    camera.x = Math.max(24, (viewport.clientWidth - width * camera.zoom) / 2);
    camera.y = Math.max(24, (viewport.clientHeight - height * camera.zoom) / 2);
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
    cameraPaint();
  }
  function changeView(next) {
    userView = next;
    paintView();
    if (next === "map" && selectedKey) reveal(selectedKey);
    invoke(options.onViewChange, next);
  }
  function paintView() {
    const mode = view();
    root.dataset.view = mode;
    viewport.hidden = mode !== "map";
    list.hidden = mode !== "list";
    cameraControls.hidden = mode !== "map";
    filterWrap.hidden = mode !== "map";
    mapButton.setAttribute("aria-pressed", String(mode === "map"));
    listButton.setAttribute("aria-pressed", String(mode === "list"));
    if (mode === "map") cameraPaint();
  }
  function select(item, { focus = false } = {}) {
    selectedKey = item.key;
    selectedEdgeKey = null;
    connectionDescription = "";
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
        const scale = uiUnit() * camera.zoom;
        camera.x =
          viewport.clientWidth / 2 -
          ((source.x + source.width / 2 + target.x + target.width / 2) / 2) *
            scale;
        camera.y =
          viewport.clientHeight / 2 -
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
    const item = layout.byKey.get(key);
    if (!item) return false;
    const scale = uiUnit() * camera.zoom;
    camera.x = viewport.clientWidth / 2 - (item.x + item.width / 2) * scale;
    camera.y = viewport.clientHeight / 2 - (item.y + item.height / 2) * scale;
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
      expand.setAttribute(
        "aria-label",
        `${item.collapsed || !item.childCount ? "Expand" : "Collapse"} ${item.title}`,
      );
      expand.setAttribute("aria-expanded", String(!item.collapsed));
      actions.append(expand);
    }
    if (item.canCreate?.length && options.onCreate) {
      const menu = node("div", undefined, "canvas-node-menu");
      menu.hidden = true;
      const add = control(
        "Add…",
        () => {
          const opening = menu.hidden;
          closeMenus();
          menu.hidden = !opening;
          add.setAttribute("aria-expanded", String(opening));
          if (opening) menu.querySelector("button")?.focus();
        },
        "quiet canvas-add",
      );
      add.setAttribute("aria-label", `Add an item within ${item.title}`);
      add.setAttribute("aria-expanded", "false");
      for (const kind of item.canCreate)
        menu.append(
          control(kind === "objective" ? "Add objective" : "New task", () => {
            menu.hidden = true;
            add.setAttribute("aria-expanded", "false");
            invoke(options.onCreate, kind, item.ref || null);
          }),
        );
      actions.append(add, menu);
    }
    if (item.canConnect && item.ref && options.onConnect) {
      const connect = control(
        "Connect",
        () =>
          invoke(options.onConnect, { sourceRef: item.ref, targetRef: null }),
        "quiet canvas-connect",
      );
      connect.setAttribute(
        "aria-label",
        `Connect ${item.title} using an explicit relationship`,
      );
      actions.append(connect);
    }
    return actions;
  }
  function render() {
    if (disposed || !model) return;
    const focused = document.activeElement;
    const focusedCard = focused?.closest?.(
      "[data-canvas-node], [data-canvas-list-node]",
    );
    const focusedKey =
      focusedCard?.dataset.canvasNode || focusedCard?.dataset.canvasListNode;
    const focusedAction = [
      "canvas-expand",
      "canvas-add",
      "canvas-connect",
    ].find((name) => focused?.classList.contains(name));
    layout = layoutCanvas(model, { collapsed, positions });
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
        `${kindName(item.kind)}: ${item.title}${item.displayState || item.lifecycle ? ", " + display(item.displayState || item.lifecycle) : ""}`,
      );
      main.setAttribute("aria-pressed", String(item.key === selectedKey));
      main.append(
        node("span", kindName(item.kind), "canvas-node-type"),
        node("span", item.title || "Untitled item", "canvas-node-title"),
      );
      const meta = node("span", undefined, "canvas-node-meta");
      if (item.displayState || item.lifecycle)
        meta.append(badge(item.displayState || item.lifecycle));
      if (item.priority)
        meta.append(
          node("span", display(item.priority), "canvas-node-priority"),
        );
      main.append(meta);
      if (item.kind === "outcome" && item.summary)
        main.append(node("span", item.summary, "canvas-node-summary"));
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
    }
  }
  function renderEdges() {
    if (!layout) return;
    const focusedEdge =
      document.activeElement?.closest?.("[data-edge-key]")?.dataset.edgeKey;
    edges.replaceChildren();
    edges.setAttribute("viewBox", `0 0 ${layout.width} ${layout.height}`);
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
      group.setAttribute(
        "aria-label",
        `${source.title} → ${target.title}: ${canvasEdgeLabel(edge)}`,
      );
      const activate = () => {
        selectedEdgeKey = edge.key;
        connectionDescription = `${source.title} → ${target.title}: ${canvasEdgeLabel(edge)}.`;
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
      title.textContent = `${source.title} → ${target.title}: ${canvasEdgeLabel(edge)}`;
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
      if (!(event.ctrlKey || event.metaKey)) return;
      event.preventDefault();
      const rect = viewport.getBoundingClientRect();
      zoom(Math.exp(-event.deltaY * 0.003), {
        x: event.clientX - rect.left,
        y: event.clientY - rect.top,
      });
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
          x: Math.max(16, gesture.position.x + dx / scale),
          y: Math.max(16, gesture.position.y + dy / scale),
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
      camera = { x: 24, y: 24, zoom: 1 };
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
      root.remove();
    },
  };
}
