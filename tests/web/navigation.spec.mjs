import { test, expect, openOutcome } from "./ux-fixture.mjs";
test("six areas and persisted Draft checklist do not automatically activate", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  await wos.login(page);
  await openOutcome(page);
  await expect(page.locator("#area-tabs button")).toHaveCount(6);
  await expect(
    page.getByRole("heading", { name: "Prepare your Outcome" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Activate outcome", exact: true }),
  ).toBeVisible();
  await page
    .locator("#area-tabs")
    .getByRole("button", { name: "Work", exact: true })
    .click();
  await expect(page.getByLabel("Work status", { exact: true })).toBeVisible();
  await expect(page.locator("#view-tabs button")).toHaveCount(2);
  expect(
    (await wos.api(`/namespaces/${wos.namespace}/outcomes/${scope.outcome_id}`))
      .value.lifecycle,
  ).toBe("draft");
});
test("deep links restore area and scoped item through reload, back and forward", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  const objective = await wos.command("create_objective", {
    scope,
    title: "Deep-linked Objective",
    priority: "normal",
  });
  await wos.login(page);
  await openOutcome(page);
  await page
    .locator("#area-tabs")
    .getByRole("button", { name: "Plan", exact: true })
    .click();
  await page
    .locator("#content")
    .getByRole("button")
    .filter({ hasText: "Deep-linked Objective" })
    .click();
  await expect(page).toHaveURL(/item=objective/);
  const url = page.url();
  await page.reload();
  await expect(page.locator("#detail-title")).toHaveText(objective.title);
  await page.keyboard.press("Escape");
  await expect(page).not.toHaveURL(/item=/);
  await page.goBack();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await page.goForward();
  await expect(page.locator("#detail-dialog")).not.toBeVisible();
  expect(url).not.toContain(wos.token);
});
test("all Task search and priority filter find an off-page item in 1,001 records", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  for (let i = 0; i < 1001; i++)
    await wos.command("create_work_item", {
      scope,
      title: i === 1000 ? "Unique remote high-priority Task" : "Other Task",
      priority: i === 1000 ? "high" : "normal",
    });
  await wos.login(page);
  await openOutcome(page);
  await page
    .locator("#area-tabs")
    .getByRole("button", { name: "Work", exact: true })
    .click();
  await page.getByLabel("Search items", { exact: true }).fill("Unique remote");
  await page.getByLabel("Priority", { exact: true }).selectOption("high");
  await expect(page.locator(".list-row")).toHaveCount(1);
  await expect(page.locator("#content")).toContainText(
    "Unique remote high-priority Task",
  );
  await page.getByLabel("Priority", { exact: true }).selectOption("low");
  await expect(page.locator(".list-row")).toHaveCount(0);
});

test("queued search cannot discard a page after explicit status navigation", async ({
  page,
  wos,
}) => {
  const { scope } = await wos.outcome();
  for (let i = 0; i < 32; i++)
    await wos.command("create_work_item", {
      scope,
      title: `Pagination task ${String(i).padStart(2, "0")}`,
      priority: "normal",
    });
  await wos.login(page);
  await openOutcome(page);
  await page
    .locator("#area-tabs")
    .getByRole("button", { name: "Work", exact: true })
    .click();
  await expect(page.locator(".list-row")).toHaveCount(25);
  await page.clock.install();
  await page.clock.pauseAt(new Date());
  await page
    .getByLabel("Search items", { exact: true })
    .fill("queued obsolete query");
  await page
    .getByLabel("Work status", { exact: true })
    .selectOption("work_items");
  await expect(page.getByLabel("Search items", { exact: true })).toHaveValue(
    "",
  );
  await page.getByRole("button", { name: "Load more", exact: true }).click();
  await expect(page.locator(".list-row")).toHaveCount(32);
  // A reference request after this point would be a stale first-page search.
  let restarted = 0;
  await page.route("**/references?**", async (route) => {
    restarted++;
    await route.continue();
  });
  await page.clock.runFor(250);
  await page.evaluate(async () => {
    const r = await fetch("/livez");
    if (!r.ok) throw Error("health fence failed");
  });
  expect(restarted).toBe(0);
  await expect(page.locator(".list-row")).toHaveCount(32);
});
