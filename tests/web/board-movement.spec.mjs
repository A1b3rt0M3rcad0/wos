import {
  test, expect, openOutcome, clickWorkspaceTool,
} from "./ux-fixture.mjs";

const board = (page) => page.locator("#operational-canvas");
const viewport = (page) => board(page).locator(".canvas-viewport");
const world = (page) => board(page).locator(".canvas-world");
const record = (page, kind, id) =>
  board(page).locator(`[data-canvas-node="${kind}:${id}"]`);
const camera = (page) => world(page).evaluate((element) => {
  const matrix = new DOMMatrixReadOnly(getComputedStyle(element).transform);
  return { x: matrix.e, y: matrix.f, zoom: matrix.a };
});
const position = (card) => card.evaluate((element) => {
  const style = getComputedStyle(element);
  return { x: parseFloat(style.left), y: parseFloat(style.top) };
});

async function openBoard(page, wos) {
  const { value, scope } = await wos.outcome("Arrange the shared work freely");
  await wos.login(page);
  await openOutcome(page, value.title, { area: null });
  await expect(viewport(page)).toBeVisible();
  await expect(world(page)).toHaveCSS("transform", /matrix/);
  return { value, scope };
}

async function dragCard(page, card, dx, dy) {
  const box = await card.locator(".canvas-node-main").boundingBox();
  const start = { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  await page.mouse.move(start.x, start.y);
  await page.mouse.down();
  await page.mouse.move(start.x + dx, start.y + dy, { steps: 8 });
  await page.mouse.up();
}

async function refreshBoard(page) {
  const refreshedGraph = page.waitForResponse((response) =>
    response.request().method() === "GET" &&
    new URL(response.url()).pathname.endsWith("/graph"));
  await clickWorkspaceTool(page, "refresh");
  const response = await refreshedGraph;
  expect(response.ok()).toBe(true);
  await response.finished();
  // A response completes before the browser publishes its parsed model. Wait
  // for that render turn before checking preserved presentation positions.
  await page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))));
}

async function fittedCards(page) {
  await board(page).getByRole("button", { name: "Fit view", exact: true }).click();
  const geometry = await viewport(page).evaluate((element) => {
    const map = element.getBoundingClientRect();
    return [...element.querySelectorAll(".canvas-node")].map((card) => {
      const box = card.getBoundingClientRect();
      return {
        left: box.left - map.left, top: box.top - map.top,
        right: map.right - box.right, bottom: map.bottom - box.bottom,
      };
    });
  });
  for (const bounds of geometry)
    for (const clearance of Object.values(bounds))
      expect(clearance).toBeGreaterThanOrEqual(-1);
}

test("Board cards cross the original origin, survive refresh and resize, and fit their real extents", async ({ page, wos }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const { value, scope } = await openBoard(page, wos);
  const root = record(page, "outcome", value.id);
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/"))
      mutations.push(request.url());
  });
  const before = await position(root);
  expect(before.x).toBeGreaterThan(0);
  expect(before.y).toBeGreaterThan(0);
  await dragCard(page, root, -180, -160);
  const negative = await position(root);
  expect(negative.x).toBeLessThan(0);
  expect(negative.y).toBeLessThan(0);
  await fittedCards(page);
  const scale = (await camera(page)).zoom;
  await board(page).getByRole("button", { name: "Center view", exact: true }).click();
  expect((await camera(page)).zoom).toBe(scale);
  await refreshBoard(page);
  await expect.poll(() => position(root)).toEqual(negative);
  await page.setViewportSize({ width: 1280, height: 720 });
  await expect.poll(() => position(root)).toEqual(negative);
  await fittedCards(page);
  await dragCard(page, root, 340, 280);
  const positive = await position(root);
  expect(positive.x).toBeGreaterThan(0);
  expect(positive.y).toBeGreaterThan(0);
  await fittedCards(page);
  // Fit must follow the cards after every card leaves the original origin,
  // rather than reserve an empty quadrant and leave the actual work offscreen.
  await refreshBoard(page);
  await expect.poll(() => position(root)).toEqual(positive);
  const objective = await wos.command("create_objective", {
    scope, title: "Load a newly recorded objective", priority: "normal",
    required_for_outcome: false,
  });
  await refreshBoard(page);
  // An initially empty expandable Outcome deliberately stays collapsed after
  // refresh; use its real disclosure rather than bypassing that user choice.
  await root.getByRole("button", { name: `Expand ${value.title}`, exact: true }).click();
  await expect(record(page, "objective", objective.id)).toHaveCount(1);
  await expect.poll(() => position(root)).toEqual(positive);
  await fittedCards(page);
  await dragCard(page, root, -450, -300);
  const connectedNegative = await position(root);
  expect(connectedNegative.x).toBeLessThan(0);
  expect(connectedNegative.y).toBeLessThan(0);
  await fittedCards(page);
  // Edges use the same signed coordinates as the cards, including hit paths.
  const edge = board(page).locator('.canvas-edge[data-kind="hierarchy"]').first();
  await expect(edge).toHaveCount(1);
  const edgeGeometry = await edge.evaluate((element) => {
    const path = element.querySelector(".canvas-edge-line"),
      svg = element.ownerSVGElement,
      linePoint = (length) => {
        const local = path.getPointAtLength(length), point = svg.createSVGPoint();
        point.x = local.x;
        point.y = local.y;
        const screen = point.matrixTransform(path.getScreenCTM());
        return { x: screen.x, y: screen.y };
      },
      source = element.closest(".canvas-world").querySelector('[data-kind="outcome"]').getBoundingClientRect(),
      target = element.closest(".canvas-world").querySelector('[data-kind="objective"]').getBoundingClientRect(),
      start = linePoint(0), end = linePoint(path.getTotalLength()),
      middle = linePoint(path.getTotalLength() / 2);
    return {
      width: element.getBoundingClientRect().width,
      sourceDelta: Math.abs(start.x - source.right) + Math.abs(start.y - (source.top + source.height / 2)),
      targetDelta: Math.abs(end.x - target.left) + Math.abs(end.y - (target.top + target.height / 2)),
      hit: document.elementFromPoint(middle.x, middle.y)?.closest("[data-edge-key]")?.dataset.edgeKey,
      key: element.dataset.edgeKey,
    };
  });
  expect(edgeGeometry.width).toBeGreaterThan(0);
  expect(edgeGeometry.sourceDelta).toBeLessThan(1);
  expect(edgeGeometry.targetDelta).toBeLessThan(1);
  expect(edgeGeometry.hit).toBe(edgeGeometry.key);
  await edge.press("Enter");
  await expect(edge).toBeFocused();
  expect(mutations).toEqual([]);
  expect(errors).toEqual([]);
});

test("trackpad, background drag and keyboard pan freely without changing cards or stealing native scroll", async ({ page, wos }) => {
  await page.setViewportSize({ width: 1366, height: 900 });
  const { value } = await openBoard(page, wos);
  const root = record(page, "outcome", value.id);
  const local = await position(root);
  const initial = await camera(page);
  const bounds = await viewport(page).boundingBox();
  await page.mouse.move(bounds.x + bounds.width - 30, bounds.y + 30);
  await page.mouse.wheel(900, 700);
  await expect.poll(() => camera(page)).toEqual({ ...initial, x: initial.x - 900, y: initial.y - 700 });
  await page.mouse.wheel(-1800, -1400);
  await expect.poll(() => camera(page)).toEqual({ ...initial, x: initial.x + 900, y: initial.y + 700 });
  await page.mouse.wheel(900, 700);
  await expect.poll(() => camera(page)).toEqual(initial);
  const point = { x: bounds.x + bounds.width - 30, y: bounds.y + 30 };
  await page.mouse.move(point.x, point.y);
  await page.mouse.down();
  await page.mouse.move(point.x - 150, point.y + 100, { steps: 5 });
  await page.mouse.up();
  await expect.poll(() => camera(page)).toEqual({ ...initial, x: initial.x - 150, y: initial.y + 100 });
  await viewport(page).focus();
  await page.keyboard.press("ArrowLeft");
  await expect.poll(() => camera(page)).toEqual({ ...initial, x: initial.x - 102, y: initial.y + 100 });
  await viewport(page).evaluate((element) => element.dispatchEvent(new WheelEvent("wheel", {
    bubbles: true, cancelable: true, deltaX: 2, deltaY: -3,
    deltaMode: WheelEvent.DOM_DELTA_LINE,
  })));
  await expect.poll(() => camera(page)).toEqual({ ...initial, x: initial.x - 134, y: initial.y + 148 });
  const current = await camera(page);
  await viewport(page).evaluate((element) => element.addEventListener("wheel", (event) => {
    const box = element.getBoundingClientRect();
    window.boardZoomAnchor = { x: event.clientX - box.left, y: event.clientY - box.top };
  }, { once: true, capture: true }));
  await page.mouse.move(point.x, point.y);
  await page.keyboard.down("Control");
  await page.mouse.wheel(0, -120);
  await page.keyboard.up("Control");
  await expect.poll(async () => (await camera(page)).zoom).toBeGreaterThan(current.zoom);
  const zoomed = await camera(page);
  const anchor = await page.evaluate(() => window.boardZoomAnchor);
  // getComputedStyle serializes matrix values to six significant digits.
  // Preserve the same board point within 0.01 unit despite that rounding.
  expect(Math.abs((anchor.x - zoomed.x) / zoomed.zoom - (anchor.x - current.x) / current.zoom)).toBeLessThan(0.01);
  expect(Math.abs((anchor.y - zoomed.y) / zoomed.zoom - (anchor.y - current.y) / current.zoom)).toBeLessThan(0.01);
  expect(await position(root)).toEqual(local);
  // The toolbar select retains its native wheel behavior; record inspection
  // owns its own scrolling and must never move the map underneath the reader.
  await board(page).getByRole("button", { name: "Fit view", exact: true }).click();
  await root.locator(".canvas-node-main").click();
  const panel = page.locator("#canvas-details");
  await expect(panel).toBeVisible();
  await page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))));
  const selected = await camera(page);
  await panel.hover();
  await page.mouse.wheel(0, 220);
  expect(await camera(page)).toEqual(selected);
  expect(await position(root)).toEqual(local);
  await page.keyboard.press("Escape");
  await board(page).getByRole("button", { name: "Fit view", exact: true }).click();
  const beforeControl = await camera(page);
  await root.locator(".canvas-add").hover();
  await page.mouse.wheel(0, 120);
  expect(await camera(page)).toEqual(beforeControl);
});
