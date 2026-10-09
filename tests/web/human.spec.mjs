import {
  test,
  expect,
  intent,
  save,
  pick,
  openOutcome,
  detail,
} from "./ux-fixture.mjs";
import assert from "node:assert/strict";
import { seedWorkspace } from "./ux-datasets.mjs";

test("F01: save a persisted Draft without command names, UUIDs or Advanced; setup never activates it", async ({
  page,
  wos,
}) => {
  await wos.login(page);
  await page
    .getByRole("button", { name: "Create your first outcome", exact: true })
    .click();
  await page
    .getByLabel("Outcome name", { exact: true })
    .fill("My first Outcome");
  await page
    .getByLabel("What does success look like?", { exact: true })
    .fill("A measurable change");
  await expect(page.locator("#command-select")).not.toBeVisible();
  await expect(page.locator("#fields")).not.toContainText("UUID");
  await save(page);
  await expect(page.locator("#title")).toHaveText("My first Outcome");
  await expect(page.locator("#state")).toHaveText("Draft");
  const pageData = await wos.api(`/namespaces/${wos.namespace}/outcomes`);
  assert.equal(pageData.items.length, 1);
  assert.equal(pageData.items[0].lifecycle, "draft");
});
test("F02/F03: add an Objective and scoped Task with readable list instructions", async ({
  page,
  wos,
}) => {
  await wos.outcome();
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "Add objective");
  await page
    .getByLabel("Objective title", { exact: true })
    .fill("Condition to verify");
  await page.getByLabel("Required for outcome", { exact: true }).check();
  await save(page);
  await intent(page, "New task");
  await page.getByLabel("Task title", { exact: true }).fill("Bounded task");
  await pick(page, "Objective", "Condition");
  await page.getByText("Execution instructions", { exact: true }).click();
  await page
    .getByRole("button", { name: "Add instructions item", exact: true })
    .click();
  await page
    .getByLabel("Instructions item", { exact: true })
    .fill("Read only the scoped files");
  await page
    .getByRole("button", { name: "Add deliverables item", exact: true })
    .click();
  await page
    .getByLabel("Deliverables item", { exact: true })
    .fill("A tested patch");
  await save(page);
  await detail(page, "Tasks", "Bounded task");
  await expect(page.locator("#detail-content")).toContainText(
    "Condition to verify",
  );
});
test("F04: visual Roadmap draft, publish and activation are distinct; replan retains revision one", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  const task = await wos.command("create_work_item", {
    scope,
    title: "Referenced task",
    priority: "normal",
    lifecycle: "todo",
  });
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "Create roadmap");
  await page
    .getByLabel("Roadmap name", { exact: true })
    .fill("Delivery roadmap");
  await save(page);
  async function roadmap() {
    await detail(page, "All roadmaps", "Delivery roadmap");
  }
  await roadmap();
  await page.getByRole("button", { name: "Edit roadmap", exact: true }).click();
  await save(page);
  await roadmap();
  await page.getByRole("button", { name: "Edit draft", exact: true }).click();
  await page.getByRole("button", { name: "Add phase", exact: true }).click();
  await page
    .locator(".roadmap-edit-row")
    .first()
    .getByLabel("Item title", { exact: true })
    .fill("Discover");
  await page
    .getByRole("button", { name: "Reference objective/task", exact: true })
    .click();
  const row = page.locator(".roadmap-edit-row").last();
  await row.getByLabel("Item title", { exact: true }).fill("Build deliverable");
  await row
    .getByLabel("Within phase", { exact: true })
    .selectOption({ label: "Discover" });
  await pick(page, "Reference objective or task", "Referenced");
  await save(page);
  await roadmap();
  await page
    .getByRole("button", { name: "Review and publish", exact: true })
    .click();
  await expect(page.locator("#fields")).toContainText(
    "Added: Build deliverable",
  );
  await page
    .getByLabel("Publication reason", { exact: true })
    .fill("First reviewed plan");
  await save(page);
  const plans = await wos.api(
    `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/roadmaps`,
  );
  let original = (
    await wos.api(
      `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/roadmaps/${plans.items[0].id}`,
    )
  ).value;
  const revision = JSON.stringify(original.revisions[0]);
  assert.equal(original.revisions.length, 1);
  await roadmap();
  await page
    .getByRole("button", { name: "Activate revision", exact: true })
    .click();
  await save(page);
  await roadmap();
  await page.getByRole("button", { name: "Edit roadmap", exact: true }).click();
  await save(page);
  await roadmap();
  await page.getByRole("button", { name: "Edit draft", exact: true }).click();
  await page
    .locator(".roadmap-edit-row")
    .last()
    .getByRole("button", { name: "Remove from draft", exact: true })
    .click();
  await save(page);
  await roadmap();
  await page
    .getByRole("button", { name: "Review and publish", exact: true })
    .click();
  await page
    .getByLabel("Publication reason", { exact: true })
    .fill("Replan, remove reference only");
  await save(page);
  original = (
    await wos.api(
      `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/roadmaps/${plans.items[0].id}`,
    )
  ).value;
  assert.equal(JSON.stringify(original.revisions[0]), revision);
  assert.equal(original.revisions.length, 2);
  assert.equal(
    (
      await wos.api(
        `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/work-items/${task.id}`,
      )
    ).value.lifecycle,
    "todo",
  );
});
test("F05: register proof then assess its exact criterion revision; Evidence alone does not certify", async ({
  page,
  wos,
}) => {
  await wos.outcome();
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "Add success criterion");
  await page
    .getByLabel("Criterion title", { exact: true })
    .fill("Delivery independently checked");
  await page
    .getByLabel("How will it be verified?", { exact: true })
    .selectOption("evidence_review");
  await save(page);
  await intent(page, "Register evidence");
  await page
    .getByLabel("Description", { exact: true })
    .fill("Successful test run");
  await page.getByLabel("Source provider", { exact: true }).fill("ci");
  await page
    .getByLabel("Source URL", { exact: true })
    .fill("https://example.test/verified-run");
  await save(page);
  await expect(page.locator("#state")).toHaveText("Draft");
  await page.locator("#view-outcome").click();
  await page
    .getByRole("button")
    .filter({ hasText: "Delivery independently checked" })
    .click();
  await pick(page, "Evidence used", "Successful test");
  await page
    .getByLabel("Rationale", { exact: true })
    .fill("The observed run covers the criterion");
  await save(page);
  await expect(page.locator("#state")).toHaveText("Draft");
});
test("F06: reporting an Issue and blocking work are explicit; resolving the Issue does not release impact", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  await wos.command("create_work_item", {
    scope,
    title: "Blocked delivery",
    priority: "normal",
    lifecycle: "todo",
  });
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "Report issue and block work");
  await page
    .getByLabel("Issue title", { exact: true })
    .fill("Access unavailable");
  await page
    .getByLabel("Blocking impact", { exact: true })
    .fill("Wait for access");
  await pick(page, "Blocked item", "Blocked delivery");
  await save(page);
  await detail(page, "Issues", "Access unavailable");
  await page
    .getByRole("button", { name: "Resolve issue", exact: true })
    .click();
  await page
    .getByLabel("Resolution summary", { exact: true })
    .fill("Credentials restored");
  await save(page);
  await detail(page, "Blockers", "Wait for access");
  await expect(page.locator("#detail-content")).toContainText("Active");
  await page
    .getByRole("button", { name: "Release blocker", exact: true })
    .click();
  await page
    .getByLabel("Resolution summary", { exact: true })
    .fill("Verified work can resume");
  await page.getByLabel("Confirm work can proceed", { exact: true }).check();
  await save(page);
  const blockers = await wos.api(
    `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/blockers`,
  );
  assert.equal(blockers.items[0].lifecycle, "resolved");
});
test("F08: conflict retains local edits, compares latest state and requires a new confirmed intent", async ({
  page,
  wos,
}) => {
  const { value, scope } = await wos.outcome();
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "Edit outcome");
  await page
    .getByLabel("Outcome name", { exact: true })
    .fill("My retained edits");
  await wos.command("update_outcome", {
    scope,
    expected_version: value.version,
    title: "Another participant changed it",
  });
  await page.locator("#submit-command").click();
  await expect(page.locator("#command-error")).toContainText(
    "A newer version exists",
  );
  await expect(page.getByLabel("Outcome name", { exact: true })).toHaveValue(
    "My retained edits",
  );
  await page
    .getByRole("button", { name: "Compare with latest state", exact: true })
    .click();
  await expect(page.locator("#conflict-review")).toContainText(
    "Another participant changed it",
  );
  await expect(page.locator("#conflict-review")).toContainText(
    "My retained edits",
  );
  await page
    .getByRole("button", {
      name: "Use latest version and save my edits",
      exact: true,
    })
    .click();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
  await expect(page.locator("#title")).toHaveText("My retained edits");
  assert.equal(
    (
      await wos.api(
        `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`,
      )
    ).value.version,
    value.version + 2,
  );
});
test("F08: lost response retries the committed command with identical bytes and key", async ({
  page,
  wos,
}) => {
  const { value, scope } = await wos.outcome();
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "Edit outcome");
  await page
    .getByLabel("Outcome name", { exact: true })
    .fill("Saved exactly once");
  const dispatched = [];
  let lose = true;
  await page.route("**/commands/update_outcome", async (route) => {
    dispatched.push({
      key: route.request().headers()["idempotency-key"],
      body: route.request().postData(),
    });
    if (lose) {
      lose = false;
      await route.fetch();
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: '{"error":{"message":"Response lost"}}',
      });
    } else await route.continue();
  });
  await page.locator("#submit-command").click();
  await expect(
    page.getByRole("button", { name: "Retry the same intent", exact: true }),
  ).toBeVisible();
  await expect(page.locator("#fields")).toHaveAttribute("inert", "");
  await page
    .getByRole("button", { name: "Retry the same intent", exact: true })
    .click();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
  await expect(page.locator("#title")).toHaveText("Saved exactly once");
  assert.equal(dispatched.length, 2);
  assert.deepEqual(dispatched[0], dispatched[1]);
  assert.equal(
    (
      await wos.api(
        `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`,
      )
    ).value.version,
    value.version + 1,
  );
});
test("reference picker searches 1,001 authorized Tasks and Evidence beyond the first page with duplicate context", async ({
  page,
  wos,
}) => {
  test.setTimeout(180000);
  const seed = await seedWorkspace({
    url: wos.url,
    token: wos.token,
    namespace: wos.namespace,
    size: 1001,
  });
  await wos.login(page);
  await openOutcome(page, "UX acceptance");
  await intent(page, "New task");
  await page
    .getByLabel("Task title", { exact: true })
    .fill("Task with distant context");
  await page.getByText("Execution instructions", { exact: true }).click();
  await pick(page, "Context references", "Off-page acceptance task");
  await pick(page, "Context references", "Off-page acceptance evidence");
  await expect(page.locator(".reference-selections").last()).toContainText(
    "Off-page acceptance task",
  );
  await expect(page.locator(".reference-selections").last()).toContainText(
    "Off-page acceptance evidence",
  );
  await save(page);
  const refs = await wos.api(
    `/namespaces/${wos.namespace}/outcomes/${seed.outcome.id}/references?kind=work_item&query=Task%20with%20distant%20context`,
  );
  const task = (
    await wos.api(
      `/namespaces/${wos.namespace}/outcomes/${seed.outcome.id}/work-items/${refs.items[0].ref.id}`,
    )
  ).value;
  assert.equal(task.execution_spec.context_refs.length, 2);
});
test("reference picker keyboard navigation selects duplicate titles using IDs and parent context", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  for (let i = 0; i < 2; i++)
    await wos.command("create_objective", {
      scope,
      title: "Same title",
      priority: "normal",
      required_for_outcome: false,
    });
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "New task");
  await page
    .getByLabel("Task title", { exact: true })
    .fill("Keyboard selection");
  const input = page.getByRole("combobox", { name: "Objective", exact: true });
  await input.focus();
  await input.fill("Same title");
  await expect(page.locator(".reference-picker [role=option]")).toHaveCount(2);
  await input.press("ArrowDown");
  await expect(input).toHaveAttribute("aria-activedescendant", /option-0$/);
  await input.press("Enter");
  await expect(input).toHaveAttribute("aria-expanded", "false");
  await save(page);
});
test("stale reference cursor asks for reload and retains selections without silent replacement", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  for (let i = 0; i < 27; i++)
    await wos.command("create_objective", {
      scope,
      title: `Objective ${i}`,
      priority: "normal",
      required_for_outcome: false,
    });
  await wos.login(page);
  await openOutcome(page);
  await intent(page, "New task");
  await page
    .getByLabel("Task title", { exact: true })
    .fill("Stale cursor draft");
  await page.getByText("Execution instructions", { exact: true }).click();
  const input = page.getByRole("combobox", {
    name: "Context references",
    exact: true,
  });
  await input.fill("Objective");
  const root = page.locator(".reference-picker").filter({ has: input });
  await expect(root.getByRole("option")).toHaveCount(25);
  await root.getByRole("option").first().click();
  await expect(root.getByRole("option")).toHaveCount(25);
  await wos.command("create_objective", {
    scope,
    title: "Newly inserted",
    priority: "normal",
    required_for_outcome: false,
  });
  await root.getByRole("button", { name: "More results", exact: true }).click();
  await expect(
    root.getByRole("button", { name: "Reload search", exact: true }),
  ).toBeVisible();
  await expect(root.locator(".reference-selections")).not.toBeEmpty();
  await root
    .getByRole("button", { name: "Reload search", exact: true })
    .click();
  await expect(root.getByRole("option")).toHaveCount(25);
  await expect(root.locator(".reference-selections")).not.toBeEmpty();
});
