import { test, expect, openOutcome, chooseArea, pick } from "./ux-fixture.mjs";
import { revealMenuAction } from "./workspace-action-fixture.mjs";
import { seedCanvas, seedCanvasRoadmap } from "./canvas-fixture.mjs";
import { nativeZoomBrowser } from "./native-browser-zoom.mjs";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { readdir, readFile, mkdir, writeFile, access } from "node:fs/promises";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// Opt-in evidence generation: the ordinary browser suite does not produce
// screenshots or add skipped artifact jobs to its functional result count.
const output = process.env.WOS_CAPTURE_OPERATIONAL_WORKSPACE;
const assets = fileURLToPath(new URL("../../packages/wos-api/web/assets/", import.meta.url));
const repository = fileURLToPath(new URL("../../", import.meta.url));
const binary = join(repository, "bin/wos");
const hash = (value) => createHash("sha256").update(value).digest("hex");
const panel = (page) => page.locator("#canvas-details");
const map = (page) => page.locator(".canvas-viewport");

async function assetPaths(directory, prefix = "") {
  const result = [];
  for (const item of await readdir(directory, { withFileTypes: true })) {
    const name = prefix + item.name;
    if (item.isDirectory()) result.push(...await assetPaths(join(directory, item.name), `${name}/`));
    else if (/\.(js|css)$/.test(name) || name === "index.html") result.push(name);
  }
  return result.sort();
}

async function qualification(page, wos) {
  const sourceCommit = execFileSync("git", ["rev-parse", "HEAD"], { cwd: repository, encoding: "utf8" }).trim();
  const expectedCommit = process.env.WOS_CAPTURE_SOURCE_COMMIT || sourceCommit;
  expect(sourceCommit, "The checked-out source is the requested capture revision").toBe(expectedCommit);
  const dirty = execFileSync("git", ["status", "--porcelain", "--untracked-files=all", "--", "packages", "deploy", "tools"],
    { cwd: repository, encoding: "utf8" });
  expect(dirty, "Commit production code before capturing source-bound evidence").toBe("");
  const version = JSON.parse(execFileSync(binary, ["--output", "json", "version"],
    { cwd: repository, encoding: "utf8" })).data;
  expect(version.commit, "The running fixture binary identifies the exact capture source").toBe(sourceCommit);
  const servedAssets = {};
  await Promise.all((await assetPaths(assets)).map(async (name) => {
    const response = await page.request.get(`${wos.url}/app/${name.split("/").map(encodeURIComponent).join("/")}`);
    expect(response.ok(), `${name} is served by the real fixture binary`).toBe(true);
    const actual = hash(await response.body());
    expect(actual, `Refuse stale embedded ${name}`).toBe(hash(await readFile(join(assets, name))));
    servedAssets[name] = actual;
  }));
  const fixtureSources = {};
  for (const name of ["operational-gallery.spec.mjs", "ux-fixture.mjs", "workspace-action-fixture.mjs", "canvas-fixture.mjs", "native-browser-zoom.mjs"])
    fixtureSources[`tests/web/${name}`] = hash(await readFile(new URL(name, import.meta.url)));
  const session = await page.context().newCDPSession(page);
  let browser;
  try { browser = await session.send("Browser.getVersion"); }
  finally { await session.detach(); }
  return { sourceCommit, binary: { ...version, sha256: hash(await readFile(binary)) },
    browser,
    servedAssetsSha256: Object.fromEntries(Object.entries(servedAssets).sort()), fixtureSourcesSha256: fixtureSources };
}

async function geometry(page) {
  return page.evaluate(() => {
    const measure = (selector) => {
      const element = document.querySelector(selector);
      if (!element || !element.checkVisibility()) return null;
      const rect = element.getBoundingClientRect(), style = getComputedStyle(element);
      return { x: rect.x, y: rect.y, width: rect.width, height: rect.height,
        right: rect.right, bottom: rect.bottom, clientHeight: element.clientHeight,
        scrollHeight: element.scrollHeight, scrollTop: element.scrollTop,
        fontSize: parseFloat(style.fontSize), fontFamily: style.fontFamily,
        lineHeight: parseFloat(style.lineHeight) || null };
    };
    return { viewport: { width: innerWidth, height: innerHeight, devicePixelRatio,
      visualViewportScale: visualViewport.scale },
      document: { width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight,
        scrollX, scrollY },
      elements: Object.fromEntries([".topbar", ".sidebar", ".outcome-heading", ".canvas-toolbar", ".canvas-viewport",
        ".canvas-footer", "#canvas-details", "#detail-title", "#detail-content", "#detail-actions",
        ".workspace-state", ".workspace-primary-actions", ".canvas-inspector-detail", ".canvas-inspector-slot",
        ".workspace-path", ".detail-description", "#command-form", "#command-form .dialog-body", "#fields", "#submit-command",
        "#area-title", "#content", "#notice", ".login-panel", "#token", "#login button"]
        .map((selector) => [selector, measure(selector)])),
      canvas: { zoom: Number(document.querySelector(".canvas-viewport")?.dataset.scale || 0),
        nodeKinds: [...document.querySelectorAll("[data-canvas-node]")]
          .map((element) => element.dataset.canvasNode.split(":")[0]) },
      inspectorFacets: [...document.querySelectorAll(".workspace-view [data-workspace-facet]")]
        .map((element) => ({ facet: element.dataset.workspaceFacet, text: element.textContent,
          visible: element.checkVisibility() })),
      activeControl: { tag: document.activeElement?.tagName, id: document.activeElement?.id,
        action: document.activeElement?.dataset.actionId || null } };
  });
}

async function unobscured(locator) {
  await expect(locator).toBeVisible();
  const value = await locator.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    const target = document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2);
    return { width: rect.width, height: rect.height, left: rect.left, right: rect.right,
      top: rect.top, bottom: rect.bottom, viewportWidth: innerWidth, viewportHeight: innerHeight,
      receivesPointer: element.contains(target) };
  });
  expect(value.left).toBeGreaterThanOrEqual(-1);
  expect(value.right).toBeLessThanOrEqual(value.viewportWidth + 1);
  expect(value.top).toBeGreaterThanOrEqual(-1);
  expect(value.bottom).toBeLessThanOrEqual(value.viewportHeight + 1);
  expect(value.receivesPointer).toBe(true);
  return value;
}

async function capture(page, metadata, name, state, extra = {}) {
  await page.evaluate(() => document.fonts.ready);
  await page.locator("img").evaluateAll((images) => Promise.all(images.filter((image) => image.checkVisibility()).map((image) => image.decode())));
  await page.mouse.move(0, 0);
  const measured = await geometry(page);
  expect(measured.document.width).toBeLessThanOrEqual(measured.viewport.width);
  expect(await page.locator("button").evaluateAll((controls) => controls
    .filter((control) => control.checkVisibility())
    .filter((control) => ![control.textContent, control.getAttribute("aria-label"), control.title]
      .some((label) => label?.trim())).length), "Visible buttons have an accessible textual name").toBe(0);
  const directory = join(resolve(output), metadata.sourceCommit.slice(0, 12));
  await mkdir(directory, { recursive: true });
  const path = join(directory, `${name}.png`);
  let exists = false;
  try { await access(path); exists = true; } catch { /* New artifact name. */ }
  expect(exists, `Capture must not overwrite existing evidence: ${name}`).toBe(false);
  let png, method;
  if (extra.nativeBrowserZoom) {
    const session = await page.context().newCDPSession(page);
    try {
      const result = await session.send("Page.captureScreenshot", { format: "png", captureBeyondViewport: false, fromSurface: true });
      png = Buffer.from(result.data, "base64");
      await writeFile(path, png, { flag: "wx" });
    } finally { await session.detach(); }
    method = "Chromium Page.captureScreenshot; actual visible native-zoom framebuffer; no clip or resizing";
  } else {
    png = await page.screenshot({ path, fullPage: false, animations: "disabled" });
    method = "Playwright visible viewport; no image resizing or content masking";
  }
  const pixelSize = { width: png.readUInt32BE(16), height: png.readUInt32BE(20) };
  if (extra.framebuffer) expect(pixelSize).toEqual(extra.framebuffer);
  else expect(pixelSize).toEqual({ width: Math.round(measured.viewport.width * measured.viewport.devicePixelRatio),
    height: Math.round(measured.viewport.height * measured.viewport.devicePixelRatio) });
  await writeFile(join(directory, `${name}.json`), JSON.stringify({ ...metadata, ...extra, state,
    image: `${name}.png`, screenshotSha256: hash(png), capturedAt: new Date().toISOString(),
    captureMethod: method, pixelSize, url: page.url(), ...measured }, null, 2) + "\n", { flag: "wx" });
}

async function seed(wos) {
  const data = await seedCanvas(wos, { title: "Plan, deliver, and verify together" });
  const owner = data.ref("work_item", data.tasks.ready);
  const criterion = await wos.command("add_criterion", { owner,
    expected_version: data.tasks.ready.version,
    title: "A new participant can resume the work from the recorded state", required: true,
    verification_mode: "evidence_review" });
  let task = (await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${owner.id}`)).value;
  await wos.command("set_work_item_assignees", { scope: data.scope, work_item_id: task.id,
    expected_version: task.version, assignee_refs: [{ kind: "agent", provider: "gallery-fixture", id: "coding-agent" }] });
  task = (await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${owner.id}`)).value;
  await wos.command("append_entity_note", { scope: data.scope, subject_ref: owner,
    type: "update", body: "The implementation is ready for a separate review. This participant update does not accept the result." });
  const observation = await wos.command("register_evidence_with_links", {
    scope: data.scope,
    evidence: { evidence_type: "attestation", description: "A participant recorded how to resume the handoff from the shared state." },
    observation: { subject_ref: owner, criterion_id: criterion.id,
      body: "A participant followed the recorded handoff. Independent criterion assessment is still pending." },
    links: [{ target_ref: owner, target_expected_version: String(task.version),
      criterion_id: criterion.id, criterion_revision: String(criterion.criterion_revision), stance: "supports",
      rationale: "Documents the participant's observation for a separate reviewer." }],
  });
  const roadmap = await seedCanvasRoadmap(wos, data, { published: true, active: true });
  const base = `/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}`;
  let reviewedObjective = (await wos.api(`${base}/objectives/${data.objective.id}`)).value;
  const objectiveOwner = data.ref("objective", reviewedObjective);
  const objectiveCriterion = await wos.command("add_criterion", { owner: objectiveOwner,
    expected_version: reviewedObjective.version, title: "The agreed handoff was independently reviewed",
    required: true, verification_mode: "attestation" });
  reviewedObjective = (await wos.api(`${base}/objectives/${reviewedObjective.id}`)).value;
  await wos.command("attest_criterion", { owner: objectiveOwner, expected_version: reviewedObjective.version,
    criterion_id: objectiveCriterion.id, criterion_revision: objectiveCriterion.criterion_revision,
    result: "met", rationale: "An authorized participant reviewed the current handoff condition. Open follow-up work remains separate." });
  reviewedObjective = (await wos.api(`${base}/objectives/${reviewedObjective.id}`)).value;
  return { ...data, criterion, observation, roadmap, reviewedObjective };
}

async function selectRecord(page, record, heading) {
  const search = page.getByRole("combobox", { name: "Search Outcome Board", exact: true });
  await search.fill(record.title || record.name || record.description);
  await page.getByRole("option").filter({ hasText: record.title || record.name || record.description }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(heading || record.title || record.name);
  await page.waitForURL((url) => url.searchParams.get("item")?.endsWith(`/${record.id}`));
  await expect(panel(page).locator(".workspace-view")).toBeVisible();
}

async function boardReady(page) {
  await expect(map(page)).toBeVisible();
  await expect.poll(() => page.locator("[data-canvas-node]").count()).toBeGreaterThan(0);
  await unobscured(page.getByRole("combobox", { name: "Search Outcome Board", exact: true }));
  const value = await geometry(page);
  expect(value.document.height).toBeLessThanOrEqual(value.viewport.height);
  expect(value.document.scrollY).toBe(0);
  expect(value.canvas.zoom).toBeGreaterThanOrEqual(.75);
}

async function inspectorReady(page) {
  await unobscured(panel(page).locator("#close-detail"));
  await panel(page).locator('[data-action-id="more_actions"]').focus();
  await expect(panel(page).locator('[data-action-id="more_actions"]')).toBeFocused();
  await unobscured(panel(page).locator('[data-action-id="more_actions"]'));
  // Return through real focus and wheel interactions after checking the footer.
  // Narrow inspectors scroll as one surface; larger ones scroll their body.
  await panel(page).locator("#close-detail").focus();
  await panel(page).locator("#detail-content").hover();
  await page.mouse.wheel(0, -10000);
  await expect.poll(() => panel(page).locator("#detail-content").evaluate((element) => element.scrollTop)).toBe(0);
  await expect.poll(() => panel(page).evaluate((element) => [element,
    ...element.querySelectorAll(".canvas-inspector-detail, #detail-content")].every((surface) => surface.scrollTop === 0))).toBe(true);
}

async function openAction(page, name) {
  const direct = panel(page).locator(`[data-action-id="${name}"]`).filter({ visible: true });
  if (await direct.count()) await direct.first().click();
  else {
    await panel(page).locator('[data-action-id="more_actions"]').click();
    const target = page.locator(`#action-options [data-action-id="${name}"]`);
    await revealMenuAction(target);
    await target.click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
}

async function mainStates(page, wos, data, metadata, label, extra = {}) {
  await page.goto(`${wos.url}/app/`);
  await page.locator("#token").focus();
  await page.keyboard.press("Tab");
  await expect(page.locator("#login button")).toBeFocused();
  await unobscured(page.locator("#login button"));
  await capture(page, metadata, `login-${label}`, "Credential login before any secret is entered", extra);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  const mapControl = page.getByRole("button", { name: "Map view", exact: true });
  const selectedCompactMap = await mapControl.getAttribute("aria-pressed") !== "true";
  if (selectedCompactMap) {
    await expect(page.getByRole("list", { name: "Outcome Board list", exact: true })).toBeVisible();
    await capture(page, metadata, `board-default-list-${label}`,
      "Default compact Outcome Board list before switching to the spatial map",
      { ...extra, boardView: "list", defaultBoardView: true });
    await unobscured(mapControl);
    await mapControl.click();
  }
  await boardReady(page);
  const kinds = (await geometry(page)).canvas.nodeKinds;
  expect(kinds.every((kind) => ["outcome", "objective", "work_item"].includes(kind))).toBe(true);
  await capture(page, metadata, `board-${label}`, selectedCompactMap
    ? "Structural work map selected through the real Map view control; compact entry defaults to List view"
    : "Default structural work map; documentary records remain contextual",
    { ...extra, boardView: "map", defaultBoardView: !selectedCompactMap });
  await selectRecord(page, data.tasks.ready);
  for (const facet of ["work", "people", "attention", "criteria", "activity"])
    await expect(panel(page).locator(`[data-workspace-facet="${facet}"]`)).toBeVisible();
  await expect(panel(page)).toContainText("Declared assignee");
  await expect(panel(page)).toContainText("Not assessed");
  await inspectorReady(page);
  if (!extra.nativeBrowserZoom) {
    const initial = await geometry(page), prose = initial.elements[".detail-description"];
    const surface = initial.elements["#canvas-details"], body = initial.elements["#detail-content"];
    const visibleHeight = Math.min(prose.bottom, surface.bottom, body.bottom)
      - Math.max(prose.y, surface.y, body.y);
    expect(visibleHeight, "The actual initial Task reading position exposes its authored prose")
      .toBeGreaterThanOrEqual(Math.min(prose.height, prose.lineHeight * 3) - 2);
  }
  await capture(page, metadata, `task-${label}`, "Task inspector with requested work, participation, obligations, pending criterion and recent documentary activity", extra);
}

async function detailedStates(page, wos, data, metadata, label) {
  await panel(page).locator('[data-action-id="more_actions"]').click();
  await expect(page.locator(".canvas-inspector-actions")).toBeVisible();
  await capture(page, metadata, `task-actions-${label}`,
    "Read-only Task action menu with contextual groups; opening the menu does not execute an action");
  await page.keyboard.press("Escape");
  await expect(panel(page)).toHaveAttribute("data-surface", "detail");
  await inspectorReady(page);
  const criteria = panel(page).locator(`[data-criterion-id="${data.criterion.id}"]`);
  await criteria.scrollIntoViewIfNeeded();
  await capture(page, metadata, `task-proof-${label}`, "Task body scrolled naturally to current success conditions and pending assessment");
  await panel(page).locator('[data-workspace-facet="activity"]').scrollIntoViewIfNeeded();
  await capture(page, metadata, `task-activity-${label}`, "Task body scrolled naturally to documentary activity, with its source and authorship");
  await criteria.locator('[data-action-id="open_criterion"]').click();
  await expect(panel(page).locator("#detail-title")).toHaveText(data.criterion.title);
  await inspectorReady(page);
  await capture(page, metadata, `criterion-${label}`, "Criterion within its owner; linked Evidence remains separate from assessment");
  await selectRecord(page, data.observation.evidence, "Evidence");
  await inspectorReady(page);
  await capture(page, metadata, `evidence-record-${label}`, "Evidence with a recorded participant observation source; no automatic acceptance");
  await selectRecord(page, data.issue);
  await expect(panel(page)).toContainText(data.tasks.blocked.title);
  await inspectorReady(page);
  await capture(page, metadata, `issue-${label}`, "Issue with named affected work and a distinct explicit blocking impact");
  await selectRecord(page, data.blocker, "Blocker");
  await inspectorReady(page);
  await capture(page, metadata, `blocker-${label}`, "Blocker identifies its exact affected work and recorded cause");
  await selectRecord(page, data.objective);
  await openAction(page, "create_work_item");
  await page.getByLabel("Task title", { exact: true }).fill("Review the recorded customer handoff");
  await expect(page.locator("#fields")).toContainText("Planned in backlog");
  await expect(page.locator("#fields")).toContainText(data.objective.title);
  await page.locator("#submit-command").focus();
  await unobscured(page.locator("#submit-command"));
  await capture(page, metadata, `quick-task-${label}`, "Unsaved quick Task draft with fixed parent and visible backlog default");
  await page.keyboard.press("Escape");
  await openAction(page, "create_plan_items");
  await page.getByLabel("One title per line", { exact: true }).fill("Review the installation route\nRecord the verification transcript\nPrepare the customer handoff");
  await page.getByRole("button", { name: "Preview plan", exact: true }).click();
  await expect(page.locator(".workspace-plan-row")).toHaveCount(3);
  await capture(page, metadata, `batch-plan-${label}`, "Explicit unsaved plan preview; no existing work is moved");
  await page.keyboard.press("Escape");
  await selectRecord(page, data.reviewedObjective);
  await openAction(page, "achieve_objective");
  await page.getByLabel("Reason", { exact: true }).fill("The agreed condition was reviewed. The listed follow-up work retains its own lifecycle.");
  const conclusion = page.locator(".workspace-conclusion-impact");
  await expect(conclusion.getByRole("heading", { name: "Review conclusion impact", exact: true })).toBeVisible();
  await expect(conclusion).toContainText(data.child.title);
  await expect(page.getByLabel("I understand that this open work is not completed by this conclusion", { exact: true })).not.toBeChecked();
  await page.locator("#submit-command").focus();
  await unobscured(page.locator("#submit-command"));
  await conclusion.getByRole("heading", { name: "Review conclusion impact", exact: true }).scrollIntoViewIfNeeded();
  await capture(page, metadata, `conclusion-impact-${label}`, "Unsaved parent conclusion deliberately scrolled to its open-work review; earlier Reason fields remain above, and no acceptance has been submitted");
  await page.keyboard.press("Escape");
  await selectRecord(page, data.tasks.ready);
  await openAction(page, "set_work_item_objective");
  await page.getByRole("button", { name: "Change placement", exact: true }).click();
  await pick(page, "New placement", data.objective.title);
  await page.getByRole("button", { name: "Preview move", exact: true }).click();
  const movement = page.locator(".workspace-move-preview");
  await expect(movement.getByRole("heading", { name: "Observed movement impact", exact: true })).toBeVisible();
  await expect(movement).toContainText(data.child.title);
  await expect(movement).toContainText(data.objective.title);
  await expect(page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true })).not.toBeChecked();
  await page.locator("#submit-command").focus();
  await unobscured(page.locator("#submit-command"));
  await movement.getByRole("heading", { name: "Observed movement impact", exact: true }).scrollIntoViewIfNeeded();
  await capture(page, metadata, `move-impact-${label}`, "Read-only move form deliberately scrolled to actual old/proposed placement, blocking and readiness; earlier destination controls remain above and no move is saved");
  await page.keyboard.press("Escape");
  await selectRecord(page, data.objective);
  await openAction(page, "create_issue");
  await page.getByLabel("Issue title", { exact: true }).fill("Review access prevents this branch from continuing");
  await page.getByLabel("Description", { exact: true }).fill("An explicit problem and its proposed blocking scope are reviewed separately from completion.");
  await page.getByLabel("Prevents continuing?", { exact: true }).check();
  await page.getByLabel("Blocking impact", { exact: true }).fill("Wait for verified access before continuing work on this branch.");
  await page.getByLabel("Propagation", { exact: true }).selectOption("subtree");
  const blocking = page.locator(".workspace-blocking-preview");
  await blocking.getByRole("button", { name: "Preview blocking impact", exact: true }).click();
  await expect(blocking.getByRole("heading", { name: "Open work affected by this Blocker", exact: true })).toBeVisible();
  await expect(blocking).toContainText(data.child.title);
  await expect(page.getByLabel("I reviewed the open work affected by this Blocker", { exact: true })).not.toBeChecked();
  await page.locator("#submit-command").focus();
  await unobscured(page.locator("#submit-command"));
  await blocking.getByRole("heading", { name: "Open work affected by this Blocker", exact: true }).scrollIntoViewIfNeeded();
  await capture(page, metadata, `blocking-impact-${label}`, "Unsaved contextual Issue deliberately scrolled to its subtree blocking review; earlier Issue fields remain above and no records are saved");
  await page.keyboard.press("Escape");
  await selectRecord(page, data.roadmap);
  await expect(panel(page)).toContainText("Publication and activation are separate");
  await inspectorReady(page);
  await capture(page, metadata, `roadmap-${label}`, "Roadmap publishes and activates a plan independently of operational acceptance");
  await page.keyboard.press("Escape");
  for (const area of ["overview", "work", "plan", "evidence", "issues", "activity"]) {
    await chooseArea(page, area);
    await expect(page.locator("#area-title")).toBeVisible();
    const view = await geometry(page);
    expect(view.document.width).toBeLessThanOrEqual(view.viewport.width);
    await capture(page, metadata, `${area}-${label}`, `Current ${area} specialist view using the retained editorial workspace`);
  }
}

if (output) {
  for (const [width, height] of [[1280, 720], [1366, 768], [1920, 1080], [2560, 1440],
    [3840, 2160], [320, 740], [390, 844], [430, 932]]) {
    test(`source-bound operational gallery at ${width} x ${height}`, async ({ page, wos }) => {
      test.setTimeout(240000);
      await page.setViewportSize({ width, height });
      await page.emulateMedia({ reducedMotion: "reduce" });
      const data = await seed(wos);
      const metadata = { ...await qualification(page, wos), fixtureScope: data.scope,
        fixtureKind: "Representative records created through authenticated public commands; not a user study" };
      const errors = [], mutations = [];
      page.on("pageerror", (error) => errors.push(error.message));
      page.on("request", (request) => {
        if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
      });
      const label = `${width}x${height}`;
      await mainStates(page, wos, data, metadata, label);
      if (width === 2560 || width === 390) await detailedStates(page, wos, data, metadata, label);
      expect(errors).toEqual([]);
      expect(mutations, "Inspecting and previewing the workspace never writes domain state").toEqual([]);
    });
  }
  for (const factor of [1.25, 1.5, 2]) {
    test(`source-bound operational gallery at native ${factor * 100}% browser zoom`, async ({ wos }) => {
      test.setTimeout(180000);
      const data = await seed(wos);
      const native = await nativeZoomBrowser({ width: 1366, height: 768 });
      try {
        const page = native.page;
        await native.setZoom(factor);
        await page.emulateMedia({ reducedMotion: "reduce" });
        const metadata = { ...await qualification(page, wos), fixtureScope: data.scope,
          fixtureKind: "Representative records created through authenticated public commands; not a user study" };
        await mainStates(page, wos, data, metadata, `native-${factor * 100}`, {
          nativeBrowserZoom: factor, framebuffer: { width: 1366, height: 768 },
        });
      } finally { await native.close(); }
    });
  }
}
