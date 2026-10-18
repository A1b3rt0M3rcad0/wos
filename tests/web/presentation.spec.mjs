import { test, expect, openOutcome, intent, save, chooseArea, clickWorkspaceTool } from "./ux-fixture.mjs";
import { seedPresentation } from "./presentation-fixture.mjs";
import AxeBuilder from "@axe-core/playwright";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";

async function reflow(page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
}
async function audit(page) {
  const result = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
  expect(result.violations.filter((v) => ["critical", "serious"].includes(v.impact)), JSON.stringify(result.violations)).toEqual([]);
}
async function capture(page, name) {
  if (!process.env.WOS_PRESENTATION_SCREENSHOTS) return;
  const directory = process.env.WOS_PRESENTATION_SCREENSHOTS;
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.locator("img").evaluateAll((images) => Promise.all(images.filter((image) => image.checkVisibility()).map((image) => image.decode())));
  if (name.startsWith("01-mobile-")) await page.evaluate(() => {
    const items = document.querySelector("#items-view");
    const header = document.querySelector(".topbar").getBoundingClientRect().height;
    scrollTo(0, scrollY + items.getBoundingClientRect().top - header - 12);
  });
  await page.mouse.move(0, 0);
  await page.screenshot({ path: join(directory, `${name}.png`), fullPage: false, animations: "disabled" });
}
async function area(page, key, section) {
  await chooseArea(page, key);
  if (section && key !== "work") await page.locator(`#tabs [data-section="${section}"]`).click();
  if (section && key === "work") await page.locator("#work-status").selectOption(section);
  await expect(page.locator(`#area-tabs [data-area="${key}"]`)).toHaveAttribute("aria-pressed", "true");
  const heading = {
    objectives: "Objectives", roadmaps: "All roadmaps", active_roadmaps: "Active roadmaps",
    active_plan_references: "Active plan tasks", work_items: "Tasks", issues: "Issues",
    blockers: "Blockers", evidence: "Evidence", artifacts: "Artifacts", decisions: "Decisions",
    timeline: "Activity", conclusion_contestations: "Contested conclusions", graph: "Relations",
  }[section || { plan: "objectives", work: "work_items", issues: "issues", evidence: "evidence", activity: "timeline" }[key]];
  if (heading) await expect(page.locator("#content .content-heading h2")).toHaveText(heading);
  await reflow(page);
}
async function rowDetail(page, key, section, item) {
  await area(page, key, section);
  await page.locator(`#content [data-entity-id="${item.id}"]`).click();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await expect(page.locator("#detail-title")).toHaveText(item.title || item.name || item.description);
}
async function dialogAccess(page, id) {
  const dialog = page.locator(id);
  await expect(dialog).toBeVisible();
  expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  await page.keyboard.press("Tab");
  expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  await reflow(page);
}
async function closeCommand(page) {
  const expanded = page.locator('#command-dialog [role="combobox"][aria-expanded="true"]');
  if (await expanded.count()) {
    await expanded.first().press("Escape");
    await expect(expanded).toHaveCount(0);
  }
  await page.keyboard.press("Escape");
  await expect(page.locator("#command-dialog")).not.toBeVisible();
}
async function roadmapRow(page, title) {
  for (const row of await page.locator(".roadmap-edit-row").all())
    if (await row.getByLabel("Item title", { exact: true }).inputValue() === title) return row;
  throw Error(`Roadmap item was not rendered: ${title}`);
}

test("specialist areas show real documentary records, section-aware filters and mobile alternatives", async ({ page, wos }) => {
  test.setTimeout(180000);
  const data = await seedPresentation(wos);
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await wos.login(page);
  await openOutcome(page, data.outcome.title);
  const views = [
    ["overview", null, "overview"],
    ["plan", "objectives", "objectives"],
    ["plan", "roadmaps", "roadmaps"],
    ["plan", "active_roadmaps", "active-roadmaps"],
    ["plan", "active_plan_references", "plan-references"],
    ["work", "work_items", "tasks"],
    ["issues", "issues", "issues"],
    ["issues", "blockers", "blockers"],
    ["evidence", "evidence", "evidence"],
    ["evidence", "artifacts", "artifacts"],
    ["evidence", "decisions", "decisions"],
    ["activity", "timeline", "timeline"],
    ["activity", "conclusion_contestations", "contested-conclusions"],
    ["activity", "graph", "relations"],
  ];
  for (const [width, height, device] of [[1440, 1000, "desktop"], [390, 844, "mobile"]]) {
    await page.setViewportSize({ width, height });
    for (const [key, section, name] of views) {
      await area(page, key, section);
      if (section) {
        await expect(page.locator("#content .content-heading h2")).toBeVisible();
        if (["work_items", "objectives"].includes(section)) await expect(page.locator("#item-priority")).toBeVisible();
        else await expect(page.locator("#item-priority")).not.toBeVisible();
        if (section === "objectives") {
          await expect(page.locator("#content [data-entity-id]")).toHaveCount(3);
          await expect(page.locator("#content button button")).toHaveCount(0);
        }
        if (section === "artifacts") await expect(page.locator("#content")).toContainText(data.artifact.name);
        if (section === "decisions") {
          await expect(page.locator("#content")).toContainText(data.decision.title);
          await expect(page.locator("#add-item")).toContainText(/propose decision/i);
          await page.locator("#add-item").click();
          await expect(page.getByLabel("Decision title", { exact: true })).toBeVisible();
          await page.keyboard.press("Escape");
        }
        if (section === "timeline") {
          const before = await page.locator(".timeline-entry").count();
          expect(before).toBeGreaterThan(1);
          // Search is explicitly limited to the loaded page. Task events are
          // present in the first chronological page; later records need paging.
          await page.locator("#item-search").fill("Task");
          await expect.poll(() => page.locator(".timeline-entry").count()).toBeGreaterThan(0);
          await expect.poll(() => page.locator(".timeline-entry").count()).toBeLessThan(before);
          for (const text of await page.locator(".timeline-entry").allTextContents()) expect(text).toMatch(/task/i);
          await page.locator("#item-search").fill("");
          await expect(page.locator(".timeline-entry")).toHaveCount(before);
        }
        if (section === "conclusion_contestations") await expect(page.locator("#content")).toContainText(/assessment contradiction/i);
        if (section === "graph") {
          await expect(page.locator(".relation-card").first()).toBeVisible();
          const before = await page.locator(".relation-card").count();
          await page.locator("#item-search").fill("depends");
          await expect.poll(() => page.locator(".relation-card").count()).toBeGreaterThan(0);
          await expect.poll(() => page.locator(".relation-card").count()).toBeLessThan(before);
          for (const text of await page.locator(".relation-card").allTextContents()) expect(text).toMatch(/depends/i);
          await page.locator("#item-search").fill("");
          await expect(page.locator(".relation-card")).toHaveCount(before);
        }
      }
      await page.evaluate(() => scrollTo(0, 0));
      const documented = device === "desktop"
        ? ["roadmaps", "plan-references", "blockers", "artifacts", "decisions", "contested-conclusions", "relations"]
        : ["objectives", "roadmaps", "issues", "blockers", "evidence", "artifacts", "decisions", "timeline", "contested-conclusions", "relations"];
      // Audit the canonical viewport before documentation deliberately scrolls
      // the content. Sticky-header occlusion is not a smaller target geometry.
      await audit(page);
      if (documented.includes(name)) await capture(page, `01-${device}-${name}`);
    }
    await area(page, "work", "work_items");
    await page.locator('#view-tabs [data-view="board"]').click();
    for (const name of ["Planned", "Ready", "In progress", "Blockers", "Done"])
      await expect(page.getByRole("region", { name, exact: true }).locator(".item").first()).toBeVisible();
    await reflow(page);
    await page.evaluate(() => scrollTo(0, 0));
    await audit(page);
    if (device === "mobile") await capture(page, `01-${device}-board`);
  }
  expect((await data.read("objectives", data.reviewed.id)).lifecycle).toBe("achieved");
  expect(mutations).toEqual([]);
});

test("standalone details, contextual forms and Roadmap revisions preserve their native modal contract", async ({ page, wos }) => {
  test.setTimeout(180000);
  const data = await seedPresentation(wos);
  const published = JSON.stringify(data.roadmap.revisions[0]);
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await wos.login(page);
  await openOutcome(page, data.outcome.title);
  const families = [
    ["plan", "objectives", data.objective, "objective", "Edit objective"],
    ["work", "work_items", data.tasks.ready, "task", "Edit task"],
    ["issues", "issues", data.issue, "issue", "Edit issue"],
    ["issues", "blockers", data.blocker, "blocker", "Edit blocking impact"],
    ["evidence", "evidence", data.evidence[0], "evidence", "Link evidence"],
    ["evidence", "artifacts", data.artifact, "artifact", "Withdraw artifact"],
    ["evidence", "decisions", data.decision, "decision", "Accept decision"],
    ["plan", "roadmaps", data.roadmap, "roadmap", "Edit draft"],
  ];
  for (const [width, height, device] of [[1440, 1000, "desktop"], [390, 844, "mobile"]]) {
    await page.setViewportSize({ width, height });
    for (const [key, section, item, name, action] of families) {
      await rowDetail(page, key, section, item);
      await dialogAccess(page, "#detail-dialog");
      if (name === "roadmap") {
        await expect(page.locator("#detail-content")).toContainText("Refine the shared handoff");
        await expect(page.locator("#detail-content")).toContainText(/Revision 1/i);
      }
      if (device === "desktop" && ["artifact", "decision", "roadmap"].includes(name) || device === "mobile" && name === "blocker")
        await capture(page, `02-${device}-${name}-detail`);
      await page.locator("#detail-actions").getByRole("button", { name: action, exact: true }).click();
      await dialogAccess(page, "#command-dialog");
      await expect(page.locator("#submit-command")).toBeInViewport();
      if (device === "desktop" && name === "roadmap" || device === "mobile" && ["evidence", "decision", "roadmap"].includes(name))
        await capture(page, `02-${device}-${name}-form`);
      await audit(page);
      await closeCommand(page);
    }
    await rowDetail(page, "plan", "roadmaps", data.roadmap);
    await page.locator("#detail-actions").getByRole("button", { name: "Review and publish", exact: true }).click();
    await expect(page.locator(".roadmap-diff")).toContainText("Refine the shared handoff");
    await expect(page.getByLabel("Publication reason", { exact: true })).toBeVisible();
    if (device === "desktop") await capture(page, `02-${device}-roadmap-publication`);
    await page.keyboard.press("Escape");
    await clickWorkspaceTool(page, "view-outcome");
    await expect(page.locator("#detail-content")).toContainText("A person can verify the delivered result");
    if (device === "desktop") await capture(page, `02-${device}-criteria`);
    await page.keyboard.press("Escape");
    await page.locator("#administration").click();
    for (const operation of ["set_grant", "issue_credential", "revoke_credential", "create_namespace"]) {
      await page.locator("#admin-operation").focus();
      await page.locator("#admin-operation").selectOption(operation);
      await dialogAccess(page, "#admin-dialog");
      if (device === "desktop" && operation === "issue_credential" || device === "mobile" && operation === "create_namespace")
        await capture(page, `02-${device}-settings-${operation.replaceAll("_", "-")}`);
    }
    await page.keyboard.press("Escape");
    if (device === "desktop") {
      await page.locator("#developer-tools").click();
      await page.locator("#command-select").focus();
      await page.locator("#command-select").selectOption("propose_decision");
      await dialogAccess(page, "#command-dialog");
      await capture(page, "02-desktop-technical-decision");
      await page.keyboard.press("Escape");
    }
  }
  expect(JSON.stringify((await data.read("roadmaps", data.roadmap.id)).revisions[0])).toBe(published);
  expect(mutations).toEqual([]);
});

test("entry, empty sections, pending reads and recoverable read errors remain usable at 320px", async ({ page, wos }) => {
  for (const [width, height, device] of [[1440, 1000, "desktop"], [390, 844, "mobile"], [320, 740, "minimum"]]) {
    await page.setViewportSize({ width, height });
    await page.goto(`${wos.url}/app/`);
    await expect(page.locator("#token")).toBeVisible();
    await reflow(page);
    await audit(page);
    if (device === "mobile") await capture(page, `03-${device}-login`);
  }
  await wos.login(page);
  await expect(page.locator("#empty")).toBeVisible();
  await page.setViewportSize({ width: 1440, height: 1000 });
  await capture(page, "03-desktop-empty-workspace");
  await page.setViewportSize({ width: 390, height: 844 });
  await capture(page, "03-mobile-empty-workspace");
  await page.locator("#empty-create").click();
  await page.getByLabel("Outcome name", { exact: true }).fill("A verifiable first delivery");
  await page.getByLabel("What does success look like?", { exact: true }).fill("People can resume and review the same work.");
  await save(page);
  await chooseArea(page, "overview");
  await expect(page.getByRole("heading", { name: "Prepare your Outcome", exact: true })).toBeVisible();
  await capture(page, "03-mobile-draft-setup");
  for (const key of ["plan", "work", "issues", "evidence", "activity"]) {
    await area(page, key, key === "activity" ? "conclusion_contestations" : undefined);
    await expect(page.locator(".empty-section").first()).toBeVisible();
    await reflow(page);
  }
  const { value: next } = await wos.outcome("A separate context loads safely");
  await page.reload();
  await expect(page.locator("#title")).toHaveText("A verifiable first delivery");
  let release;
  const pending = new Promise((resolve) => { release = resolve; });
  const pattern = `**/outcomes/${next.id}/continuity?**`;
  await page.route(pattern, async (route) => { await pending; await route.continue(); });
  try {
    const switcher = page.locator("#navigation-toggle");
    if (await switcher.isVisible() && await switcher.getAttribute("aria-expanded") === "false") await switcher.click();
    await page.locator("#outcomes button").filter({ hasText: next.title }).click();
    await expect(page.locator("#workspace-loading")).toBeVisible();
    await capture(page, "03-mobile-loading");
    await page.setViewportSize({ width: 1440, height: 1000 });
  } finally { release(); }
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await page.unroute(pattern);
  await chooseArea(page, "overview");
  await page.route(pattern, (route) => route.fulfill({ status: 503, contentType: "application/json", body: '{"error":{"message":"The state service is temporarily unavailable. Refresh to try again."}}' }));
  await clickWorkspaceTool(page, "refresh");
  await expect(page.locator("#notice")).toHaveClass(/error/);
  await expect(page.locator("#notice")).toContainText("temporarily unavailable");
  await capture(page, "03-desktop-read-error");
  await page.unroute(pattern);
  await clickWorkspaceTool(page, "refresh");
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  await expect(page.locator("#notice")).toContainText("Shared state refreshed");
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await expect(page.locator("#title")).toHaveText(next.title);
  await page.setViewportSize({ width: 320, height: 740 });
  await reflow(page);
  await audit(page);
});

test("native conflict review and uncertain response preserve the exact recoverable intent on mobile", async ({ page, wos }) => {
  const { value, scope } = await wos.outcome("A recoverable shared intention");
  await page.setViewportSize({ width: 390, height: 844 });
  await wos.login(page);
  await openOutcome(page, value.title);
  await intent(page, "Edit outcome");
  await page.getByLabel("Outcome name", { exact: true }).fill("My retained handoff description");
  await wos.command("update_outcome", { scope, expected_version: value.version, title: "Another participant clarified the handoff" });
  await page.locator("#submit-command").click();
  await expect(page.locator("#command-error")).toContainText("A newer version exists");
  await capture(page, "04-mobile-conflict");
  await page.getByRole("button", { name: "Compare with latest state", exact: true }).click();
  await expect(page.locator("#conflict-review")).toContainText("Another participant clarified the handoff");
  await expect(page.locator("#conflict-review")).toContainText("My retained handoff description");
  await capture(page, "04-mobile-conflict-comparison");
  await reflow(page);
  await audit(page);
  await page.getByRole("button", { name: "Use latest version and save my edits", exact: true }).click();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
  await expect(page.locator("#submit-command")).toBeEnabled();
  await expect(page.locator("#title")).toHaveText("My retained handoff description");
  await intent(page, "Edit outcome");
  await page.getByLabel("Outcome name", { exact: true }).fill("A single persisted handoff update");
  const dispatched = [];
  let lose = true;
  await page.route("**/commands/update_outcome", async (route) => {
    dispatched.push({ body: route.request().postData(), key: route.request().headers()["idempotency-key"] });
    if (lose) {
      lose = false;
      await route.fetch();
      await route.fulfill({ status: 503, contentType: "application/json", body: '{"error":{"message":"Response lost after commit"}}' });
    } else await route.continue();
  });
  await page.locator("#submit-command").click();
  await expect(page.locator("#retry-command")).toBeVisible();
  await expect(page.locator("#fields")).toHaveAttribute("inert", "");
  await capture(page, "04-mobile-uncertain-intent");
  await page.locator("#retry-command").click();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
  expect(dispatched).toHaveLength(2);
  expect(dispatched[0]).toEqual(dispatched[1]);
  const persisted = (await wos.api(`/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`)).value;
  expect(persisted.title).toBe("A single persisted handoff update");
  expect(persisted.version).toBe(value.version + 3);
});

test("native Artifact and Decision creation use canonical commands and return to their real section", async ({ page, wos }) => {
  const { value, scope } = await wos.outcome("Documentary records preserve their authorship");
  await wos.login(page);
  await openOutcome(page, value.title);
  await area(page, "evidence", "artifacts");
  await page.locator("#add-item").click();
  await page.getByLabel("Artifact name", { exact: true }).fill("An independently reviewable report");
  await page.getByLabel("Artifact type", { exact: true }).fill("test_report");
  await page.getByLabel("URL", { exact: true }).fill("https://example.test/reports/reviewable-delivery");
  const request = page.waitForRequest((entry) => entry.method() === "POST" && entry.url().endsWith("/commands/register_artifact"));
  await save(page);
  const sent = (await request).postDataJSON().command;
  expect(sent).not.toHaveProperty("description");
  expect(sent).not.toHaveProperty("producer_ref");
  const base = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
  const artifact = (await wos.api(`${base}/artifacts`)).items.find((item) => item.name === "An independently reviewable report");
  expect(artifact).toBeTruthy();
  expect(artifact.lifecycle).toBe("registered");
  expect(artifact.producer_ref.id).toBeTruthy();
  await expect(page.locator('#tabs [data-section="artifacts"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator(`#content [data-entity-id="${artifact.id}"]`)).toContainText(artifact.name);
  await area(page, "evidence", "decisions");
  await page.locator("#add-item").click();
  await page.getByLabel("Decision title", { exact: true }).fill("Require an independent delivery review");
  await page.getByLabel("Proposal", { exact: true }).fill("A reviewer checks the actual result against agreed criteria.");
  await page.getByLabel("Chosen alternative", { exact: true }).fill("Independent review");
  await page.getByLabel("Rationale", { exact: true }).fill("Delivery and acceptance need distinct recorded decisions.");
  await page.getByRole("button", { name: "Add alternatives item", exact: true }).click();
  await page.getByLabel("Alternatives item", { exact: true }).fill("Independent review");
  await save(page);
  const decision = (await wos.api(`${base}/decisions`)).items.find((item) => item.title === "Require an independent delivery review");
  expect(decision).toBeTruthy();
  expect(decision.lifecycle).toBe("proposed");
  await expect(page.locator('#tabs [data-section="decisions"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator(`#content [data-entity-id="${decision.id}"]`)).toContainText(decision.title);
});

test("opening and saving an existing Roadmap Draft retains child phases and immutable publication", async ({ page, wos }) => {
  const data = await seedPresentation(wos);
  const original = data.roadmap;
  const published = JSON.stringify(original.revisions[0]);
  const children = original.draft.nodes.filter((item) => item.parent_node_key);
  expect(children).toHaveLength(2);
  await wos.login(page);
  await openOutcome(page, data.outcome.title);
  await rowDetail(page, "plan", "roadmaps", original);
  await page.locator("#detail-actions").getByRole("button", { name: "Edit draft", exact: true }).click();
  // Canonical API nodes can precede their parent phase in the response. Merely
  // opening the editor must not turn a real child into a root-level reference.
  for (const child of children)
    await expect((await roadmapRow(page, child.title)).getByLabel("Within phase", { exact: true })).toHaveValue(child.parent_node_key);
  await save(page);
  const persisted = await data.read("roadmaps", original.id);
  for (const child of children) {
    const saved = persisted.draft.nodes.find((item) => item.node_key === child.node_key);
    expect(saved.parent_node_key).toBe(child.parent_node_key);
    expect(saved.target_ref).toEqual(child.target_ref);
  }
  expect(persisted.draft.after_links).toEqual(original.draft.after_links);
  expect(JSON.stringify(persisted.revisions[0])).toBe(published);
  await rowDetail(page, "plan", "roadmaps", persisted);
  await page.locator("#detail-actions").getByRole("button", { name: "Edit draft", exact: true }).click();
  const removed = children[0];
  const row = await roadmapRow(page, removed.title);
  await row.getByLabel("Within phase", { exact: true }).focus();
  await row.getByLabel("Within phase", { exact: true }).selectOption("");
  await row.getByRole("button", { name: "Move down", exact: true }).click();
  await expect((await roadmapRow(page, removed.title)).getByLabel("Within phase", { exact: true })).toHaveValue("");
  await save(page);
  const changed = await data.read("roadmaps", original.id);
  for (const child of children) {
    const saved = changed.draft.nodes.find((item) => item.node_key === child.node_key);
    expect(saved.parent_node_key || "").toBe(child.node_key === removed.node_key ? "" : child.parent_node_key);
    expect(saved.target_ref).toEqual(child.target_ref);
  }
  expect(changed.draft.after_links).toEqual(original.draft.after_links);
  expect(JSON.stringify(changed.revisions[0])).toBe(published);
  expect((await wos.api(`${data.base}/relations`)).items.filter((item) => item.relation_type === "depends_on")).toHaveLength(2);
});
