import {
  test, expect, openOutcome, outcomeActions, chooseArea, clickWorkspaceTool, pick,
} from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";

const panel = (page) => page.locator("#canvas-details");
const menu = (page) => panel(page).locator("#action-options");

async function openBoard(page, wos, data) {
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await expect(page.locator("#operational-canvas")).toBeVisible();
}

async function selectEvidence(page, evidence) {
  await page.getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(evidence.description);
  await page.getByRole("option").filter({ hasText: evidence.description }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText("Evidence");
  // Entity expansion finishes by recording the deep link. The heading appears
  // earlier and cannot alone establish that selection has settled.
  await page.waitForURL((url) => url.searchParams.get("item") === `evidence/${evidence.id}`);
}

async function wholeControl(locator) {
  await locator.scrollIntoViewIfNeeded();
  await expect(locator).toBeVisible();
  expect(await locator.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    const hit = document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2);
    return rect.height >= 44 && rect.width >= 44 && rect.left >= 0 &&
      rect.right <= innerWidth && rect.top >= 0 && rect.bottom <= innerHeight &&
      element.contains(hit);
  }), "A complete 44 px action receives input inside the visible viewport").toBe(true);
}

for (const viewport of [{ width: 2560, height: 1440 }, { width: 390, height: 844 }, { width: 360, height: 640 }]) {
  test(`Evidence actions remain compact and contextual at ${viewport.width} x ${viewport.height}`, async ({ page, wos }) => {
    await page.setViewportSize(viewport);
    const data = await seedCanvas(wos);
    await openBoard(page, wos, data);
    await selectEvidence(page, data.evidence[0]);
    const actionTrigger = panel(page).locator("#entity-actions");
    await wholeControl(actionTrigger);
    await actionTrigger.click();
    await expect(panel(page).locator(".canvas-inspector-actions")).toBeVisible();
    await expect(menu(page).locator(".action-context")).toContainText("Evidence");
    const retract = menu(page).getByRole("button", { name: /^Retract evidence\b/ });
    const link = menu(page).getByRole("button", { name: /^Link evidence\b/ });
    for (const button of [retract, link]) {
      await wholeControl(button);
      expect(await button.evaluate((element) => element.getBoundingClientRect().height),
        "Two actions must not grow into page-height tiles").toBeLessThanOrEqual(120);
    }
    const bounds = await menu(page).evaluate((element) => {
      const buttons = Array.from(element.querySelectorAll("button"));
      return buttons.slice(1).map((button, index) =>
        button.getBoundingClientRect().top - buttons[index].getBoundingClientRect().bottom,
      );
    });
    expect(Math.max(0, ...bounds), "Adjacent action rows remain grouped rather than filling the inspector").toBeLessThanOrEqual(80);
    await expect(menu(page).getByRole("button", { name: /^New task\b/ })).toHaveCount(0);
    await retract.focus();
    await page.keyboard.press("Tab");
    await expect(link).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(panel(page).locator(".canvas-inspector-form")).toBeVisible();
    await expect(panel(page).locator("#command-title")).toHaveText("Link evidence");
    await page.keyboard.press("Escape");
    await expect(panel(page).locator(".canvas-inspector-detail")).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBe(0);
    expect(await page.evaluate(() => document.documentElement.scrollHeight - innerHeight)).toBe(0);
  });
}

test("Outcome actions group real available commands and keep the final actions reachable", async ({ page, wos }) => {
  await page.setViewportSize({ width: 360, height: 640 });
  const data = await seedCanvas(wos);
  await openBoard(page, wos, data);
  await page.locator("#actions").click();
  await expect(menu(page).locator(".action-context")).toContainText(data.outcome.title);
  expect(await menu(page).locator(".action-group").count()).toBeGreaterThan(1);
  const groups = await menu(page).locator(".action-group > h3").allTextContents();
  expect(groups.every((label) => label.trim().length > 0)).toBe(true);
  const buttons = menu(page).getByRole("button");
  expect(await buttons.count()).toBeGreaterThan(3);
  const first = buttons.first();
  await wholeControl(first);
  await first.focus();
  await page.keyboard.press("Tab");
  await expect(buttons.nth(1)).toBeFocused();
  const advanced = menu(page).locator(":scope > details > summary");
  if (await advanced.count()) {
    await wholeControl(advanced);
    await advanced.click();
  }
  await wholeControl(buttons.last());
  expect(await buttons.last().evaluate((element) => element.getBoundingClientRect().height)).toBeLessThanOrEqual(120);
  await page.keyboard.press("Escape");
  await expect(panel(page)).toBeHidden();
  await expect(page.locator("#actions")).toBeFocused();
});

test("Dismissing global Outcome actions preserves the selected Evidence and its command scope", async ({ page, wos }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const data = await seedCanvas(wos);
  await openBoard(page, wos, data);
  await selectEvidence(page, data.evidence[0]);
  const evidenceRoute = page.url();
  await page.locator("#actions").click();
  await expect(menu(page).locator(".action-context")).toContainText(`Outcome · ${data.outcome.title}`);
  // The unselected Outcome action remains generic; the specific lifecycle is
  // chosen when a work item is created from this context.
  await expect(menu(page).getByRole("button", { name: "New task", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(panel(page).locator(".canvas-inspector-detail")).toBeVisible();
  await expect(panel(page).locator("#detail-title")).toHaveText("Evidence");
  await expect(panel(page).locator("#detail-content .detail-description")).toHaveText(data.evidence[0].description);
  expect(page.url()).toBe(evidenceRoute);
  await panel(page).locator("#entity-actions").click();
  await expect(menu(page).locator(".action-context")).toContainText("Evidence");
  expect(await menu(page).getByRole("button").evaluateAll((elements) => elements
    .map((element) => element.getAttribute("aria-label")).sort(),
  )).toEqual(["Add artifact here", "Link evidence", "Propose decision here", "Record an update", "Report issue", "Retract evidence"]);
  await page.keyboard.press("Escape");

  await page.locator("#actions").click();
  await menu(page).getByRole("button", { name: "New task", exact: true }).click();
  await expect(panel(page).locator(".canvas-inspector-form")).toBeVisible();
  await expect(panel(page).locator("#command-title")).toHaveText("New task");
  await expect(panel(page).locator(".workspace-placement-summary")).toContainText("Choose an Objective");
  await panel(page).getByRole("button", { name: "Choose Objective", exact: true }).click();
  await pick(page, "New placement", data.objective.title);
  const title = "A Task created from the global Outcome action";
  await panel(page).getByLabel("Task title", { exact: true }).fill(title);
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator(".canvas-inspector-form")).toBeHidden();
  await expect(page.locator("#submit-command")).toBeEnabled();
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  const tasks = await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items`);
  const created = tasks.items.find((record) => record.title === title);
  expect(created).toBeTruthy();
  expect(created.scope).toEqual(data.scope);
  expect(created.objective_id).toBe(data.objective.id);
});

for (const viewport of [{ width: 1280, height: 900 }, { width: 390, height: 844 }]) {
  test(`Native action dialog keeps grouped commands readable at ${viewport.width} x ${viewport.height}`, async ({ page, wos }) => {
    await page.setViewportSize(viewport);
    const data = await seedCanvas(wos);
    await wos.login(page);
    await openOutcome(page, data.outcome.title);
    await outcomeActions(page);
    const dialog = page.locator("#action-dialog");
    const options = dialog.locator("#action-options");
    await expect(options.locator(".action-context")).toContainText(data.outcome.title);
    const groups = options.locator(".action-group");
    expect(await groups.count()).toBeGreaterThan(1);
    const columns = await options.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").length);
    expect(columns).toBe(viewport.width > 760 ? 2 : 1);
    const context = await options.locator(".action-context").boundingBox();
    const firstGroup = await groups.first().boundingBox();
    expect(context.width).toBeGreaterThanOrEqual(firstGroup.width);
    const action = options.getByRole("button", { name: "New task · Backlog", exact: true });
    await wholeControl(action);
    expect(await action.evaluate((element) => element.getBoundingClientRect().height)).toBeLessThanOrEqual(150);
    await action.click();
    await expect(page.locator("#action-dialog")).toBeHidden();
    await expect(page.locator("#command-dialog")).toBeVisible();
    await expect(page.locator("#command-title")).toHaveText("New task");
    await wholeControl(page.locator("#submit-command"));
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBe(0);
  });
}

test("Partial map coverage stays in the footer and other feedback never covers inspection actions", async ({ page, wos, context }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  await context.grantPermissions(["clipboard-write", "clipboard-read"], { origin: wos.url });
  const data = await seedCanvas(wos, { extraTasks: 35 });
  await openBoard(page, wos, data);
  await expect(page.locator(".canvas-status")).toContainText("Partial map");
  await expect(page.getByRole("button", { name: "Load more map items", exact: true })).toBeVisible();
  await expect(page.locator("#notice")).not.toContainText("Partial view");
  await selectEvidence(page, data.evidence[0]);
  await clickWorkspaceTool(page, "copy-workspace-link");
  const notice = page.locator("#canvas-feedback > #notice");
  await expect(notice).toContainText("copied");
  expect(await notice.evaluate((element) => getComputedStyle(element).position)).toBe("static");
  const geometry = await page.evaluate(() => {
    const read = (selector) => {
      const rect = document.querySelector(selector).getBoundingClientRect();
      return { top: rect.top, bottom: rect.bottom };
    };
    return { stage: read(".canvas-stage"), feedback: read("#canvas-feedback"), notice: read("#notice") };
  });
  expect(geometry.feedback.top).toBeGreaterThanOrEqual(geometry.stage.bottom);
  expect(geometry.notice.bottom).toBeLessThanOrEqual(768);
  await wholeControl(panel(page).locator("#entity-actions"));
  await panel(page).locator("#entity-actions").click();
  await wholeControl(menu(page).getByRole("button", { name: /^Link evidence\b/ }));
  await page.keyboard.press("Escape");
  await chooseArea(page, "overview");
  await expect(page.locator(".workspace-main > #notice")).toHaveCount(1);
  await expect(page.locator("#canvas-feedback > #notice")).toHaveCount(0);
  await clickWorkspaceTool(page, "refresh");
  await expect(page.locator("#notice")).toContainText("Partial view");
  await chooseArea(page, "canvas");
  await expect(page.locator("#canvas-feedback > #notice")).toBeEmpty();
  await expect(page.locator("#canvas-feedback")).toBeHidden();
  await expect(page.locator(".canvas-status")).toContainText("Partial map");
});
