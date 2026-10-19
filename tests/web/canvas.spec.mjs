import { expandWorkspaceDetails } from './workspace-action-fixture.mjs';
import { test, expect, openOutcome, pick, chooseArea, clickWorkspaceTool, openCanvasContext } from "./ux-fixture.mjs";
import { clickRecordAction } from "./workspace-action-fixture.mjs";
import { seedCanvas, provisionCanvasViewer, seedCanvasRoadmap } from "./canvas-fixture.mjs";
import AxeBuilder from "@axe-core/playwright";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";

const canvas = (page) => page.locator("#operational-canvas");
const panel = (page) => page.locator("#canvas-details");
const node = (page, kind, value) => canvas(page).locator(
  `.workspace-canvas[data-view="map"] [data-canvas-node="${kind}:${value.id}"] .canvas-node-main, .workspace-canvas[data-view="list"] [data-canvas-list-node="${kind}:${value.id}"] .canvas-list-main`,
);
const endpoint = (scope) => `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
async function openCanvas(page, wos, data, credential) {
  await wos.login(page, credential);
  await openOutcome(page, data.outcome.title, { area: null });
  await expect(page.locator('#area-tabs [data-area="canvas"]')).toHaveAttribute("aria-pressed", "true");
  await expect(canvas(page)).toBeVisible();
}
async function choose(page, kind, value) {
  // Search preserves the authored label; the inspector distinguishes concise
  // record identity from the full description of records without a title.
  const searchLabel = value.title || value.name || value.description;
  await canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(searchLabel);
  await canvas(page).getByRole("option").filter({ hasText: searchLabel }).first().click();
  const kindHeading = {
    outcome: "Outcome", objective: "Objective", work_item: "Task", issue: "Issue",
    blocker: "Blocker", evidence: "Evidence", artifact: "Artifact", decision: "Decision", roadmap: "Roadmap",
  }[kind];
  await expect(panel(page).locator("#detail-title")).toHaveText(value.title || value.name || kindHeading);
  await expect(panel(page)).toBeVisible();
  if (!value.title && !value.name) {
    const body = panel(page).locator("#detail-content");
    await expect(body.locator(".detail-description")).toHaveText(value.description);
    // Named relationship endpoints can repeat the record's authored label.
    // The primary description itself must still be rendered exactly once.
    await expect(body.locator(".detail-description")).toHaveCount(1);
  }
  await expect(page.locator("#detail-dialog")).not.toBeVisible();
}
async function finishInline(page) {
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator("#command-form")).not.toBeVisible();
  // The form closes after commit; discovery, refresh and detail restoration
  // still hold navigation until this real completion boundary is released.
  await expect(page.locator("#submit-command")).toBeEnabled();
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  await expect(canvas(page)).toBeVisible();
}
async function originalPapers(page) {
  const papers = canvas(page).locator(".canvas-node > .canvas-node-paper");
  await expect(papers).toHaveCount(await canvas(page).locator(".canvas-node").count());
  await expect.poll(() => papers.evaluateAll((images) => images.every((image) =>
    image.complete && image.naturalWidth > 0 &&
    new URL(image.src).origin === location.origin && image.src.includes("/postits/svg/") && image.src.endsWith(".svg") &&
    image.alt === "" && image.getAttribute("aria-hidden") === "true" && !image.draggable,
  ))).toBe(true);
}
async function shot(page, name, { preserveScroll = false } = {}) {
  if (!process.env.WOS_CAPTURE_CANVAS) return;
  const directory = join(process.env.WOS_CAPTURE_CANVAS, "screenshots");
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await canvas(page).locator(".canvas-node-paper").evaluateAll((images) => Promise.all(images.map((image) => image.decode())));
  if (!preserveScroll && !name.startsWith("04-canvas-mobile")) await page.evaluate(() => scrollTo(0, 0));
  await page.screenshot({ path: join(directory, `${name}.png`), fullPage: false, animations: "disabled" });
  if (["01-canvas-workspace", "02-canvas-relationships", "06-canvas-roadmap"].includes(name))
    await page.locator(".canvas-workspace-shell").screenshot({ path: join(directory, `${name}-board.png`), animations: "disabled" });
}

test("Outcome Board is the default operational view without a Roadmap and keeps contextual deep links", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.setViewportSize({ width: 1440, height: 1000 });
  await openCanvas(page, wos, data);
  for (const [kind, value] of [["outcome", data.outcome], ["objective", data.objective], ["objective", data.child], ["work_item", data.tasks.unscoped]])
    await expect(node(page, kind, value)).toBeVisible();
  await originalPapers(page);
  expect((await wos.api(`${endpoint(data.scope)}/roadmaps`)).items).toEqual([]);
  await node(page, "outcome", data.outcome).click();
  await expect(panel(page).locator("#detail-title")).toHaveText(data.outcome.title);
  await choose(page, "work_item", data.tasks.ready);
  await expandWorkspaceDetails(panel(page), {history:false});
  await expect(panel(page).getByText("Perform only the scoped task", { exact: true })).toBeVisible();
  await expect(panel(page).locator(".workspace-state")).toBeVisible();
  await expect(panel(page).locator('[data-workspace-facet="attention"]')).toBeVisible();
  await expect(page).toHaveURL(new RegExp(`area=canvas.*item=work_item(?:%2F|/)${data.tasks.ready.id}`));
  await shot(page, "01-canvas-workspace");
  const selectedRoute = page.url();
  data.tasks.ready = await wos.command("update_work_item", {
    scope: data.scope, work_item_id: data.tasks.ready.id,
    expected_version: data.tasks.ready.version, title: "A shared Task updated by another participant",
  });
  await clickWorkspaceTool(page, "refresh");
  await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
  await expect(node(page, "work_item", data.tasks.ready)).toContainText(data.tasks.ready.title);
  expect(page.url()).toBe(selectedRoute);
  await page.reload();
  await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
  await expect(canvas(page)).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page).not.toHaveURL(/item=/);
  await expect(node(page, "work_item", data.tasks.ready)).toBeFocused();
  await page.goBack();
  await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
  await chooseArea(page, "Work");
  await expect(page.getByLabel("Work status", { exact: true })).toBeVisible();
  await chooseArea(page, "Canvas");
  await expect(canvas(page)).toBeVisible();
  // A real context response that arrives after dismissal must not reopen the
  // inspector or restore a selection the person already closed.
  await choose(page, "outcome", data.outcome);
  let releaseContext, reachedContext;
  const held = new Promise((resolve) => { releaseContext = resolve; });
  const reached = new Promise((resolve) => { reachedContext = resolve; });
  const contextPath = "**/workspace-view";
  await page.route(contextPath, async (route) => {
    if (route.request().postDataJSON()?.subject?.ref?.id !== data.tasks.ready.id)
      return route.continue();
    const response = await route.fetch();
    reachedContext();
    await held;
    await route.fulfill({ response });
  });
  try {
    await canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(data.tasks.ready.title);
    await canvas(page).getByRole("option").filter({ hasText: data.tasks.ready.title }).first().click();
    await reached;
    await panel(page).locator("#close-detail").click();
    await expect(panel(page)).not.toBeVisible();
    const response = page.waitForResponse((value) => value.url().endsWith("/workspace-view")
      && value.request().postDataJSON()?.subject?.ref?.id === data.tasks.ready.id);
    releaseContext();
    await response;
    await page.evaluate(async () => { if (!(await fetch("/livez")).ok) throw Error("health fence failed"); });
    await expect(panel(page)).not.toBeVisible();
    await expect(page).not.toHaveURL(/item=/);
  } finally {
    releaseContext();
    await page.unroute(contextPath);
  }
  expect(errors).toEqual([]);
});

test("contextual creation and editing reuse the real form and preserve Objective association", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await openCanvas(page, wos, data);
  await choose(page, "objective", data.child);
  await clickRecordAction(page, "New task");
  await expect(panel(page).locator("#command-form")).toBeVisible();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
  await page.getByLabel("Task title", { exact: true }).fill("Created in the review context");
  await finishInline(page);
  const task = (await wos.api(`${endpoint(data.scope)}/work-items`)).items.find((item) => item.title === "Created in the review context");
  expect(task).toBeTruthy();
  expect(task.objective_id).toBe(data.child.id);
  await choose(page, "work_item", task);
  await clickRecordAction(page, "Edit task");
  await page.getByLabel("Task title", { exact: true }).fill("A precisely edited contextual Task");
  await finishInline(page);
  await expect(node(page, "work_item", task)).toContainText("A precisely edited contextual Task");
  const persisted = (await wos.api(`${endpoint(data.scope)}/work-items/${task.id}`)).value;
  expect(persisted.objective_id).toBe(data.child.id);
  expect(persisted.version).toBe(task.version + 1);
  expect((await wos.api(`${endpoint(data.scope)}/objectives/${data.child.id}`)).value.lifecycle).toBe(data.child.lifecycle);
});

test("real connections distinguish hierarchy, dependency strength and Evidence stance; Issue resolution retains its Blocker", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await page.setViewportSize({ width: 1440, height: 1000 });
  await openCanvas(page, wos, data);
  // The initial map has a declared depth bound. Selecting the real Blocker
  // loads one authorized neighborhood so its Issue cause is revealed.
  await choose(page, "blocker", data.blocker);
  await canvas(page).getByLabel("Map layers", { exact: true }).selectOption("records");
  await canvas(page).getByLabel("Connections", { exact: true }).selectOption("all");
  for (const kind of ["hierarchy", "ownership", "blocks", "causes", "supports", "contradicts", "context"])
    await expect(canvas(page).locator(`[data-canvas-edge="${kind}"]`).first()).toBeAttached();
  await expect(canvas(page).locator('[data-canvas-edge="depends_on"][data-strength="hard"]')).toHaveCount(1);
  await expect(canvas(page).locator('[data-canvas-edge="depends_on"][data-strength="advisory"]')).toHaveCount(1);
  await choose(page, "issue", data.issue);
  await clickRecordAction(page, "Resolve issue");
  await page.getByLabel("Resolution summary", { exact: true }).fill("Access was restored; blocking impact is reviewed separately");
  await finishInline(page);
  const blocker = (await wos.api(`${endpoint(data.scope)}/blockers/${data.blocker.id}`)).value;
  expect(blocker.lifecycle).toBe("active");
  await choose(page, "work_item", data.tasks.blocked);
  await expect(panel(page)).toContainText("Blocked");
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.blocked.id}/operational-state`)).state.is_blocked).toBe(true);
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.advisory.id}/operational-state`)).state.display_state).toBe("ready");
  await choose(page, "evidence", data.evidence[2]);
  await clickRecordAction(page, "Link evidence");
  await pick(page, "Related item", data.tasks.unscoped.title);
  await page.getByLabel("Evidence stance", { exact: true }).selectOption("context");
  await page.getByLabel("Rationale", { exact: true }).fill("The observation provides context, without claiming that a criterion is met");
  await finishInline(page);
  const links = (await wos.api(`${endpoint(data.scope)}/evidence-links`)).items;
  expect(links.some((link) => link.evidence_id === data.evidence[2].id && link.target_ref.id === data.tasks.unscoped.id && link.stance === "context")).toBe(true);
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.unscoped.id}`)).value.lifecycle).toBe("todo");
  await expandWorkspaceDetails(panel(page), {history:false});
  const provider = panel(page).getByText("canvas-acceptance", { exact: true });
  await expect(provider).toBeVisible();
  expect(await provider.evaluate((element) => element.getClientRects().length)).toBeLessThanOrEqual(2);
  await shot(page, "02-canvas-relationships");
});

test("large Outcomes declare partial data, load incrementally and find authorized off-page Tasks", async ({ page, wos }) => {
  const data = await seedCanvas(wos, { extraTasks: 130 });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await openCanvas(page, wos, data);
  await expect(canvas(page)).toContainText("Partial map");
  const loadedCount = async () => Number((await canvas(page).locator(".canvas-status").textContent()).match(/(\d+) loaded items/)[1]);
  const before = await loadedCount();
  expect(before).toBeLessThan(143);
  await canvas(page).getByRole("button", { name: "Load more map items", exact: true }).click();
  await expect.poll(loadedCount).toBeGreaterThan(before);
  const remote = data.additional.at(-1);
  await canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill("off-page verification");
  await canvas(page).getByRole("option").filter({ hasText: remote.title }).click();
  await expect(panel(page).locator("#detail-title")).toHaveText(remote.title);
  await expect(node(page, "work_item", remote)).toBeVisible();
  await expect(page).toHaveURL(new RegExp(remote.id));
  await shot(page, "03-canvas-large-outcome");
});

test("authorized dependency editing persists advisory links, refuses cycles and explicitly removes the selected relation", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await openCanvas(page, wos, data);
  await choose(page, "work_item", data.tasks.unscoped);
  await clickRecordAction(page, "Add dependency");
  await pick(page, "Depends on", data.tasks.ready.title);
  await page.getByLabel("Dependency type", { exact: true }).selectOption("advisory");
  await page.getByLabel("Why is this prerequisite required?", { exact: true }).fill("Share context without preventing this work from starting");
  await finishInline(page);
  const relation = (await wos.api(`${endpoint(data.scope)}/relations`)).items.find((item) => item.source_ref.id === data.tasks.unscoped.id && item.target_ref.id === data.tasks.ready.id);
  expect(relation.strength).toBe("advisory");
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.unscoped.id}/operational-state`)).state.display_state).toBe("ready");
  await choose(page, "work_item", data.tasks.ready);
  await clickRecordAction(page, "Add dependency");
  await pick(page, "Depends on", data.tasks.unscoped.title);
  await page.getByLabel("Why is this prerequisite required?", { exact: true }).fill("Attempting the reverse edge must still be validated by the server");
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator("#command-error")).toContainText(/cycl/i);
  expect((await wos.api(`${endpoint(data.scope)}/relations`)).items.filter((item) => item.lifecycle === "active")).toHaveLength(3);
  await page.keyboard.press("Escape");
  await canvas(page).getByLabel("Connections", { exact: true }).selectOption("all");
  const edge = canvas(page).locator(`[data-edge-key="graph:relation:${relation.id}"]`);
  await edge.press("Enter");
  await clickRecordAction(page, "Remove dependency");
  await page.getByLabel("Reason", { exact: true }).fill("The advisory context no longer needs a persisted dependency");
  await finishInline(page);
  await expect(edge).toHaveCount(0);
  const persisted = (await wos.api(`${endpoint(data.scope)}/relations`)).items.find((item) => item.id === relation.id);
  expect(persisted).toBeTruthy();
  expect(persisted.lifecycle).toBe("removed");
  expect(persisted.version).toBe(relation.version + 1);
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.unscoped.id}`)).value.lifecycle).toBe("todo");
});

test("read-only Outcome Board has usable keyboard and mobile views while zoom and layout never mutate state", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  const viewer = await provisionCanvasViewer(wos);
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await openCanvas(page, wos, data, viewer.token);
  await expect(page.locator("#canvas-view").getByRole("button", { name: "Add objective", exact: true })).toHaveCount(0);
  await expect(page.locator("#canvas-view").getByRole("button", { name: "New task", exact: true })).toHaveCount(0);
  await node(page, "work_item", data.tasks.ready).focus();
  await page.keyboard.press("Enter");
  await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
  await expect(panel(page).getByRole("button", { name: "Edit task", exact: true })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(node(page, "work_item", data.tasks.ready)).toBeFocused();
  const zoom = canvas(page).getByLabel("Outcome Board zoom", { exact: true });
  const start = await zoom.textContent();
  await canvas(page).getByRole("button", { name: "Zoom in", exact: true }).click();
  await expect(zoom).not.toHaveText(start);
  await canvas(page).getByRole("button", { name: "Zoom out", exact: true }).click();
  await canvas(page).getByRole("button", { name: "Fit view", exact: true }).click();
  const world = canvas(page).locator(".canvas-world");
  const beforePan = await world.evaluate((element) => getComputedStyle(element).transform);
  await canvas(page).locator(".canvas-viewport").focus();
  await page.keyboard.press("ArrowRight");
  await expect(world).not.toHaveCSS("transform", beforePan);
  await canvas(page).getByRole("button", { name: "Auto-layout", exact: true }).click();
  // Secondary node actions are used at a readable map scale. The subsequent
  // Auto-layout returns to the fitted overview for its separate axe audit.
  for (let step = 0; step < 12 && parseFloat(await zoom.textContent()) < 75; step++)
    await canvas(page).getByRole("button", { name: "Zoom in", exact: true }).click();
  await choose(page, "objective", data.objective);
  await page.keyboard.press("Escape");
  await canvas(page).getByRole("button", { name: `Collapse ${data.objective.title}`, exact: true }).click();
  await expect(node(page, "work_item", data.tasks.ready)).toHaveCount(0);
  await canvas(page).getByRole("button", { name: `Expand ${data.objective.title}`, exact: true }).click();
  await expect(node(page, "work_item", data.tasks.ready)).toBeVisible();
  await choose(page, "outcome", data.outcome);
  await page.keyboard.press("Escape");
  const card = canvas(page).locator(`[data-canvas-node="outcome:${data.outcome.id}"]`);
  const position = await card.evaluate((element) => ({ left: element.style.left, top: element.style.top }));
  const bounds = await node(page, "outcome", data.outcome).boundingBox();
  await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
  await page.mouse.down();
  await page.mouse.move(bounds.x + bounds.width / 2 + 50, bounds.y + bounds.height / 2 + 30, { steps: 5 });
  await page.mouse.up();
  await expect.poll(() => card.evaluate((element) => ({ left: element.style.left, top: element.style.top }))).not.toEqual(position);
  await canvas(page).getByRole("button", { name: "Auto-layout", exact: true }).click();
  const audit = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
  expect(audit.violations.filter((violation) => ["critical", "serious"].includes(violation.impact)), JSON.stringify(audit.violations)).toEqual([]);
  const internalZoom = await canvas(page).locator(".canvas-viewport").getAttribute("data-scale");
  let referenceFont;
  for (const [width, height] of [[1440, 1000], [1920, 1080], [2560, 1440], [3840, 2160], [390, 844]]) {
    await page.setViewportSize({ width, height });
    await expect(canvas(page).locator(".canvas-viewport")).toHaveAttribute("data-scale", internalZoom);
    const titleFont = await canvas(page).locator(".canvas-node-title").first().evaluate((element) => parseFloat(getComputedStyle(element).fontSize));
    if (width === 1440) referenceFont = titleFont;
    else if (width > 1440) expect(titleFont).toBeCloseTo(referenceFont, 2);
    if (width === 390) await canvas(page).getByRole("button", { name: "List view", exact: true }).click();
    await expect(node(page, "work_item", data.tasks.ready)).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await choose(page, "work_item", data.tasks.ready);
    await expect(panel(page)).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    if (width === 390) await shot(page, "04-canvas-mobile");
    await page.keyboard.press("Escape");
    if (width === 390) {
      await canvas(page).getByRole("button", { name: "Map view", exact: true }).click();
      // The viewport checks above preserve the intentional fitted camera.
      // Use explicit controls for a readable Map demonstration afterwards.
      for (let step = 0; step < 12 && parseFloat(await zoom.textContent()) < 75; step++)
        await canvas(page).getByRole("button", { name: "Zoom in", exact: true }).click();
      expect(parseFloat(await zoom.textContent())).toBeGreaterThanOrEqual(75);
      await choose(page, "work_item", data.tasks.ready);
      await page.keyboard.press("Escape");
      await canvas(page).locator(".canvas-viewport").scrollIntoViewIfNeeded();
      await expect(canvas(page).getByRole("button", { name: "Map view", exact: true })).toHaveAttribute("aria-pressed", "true");
      await expect(canvas(page).locator(`[data-canvas-node="work_item:${data.tasks.ready.id}"]`)).toBeInViewport({ ratio: 0.9 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await shot(page, "04-canvas-mobile-map");
    }
  }
  await page.setViewportSize({ width: 2560, height: 1440 });
  await choose(page, "work_item", data.tasks.ready);
  await canvas(page).locator(".canvas-viewport").scrollIntoViewIfNeeded();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await shot(page, "05-canvas-2k", { preserveScroll: true });
  expect(mutations).toEqual([]);
});

test("inline conflicts preserve local edits and uncertain retries keep their exact bytes and key", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await openCanvas(page, wos, data);
  await choose(page, "work_item", data.tasks.ready);
  await clickRecordAction(page, "Edit task");
  await page.getByLabel("Task title", { exact: true }).fill("My retained canvas edit");
  await wos.command("update_work_item", { scope: data.scope, work_item_id: data.tasks.ready.id, expected_version: data.tasks.ready.version, title: "Another participant changed this Task" });
  await clickWorkspaceTool(page, "refresh");
  await expect(page.getByLabel("Task title", { exact: true })).toHaveValue("My retained canvas edit");
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator("#command-error")).toContainText("A newer version exists");
  await page.getByRole("button", { name: "Compare with latest state", exact: true }).click();
  await expect(page.locator("#conflict-review")).toContainText("Another participant changed this Task");
  await expect(page.locator("#conflict-review")).toContainText("My retained canvas edit");
  await page.getByRole("button", { name: "Use latest version and save my edits", exact: true }).click();
  await expect(panel(page).locator("#command-form")).not.toBeVisible();
  await choose(page, "work_item", { ...data.tasks.ready, title: "My retained canvas edit" });
  await clickRecordAction(page, "Edit task");
  await page.getByLabel("Task title", { exact: true }).fill("Saved once from the Canvas");
  const dispatched = [];
  let lose = true;
  await page.route("**/commands/update_work_item", async (route) => {
    dispatched.push({ key: route.request().headers()["idempotency-key"], body: route.request().postData() });
    if (lose) {
      lose = false;
      await route.fetch();
      await route.fulfill({ status: 503, contentType: "application/json", body: '{"error":{"message":"Response lost after commit"}}' });
    } else await route.continue();
  });
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator("#retry-command")).toBeVisible();
  await expect(panel(page).locator("#fields")).toHaveAttribute("inert", "");
  await chooseArea(page, "Work");
  await expect(canvas(page)).toBeVisible();
  await expect(panel(page).locator("#retry-command")).toBeVisible();
  await panel(page).locator("#retry-command").click();
  await expect(panel(page).locator("#command-form")).not.toBeVisible();
  expect(dispatched).toHaveLength(2);
  expect(dispatched[0]).toEqual(dispatched[1]);
  const current = (await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.ready.id}`)).value;
  expect(current.title).toBe("Saved once from the Canvas");
  expect(current.version).toBe(data.tasks.ready.version + 3);
});

test("Roadmap Draft, immutable publication and activation stay separate from operational dependency", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  const roadmap = await seedCanvasRoadmap(wos, data);
  await page.setViewportSize({ width: 1440, height: 1000 });
  await openCanvas(page, wos, data);
  await choose(page, "roadmap", roadmap);
  await expect(panel(page).locator('[data-workspace-facet="roadmap"]')).toContainText("Draft");
  await expandWorkspaceDetails(panel(page), {history:false});
  await panel(page).locator('[data-action-id="view_roadmap_draft"]').click();
  await expect(canvas(page)).toContainText("Discover and verify");
  await canvas(page).getByLabel("Connections", { exact: true }).selectOption("planning");
  await expect(canvas(page).locator('[data-canvas-edge="plan_sequence"]')).toHaveCount(1);
  await expect(canvas(page).locator('[data-canvas-edge="plan_reference"]')).toHaveCount(2);
  await expect(canvas(page).locator('.canvas-node[data-kind="phase"]')).toBeVisible();
  await choose(page, "roadmap", roadmap);
  await clickRecordAction(page, "Review and publish");
  await expect(page.locator("#fields")).toContainText("Publish");
  await page.getByLabel("Publication reason", { exact: true }).fill("Review the planning references explicitly");
  await finishInline(page);
  const path = `${endpoint(data.scope)}/roadmaps/${roadmap.id}`;
  const published = (await wos.api(path)).value;
  expect(published.revisions).toHaveLength(1);
  const original = JSON.stringify(published.revisions[0]);
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.unscoped.id}/operational-state`)).state.display_state).toBe("ready");
  await choose(page, "roadmap", published);
  await expandWorkspaceDetails(panel(page), {history:false});
  await panel(page).locator('[data-action-id="view_roadmap_revision"]').click();
  await canvas(page).getByLabel("Connections", { exact: true }).selectOption("planning");
  await expect(canvas(page).locator('[data-canvas-edge="plan_sequence"]')).toHaveCount(1);
  await expect.poll(async () => parseFloat(await canvas(page).getByLabel("Outcome Board zoom", { exact: true }).textContent())).toBeGreaterThanOrEqual(75);
  await expect(canvas(page).locator('.canvas-node[data-kind="phase"]')).toBeVisible();
  await originalPapers(page);
  await shot(page, "06-canvas-roadmap");
  await choose(page, "roadmap", published);
  await clickRecordAction(page, "Activate revision");
  await finishInline(page);
  await openCanvasContext(page);
  await expect(page.locator("#canvas-planning").getByRole("button", { name: "Active plan · revision 1", exact: true })).toBeVisible();
  await expect(canvas(page).locator('[data-canvas-edge="plan_sequence"]')).toHaveCount(1);
  expect(JSON.stringify((await wos.api(path)).value.revisions[0])).toBe(original);
  const graph = await wos.api(`${endpoint(data.scope)}/graph?depth=8&limit=100`);
  expect(graph.edges.filter((edge) => edge.kind === "depends_on")).toHaveLength(2);
  expect((await wos.api(`${endpoint(data.scope)}/work-items/${data.tasks.unscoped.id}`)).value.lifecycle).toBe("todo");
  await page.locator("#canvas-planning").getByRole("button", { name: "Hide plan", exact: true }).click();
  await expect(canvas(page).locator('[data-canvas-edge="plan_sequence"]')).toHaveCount(0);
  await canvas(page).getByLabel("Connections", { exact: true }).selectOption("all");
  await expect(canvas(page).locator('[data-canvas-edge="depends_on"]')).toHaveCount(2);
  await expect(node(page, "work_item", data.tasks.unscoped)).toBeVisible();
});
