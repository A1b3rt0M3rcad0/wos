import { test, expect, openOutcome } from "./ux-fixture.mjs";

const panel = (page) => page.locator("#canvas-details");
const base = (scope) => `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;

async function select(page, title) {
  await page.getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(title);
  await page.getByRole("option").filter({ hasText: title }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(title);
}

async function taskForm(page) {
  const direct = panel(page).locator('[data-action-id="create_work_item"]').filter({ visible: true });
  if (await direct.count()) await direct.first().click();
  else {
    await panel(page).locator("#entity-actions").click();
    await page.locator('#action-options [data-action-id="create_work_item"]').click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
  return page.getByLabel("Task title", { exact: true });
}

async function created(page, trigger) {
  const response = page.waitForResponse((value) => value.request().method() === "POST"
    && value.url().endsWith("/commands/create_work_item"));
  await trigger();
  const result = await response;
  const data = await result.json();
  expect(result.ok(), JSON.stringify(data)).toBe(true);
  return data.value;
}

test("quick Task Enter repeats the same placement, Save inspects the result, and Escape never creates a draft", async ({ page, wos }) => {
  const { scope, value: outcome } = await wos.outcome("Plan several independent steps");
  const objective = await wos.command("create_objective", {
    scope, title: "Every step has an explicit home", priority: "normal", required_for_outcome: true,
  });
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/"))
      mutations.push({ name: request.url().split("/").at(-1), payload: request.postDataJSON().command });
  });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await select(page, objective.title);
  let title = await taskForm(page);
  await title.fill("Check the documented prerequisites");
  const first = await created(page, () => title.press("Enter"));
  await expect(page.locator("#command-form")).toBeVisible();
  title = page.getByLabel("Task title", { exact: true });
  await expect(title).toHaveValue("");
  await expect(title).toBeFocused();
  await expect(page.locator("#fields")).toContainText(objective.title);
  await title.fill("Record a usable installation transcript");
  const second = await created(page, () => title.press("Enter"));
  await expect(title).toHaveValue("");
  await expect(title).toBeFocused();
  await title.fill("Prepare an independent review");
  const third = await created(page, () => page.locator("#submit-command").click());
  await expect(page.locator("#command-form")).not.toBeVisible();
  await expect(panel(page).locator("#detail-title")).toHaveText(third.title);

  await select(page, objective.title);
  title = await taskForm(page);
  await title.fill("This unsaved Task must not exist");
  await title.press("Escape");
  await expect(page.locator("#command-form")).not.toBeVisible();
  await expect(panel(page).locator("#detail-title")).toHaveText(objective.title);
  const list = await wos.api(`${base(scope)}/work-items`);
  const records = list.items || list.value?.items || list.value || [];
  expect(records.map((item) => item.title).sort()).toEqual([first, second, third].map((item) => item.title).sort());
  expect(mutations.map((item) => item.name)).toEqual(["create_work_item", "create_work_item", "create_work_item"]);
  for (const { payload } of mutations) expect(payload).toMatchObject({ objective_id: objective.id, lifecycle: "backlog" });
  for (const task of [first, second, third]) {
    expect(task.objective_id).toBe(objective.id);
    expect(task.lifecycle).toBe("backlog");
    expect(task.current_contract_id).toBeUndefined();
    expect(task.current_lease).toBeUndefined();
  }
});

test("a criterion deep link reloads the typed subject and retains its real Task owner", async ({ page, wos }) => {
  const { scope, value: outcome } = await wos.outcome("Restore the precise review context");
  const task = await wos.command("create_work_item", { scope, title: "Review the documented startup", lifecycle: "backlog", priority: "normal" });
  const owner = { ...scope, kind: "work_item", id: task.id };
  const criterion = await wos.command("add_criterion", { owner, expected_version: task.version,
    title: "The documented command starts a usable workspace", required: true, verification_mode: "evidence_review" });
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await select(page, task.title);
  await panel(page).locator(`[data-criterion-id="${criterion.id}"] [data-action-id="open_criterion"]`).click();
  await expect(panel(page).locator("#detail-title")).toHaveText(criterion.title);
  await page.waitForURL((url) => url.searchParams.get("subject") === "criterion"
    && url.searchParams.get("criterion") === criterion.id
    && url.searchParams.get("item") === `work_item/${task.id}`);
  const link = page.url();
  await page.reload();
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await expect(panel(page).locator("#detail-title")).toHaveText(criterion.title);
  await expect(panel(page)).toContainText("Not assessed");
  expect(page.url()).toBe(link);
  await panel(page).locator('.workspace-path [data-reference-kind="work_item"]').click();
  await expect(panel(page).locator("#detail-title")).toHaveText(task.title);
  await page.waitForURL((url) => !url.searchParams.has("subject") && !url.searchParams.has("criterion"));
  expect(mutations).toEqual([]);
});
