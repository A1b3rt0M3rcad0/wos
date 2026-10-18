import { test, expect, openOutcome, chooseArea } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { join } from "node:path";

const board = (page) => page.locator("#operational-canvas");
const panel = (page) => page.locator("#canvas-details");
const context = (page) => panel(page).locator(".entity-context");
const digest = (bytes) => createHash("sha256").update(bytes).digest("hex");
const production = [
  "app.js", "canvas.css", "canvas-context.js", "editorial.css", "en-US.js",
  "index.html", "style.css", "colors.css", "workspace-entry.css",
  "workspace-canvas.js", "workspace-entity-context.js", "workspace-action-menu.js",
];
const qualified = new WeakMap();
async function qualify(page, wos) {
  if (!process.env.WOS_CAPTURE_BOARD_COMMUNICATION) return;
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
  const directory = process.env.WOS_CAPTURE_BOARD_COMMUNICATION;
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
}
async function group(page, title) {
  return context(page).locator(".canvas-context-section").filter({
    has: page.getByRole("heading", { name: title, exact: true }),
  });
}

for (const viewport of [{ width: 2560, height: 1440 }, { width: 390, height: 844 }]) {
  test(`relationships communicate placement, evidence and real blocking at ${viewport.width} x ${viewport.height}`, async ({ page, wos }) => {
    await page.setViewportSize(viewport);
    const data = await seedCanvas(wos);
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
    await expect(context(page)).toContainText("A verifiable condition within this Outcome");
    await expect(context(page).getByRole("button", { name: data.outcome.title, exact: true })).toHaveCount(1);
    await expect(context(page).getByRole("button", { name: data.objective.title, exact: true })).toHaveCount(1);
    const work = await group(page, "Objectives and Tasks");
    await expect(work.getByRole("button", { name: data.tasks.ready.title, exact: true })).toHaveCount(1);
    await capture(page, `relationships-${viewport.width}x${viewport.height}-objective`,
      "Objective membership and clickable associated Tasks, with explicit role and direct parent; loaded-only scope");
    await work.getByRole("button", { name: data.tasks.ready.title, exact: true }).click();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await expect(context(page)).toContainText("does not automatically achieve its Objective or Outcome");
    await expect(context(page).getByRole("button", { name: data.child.title, exact: true })).toHaveCount(1);
    await expect(panel(page).getByRole("heading", { name: "Outcome and Objective", exact: true })).toHaveCount(0);
    await expect(panel(page).getByRole("heading", { name: "Task context", exact: true })).toHaveCount(1);
    await expect(panel(page).locator(".entity-context .canvas-context-section")).toHaveCount(0);
    const taskEvidence = panel(page).locator(".canvas-task-context .canvas-context-section").filter({
      has: page.getByRole("heading", { name: "Evidence relationships", exact: true }),
    });
    await expect(taskEvidence).toContainText("Supports");
    await expect(taskEvidence).toContainText("Contradicts");
    await expect(taskEvidence).toContainText("not an assessment or acceptance decision");
    const supporting = taskEvidence.locator(".canvas-context-card").filter({
      has: page.getByText("Supports", { exact: true }),
    });
    await supporting.getByRole("button", { name: /^Evidence / }).click();
    await expect(panel(page).locator("#detail-title")).toHaveText("Evidence");
    await expect(panel(page).locator(".detail-description")).toHaveText(data.evidence[0].description);
    await expect(context(page)).toContainText("linking it does not verify a criterion");
    const linked = await group(page, "Evidence relationships");
    await expect(linked).toContainText("Supports");
    await expect(linked.getByRole("button", { name: data.tasks.ready.title, exact: true })).toHaveCount(1);
    await linked.getByRole("button", { name: "Inspect link", exact: true }).click();
    await expect(context(page)).toContainText("One observation related to one target, with a specific stance");
    const endpoints = await group(page, "Relationship endpoints");
    await expect(endpoints).toContainText("Supports");
    await endpoints.getByRole("button", { name: data.tasks.ready.title, exact: true }).click();
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);

    await select(page, data.issue);
    await expect(context(page)).toContainText("a separate Blocker records an actual impediment");
    const impacts = await group(page, "Problems and blocking");
    await expect(impacts).toContainText("Affected item · not a blocking statement");
    await expect(impacts.getByRole("button", { name: data.tasks.blocked.title, exact: true })).toHaveCount(1);
    await capture(page, `relationships-${viewport.width}x${viewport.height}-issue`,
      "Issue affected scope is distinct from actual explicit Blocker causation; inspector remains within working stage");
    // The initial bounded graph need not include the Issue->Blocker cause
    // edge. Follow the recorded affected Task and its authoritative blocking
    // response rather than treating omission as proof of no impact.
    await impacts.getByRole("button", { name: data.tasks.blocked.title, exact: true }).click();
    await expect(panel(page).getByText("Blocked: Yes", { exact: true })).toHaveCount(1);
    const blocking = panel(page).locator(".canvas-task-context .canvas-context-section").filter({
      has: page.getByRole("heading", { name: "Blocking impact", exact: true }),
    });
    await blocking.getByRole("button", { name: /^Blocker / }).click();
    await expect(panel(page).locator("#detail-title")).toHaveText("Blocker");
    const blocker = await group(page, "Problems and blocking");
    await expect(blocker).toContainText("Blocks");
    await expect(blocker).toContainText("Recorded cause");
    await expect(blocker.getByRole("button", { name: data.issue.title, exact: true })).toHaveCount(1);
    await blocker.getByRole("button", { name: data.tasks.blocked.title, exact: true }).click();
    await expect(panel(page).getByText("Blocked: Yes", { exact: true })).toHaveCount(1);

    await select(page, data.evidence[0]);
    const trigger = panel(page).locator("#entity-actions");
    await trigger.scrollIntoViewIfNeeded();
    await trigger.click();
    await expect(panel(page).locator(".canvas-inspector-actions")).toBeVisible();
    await expect(panel(page).locator("#action-options")).toContainText("Evidence");
    await capture(page, `relationships-${viewport.width}x${viewport.height}-actions`,
      "Compact available actions for the selected Evidence with context and consequence descriptions, without viewport-filling tiles");
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
    expect(mutations).toEqual([]);
    expect(errors).toEqual([]);
  });
}

test("specialist entity details explain direct recorded scope without claiming a loaded Board index", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title);
  await chooseArea(page, "issues");
  await page.locator("#content").getByRole("button").filter({ hasText: data.issue.title }).first().click();
  const detail = page.locator("#detail-dialog");
  await expect(detail).toBeVisible();
  const summary = detail.locator(".entity-context");
  await expect(summary).toContainText("Affected item · not a blocking statement");
  await summary.getByText("Loaded connections and coverage", { exact: true }).click();
  await expect(summary).toContainText("The Board connection index is not included here");
  await expect(summary.getByRole("button", { name: /^Task / })).toHaveCount(1);
});

test("a late bounded graph expansion refreshes actual Objective connections without losing focus or disclosure state", async ({ page, wos }) => {
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
    if (parameters.get("root_kind") !== "objective" ||
      parameters.get("root_id") !== data.objective.id) return route.continue();
    held = true;
    try {
      const response = await route.fetch();
      expect(response.ok()).toBe(true);
      arrived();
      await gate;
      await route.fulfill({ response });
    } finally {
      finished();
    }
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
    const dependencies = await group(page, "Dependencies");
    const originalCount = await dependencies.locator(".canvas-context-card").count();
    expect(originalCount).toBeGreaterThan(4);
    expect(originalCount).toBeLessThan(30);
    for (const id of ["PlanWork", "Dependencies", "coverage"]) {
      const disclosure = context(page).locator(`details[data-entity-context-section="${id}"]`);
      await disclosure.locator(":scope > summary").click();
      await expect(disclosure).toHaveAttribute("open", "");
    }
    const focused = dependencies.getByRole("button", { name: "Inspect link", exact: true }).first();
    await focused.focus();
    const identity = await focused.getAttribute("data-entity-context-ref");
    await expect(focused).toBeFocused();
    release();
    await done;
    await expect(dependencies.locator(".canvas-context-card")).toHaveCount(30);
    for (const id of ["PlanWork", "Dependencies", "coverage"])
      await expect(context(page).locator(`details[data-entity-context-section="${id}"]`)).toHaveAttribute("open", "");
    await expect(context(page).locator(`[data-entity-context-ref="${identity}"]`)).toBeFocused();
    await expect(dependencies).toContainText("Advisory dependency");
    expect(mutations).toEqual([]);
  } finally {
    release();
    if (held) await done;
    await page.unroute(match);
  }
});
