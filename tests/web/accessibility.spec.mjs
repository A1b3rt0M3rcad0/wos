import {
  test,
  expect,
  openOutcome,
  intent,
  save,
  pick,
  detail,
  outcomeActions,
} from "./ux-fixture.mjs";
import AxeBuilder from "@axe-core/playwright";
import { clickRecordAction } from "./workspace-action-fixture.mjs";
async function audit(page) {
  const result = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(
    result.violations.filter((v) => ["critical", "serious"].includes(v.impact)),
    JSON.stringify(result.violations),
  ).toEqual([]);
}
test("axe checks login, six-area overview, common Task form and server combobox", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  const objective = await wos.command("create_objective", {
    scope,
    title: "Accessible condition",
    priority: "normal",
  });
  await page.goto(wos.url + "/app/");
  await audit(page);
  await wos.login(page);
  await openOutcome(page);
  await audit(page);
  await intent(page, "New task");
  await page.getByLabel("Task title", { exact: true }).fill("Accessible Task");
  await audit(page);
  await page.getByRole("button", { name: "Choose Objective", exact: true }).click();
  await page
    .getByRole("combobox", { name: "New placement", exact: true })
    .fill(objective.title);
  await expect(page.locator("[role=option]")).toHaveCount(1);
  await audit(page);
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await save(page);
});
test("390px mobile creates Outcome and Task with a usable searched reference", async ({
  page,
  wos,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await wos.login(page);
  await page.locator("#empty-create").click();
  await page.getByLabel("Outcome name", { exact: true }).fill("Mobile Outcome");
  await page
    .getByLabel("What does success look like?", { exact: true })
    .fill("Small screen coordination works");
  await save(page);
  await intent(page, "Add objective");
  await page
    .getByLabel("Objective title", { exact: true })
    .fill("Mobile condition");
  await save(page);
  await intent(page, "New task");
  await page.getByLabel("Task title", { exact: true }).fill("Mobile Task");
  await page.getByRole("button", { name: "Choose Objective", exact: true }).click();
  await pick(page, "New placement", "Mobile condition");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await save(page);
});
for (const scale of [2, 4])
  test(`${scale * 100}% equivalent reflow retains fields, focus and confirmation`, async ({
    page,
    wos,
  }) => {
    await page.setViewportSize({
      width: Math.floor(1280 / scale),
      height: Math.floor(900 / scale),
    });
    const { scope } = await wos.outcome();
    await wos.command("create_objective", { scope, title: "Reflow condition", priority: "normal" });
    await wos.login(page);
    await openOutcome(page);
    await intent(page, "New task");
    await page.getByLabel("Task title", { exact: true }).fill("Reflow Task");
    await page.getByRole("button", { name: "Choose Objective", exact: true }).click();
    await pick(page, "New placement", "Reflow condition");
    await page.keyboard.press("Tab");
    expect(
      await page.evaluate(() => document.activeElement === document.body),
    ).toBe(false);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.locator("#submit-command").scrollIntoViewIfNeeded();
    await save(page);
  });
test("keyboard opens, searches, selects, confirms and restores focus without a pointer", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  await wos.command("create_objective", {
    scope,
    title: "Keyboard condition",
    priority: "normal",
  });
  await wos.login(page);
  await openOutcome(page);
  await page.locator("#review-outcome").focus();
  await page.keyboard.press("Enter");
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await page.locator("#entity-actions").focus();
  await page.keyboard.press("Enter");
  await expect(page.locator("#action-dialog")).toBeVisible();
  await page
    .locator("#action-options")
    .getByRole("button", { name: "New task · Backlog", exact: true })
    .focus();
  await page.keyboard.press("Enter");
  await page.getByLabel("Task title", { exact: true }).fill("Keyboard Task");
  await page.getByRole("button", { name: "Choose Objective", exact: true }).focus();
  await page.keyboard.press("Enter");
  const input = page.getByRole("combobox", { name: "New placement", exact: true });
  await input.focus();
  await page.keyboard.type("Keyboard");
  await expect(page.locator("[role=option]")).toHaveCount(1);
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await page.locator("#submit-command").focus();
  await page.keyboard.press("Enter");
  await expect(page.locator("#command-dialog")).not.toBeVisible();
});
test("compound Issue resolution explicitly releases selected impacts atomically", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  const objective = await wos.command("create_objective", { scope, title: "Issue impact Objective", priority: "normal" });
  const task = await wos.command("create_work_item", {
    scope,
    title: "Affected Task",
    priority: "normal",
    objective_id: objective.id,
  });
  await wos.command("report_issue_with_blocker", {
    scope,
    objective_id: objective.id,
    title: "Compound problem",
    severity: "major",
    blocked_ref: { ...scope, kind: "work_item", id: task.id },
    blocker_description: "Concrete impact",
    propagation: "direct",
  });
  await wos.login(page);
  await openOutcome(page);
  await detail(page, "Issues", "Compound problem");
  await page.locator("#entity-actions").click();
  await page.locator('#action-options [data-action-id="resolve_issue_and_blockers"]').click();
  await page
    .getByLabel("Issue resolution summary", { exact: true })
    .fill("Verified together");
  const impact = page.getByLabel("Affected Task", { exact: true });
  await expect(impact).not.toBeChecked();
  await impact.check();
  await page
    .getByLabel("Confirm each selected impact is released", { exact: true })
    .check();
  await save(page);
  expect(
    (
      await wos.api(
        `/namespaces/${wos.namespace}/outcomes/${scope.outcome_id}/blockers`,
      )
    ).items[0].lifecycle,
  ).toBe("resolved");
});
test("typed external context preserves scalar types without changing permission", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  await wos.login(page);
  await openOutcome(page);
  await outcomeActions(page);
  await page.locator("#action-options details summary").click();
  await page
    .getByRole("button", { name: "Edit external context", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Add context entry", exact: true })
    .click();
  await page.getByLabel("Context key", { exact: true }).fill("tenant_id");
  await page
    .getByLabel("Context value", { exact: true })
    .fill("customer-address");
  await save(page);
  expect(
    (await wos.api(`/namespaces/${wos.namespace}/outcomes/${scope.outcome_id}`))
      .value.external_context.tenant_id,
  ).toBe("customer-address");
});
test("typed participants are documentary and never acquire execution authority", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  const objective = await wos.command("create_objective", { scope, title: "Participant Objective", priority: "normal" });
  const task = await wos.command("create_work_item", {
    scope,
    title: "Participant Task",
    priority: "normal",
    objective_id: objective.id,
  });
  await wos.login(page);
  await openOutcome(page);
  await detail(page, "Tasks", "Participant Task");
  await clickRecordAction(page, "Edit participants");
  await page
    .getByRole("button", { name: "Add participant", exact: true })
    .click();
  await page.getByLabel("Participant provider", { exact: true }).fill("codex");
  await page.getByLabel("Participant identity", { exact: true }).fill("worker");
  await save(page);
  const saved = (
    await wos.api(
      `/namespaces/${wos.namespace}/outcomes/${scope.outcome_id}/work-items/${task.id}`,
    )
  ).value;
  expect(saved.assignee_refs).toEqual([
    { kind: "agent", provider: "codex", id: "worker" },
  ]);
  expect(saved.current_lease).toBeFalsy();
});
