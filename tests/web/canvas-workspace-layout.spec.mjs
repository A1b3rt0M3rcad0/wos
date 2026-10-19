import {
  test, expect, openOutcome, chooseArea, openViews, openCanvasContext,
} from "./ux-fixture.mjs";
import { seedCanvas, seedCanvasRoadmap } from "./canvas-fixture.mjs";
import { clickRecordAction } from "./workspace-action-fixture.mjs";
import { nativeZoomBrowser } from "./native-browser-zoom.mjs";
import AxeBuilder from "@axe-core/playwright";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { createHash } from "node:crypto";

const verifiedAssets = new WeakMap();
const sha256 = (value) => createHash("sha256").update(value).digest("hex");

for (const [width, height] of [[1280, 720], [1366, 768], [1024, 768], [390, 844], [360, 640], [320, 740]]) {
  test(`Compact login fits its complete default composition at ${width} x ${height}`, async ({ page, wos }) => {
    await page.setViewportSize({ width, height });
    await page.goto(`${wos.url}/app/`);
    await page.evaluate(() => document.fonts.ready);
    await verifyCaptureAssets(page, wos);
    const measurement = await entryBounds(page);
    await capture(page, `login-${width}x${height}`, { ...measurement, state: "default credential login; local access collapsed" });
    expect(measurement.document.width).toBeLessThanOrEqual(width);
    expect(measurement.document.height).toBeLessThanOrEqual(height);
    await fullyVisible(page.locator(".login-panel"));
    await fullyVisible(page.locator("#token"));
    await fullyVisible(page.locator("#login button"));
    expect(measurement.input.height).toBeGreaterThanOrEqual(44);
    expect(measurement.submit.height).toBeGreaterThanOrEqual(44);
    await page.locator("#token").focus();
    await page.keyboard.press("Tab");
    await expect(page.locator("#login button")).toBeFocused();
    await wos.login(page);
    await expect(page.locator("#connect")).toBeHidden();
  });
}

test("Expanded local login and enlarged text keep natural scrolling and whole controls", async ({ page, wos }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  await page.goto(`${wos.url}/app/`);
  await page.locator(".entry-local > summary").click();
  await page.locator("#local-ns").fill(wos.namespace);
  await page.locator("#local button").scrollIntoViewIfNeeded();
  await fullyVisible(page.locator("#local button"));
  await page.locator("html").evaluate((element) => { element.style.fontSize = "200%"; });
  await page.locator("#token").scrollIntoViewIfNeeded();
  await page.locator("#token").fill("a".repeat(32));
  await page.locator("#login button").scrollIntoViewIfNeeded();
  await fullyVisible(page.locator("#login button"));
  const bounds = await entryBounds(page);
  expect(bounds.document.width).toBeLessThanOrEqual(bounds.viewport.width);
  expect(bounds.document.height).toBeGreaterThan(bounds.viewport.height);
  await page.locator("#local button").scrollIntoViewIfNeeded();
  await fullyVisible(page.locator("#local button"));
});

const canvas = (page) => page.locator("#operational-canvas");
const viewport = (page) => canvas(page).locator(".canvas-viewport");
const panel = (page) => page.locator("#canvas-details");
const task = (page, value, view = "map") => canvas(page).locator(view === "map"
  ? `[data-canvas-node="work_item:${value.id}"] .canvas-node-main`
  : `[data-canvas-list-node="work_item:${value.id}"] .canvas-list-main`);

async function fullyVisible(locator) {
  await expect(locator).toBeVisible();
  const box = await locator.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    return { left: rect.left, top: rect.top, right: rect.right, bottom: rect.bottom, width: innerWidth, height: innerHeight };
  });
  expect(box.left).toBeGreaterThanOrEqual(-1);
  expect(box.top).toBeGreaterThanOrEqual(-1);
  expect(box.right).toBeLessThanOrEqual(box.width + 1);
  expect(box.bottom).toBeLessThanOrEqual(box.height + 1);
}
async function workspaceBounds(page) {
  await expect.poll(() => page.evaluate(() => ({
    x: scrollX, y: scrollY,
    excessX: document.documentElement.scrollWidth - innerWidth,
    excessY: document.documentElement.scrollHeight - innerHeight,
  }))).toEqual({ x: 0, y: 0, excessX: 0, excessY: 0 });
  const surface = await canvas(page).locator('.workspace-canvas[data-view="list"]').count()
    ? canvas(page).locator(".canvas-list") : viewport(page);
  const box = await surface.boundingBox();
  await fullyVisible(surface);
  const chrome = await page.evaluate(() => Object.fromEntries([
    ".topbar", ".outcome-heading", ".canvas-toolbar", ".canvas-footer", "#notice", ".workspace-main",
  ].map((selector) => {
    const element = document.querySelector(selector);
    const rect = element.getBoundingClientRect();
    return [selector, { top: rect.top, height: rect.height, bottom: rect.bottom }];
  })));
  const dimensions = await page.evaluate(() => ({
    width: innerWidth, height: innerHeight,
    devicePixelRatio, visualViewportScale: visualViewport.scale,
  }));
  return {
    width: box.width, height: box.height, heightRatio: box.height / dimensions.height,
    rect: { ...box, left: box.x, top: box.y, right: box.x + box.width, bottom: box.y + box.height },
    viewport: dimensions, chrome,
  };
}
async function entryBounds(page) {
  return page.evaluate(() => {
    const box = (selector) => {
      const r = document.querySelector(selector).getBoundingClientRect();
      return { x: r.x, y: r.y, width: r.width, height: r.height, top: r.top, bottom: r.bottom, left: r.left, right: r.right };
    };
    return {
      entry: true,
      viewport: { width: innerWidth, height: innerHeight, devicePixelRatio, visualViewportScale: visualViewport.scale },
      document: { width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight, scrollX, scrollY },
      panel: box(".login-panel"), story: box(".login-story"), input: box("#token"), submit: box("#login button"),
    };
  });
}
async function verifyCaptureAssets(page, wos) {
  if (!process.env.WOS_CAPTURE_CANVAS_FIRST) return;
  const entries = await Promise.all(["app.js", "canvas.css", "editorial.css", "en-US.js", "index.html", "workspace-canvas.js", "workspace-entry.css"].map(async (file) => {
    const [response, source] = await Promise.all([
      page.request.get(`${wos.url}/app/${file}`),
      readFile(new URL(`../../packages/wos-api/web/assets/${file}`, import.meta.url)),
    ]);
    expect(response.ok(), `Capture asset ${file} is served`).toBe(true);
    const actual = sha256(await response.body());
    expect(actual, `Capture refuses a stale embedded ${file}`).toBe(sha256(source));
    return [file, actual];
  }));
  verifiedAssets.set(page, Object.fromEntries(entries));
  await test.info().attach("verified-embedded-assets", { body: JSON.stringify(verifiedAssets.get(page), null, 2), contentType: "application/json" });
}
async function controls(page) {
  await fullyVisible(page.locator('#area-tabs [data-area="canvas"]'));
  await fullyVisible(page.locator("#outcome-views > summary"));
  await fullyVisible(canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }));
  for (const name of ["Zoom in", "Zoom out", "Center view", "Fit view"]) {
    const button = canvas(page).getByRole("button", { name, exact: true });
    await fullyVisible(button);
    expect(await button.evaluate((element) => {
      const r = element.getBoundingClientRect();
      return r.width >= 44 && r.height >= 44 && element.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2));
    }), `${name} has a whole, unobscured touch target`).toBe(true);
  }
}
async function exerciseCameraWithInspection(page) {
  const scale = () => viewport(page).evaluate((element) => Number(element.dataset.scale));
  const initialScale = await scale();
  await canvas(page).getByRole("button", { name: "Zoom in", exact: true }).click();
  expect(await scale()).toBeGreaterThan(initialScale);
  await canvas(page).getByRole("button", { name: "Zoom out", exact: true }).click();
  expect(await scale()).toBeCloseTo(initialScale, 4);
  await viewport(page).focus();
  await page.keyboard.press("ArrowRight");
  const panned = await canvas(page).locator(".canvas-world").getAttribute("style");
  await canvas(page).getByRole("button", { name: "Center view", exact: true }).click();
  expect(await scale()).toBeCloseTo(initialScale, 4);
  await expect(canvas(page).locator(".canvas-world")).not.toHaveAttribute("style", panned);
  const geometry = () => canvas(page).evaluate((element) => {
    const map = element.querySelector(".canvas-viewport").getBoundingClientRect();
    const world = element.querySelector(".canvas-world").getBoundingClientRect();
    const inspector = element.querySelector(".canvas-context-panel").getBoundingClientRect();
    const unit = parseFloat(getComputedStyle(document.documentElement).fontSize) / 14;
    const right = inspector.left < map.right ? inspector.left - 12 * unit : map.right;
    return { map: { left: map.left, right, top: map.top, bottom: map.bottom }, world: { left: world.left, right: world.right, top: world.top, bottom: world.bottom } };
  });
  let position = await geometry();
  expect((position.world.left + position.world.right) / 2).toBeCloseTo((position.map.left + position.map.right) / 2, 0);
  expect((position.world.top + position.world.bottom) / 2).toBeCloseTo((position.map.top + position.map.bottom) / 2, 0);
  await canvas(page).getByRole("button", { name: "Fit view", exact: true }).click();
  position = await geometry();
  expect(position.world.left).toBeGreaterThanOrEqual(position.map.left);
  expect(position.world.right).toBeLessThanOrEqual(position.map.right + 1);
  expect(position.world.top).toBeGreaterThanOrEqual(position.map.top);
  expect(position.world.bottom).toBeLessThanOrEqual(position.map.bottom + 1);
  await expect(panel(page)).toBeVisible();
  await panel(page).locator(".canvas-inspector-detail").focus();
}
async function focusedBoardOwnsNavigation(page) {
  const sidebar = page.locator(".sidebar");
  const board = page.locator('#area-tabs [data-area="canvas"]');
  await expect(sidebar).toBeHidden();
  await expect(page.getByRole("complementary", { name: "Workspace navigation", exact: true })).toHaveCount(0);
  expect(await sidebar.locator("button, input, select, a[href], [tabindex]").evaluateAll((controls) =>
    controls.every((control) => {
      control.focus();
      return !control.closest(".sidebar").contains(document.activeElement);
    }),
  )).toBe(true);
  await page.locator("#logout").focus();
  await page.keyboard.press("Tab");
  await expect(board).toBeFocused();
}
async function inspectorActionsRemainReachable(page) {
  const body = panel(page).locator("#detail-content");
  expect(await body.evaluate((element) => element.clientHeight)).toBeGreaterThan(48);
  const footer = panel(page).locator("#detail-actions");
  const actions = footer.locator("#entity-actions");
  // A long action list must scroll inside its own bounded footer. Merely
  // checking the outer panel box misses buttons clipped beneath that panel.
  await footer.evaluate((element) => { element.scrollTop = element.scrollHeight; });
  await actions.focus();
  await fullyVisible(actions);
  expect(await actions.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    const footer = element.closest("footer").getBoundingClientRect();
    const hit = document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2);
    return rect.top >= footer.top - 1 && rect.bottom <= footer.bottom + 1 && element.contains(hit);
  })).toBe(true);
  await workspaceBounds(page);
  await actions.click();
  await expect(panel(page).locator(".canvas-inspector-actions")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(panel(page).locator(".canvas-inspector-detail")).toBeVisible();
  await expect(actions).toBeFocused();
}
async function openCanvas(page, wos, data) {
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toHaveText("Outcome Board");
  await expect(page.locator("#outcome-tools")).toBeVisible();
  await expect(page.locator("#actions")).toBeVisible();
  await expect(page.locator("#refresh")).toHaveCount(1);
  await expect(page.locator("#outcome-tools #refresh")).toHaveCount(1);
  await expect(canvas(page)).toHaveAttribute("aria-label", "Outcome Board workspace");
  await expect(canvas(page).locator(".workspace-canvas")).toBeVisible();
  await expect(canvas(page).locator(".workspace-canvas")).toHaveAttribute("aria-label", "Outcome Board");
  await expect(viewport(page)).toHaveAttribute("aria-label", /^Outcome Board map\./);
  await expect(canvas(page).locator(".canvas-list")).toHaveAttribute("aria-label", "Outcome Board list");
  await page.evaluate(() => document.fonts.ready);
  // Selecting an Outcome must move focus out of navigation that is now hidden.
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toBeFocused();
  await focusedBoardOwnsNavigation(page);
  await verifyCaptureAssets(page, wos);
}
async function sidebarBounds(page) {
  await expect(page.locator(".sidebar")).toBeVisible();
  const geometry = await page.evaluate(() => {
    const rect = document.querySelector(".sidebar").getBoundingClientRect();
    return {
      viewport: { width: innerWidth, height: innerHeight, devicePixelRatio, visualViewportScale: visualViewport.scale },
      sidebar: { x: rect.x, y: rect.y, width: rect.width, height: rect.height, top: rect.top, right: rect.right, bottom: rect.bottom, left: rect.left },
      document: { scrollX, scrollY, width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight },
    };
  });
  expect(geometry.document.width).toBeLessThanOrEqual(geometry.viewport.width);
  return geometry;
}
async function capture(page, name, measurement) {
  await test.info().attach(`${name}-geometry`, { body: JSON.stringify(measurement, null, 2), contentType: "application/json" });
  const directory = process.env.WOS_CAPTURE_CANVAS_FIRST;
  if (!directory) return;
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.locator("img").evaluateAll((images) => Promise.all(images.filter((image) => image.checkVisibility()).map((image) => image.decode())));
  await page.mouse.move(0, 0);
  let png, captureMethod;
  if (measurement.nativeBrowserZoom) {
    // Playwright's CSS-sized clip crops Chromium's native zoom framebuffer.
    // Capture the actual visible browser surface without a clip or resizing.
    const session = await page.context().newCDPSession(page);
    try {
      const result = await session.send("Page.captureScreenshot", { format: "png", captureBeyondViewport: false, fromSurface: true });
      png = Buffer.from(result.data, "base64");
      await writeFile(join(directory, `${name}.png`), png);
    } finally { await session.detach(); }
    captureMethod = "CDP Page.captureScreenshot; visible surface; no clip or resizing";
  } else {
    png = await page.screenshot({ path: join(directory, `${name}.png`), fullPage: false, animations: "disabled" });
    captureMethod = "Playwright viewport screenshot";
  }
  const pixelSize = { width: png.readUInt32BE(16), height: png.readUInt32BE(20) };
  if (measurement.nativeBrowserZoom) expect(pixelSize).toEqual(measurement.unzoomedViewport);
  expect(verifiedAssets.has(page), "The captured server's modified assets were checked against source").toBe(true);
  await writeFile(join(directory, `${name}.json`), JSON.stringify({
    ...(measurement.entry ? await entryBounds(page) : measurement.specialist ? await sidebarBounds(page) : await workspaceBounds(page)), ...measurement, pixelSize, captureMethod,
    servedAssetsSha256: verifiedAssets.get(page), capturedAt: new Date().toISOString(),
  }, null, 2) + "\n");
}

for (const [width, height] of [[1280, 720], [1366, 768], [1920, 1080], [2560, 1440]]) {
  test(`Outcome Board uses the available desktop workspace at ${width} x ${height}`, async ({ page, wos }) => {
    await page.setViewportSize({ width, height });
    const data = await seedCanvas(wos, width === 1366 ? {
      title: "Coordinate a complete customer delivery across people, agents, tools and sessions while preserving a shared verification trail",
    } : {});
    const mutations = [];
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
    });
    await openCanvas(page, wos, data);
    await expect(page.locator("#canvas-counts")).toHaveText("2 Objectives · 5 Tasks · 1 Issue");
    if (width === 1366) {
      await expect(page.locator("#title")).toHaveText(data.outcome.title);
      await expect(page.locator("#title")).toHaveAttribute("title", data.outcome.title);
      await openCanvasContext(page);
      await expect(page.locator("#canvas-summary .canvas-context-title")).toHaveText(data.outcome.title);
      await expect(page.locator("#canvas-summary .canvas-context-title")).toBeVisible();
      await page.keyboard.press("Escape");
      await expect(page.locator("#canvas-context")).not.toHaveAttribute("open", "");
    }
    await controls(page);
    const initial = await workspaceBounds(page);
    const headerBefore = await page.locator(".outcome-heading").boundingBox();
    const toolbarBefore = await canvas(page).locator(".canvas-toolbar").boundingBox();
    await capture(page, `canvas-${width}x${height}`, { state: "default", ...initial });
    expect(initial.heightRatio).toBeGreaterThanOrEqual(0.7);
    await task(page, data.tasks.ready).focus();
    await page.keyboard.press("Enter");
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await fullyVisible(panel(page));
    const selected = await workspaceBounds(page);
    expect(await page.locator(".outcome-heading").boundingBox()).toEqual(headerBefore);
    expect(await canvas(page).locator(".canvas-toolbar").boundingBox()).toEqual(toolbarBefore);
    const inspection = await panel(page).boundingBox();
    const footer = await canvas(page).locator(".canvas-footer").boundingBox();
    expect(inspection.y).toBeGreaterThanOrEqual(toolbarBefore.y + toolbarBefore.height);
    expect(inspection.y + inspection.height).toBeLessThanOrEqual(footer.y + 1);
    await controls(page);
    if (width <= 1400 || height <= 800) expect(selected.width).toBeCloseTo(initial.width, 0);
    await capture(page, `canvas-${width}x${height}-inspector`, { state: "selected Task, initial context", ...selected });
    await inspectorActionsRemainReachable(page);
    await exerciseCameraWithInspection(page);
    if (width === 1366)
      await capture(page, "canvas-1366x768-fitted", { state: "complete loaded map fitted in the area beside the open inspector" });
    const body = panel(page).locator(".dialog-body").first();
    await body.evaluate((element) => { element.scrollTop = element.scrollHeight; });
    await workspaceBounds(page);
    await page.keyboard.press("Escape");
    await expect(panel(page)).toBeHidden();
    await expect(task(page, data.tasks.ready)).toBeFocused();
    await controls(page);
    await workspaceBounds(page);
    await page.goBack();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await page.reload();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await workspaceBounds(page);
    await page.keyboard.press("Escape");
    await canvas(page).getByRole("button", { name: "Zoom in", exact: true }).click();
    await canvas(page).getByRole("button", { name: "Fit view", exact: true }).click();
    await viewport(page).focus();
    const camera = await canvas(page).locator(".canvas-world").getAttribute("style");
    await page.keyboard.press("ArrowRight");
    await expect(canvas(page).locator(".canvas-world")).not.toHaveAttribute("style", camera);
    await workspaceBounds(page);
    expect(mutations).toEqual([]);
  });
}

test("Views and Outcome context stay progressive, keyboard accessible and separate from verification", async ({ page, wos }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  const data = await seedCanvas(wos);
  await seedCanvasRoadmap(wos, data, { published: true, active: true });
  const other = await wos.outcome("A separate customer delivery");
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await openCanvas(page, wos, data);
  await expect(page.locator("#canvas-context")).not.toHaveAttribute("open", "");
  await openCanvasContext(page);
  await expect(page.locator("#canvas-summary")).toBeVisible();
  await expect(page.locator("#canvas-planning")).toBeVisible();
  await fullyVisible(page.locator("#canvas-context > summary"));
  await capture(page, "canvas-outcome-context", { state: "expanded Outcome context and active planning" });
  await page.keyboard.press("Escape");
  await expect(page.locator("#canvas-context")).not.toHaveAttribute("open", "");
  await expect(page.locator("#canvas-context > summary")).toBeFocused();
  await openViews(page);
  for (const key of ["overview", "plan", "work", "issues", "evidence", "activity"])
    await fullyVisible(page.locator(`#area-tabs [data-area="${key}"]`));
  await capture(page, "canvas-views", { state: "expanded specialist Views navigation" });
  await page.keyboard.press("Escape");
  await expect(page.locator("#outcome-views")).not.toHaveAttribute("open", "");
  await expect(page.locator("#outcome-views > summary")).toBeFocused();
  for (const area of ["overview", "plan", "work", "issues", "evidence", "activity"]) {
    await chooseArea(page, area);
    await expect(page.locator(".sidebar")).toBeVisible();
    await expect(page.locator("#outcome-tools")).toBeHidden();
    await expect(page.locator("#actions")).toBeHidden();
    await expect(page.locator("#refresh")).toHaveCount(1);
    await expect(page.locator(".heading-actions > #refresh")).toBeVisible();
    await expect(page.locator("#refresh")).toHaveText("Refresh view");
    if (area === "overview") {
      await expect(page.locator(".summary-quick > #review-outcome")).toBeVisible();
      await page.locator("#review-outcome").click();
      await expect(page.locator("#detail-title")).toHaveText(data.outcome.title);
      await expect(page.locator("#detail-dialog")).toBeVisible();
      await page.keyboard.press("Escape");
    } else if (area !== "activity")
      await expect(page.locator("#add-item")).toBeVisible();
    await expect(page.getByRole("complementary", { name: "Workspace navigation", exact: true })).toBeVisible();
    await expect(page.locator(`#area-tabs [data-area="${area}"]`)).toHaveAttribute("aria-pressed", "true");
    if (["overview", "work"].includes(area)) {
      await page.evaluate(() => scrollTo(0, 0));
      await capture(page, `canvas-1366x768-${area}-sidebar`, { specialist: true, state: `${area} with retained live-search sidebar and local actions` });
    }
  }
  await chooseArea(page, "overview");
  await openOutcome(page, other.value.title, { area: "overview" });
  await expect(page.locator(".sidebar")).toBeVisible();
  await expect(page.locator("#title")).toHaveText(other.value.title);
  await chooseArea(page, "canvas");
  await expect(page).toHaveURL(new RegExp(`outcome=${other.value.id}.*area=canvas`));
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toBeFocused();
  await focusedBoardOwnsNavigation(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await expect(page).toHaveURL(new RegExp(`outcome=${data.outcome.id}.*area=canvas`));
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toBeFocused();
  await focusedBoardOwnsNavigation(page);
  await workspaceBounds(page);
  await chooseArea(page, "work");
  await expect(page.getByLabel("Work status", { exact: true })).toBeVisible();
  await expect(page.locator("#outcome-views > summary")).toBeFocused();
  await expect(page.locator("#outcome-views")).not.toHaveAttribute("open", "");
  await page.goBack();
  await expect(viewport(page)).toBeVisible();
  await workspaceBounds(page);
  const search = canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true });
  await search.fill(data.evidence[0].description);
  await canvas(page).getByRole("option").filter({ hasText: data.evidence[0].description }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText("Evidence");
  await expect(panel(page).locator(".detail-description")).toHaveText(data.evidence[0].description);
  const persisted = await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}`);
  expect(persisted.value.lifecycle).toBe("active");
  expect(mutations).toEqual([]);
  const audit = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
  expect(audit.violations.filter((violation) => ["critical", "serious"].includes(violation.impact)), JSON.stringify(audit.violations)).toEqual([]);
});

for (const [width, height] of [[320, 740], [390, 844]]) {
  test(`Compact Outcome Board keeps its own list and inspector scroll at ${width} x ${height}`, async ({ page, wos }) => {
    await page.setViewportSize({ width, height });
    const data = await seedCanvas(wos, { extraTasks: 35 });
    await openCanvas(page, wos, data);
    await expect(canvas(page).getByRole("button", { name: "List view", exact: true })).toHaveAttribute("aria-pressed", "true");
    const list = canvas(page).locator(".canvas-list");
    await fullyVisible(list);
    const before = await list.evaluate((element) => ({ top: element.scrollTop, overflow: element.scrollHeight - element.clientHeight }));
    expect(before.overflow).toBeGreaterThan(0);
    await list.hover();
    await page.mouse.wheel(0, 500);
    await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBeGreaterThan(before.top);
    await workspaceBounds(page);
    await capture(page, `canvas-${width}x${height}-list`, { state: "independently scrolled list" });
    await list.evaluate((element) => { element.scrollTop = 0; });
    await task(page, data.tasks.unscoped, "list").click();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.unscoped.title);
    await fullyVisible(panel(page));
    await capture(page, `canvas-${width}x${height}-inspector`, { state: "selected Task, initial context" });
    await inspectorActionsRemainReachable(page);
    await clickRecordAction(page, "Edit task");
    await expect(panel(page).locator("#command-form")).toBeVisible();
    const body = panel(page).locator("#fields");
    await body.evaluate((element) => { element.scrollTop = element.scrollHeight; });
    // Narrow/short inspectors scroll as one surface; native focus must reveal
    // the submit action there as well as in the independently scrolling body.
    await panel(page).locator("#submit-command").focus();
    await fullyVisible(panel(page).locator("#submit-command"));
    await workspaceBounds(page);
    await capture(page, `canvas-${width}x${height}-form`, { state: "contextual form with independent scroll" });
    await page.keyboard.press("Escape");
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.unscoped.title);
    await page.keyboard.press("Escape");
    await expect(panel(page)).toBeHidden();
    await expect(task(page, data.tasks.unscoped, "list")).toBeFocused();
    await workspaceBounds(page);
    await canvas(page).getByRole("button", { name: "Map view", exact: true }).click();
    await controls(page);
    await workspaceBounds(page);
    await capture(page, `canvas-${width}x${height}-map`, { state: "explicit Map alternative" });
    if (width === 320) {
      await page.setViewportSize({ width: 360, height: 640 });
      await controls(page);
      const map = await viewport(page).boundingBox();
      const camera = await canvas(page).locator(".canvas-camera-controls").boundingBox();
      expect(camera.y).toBeGreaterThanOrEqual(map.y);
      expect(camera.y + camera.height).toBeLessThanOrEqual(map.y + map.height);
      const beforeZoom = Number(await viewport(page).getAttribute("data-scale"));
      await canvas(page).getByRole("button", { name: "Zoom in", exact: true }).click();
      expect(Number(await viewport(page).getAttribute("data-scale"))).toBeGreaterThan(beforeZoom);
      await canvas(page).getByRole("button", { name: "Center view", exact: true }).click();
      await canvas(page).getByRole("button", { name: "Fit view", exact: true }).click();
      await workspaceBounds(page);
      await capture(page, "canvas-360x640-map", { state: "short mobile Map alternative with horizontal camera controls and explicit fit" });
      // The bounded large fixture need not include every other Task. Reopen
      // the same loaded Task already inspected through its list alternative.
      await task(page, data.tasks.unscoped).focus();
      await page.keyboard.press("Enter");
      await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.unscoped.title);
      await fullyVisible(panel(page));
      const detailSlot = panel(page).locator(".canvas-inspector-detail");
      const reading = panel(page).getByRole("heading", { name: "What needs attention", exact: true });
      await reading.scrollIntoViewIfNeeded();
      await fullyVisible(reading);
      expect(await detailSlot.evaluate((element) => element.scrollHeight > element.clientHeight)).toBe(true);
      await inspectorActionsRemainReachable(page);
      await panel(page).locator("#close-detail").scrollIntoViewIfNeeded();
      await fullyVisible(panel(page).locator("#close-detail"));
      await workspaceBounds(page);
      await capture(page, "canvas-360x640-inspector", { state: "short mobile inspection; its whole detail surface scrolls inside the stage" });
      await panel(page).locator("#close-detail").click();
      await expect(panel(page)).toBeHidden();
      await page.setViewportSize({ width, height });
    }
    await chooseArea(page, "overview");
    const navigation = page.locator("#navigation-toggle");
    if (await navigation.getAttribute("aria-expanded") === "false") await navigation.click();
    await page.evaluate(() => scrollTo(0, 0));
    await capture(page, `canvas-${width}x${height}-overview-sidebar`, { specialist: true, state: "expanded mobile Overview navigation and live search" });
    await navigation.click();
    await chooseArea(page, "canvas");
    await workspaceBounds(page);
  });
}

test("native browser zoom at 125% and 150% preserves Outcome Board controls and contextual return", async ({ wos }) => {
  test.setTimeout(180000);
  const data = await seedCanvas(wos);
  const browser = await nativeZoomBrowser({ width: 1366, height: 768 });
  try {
    const page = browser.page;
    await openCanvas(page, wos, data);
    const unzoomedViewport = await page.evaluate(() => ({ width: innerWidth, height: innerHeight }));
    const outer = await page.evaluate(() => ({ width: outerWidth, height: outerHeight }));
    for (const factor of [1.25, 1.5]) {
      await browser.setZoom(factor);
      await expect(page.locator(".sidebar")).toBeHidden();
      await controls(page);
      const bounds = await workspaceBounds(page);
      const metrics = await page.evaluate(() => ({ width: innerWidth, height: innerHeight, outerWidth, outerHeight, devicePixelRatio, visualScale: visualViewport.scale }));
      expect(metrics.outerWidth).toBe(outer.width);
      expect(metrics.outerHeight).toBe(outer.height);
      expect(metrics.devicePixelRatio).toBeCloseTo(factor, 5);
      expect(metrics.visualScale).toBe(1);
      await capture(page, `canvas-native-zoom-${factor * 100}`, { nativeBrowserZoom: factor, unzoomedViewport, metrics, ...bounds });
      expect(bounds.height).toBeGreaterThan(200);
      await task(page, data.tasks.ready).focus();
      await page.keyboard.press("Enter");
      await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
      await fullyVisible(panel(page));
      await workspaceBounds(page);
      await capture(page, `canvas-native-zoom-${factor * 100}-inspector`, { state: "selected Task, initial context", nativeBrowserZoom: factor, unzoomedViewport, metrics });
      await inspectorActionsRemainReachable(page);
      await page.keyboard.press("Escape");
      await expect(task(page, data.tasks.ready)).toBeFocused();
      await workspaceBounds(page);
    }
  } finally { await browser.close(); }
});
