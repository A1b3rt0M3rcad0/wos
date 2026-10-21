import { expandWorkspaceDetails } from './workspace-action-fixture.mjs';
import { test, expect, openOutcome, chooseArea } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { join } from "node:path";

const board = (page) => page.locator("#operational-canvas");
const panel = (page) => page.locator("#canvas-details");
const context = (page) => panel(page).locator(".workspace-view");
const facet = (page, name, host = panel(page)) => host.locator(`[data-workspace-facet="${name}"]`);
const reference = (host, kind, id) => host.locator(`button[data-reference-kind="${kind}"][data-reference-id="${id}"]`);
const rows = (page, label) => facet(page, "relationships").locator(".workspace-relationship").filter({ has: page.locator(".workspace-relationship-label").filter({ hasText: new RegExp(`^${label}$`) }) });
const digest = (bytes) => createHash("sha256").update(bytes).digest("hex");
const production = [
  "app.js", "canvas.css", "editorial.css", "en-US.js",
  "index.html", "style.css", "colors.css", "workspace-entry.css",
  "workspace-canvas.js", "canvas-data.js", "workspace-card-summary.js", "workspace-board.css",
  "workspace-view.js", "workspace-view.css", "workspace-inspector.js", "workspace-action-menu.js",
  "workspace-workflows.css", "workspace-actions.js",
];
const qualified = new WeakMap();
async function qualify(page, wos) {
  if (!process.env.WOS_CAPTURE_WORKSPACE_RELATIONSHIPS) return;
  const assets = await Promise.all(production.map(async (file) => {
    const [response, source] = await Promise.all([
      page.request.get(`${wos.url}/app/${file}`),
      readFile(new URL(`../../packages/wos-api/web/assets/${file}`, import.meta.url)),
    ]);
    expect(response.ok(), `Capture source ${file} is actually served`).toBe(true);
    const actual = digest(await response.body());
    expect(actual, `Refuse stale embedded ${file}`).toBe(digest(source));
    return [file, actual];
  }));
  qualified.set(page, Object.fromEntries(assets));
}
async function capture(page, name, state) {
  const directory = process.env.WOS_CAPTURE_WORKSPACE_RELATIONSHIPS;
  if (!directory) return;
  expect(qualified.has(page)).toBe(true);
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.locator("img").evaluateAll((images) => Promise.all(images
    .filter((image) => image.checkVisibility()).map((image) => image.decode())));
  await page.mouse.move(0, 0);
  const geometry = await page.evaluate(() => {
    const measure = (selector) => {
      const element = document.querySelector(selector);
      if (!element?.checkVisibility()) return null;
      const bounds = element.getBoundingClientRect(), style = getComputedStyle(element);
      return { x: bounds.x, y: bounds.y, width: bounds.width, height: bounds.height,
        bottom: bounds.bottom, right: bounds.right, fontSize: parseFloat(style.fontSize),
        clientHeight: element.clientHeight, scrollHeight: element.scrollHeight };
    };
    return {
      viewport: { width: innerWidth, height: innerHeight, devicePixelRatio,
        visualViewportScale: visualViewport.scale },
      document: { width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight,
        scrollX, scrollY },
      elements: Object.fromEntries([".outcome-heading", ".canvas-toolbar", ".canvas-stage",
        ".canvas-footer", "#canvas-details", "#detail-content", "#detail-actions", "#action-options",
        "#notice"].map((selector) => [selector, measure(selector)])),
    };
  });
  const png = await page.screenshot({ path: join(directory, `${name}.png`), fullPage: false, animations: "disabled" });
  const pixelSize = { width: png.readUInt32BE(16), height: png.readUInt32BE(20) };
  expect(pixelSize.width).toBe(Math.round(geometry.viewport.width * geometry.viewport.devicePixelRatio));
  expect(pixelSize.height).toBe(Math.round(geometry.viewport.height * geometry.viewport.devicePixelRatio));
  await writeFile(join(directory, `${name}.json`), JSON.stringify({ state, ...geometry, pixelSize,
    screenshotSha256: digest(png), servedAssetsSha256: qualified.get(page),
    captureMethod: "Real isolated WOS server, visible Chromium viewport, no image resizing",
    capturedAt: new Date().toISOString() }, null, 2) + "\n");
}
async function select(page, record, title = record.title || record.name || "Evidence") {
  const label = record.title || record.name || record.description;
  await board(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(label);
  await board(page).getByRole("option").filter({ hasText: label }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(title);
  await expect(context(page)).toBeVisible();
  await expandWorkspaceDetails(panel(page), {history:false});
}
for (const viewport of [{ width: 2560, height: 1440 }, { width: 390, height: 844 }]) {
  test(`relationships communicate placement, evidence and real blocking at ${viewport.width} x ${viewport.height}`, async ({ page, wos }) => {
    await page.setViewportSize(viewport);
    const data = await seedCanvas(wos,{planningTasks:true});
    const criterion = await wos.command("add_criterion", {
      owner: data.ref("work_item", data.tasks.ready), expected_version: data.tasks.ready.version,
      title: "The requested result has been independently reviewed", required: true,
      verification_mode: "evidence_review",
    });
    data.tasks.ready.version++;
    for(const [name, task] of Object.entries(data.tasks)) data.tasks[name]=await wos.command('activate_work_item',{scope:data.scope,work_item_id:task.id,expected_version:task.version});
    const mutations = [], errors = [];
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
    });
    page.on("pageerror", (error) => errors.push(error.message));
    await wos.login(page);
    await qualify(page, wos);
    await openOutcome(page, data.outcome.title, { area: null });
    await expect(board(page)).toBeVisible();

    await select(page, data.child);
    await expect(panel(page).locator("#detail-kind")).toContainText("Objective");
    await expect(context(page).locator(".workspace-state")).toContainText("Optional for Outcome");
    const path = context(page).locator(".workspace-path");
    await expect(reference(path, "outcome", data.outcome.id)).toHaveCount(1);
    await expect(reference(path, "objective", data.objective.id)).toHaveCount(1);
    const work = facet(page, "relationships");
    await expect(reference(work, "work_item", data.tasks.ready.id)).toHaveCount(1);
    await expect(work).toContainText("Belongs to");
    await capture(page, `relationships-${viewport.width}x${viewport.height}-objective`,
      "Canonical Objective ownership and bounded, clickable related work, independent from Board loading");
    await reference(work, "work_item", data.tasks.ready.id).click();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await expandWorkspaceDetails(panel(page), {history:false});
    await expect(reference(context(page).locator(".workspace-path"), "objective", data.child.id)).toHaveCount(1);
    // The old focal detail renderer is gone. The same canonical inspector
    // exposes every pertinent group without five unrelated tabs.
    await expect(panel(page).locator(".canvas-task-context")).toHaveCount(0);
    for (const name of ["work", "people", "attention", "criteria", "delivery", "relationships", "activity"])
      await expect(facet(page, name)).toHaveCount(1);
    await expect(facet(page, "work")).toContainText("Perform only the scoped task");
    await expect(facet(page, "people")).toContainText("Declared responsibility does not grant execution authority");
    await expect(facet(page, "criteria").locator(`[data-criterion-id="${criterion.id}"]`)).toContainText("Not assessed");
    await expect(rows(page, "Supports")).toHaveCount(1);
    await expect(rows(page, "Contradicts")).toHaveCount(1);
    await expect(rows(page, "Provides context")).toHaveCount(1);
    await expect(rows(page, "Supports")).toContainText("does not assess a criterion");
    await reference(rows(page, "Supports"), "evidence", data.evidence[0].id).click();
    await expect(panel(page).locator("#detail-title")).toHaveText("Evidence");
    await expandWorkspaceDetails(panel(page), {history:false});
    await expect(context(page).locator(".detail-description")).toHaveText(data.evidence[0].description);
    await expect(context(page)).toContainText("A link, an assessment and acceptance are distinct records");
    await expect(reference(rows(page, "Supports"), "work_item", data.tasks.ready.id)).toHaveCount(1);
    await rows(page, "Supports").getByRole("button", { name: "Inspect relationship", exact: true }).click();
    await expect(panel(page).locator("#detail-kind")).toContainText(/evidence.*link/i);
    await expect(panel(page).locator("#detail-title")).toHaveText("supports");
    await expandWorkspaceDetails(panel(page), {history:false});
    const endpoints = facet(page, "record");
    await expect(reference(endpoints, "evidence", data.evidence[0].id)).toHaveCount(1);
    await expect(reference(endpoints, "work_item", data.tasks.ready.id)).toHaveCount(1);
    await reference(endpoints, "work_item", data.tasks.ready.id).click();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await expandWorkspaceDetails(panel(page), {history:false});
    await expect(facet(page, "criteria").locator(`[data-criterion-id="${criterion.id}"]`)).toContainText("Not assessed");

    await select(page, data.issue);
    await expect(context(page)).toContainText("Only explicit Blockers record an impediment");
    const impacts = rows(page, "Affects; does not itself block");
    await expect(reference(impacts, "work_item", data.tasks.blocked.id)).toHaveCount(1);
    await capture(page, `relationships-${viewport.width}x${viewport.height}-issue`,
      "Issue affected scope remains distinct from explicit Blocker impacts and their recorded causes");
    await reference(impacts, "work_item", data.tasks.blocked.id).click();
    await expect(context(page).locator(".workspace-state")).toContainText("Blocked");
    await expect(facet(page, "attention")).toContainText(/block/i);
    await expandWorkspaceDetails(panel(page), {history:false});
    await reference(rows(page, "Blocks"), "blocker", data.blocker.id).click();
    await expect(panel(page).locator("#detail-title")).toHaveText("Blocker");
    await expandWorkspaceDetails(panel(page), {history:false});
    await expect(context(page)).toContainText("Evidence belongs to the cause or affected work");
    await expect(rows(page, "Cause of impediment")).toHaveCount(1);
    await expect(reference(rows(page, "Cause of impediment"), "issue", data.issue.id)).toHaveCount(1);
    await reference(rows(page, "Blocks"), "work_item", data.tasks.blocked.id).click();
    await expect(context(page).locator(".workspace-state")).toContainText("Blocked");

    await select(page, data.evidence[0]);
    await panel(page).locator('#entity-actions').click();
    await expect(panel(page).locator(".canvas-inspector-actions")).toBeVisible();
    await expect(panel(page).locator("#action-options")).toContainText("Evidence");
    await capture(page, `relationships-${viewport.width}x${viewport.height}-actions`,
      "Compact contextual actions retain the selected Evidence and its documentary consequences");
    await page.keyboard.press("Escape");
    await expect(panel(page).locator(".canvas-inspector-detail")).toBeVisible();
    expect(await page.evaluate(() => ({
      x: document.documentElement.scrollWidth - innerWidth,
      y: document.documentElement.scrollHeight - innerHeight,
    }))).toEqual({ x: 0, y: 0 });
    const response = await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${data.tasks.blocked.id}/operational-state`);
    expect(response.state.is_blocked).toBe(true);
    const issue = await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/issues/${data.issue.id}`);
    expect(issue.value.lifecycle).toBe("open");
    const owner = await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${data.tasks.ready.id}`);
    expect(owner.value.criteria.current_assessments || {}).toEqual({});
    expect((await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/objectives/${data.child.id}`)).value.lifecycle).toBe("planned");
    expect(mutations).toEqual([]);
    expect(errors).toEqual([]);
  });
}

test("specialist entity details use the same canonical relationships without depending on a loaded Board index", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await select(page, data.issue);
  const boardMeaning = await context(page).locator(".workspace-meaning").textContent();
  const boardImpact = await rows(page, "Affects; does not itself block").textContent();
  await panel(page).locator("#close-detail").click();
  await chooseArea(page, "issues");
  await page.locator("#content").getByRole("button").filter({ hasText: data.issue.title }).first().click();
  const detail = page.locator("#detail-dialog");
  await expect(detail).toBeVisible();
  const view = detail.locator(".workspace-view");
  await expect(view.locator(".workspace-meaning")).toHaveText(boardMeaning);
  const affected = facet(page, "relationships", detail).locator(".workspace-relationship").filter({
    has: page.locator(".workspace-relationship-label").filter({ hasText: /^Affects; does not itself block$/ }),
  });
  await expect(affected).toHaveText(boardImpact);
  await expect(reference(affected, "work_item", data.tasks.blocked.id)).toHaveCount(1);
  await expect(detail.locator(".entity-context, .canvas-task-context")).toHaveCount(0);
});

test("a late bounded graph expansion preserves coherent relationship pages, keyboard focus and disclosure state", async ({ page, wos }) => {
  const data = await seedCanvas(wos, { extraTasks: 35 });
  for (const task of data.additional.slice(0, 30))
    await wos.command("add_dependency", {
      scope: data.scope, source_ref: data.ref("objective", data.objective),
      target_ref: data.ref("work_item", task), strength: "advisory",
      reason: "A real optional prerequisite for the selected Objective",
    });
  let release, arrived, finished, held = false;
  const gate = new Promise((resolve) => { release = resolve; });
  const seen = new Promise((resolve) => { arrived = resolve; });
  const done = new Promise((resolve) => { finished = resolve; });
  const match = "**/graph?**";
  await page.route(match, async (route) => {
    const parameters = new URL(route.request().url()).searchParams;
    if (parameters.get("root_kind") !== "objective" || parameters.get("root_id") !== data.objective.id)
      return route.continue();
    held = true;
    try {
      const response = await route.fetch();
      expect(response.ok()).toBe(true);
      arrived();
      await gate;
      await route.fulfill({ response });
    } finally { finished(); }
  });
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  try {
    await wos.login(page);
    await openOutcome(page, data.outcome.title, { area: null });
    await select(page, data.objective);
    await seen;
    const relationships = facet(page, "relationships");
    const originalCount = await relationships.locator(".workspace-relationship").count();
    expect(originalCount).toBeGreaterThan(4);
    expect(originalCount).toBeLessThan(30);
    await expect(relationships).toContainText("Partial view of this information");
    const audit = context(page).locator(".workspace-management");
    await expect(audit).toHaveAttribute("open", "");
    const focused = relationships.getByRole("button", { name: "Inspect relationship", exact: true }).first();
    await focused.focus();
    await expect(focused).toBeFocused();
    release();
    await done;
    // Graph exploration enriches the Board only. It must neither splice a
    // separate snapshot into the inspector nor steal an established focus.
    await expect(relationships.locator(".workspace-relationship")).toHaveCount(originalCount);
    await expect(audit).toHaveAttribute("open", "");
    await expect(focused).toBeFocused();
    await expect(relationships).toContainText("Advisory dependency");
    await relationships.locator('[data-action-id="load_more_relationships"]').click();
    await expect.poll(() => relationships.locator(".workspace-relationship").count()).toBeGreaterThan(originalCount);
    expect(mutations).toEqual([]);
  } finally {
    release();
    if (held) await done;
    await page.unroute(match);
  }
});
