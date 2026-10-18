import {
  test, expect, openOutcome, chooseArea, openViews, openCanvasContext,
} from "./ux-fixture.mjs";
import { seedCanvas, seedCanvasRoadmap } from "./canvas-fixture.mjs";
import { nativeZoomBrowser } from "./native-browser-zoom.mjs";
import AxeBuilder from "@axe-core/playwright";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { createHash } from "node:crypto";

const verifiedAssets = new WeakMap();
const sha256 = (value) => createHash("sha256").update(value).digest("hex");

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
async function controls(page) {
  await fullyVisible(page.locator('#area-tabs [data-area="canvas"]'));
  await fullyVisible(page.locator("#outcome-views > summary"));
  await fullyVisible(canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }));
  for (const name of ["Zoom in", "Zoom out", "Fit view"])
    await fullyVisible(canvas(page).getByRole("button", { name, exact: true }));
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
  await expect(canvas(page)).toHaveAttribute("aria-label", "Outcome Board workspace");
  await expect(canvas(page).locator(".workspace-canvas")).toBeVisible();
  await expect(canvas(page).locator(".workspace-canvas")).toHaveAttribute("aria-label", "Outcome Board");
  await expect(viewport(page)).toHaveAttribute("aria-label", /^Outcome Board map\./);
  await expect(canvas(page).locator(".canvas-list")).toHaveAttribute("aria-label", "Outcome Board list");
  await page.evaluate(() => document.fonts.ready);
  // Selecting an Outcome must move focus out of navigation that is now hidden.
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toBeFocused();
  await focusedBoardOwnsNavigation(page);
  if (process.env.WOS_CAPTURE_CANVAS_FIRST) {
    const entries = await Promise.all(["app.js", "canvas.css", "en-US.js", "index.html", "workspace-canvas.js"].map(async (file) => {
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
    ...await workspaceBounds(page), ...measurement, pixelSize, captureMethod,
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
    await capture(page, `canvas-${width}x${height}`, { state: "default", ...initial });
    expect(initial.heightRatio).toBeGreaterThanOrEqual(0.7);
    await task(page, data.tasks.ready).focus();
    await page.keyboard.press("Enter");
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await fullyVisible(panel(page));
    const selected = await workspaceBounds(page);
    if (width <= 1400 || height <= 800) expect(selected.width).toBeCloseTo(initial.width, 0);
    await capture(page, `canvas-${width}x${height}-inspector`, { state: "selected Task, initial context", ...selected });
    await inspectorActionsRemainReachable(page);
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
    await expect(page.getByRole("complementary", { name: "Workspace navigation", exact: true })).toBeVisible();
    await expect(page.locator(`#area-tabs [data-area="${area}"]`)).toHaveAttribute("aria-pressed", "true");
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
  await expect(panel(page).locator("#detail-title")).toHaveText(data.evidence[0].description);
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
    await panel(page).getByRole("button", { name: "Edit task", exact: true }).click();
    await expect(panel(page).locator("#command-form")).toBeVisible();
    const body = panel(page).locator("#fields");
    await body.evaluate((element) => { element.scrollTop = element.scrollHeight; });
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
